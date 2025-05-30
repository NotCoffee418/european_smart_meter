#!/usr/bin/env python3

import asyncio
import json
import os
import re
import time
from datetime import datetime
from typing import Optional, Dict, Any
from dataclasses import dataclass, asdict
from contextlib import asynccontextmanager

import serial
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.responses import JSONResponse
import uvicorn
import crcmod


@dataclass
class MeterReading:
    timestamp: str
    # Consumption (current)
    current_consumption_kw: float = 0.0
    current_production_kw: float = 0.0
    l1_consumption_kw: float = 0.0
    l2_consumption_kw: float = 0.0
    l3_consumption_kw: float = 0.0
    l1_production_kw: float = 0.0
    l2_production_kw: float = 0.0
    l3_production_kw: float = 0.0
    
    # Totals
    total_consumption_day_kwh: float = 0.0
    total_consumption_night_kwh: float = 0.0
    total_production_day_kwh: float = 0.0
    total_production_night_kwh: float = 0.0
    
    # Electrical info
    current_tariff: int = 0  # 1=day, 2=night
    l1_voltage_v: float = 0.0
    l2_voltage_v: float = 0.0
    l3_voltage_v: float = 0.0
    l1_current_a: float = 0.0
    l2_current_a: float = 0.0
    l3_current_a: float = 0.0
    
    # Switches/status
    switch_electricity: int = 0
    switch_gas: int = 0
    
    # Serial numbers
    meter_serial_electricity: str = ""
    meter_serial_gas: str = ""
    
    # Gas
    gas_consumption_m3: float = 0.0
    
    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)


