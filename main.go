package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jacobsa/go-serial/serial"
	"github.com/sigurn/crc16"
)

type MeterReading struct {
	Timestamp string `json:"timestamp"`

	// Current consumption/production
	CurrentConsumptionKW float64 `json:"current_consumption_kw"`
	CurrentProductionKW  float64 `json:"current_production_kw"`
	L1ConsumptionKW      float64 `json:"l1_consumption_kw"`
	L2ConsumptionKW      float64 `json:"l2_consumption_kw"`
	L3ConsumptionKW      float64 `json:"l3_consumption_kw"`
	L1ProductionKW       float64 `json:"l1_production_kw"`
	L2ProductionKW       float64 `json:"l2_production_kw"`
	L3ProductionKW       float64 `json:"l3_production_kw"`

	// Totals
	TotalConsumptionDayKWH   float64 `json:"total_consumption_day_kwh"`
	TotalConsumptionNightKWH float64 `json:"total_consumption_night_kwh"`
	TotalProductionDayKWH    float64 `json:"total_production_day_kwh"`
	TotalProductionNightKWH  float64 `json:"total_production_night_kwh"`

	// Electrical info
	CurrentTariff int     `json:"current_tariff"`
	L1VoltageV    float64 `json:"l1_voltage_v"`
	L2VoltageV    float64 `json:"l2_voltage_v"`
	L3VoltageV    float64 `json:"l3_voltage_v"`
	L1CurrentA    float64 `json:"l1_current_a"`
	L2CurrentA    float64 `json:"l2_current_a"`
	L3CurrentA    float64 `json:"l3_current_a"`

	// Switches/status
	SwitchElectricity int `json:"switch_electricity"`
	SwitchGas         int `json:"switch_gas"`

	// Serial numbers
	MeterSerialElectricity string `json:"meter_serial_electricity"`
	MeterSerialGas         string `json:"meter_serial_gas"`

	// Gas
	GasConsumptionM3 float64 `json:"gas_consumption_m3"`
}

type P1Reader struct {
	port           string
	baudrate       uint
	serialPort     io.ReadWriteCloser
	latestReading  *MeterReading
	readingMutex   sync.RWMutex
	wsClients      map[*websocket.Conn]bool
	wsClientsMutex sync.RWMutex

	// Pre-compiled regex patterns
	obisPatterns    map[string]*regexp.Regexp
	specialPatterns map[string]*regexp.Regexp
}

func NewP1Reader(port string, baudrate uint) *P1Reader {
	reader := &P1Reader{
		port:      port,
		baudrate:  baudrate,
		wsClients: make(map[*websocket.Conn]bool),
	}

	// Pre-compile regex patterns
	reader.obisPatterns = map[string]*regexp.Regexp{
		"current_consumption":     regexp.MustCompile(`1-0:1\.7\.0\((\d+\.\d+)\*kW\)`),
		"current_production":      regexp.MustCompile(`1-0:2\.7\.0\((\d+\.\d+)\*kW\)`),
		"l1_consumption":          regexp.MustCompile(`1-0:21\.7\.0\((\d+\.\d+)\*kW\)`),
		"l2_consumption":          regexp.MustCompile(`1-0:41\.7\.0\((\d+\.\d+)\*kW\)`),
		"l3_consumption":          regexp.MustCompile(`1-0:61\.7\.0\((\d+\.\d+)\*kW\)`),
		"l1_production":           regexp.MustCompile(`1-0:22\.7\.0\((\d+\.\d+)\*kW\)`),
		"l2_production":           regexp.MustCompile(`1-0:42\.7\.0\((\d+\.\d+)\*kW\)`),
		"l3_production":           regexp.MustCompile(`1-0:62\.7\.0\((\d+\.\d+)\*kW\)`),
		"total_consumption_day":   regexp.MustCompile(`1-0:1\.8\.1\((\d+\.\d+)\*kWh\)`),
		"total_consumption_night": regexp.MustCompile(`1-0:1\.8\.2\((\d+\.\d+)\*kWh\)`),
		"total_production_day":    regexp.MustCompile(`1-0:2\.8\.1\((\d+\.\d+)\*kWh\)`),
		"total_production_night":  regexp.MustCompile(`1-0:2\.8\.2\((\d+\.\d+)\*kWh\)`),
		"l1_voltage":              regexp.MustCompile(`1-0:32\.7\.0\((\d+\.\d+)\*V\)`),
		"l2_voltage":              regexp.MustCompile(`1-0:52\.7\.0\((\d+\.\d+)\*V\)`),
		"l3_voltage":              regexp.MustCompile(`1-0:72\.7\.0\((\d+\.\d+)\*V\)`),
		"l1_current":              regexp.MustCompile(`1-0:31\.7\.0\((\d+\.\d+)\*A\)`),
		"l2_current":              regexp.MustCompile(`1-0:51\.7\.0\((\d+\.\d+)\*A\)`),
		"l3_current":              regexp.MustCompile(`1-0:71\.7\.0\((\d+\.\d+)\*A\)`),
		"switch_electricity":      regexp.MustCompile(`0-0:96\.3\.10\((\d+)\)`),
		"switch_gas":              regexp.MustCompile(`0-1:24\.4\.0\((\d+)\)`),
		"gas_consumption":         regexp.MustCompile(`0-1:24\.2\.3\(\d{12}[WS]\)\((\d+\.\d+)\*m3\)`),
	}

	reader.specialPatterns = map[string]*regexp.Regexp{
		"timestamp":                regexp.MustCompile(`0-0:1\.0\.0\((\d{12}[WS])\)`),
		"current_tariff":           regexp.MustCompile(`0-0:96\.14\.0\((\d{4})\)`),
		"meter_serial_electricity": regexp.MustCompile(`0-0:96\.1\.1\(([A-F0-9]+)\)`),
		"meter_serial_gas":         regexp.MustCompile(`0-1:96\.1\.1\(([A-F0-9]+)\)`),
	}

	return reader
}

