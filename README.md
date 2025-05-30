# Smart Meter API

This is a simple API to get data from the European smart meter.  
This was designed for Belgian meters but should work for other countries as well.

## Requirements

- Raspberry Pi 3+ (or other Linux machine)
- [P1 Smart Meter Cable](https://webshop.cedel.nl/nl/Slimme-meter-kabel-P1-naar-USB)
- [Activated P1 port on your smart meter](https://www.stroohm.be/en/help/using-and-accessing-the-p1-port-of-the-digital-meter-in-belgium/)
- Docker installed (Optional)

## Installation

1. [Set up Raspberry Pi](https://www.raspberrypi.com/documentation/computers/getting-started.html)
2. Connect the cable to the P1 port on the meter and a USB port on the Raspberry Pi
3. SSH into the Raspberry Pi or open a terminal on the Raspberry Pi
4. [Install Docker](https://docs.docker.com/engine/install/debian/)
5. Give user Docker access

    ```bash
    sudo usermod -aG docker $USER
    newgrp docker # Probably not needed
    ```

6. Logout and login again to apply the changes (or close and reopen the terminal).
7. Clone the repository to your Raspberry Pi

    ```bash
    git clone https://github.com/NotCoffee418/belgian_smart_meter_api
    cd belgian_smart_meter_api
    ```

8. **(Optional)** Confirm which port the P1 cable is connected to (probably `/dev/ttyUSB0`).

   ```bash
   # List all USB devices. If only one it will be /dev/ttyUSB0.
   ls /dev/ttyUSB*

   # Expected output:
   # pi@raspberrypi:~ $ ls /dev/ttyUSB*
   # /dev/ttyUSB0

   # Test the connection with this command (replace /dev/ttyUSB0 with your port).
   cat /dev/ttyUSB0 | grep "\:1\.7\.0"

   # You should see your current consumption at the bottom:
   # 1-0:1.7.0(00.342*kW)
   
   # If it shows nothing, you may be on the wrong device or your P1 port is not activated yet.
   ```

9. **(Optional)** Confirm the correct device in `compose.yml` (default: `/dev/ttyUSB0`)
10. **(Optional)** Confirm the correct baudrate in `compose.yml` (default: `115200`)
    - Older meters: `9600`
    - Newer meters (DSMR 4.0+/ESMR 5.x+): `115200`

### Docker Compose (Recommended)

```bash
git clone https://github.com/NotCoffee418/belgian_smart_meter_api
cd belgian_smart_meter_api
docker compose up -d
```

### Directly (Not recommended)

```bash
git clone https://github.com/NotCoffee418/belgian_smart_meter_api
cd belgian_smart_meter_api
python3 -m venv .venv
source .venv/bin/activate
pip3 install -r requirements.txt
python3 main.py
```

## Test the connection

```bash
# Run this command from the Raspberry Pi.
# You should see a JSON response with the latest data from the smart meter.
curl http://localhost:9039/latest | python3 -m json.tool
```



## API Usage

Additional tools for the API (still being built or available in my github if i forget to update this message)


## Endpoints

- **/latest**: Get the latest data from the smart meter
- **/ws**: Subscribe to the websocket endpoint to get real-time data from the smart meter

Both output the following JSON response structure:

```json
{
  "timestamp": "2025-05-30T15:52:07", // Local Time
  "current_consumption_kw": 0.150, // Combined Consumption (L1+L2+L3)
  "current_production_kw": 0.0,
  "l1_consumption_kw": 0.150,
  "l2_consumption_kw": 0.0, // L2 and L3 is for 3 phase meters (industrial)
  "l3_consumption_kw": 0.0, // L2 and L3 is for 3 phase meters (industrial)
  "l1_production_kw": 0.0,
  "l2_production_kw": 0.0,
  "l3_production_kw": 0.0,
  "total_consumption_day_kwh": 9999.999,
  "total_consumption_night_kwh": 9999.99,
  "total_production_day_kwh": 9999.99,
  "total_production_night_kwh": 9999.99,
  "current_tariff": 1, // 1=day, 2=night
  "l1_voltage_v": 237.8,
  "l2_voltage_v": 0.0,
  "l3_voltage_v": 0.0,
  "l1_current_a": 2.28,
  "l2_current_a": 0.0,
  "l3_current_a": 0.0,
  "switch_electricity": 1, // 1=on, 0=off - Physical switch on the meter
  "switch_gas": 1, // 1=on, 0=off - Physical switch on the meter
  "meter_serial_electricity": "XXXXXXXXX",
  "meter_serial_gas": "XXXXXXXXX",
  "gas_consumption_m3": 9999.99 // Updated every 10 minutes
}
```

## Troubleshooting

### Device not found

If you have multiple USB ports, you need to check which one is the P1 port and adjust it in the docker-compose.yml file if it's not /dev/ttyUSB0.

```bash
# List all USB devices
ls /dev/ttyUSB*
```

### No data received
It may be that you have submitted the request to activate the P1 port but it's not activated yet.
You can check if you're receiving data by running the following command:

```bash
python3 -m serial.tools.miniterm /dev/ttyUSB0 115200 --xonxoff
```