class P1Reader:
    def __init__(self, port: str = "/dev/ttyUSB0", baudrate: int = 115200):
        self.port = port
        self.baudrate = baudrate
        self.serial_conn = None
        self.latest_reading: Optional[MeterReading] = None
        self.websocket_clients = set()
        
    def connect(self):
        """Connect to the P1 port"""
        try:
            self.serial_conn = serial.Serial(
                port=self.port,
                baudrate=self.baudrate,
                bytesize=serial.EIGHTBITS,
                parity=serial.PARITY_NONE,
                stopbits=serial.STOPBITS_ONE,
                xonxoff=True,
                timeout=1
            )
            print(f"Connected to P1 port on {self.port}")
        except Exception as e:
            print(f"Failed to connect to P1 port: {e}")
            raise
    
    def disconnect(self):
        """Disconnect from P1 port"""
        if self.serial_conn and self.serial_conn.is_open:
            self.serial_conn.close()
            print("Disconnected from P1 port")
    
    def validate_crc(self, telegram: str) -> bool:
        """Validate CRC16 checksum of telegram"""
        try:
            if '!' not in telegram:
                return False
            
            parts = telegram.split('!')
            if len(parts) != 2 or len(parts[1]) < 4:
                return False
            
            data = parts[0] + '!'
            given_crc = parts[1][:4]
            
            # CRC16 with polynomial x^16 + x^15 + x^2 + 1 (DSMR standard)
            crc16 = crcmod.mkCrcFun(0x18005, initCrc=0x0000, xorOut=0x0000)
            calc_crc = f"{crc16(data.encode('ascii')):04X}"
            
            return given_crc.upper() == calc_crc.upper()
        except Exception as e:
            print(f"CRC validation error: {e}")
            return False
        """Parse a P1 telegram into a MeterReading"""
        try:
            # Extract timestamp from telegram
            timestamp_match = re.search(r'0-0:1\.0\.0\((\d{12}[WS])\)', telegram)
            if timestamp_match:
                ts_str = timestamp_match.group(1)
                # Parse YYMMDDHHMMSS[WS] format
                dt = datetime.strptime(ts_str[:-1], '%y%m%d%H%M%S')
                timestamp = dt.isoformat()
            else:
                timestamp = datetime.now().isoformat()
            
            reading = MeterReading(timestamp=timestamp)
            
            # OBIS patterns with their target fields and conversion functions
            obis_mapping = {
                # Current consumption/production
                r'1-0:1\.7\.0\((\d+\.\d+)\*kW\)': ('current_consumption_kw', float),
                r'1-0:2\.7\.0\((\d+\.\d+)\*kW\)': ('current_production_kw', float),
                r'1-0:21\.7\.0\((\d+\.\d+)\*kW\)': ('l1_consumption_kw', float),
                r'1-0:41\.7\.0\((\d+\.\d+)\*kW\)': ('l2_consumption_kw', float),
                r'1-0:61\.7\.0\((\d+\.\d+)\*kW\)': ('l3_consumption_kw', float),
                r'1-0:22\.7\.0\((\d+\.\d+)\*kW\)': ('l1_production_kw', float),
                r'1-0:42\.7\.0\((\d+\.\d+)\*kW\)': ('l2_production_kw', float),
                r'1-0:62\.7\.0\((\d+\.\d+)\*kW\)': ('l3_production_kw', float),
                
                # Total consumption/production
                r'1-0:1\.8\.1\((\d+\.\d+)\*kWh\)': ('total_consumption_day_kwh', float),
                r'1-0:1\.8\.2\((\d+\.\d+)\*kWh\)': ('total_consumption_night_kwh', float),
                r'1-0:2\.8\.1\((\d+\.\d+)\*kWh\)': ('total_production_day_kwh', float),
                r'1-0:2\.8\.2\((\d+\.\d+)\*kWh\)': ('total_production_night_kwh', float),
                
                # Electrical measurements
                r'1-0:32\.7\.0\((\d+\.\d+)\*V\)': ('l1_voltage_v', float),
                r'1-0:52\.7\.0\((\d+\.\d+)\*V\)': ('l2_voltage_v', float),
                r'1-0:72\.7\.0\((\d+\.\d+)\*V\)': ('l3_voltage_v', float),
                r'1-0:31\.7\.0\((\d+\.\d+)\*A\)': ('l1_current_a', float),
                r'1-0:51\.7\.0\((\d+\.\d+)\*A\)': ('l2_current_a', float),
                r'1-0:71\.7\.0\((\d+\.\d+)\*A\)': ('l3_current_a', float),
                
                # Switches/status
                r'0-0:96\.3\.10\((\d+)\)': ('switch_electricity', int),
                r'0-1:24\.4\.0\((\d+)\)': ('switch_gas', int),
                
                # Gas consumption
                r'0-1:24\.2\.3\(\d{12}[WS]\)\((\d+\.\d+)\*m3\)': ('gas_consumption_m3', float),
            }
            
            # Special handlers for complex cases
            def parse_tariff(value: str) -> int:
                """Convert 0001 to 1, 0002 to 2"""
                int_val = int(value)
                return int_val if int_val < 10 else int(str(int_val)[-1:])
            
            def parse_hex_serial(value: str) -> str:
                """Decode hex to ASCII"""
                try:
                    return bytes.fromhex(value).decode('ascii')
                except:
                    return value  # Keep hex if decode fails
            
            # Special cases with custom parsers
            special_cases = {
                r'0-0:96\.14\.0\((\d{4})\)': ('current_tariff', parse_tariff),
                r'0-0:96\.1\.1\(([A-F0-9]+)\)': ('meter_serial_electricity', parse_hex_serial),
                r'0-1:96\.1\.1\(([A-F0-9]+)\)': ('meter_serial_gas', parse_hex_serial),
            }
            
            # Process regular mappings
            for pattern, (field_name, converter) in obis_mapping.items():
                match = re.search(pattern, telegram)
                if match:
                    setattr(reading, field_name, converter(match.group(1)))
            
            # Process special cases
            for pattern, (field_name, parser_func) in special_cases.items():
                match = re.search(pattern, telegram)
                if match:
                    setattr(reading, field_name, parser_func(match.group(1)))
            
            return reading
            
        except Exception as e:
            print(f"Error parsing telegram: {e}")
            return None
    
    def read_telegram(self) -> Optional[str]:
        """Read a complete telegram from the P1 port"""
        if not self.serial_conn or not self.serial_conn.is_open:
            return None
            
        try:
            # Read until we find the start of a telegram
            buffer = ""
            while True:
                data = self.serial_conn.read(1)
                if not data:
                    continue
                    
                char = data.decode('ascii', errors='ignore')
                buffer += char
                
                # Look for start of telegram
                if char == '/' and len(buffer) > 1:
                    buffer = char  # Reset buffer to start fresh
                elif char == '!' and '/' in buffer:
                    # Read the CRC (4 hex chars after !)
                    crc = self.serial_conn.read(4).decode('ascii', errors='ignore')
                    buffer += crc
                    return buffer
                    
        except Exception as e:
            print(f"Error reading telegram: {e}")
            return None
    
    async def start_reading(self):
        """Start reading P1 data in a loop"""
        consecutive_errors = 0
        max_errors = 10
        
        while consecutive_errors < max_errors:
            try:
                telegram = self.read_telegram()
                if telegram:
                    reading = self.parse_telegram(telegram)
                    if reading:
                        self.latest_reading = reading
                        # Broadcast to all WebSocket clients
                        await self.broadcast_to_websockets(reading.to_dict())
                        consecutive_errors = 0  # Reset error counter on success
                        
                await asyncio.sleep(0.1)  # Small delay to prevent CPU spinning
                
            except Exception as e:
                consecutive_errors += 1
                print(f"Error in reading loop ({consecutive_errors}/{max_errors}): {e}")
                await asyncio.sleep(1)
        
        print(f"Too many consecutive errors ({max_errors}), stopping reader")
    
    async def broadcast_to_websockets(self, data: Dict[str, Any]):
        """Broadcast data to all connected WebSocket clients"""
        if not self.websocket_clients:
            return
            
        disconnected = set()
        for websocket in self.websocket_clients:
            try:
                await websocket.send_text(json.dumps(data))
            except:
                disconnected.add(websocket)
        
        # Remove disconnected clients
        self.websocket_clients -= disconnected
    
    def add_websocket_client(self, websocket: WebSocket):
        """Add a WebSocket client"""
        self.websocket_clients.add(websocket)
    
    def remove_websocket_client(self, websocket: WebSocket):
        """Remove a WebSocket client"""
        self.websocket_clients.discard(websocket)