func (p *P1Reader) Connect() error {
	options := serial.OpenOptions{
		PortName:        p.port,
		BaudRate:        p.baudrate,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 1,
	}

	port, err := serial.Open(options)
	if err != nil {
		return fmt.Errorf("failed to open serial port: %w", err)
	}

	p.serialPort = port
	log.Printf("Connected to P1 port on %s", p.port)
	return nil
}

func (p *P1Reader) Disconnect() {
	if p.serialPort != nil {
		p.serialPort.Close()
		log.Println("Disconnected from P1 port")
	}
}

func (p *P1Reader) ValidateCRC(telegram string) bool {
	parts := strings.Split(telegram, "!")
	if len(parts) != 2 || len(parts[1]) < 4 {
		return false
	}

	data := parts[0] + "!"
	givenCRC := parts[1][:4]

	// Use CRC16_ARC which matches Belgian DSMR specification
	table := crc16.MakeTable(crc16.CRC16_ARC)
	calcCRC := crc16.Checksum([]byte(data), table)
	calcCRCHex := fmt.Sprintf("%04X", calcCRC)

	return strings.ToUpper(givenCRC) == calcCRCHex
}

func (p *P1Reader) ParseTelegram(telegram string) *MeterReading {
	if !p.ValidateCRC(telegram) {
		log.Println("Invalid CRC, skipping telegram")
		return nil
	}

	reading := &MeterReading{
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Parse timestamp
	if match := p.specialPatterns["timestamp"].FindStringSubmatch(telegram); match != nil {
		tsStr := match[1]
		if t, err := time.Parse("060102150405", tsStr[:12]); err == nil {
			reading.Timestamp = t.Format(time.RFC3339)
		}
	}

	// Parse regular OBIS codes
	obisMap := map[string]func(float64){
		"current_consumption":     func(v float64) { reading.CurrentConsumptionKW = v },
		"current_production":      func(v float64) { reading.CurrentProductionKW = v },
		"l1_consumption":          func(v float64) { reading.L1ConsumptionKW = v },
		"l2_consumption":          func(v float64) { reading.L2ConsumptionKW = v },
		"l3_consumption":          func(v float64) { reading.L3ConsumptionKW = v },
		"l1_production":           func(v float64) { reading.L1ProductionKW = v },
		"l2_production":           func(v float64) { reading.L2ProductionKW = v },
		"l3_production":           func(v float64) { reading.L3ProductionKW = v },
		"total_consumption_day":   func(v float64) { reading.TotalConsumptionDayKWH = v },
		"total_consumption_night": func(v float64) { reading.TotalConsumptionNightKWH = v },
		"total_production_day":    func(v float64) { reading.TotalProductionDayKWH = v },
		"total_production_night":  func(v float64) { reading.TotalProductionNightKWH = v },
		"l1_voltage":              func(v float64) { reading.L1VoltageV = v },
		"l2_voltage":              func(v float64) { reading.L2VoltageV = v },
		"l3_voltage":              func(v float64) { reading.L3VoltageV = v },
		"l1_current":              func(v float64) { reading.L1CurrentA = v },
		"l2_current":              func(v float64) { reading.L2CurrentA = v },
		"l3_current":              func(v float64) { reading.L3CurrentA = v },
		"gas_consumption":         func(v float64) { reading.GasConsumptionM3 = v },
	}

	for field, setter := range obisMap {
		if pattern, exists := p.obisPatterns[field]; exists {
			if match := pattern.FindStringSubmatch(telegram); match != nil {
				if value, err := strconv.ParseFloat(match[1], 64); err == nil {
					setter(value)
				}
			}
		}
	}

	// Parse integer fields
	intMap := map[string]func(int){
		"switch_electricity": func(v int) { reading.SwitchElectricity = v },
		"switch_gas":         func(v int) { reading.SwitchGas = v },
	}

	for field, setter := range intMap {
		if pattern, exists := p.obisPatterns[field]; exists {
			if match := pattern.FindStringSubmatch(telegram); match != nil {
				if value, err := strconv.Atoi(match[1]); err == nil {
					setter(value)
				}
			}
		}
	}

	// Parse special cases
	if match := p.specialPatterns["current_tariff"].FindStringSubmatch(telegram); match != nil {
		if value, err := strconv.Atoi(match[1]); err == nil {
			if value < 10 {
				reading.CurrentTariff = value
			} else {
				// Convert 0001 to 1, 0002 to 2
				reading.CurrentTariff = value % 10
			}
		}
	}

	// Parse hex serial numbers
	if match := p.specialPatterns["meter_serial_electricity"].FindStringSubmatch(telegram); match != nil {
		if decoded, err := hex.DecodeString(match[1]); err == nil {
			reading.MeterSerialElectricity = string(decoded)
		} else {
			reading.MeterSerialElectricity = match[1]
		}
	}

	if match := p.specialPatterns["meter_serial_gas"].FindStringSubmatch(telegram); match != nil {
		if decoded, err := hex.DecodeString(match[1]); err == nil {
			reading.MeterSerialGas = string(decoded)
		} else {
			reading.MeterSerialGas = match[1]
		}
	}

	return reading
}

func (p *P1Reader) ReadTelegram() (string, error) {
	if p.serialPort == nil {
		return "", fmt.Errorf("serial port not connected")
	}

	var buffer strings.Builder
	var inTelegram bool
	reader := bufio.NewReader(p.serialPort)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		if strings.HasPrefix(line, "/") {
			// Start of telegram
			buffer.Reset()
			buffer.WriteString(line)
			inTelegram = true
		} else if inTelegram {
			buffer.WriteString(line)
			if strings.HasPrefix(strings.TrimSpace(line), "!") {
				// End of telegram
				return buffer.String(), nil
			}
		}
	}
}

