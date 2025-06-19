package port_reader

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/NotCoffee418/european_smart_meter/pkg/interpreter"
	"github.com/jacobsa/go-serial/serial"
	"github.com/sigurn/crc16"
)

// Initialize a new P1Reader client.
func NewP1Reader(port string, baudrate uint) *P1Reader {
	reader := &P1Reader{
		port:       port,
		baudrate:   baudrate,
		stopSignal: false,
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

// Start listening for readings. Messages are sent every second.
// Runs in goroutine. handleReading() also runs in goroutine.
func (p *P1Reader) StartReading(
	handleReading func(reading *interpreter.RawMeterReading),
	handleError func(error),
) {
	p.stopSignal = false

	go func() {
		// Tolerance before we report error.
		consecutiveErrors := 0
		maxErrors := 10
		var lastError error

		// Initialize the connection
		openConnError := p.connect()
		if openConnError != nil {
			handleError(openConnError)
			return
		}

		for consecutiveErrors < maxErrors {
			// Check for Stop command
			if p.stopSignal {
				fmt.Println("Stop signal received, disconnecting")
				p.disconnect()
				return
			}

			// Read the telegram
			telegram, err := p.readTelegram()
			if err != nil {
				consecutiveErrors++
				lastError = err
				log.Printf("Error reading telegram (%d/%d): %v", consecutiveErrors, maxErrors, err)
				time.Sleep(time.Second)
				continue
			}

			if reading := p.parseTelegram(telegram); reading != nil {
				p.readingMutex.Lock()
				p.latestReading = reading
				p.readingMutex.Unlock()

				go handleReading(reading)
				consecutiveErrors = 0
			}
		}

		log.Printf("Too many consecutive errors (%d), stopping reader: %v", maxErrors, lastError)
		handleError(lastError)
		p.disconnect()
	}()
}

func (p *P1Reader) StopReading() {
	p.stopSignal = true
	p.disconnect()
}

func (p *P1Reader) GetLatestReading() *interpreter.RawMeterReading {
	p.readingMutex.RLock()
	defer p.readingMutex.RUnlock()
	return p.latestReading
}

// Open the connection to the P1 port.
func (p *P1Reader) connect() error {
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

func (p *P1Reader) disconnect() {
	if p.serialPort != nil {
		p.serialPort.Close()
		log.Println("Disconnected from P1 port")
	}
}

func (p *P1Reader) readTelegram() (string, error) {
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

func (p *P1Reader) validateCRC(telegram string) bool {
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

func (p *P1Reader) parseTelegram(telegram string) *interpreter.RawMeterReading {
	if !p.validateCRC(telegram) {
		log.Println("Invalid CRC, skipping telegram")
		return nil
	}

	reading := &interpreter.RawMeterReading{
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