# Global P1 reader instance
p1_reader = P1Reader(
    port=os.getenv("P1_PORT", "/dev/ttyUSB0"),
    baudrate=int(os.getenv("P1_BAUDRATE", "115200"))
)


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    reading_task = None
    try:
        p1_reader.connect()
        # Start the reading task
        reading_task = asyncio.create_task(p1_reader.start_reading())
        yield
    except Exception as e:
        print(f"Failed to start P1 reader: {e}")
        print("API will run but no meter data will be available")
        yield
    finally:
        # Shutdown
        if reading_task:
            reading_task.cancel()
        p1_reader.disconnect()


app = FastAPI(
    title="P1 Meter API",
    description="Belgian smart meter P1 port reader",
    version="1.0.0",
    lifespan=lifespan
)


@app.get("/")
async def root():
    return {"message": "P1 Meter API", "status": "running"}


@app.get("/latest")
async def get_latest_reading():
    """Get the latest meter reading"""
    if p1_reader.latest_reading is None:
        return JSONResponse(
            status_code=404,
            content={"error": "No readings available yet"}
        )
    
    return p1_reader.latest_reading.to_dict()


@app.websocket("/ws")
async def websocket_endpoint(websocket: WebSocket):
    """WebSocket endpoint for real-time meter updates"""
    await websocket.accept()
    p1_reader.add_websocket_client(websocket)
    
    try:
        # Send current reading immediately if available
        if p1_reader.latest_reading:
            await websocket.send_text(json.dumps(p1_reader.latest_reading.to_dict()))
        
        # Keep connection alive and handle messages
        while True:
            try:
                # Wait for client messages (ping/pong, etc.)
                await websocket.receive_text()
            except WebSocketDisconnect:
                break
    except Exception as e:
        print(f"WebSocket error: {e}")
    finally:
        p1_reader.remove_websocket_client(websocket)


if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8000,
        reload=False
    )