func (p *P1Reader) StartReading() {
	consecutiveErrors := 0
	maxErrors := 10

	for consecutiveErrors < maxErrors {
		telegram, err := p.ReadTelegram()
		if err != nil {
			consecutiveErrors++
			log.Printf("Error reading telegram (%d/%d): %v", consecutiveErrors, maxErrors, err)
			time.Sleep(time.Second)
			continue
		}

		if reading := p.ParseTelegram(telegram); reading != nil {
			p.readingMutex.Lock()
			p.latestReading = reading
			p.readingMutex.Unlock()

			p.BroadcastToWebSockets(reading)
			consecutiveErrors = 0
			log.Printf("Successfully parsed telegram - Current consumption: %.3f kW", reading.CurrentConsumptionKW)
		}

		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Too many consecutive errors (%d), stopping reader", maxErrors)
}

func (p *P1Reader) GetLatestReading() *MeterReading {
	p.readingMutex.RLock()
	defer p.readingMutex.RUnlock()
	return p.latestReading
}

func (p *P1Reader) BroadcastToWebSockets(reading *MeterReading) {
	p.wsClientsMutex.RLock()
	clients := make([]*websocket.Conn, 0, len(p.wsClients))
	for client := range p.wsClients {
		clients = append(clients, client)
	}
	p.wsClientsMutex.RUnlock()

	data, err := json.Marshal(reading)
	if err != nil {
		log.Printf("Error marshaling reading: %v", err)
		return
	}

	for _, client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			p.RemoveWebSocketClient(client)
		}
	}
}

func (p *P1Reader) AddWebSocketClient(conn *websocket.Conn) {
	p.wsClientsMutex.Lock()
	p.wsClients[conn] = true
	p.wsClientsMutex.Unlock()
}

func (p *P1Reader) RemoveWebSocketClient(conn *websocket.Conn) {
	p.wsClientsMutex.Lock()
	delete(p.wsClients, conn)
	p.wsClientsMutex.Unlock()
	conn.Close()
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

var p1Reader *P1Reader

func main() {
	port := os.Getenv("P1_PORT")
	if port == "" {
		port = "/dev/ttyUSB0"
	}

	baudrateStr := os.Getenv("P1_BAUDRATE")
	if baudrateStr == "" {
		baudrateStr = "115200"
	}

	baudrate, err := strconv.ParseUint(baudrateStr, 10, 32)
	if err != nil {
		log.Fatalf("Invalid baudrate: %v", err)
	}

	p1Reader = NewP1Reader(port, uint(baudrate))

	// Try to connect to P1 port
	if err := p1Reader.Connect(); err != nil {
		log.Printf("Failed to start P1 reader: %v", err)
		log.Println("API will run but no meter data will be available")
	} else {
		go p1Reader.StartReading()
	}

	// Setup HTTP handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]string{
			"message": "Belgian Smart Meter API",
			"status":  "running",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/latest", func(w http.ResponseWriter, r *http.Request) {
		reading := p1Reader.GetLatestReading()
		if reading == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "No readings available yet",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reading)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		p1Reader.AddWebSocketClient(conn)

		// Send current reading immediately if available
		if reading := p1Reader.GetLatestReading(); reading != nil {
			if data, err := json.Marshal(reading); err == nil {
				conn.WriteMessage(websocket.TextMessage, data)
			}
		}

		// Keep connection alive
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				p1Reader.RemoveWebSocketClient(conn)
				break
			}
		}
	})

	log.Println("Starting Belgian Smart Meter API on :9039")
	log.Fatal(http.ListenAndServe(":9039", nil))
}
