package solarinverter

import (
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/NotCoffee418/european_smart_meter/pkg/config"
	"github.com/goburrow/modbus"
	probing "github.com/prometheus-community/pro-bing"
)

var (
	ErrModbusNotConfigured = fmt.Errorf("modbus not configured") // may be intended
	ErrModbusReadFailed    = fmt.Errorf("modbus read failed")
	ErrModbusNotConnected  = fmt.Errorf("modbus not connected")
)

var (
	solarPowerMu      sync.Mutex
	lastSolarReadWatt int32 = 0
	lastSolarReadTime time.Time
)

// IsModbusConfigured checks if the modbus configuration is set.
// This feature is optional, Empty values as config are acceptable.
func IsModbusConfigured() bool {
	return config.ActiveInterpreterAPIConfig.SolarInverterIp != "" &&
		config.ActiveInterpreterAPIConfig.SolarInverterModbusPort != 0 &&
		config.ActiveInterpreterAPIConfig.WlanConnectionId != ""
}

func ReadSolarData() (int32, error) {
	// Check if configured
	if !IsModbusConfigured() {
		return 0, ErrModbusNotConfigured
	}

	// Use cached reads to avoid spamming the poor inverter
	solarPowerMu.Lock()
	defer solarPowerMu.Unlock()
	if lastSolarReadTime.After(time.Now().Add(-10 * time.Second)) {
		return lastSolarReadWatt, nil
	}

	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Try reconnecting on retry attempts
			if err := tryReconnect(); err != nil {
				lastErr = fmt.Errorf("reconnect failed on attempt %d: %w", attempt+1, err)
				continue
			}
		}

		// Ping check before attempting modbus connection
		if ok, _, err := ping(config.ActiveInterpreterAPIConfig.SolarInverterIp); !ok || err != nil {
			lastErr = fmt.Errorf("ping failed on attempt %d: %w", attempt+1, err)
			if attempt < maxRetries-1 {
				time.Sleep(2 * time.Second)
			}
			continue
		}

		host := config.ActiveInterpreterAPIConfig.SolarInverterIp
		port := config.ActiveInterpreterAPIConfig.SolarInverterModbusPort

		handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%d", host, port))
		handler.Timeout = 10 * time.Second
		handler.SlaveId = 0

		if err := handler.Connect(); err != nil {
			lastErr = fmt.Errorf("connection failed on attempt %d: %w", attempt+1, err)
			handler.Close()
			if attempt < maxRetries-1 {
				time.Sleep(2 * time.Second)
			}
			continue
		}

		// The 2s delay after connecting causes everything to not implode as much
		time.Sleep(2 * time.Second)
		client := modbus.NewClient(handler)

		// Read Active Power
		result, err := client.ReadHoldingRegisters(32080, 2)
		handler.Close()

		if err != nil {
			lastErr = fmt.Errorf("read power failed on attempt %d: %w", attempt+1, err)
			if attempt < maxRetries-1 {
				time.Sleep(2 * time.Second)
			}
			continue
		}

		// Success - calculate power and return
		power := int32(result[0])<<24 | int32(result[1])<<16 | int32(result[2])<<8 | int32(result[3])
		lastSolarReadWatt = power
		lastSolarReadTime = time.Now()
		return power, nil
	}

	return 0, errors.Join(ErrModbusReadFailed, lastErr)
}

func tryReconnect() error {
	if !IsModbusConfigured() {
		return ErrModbusNotConfigured
	}

	// Check if already connected
	ok, _, err := ping(config.ActiveInterpreterAPIConfig.SolarInverterIp)
	if err != nil {
		return err
	}
	if ok {
		return nil // Already connected, no need to reconnect
	}

	// Try reconnecting to wifi
	cmd := exec.Command("nmcli", "connection", "up", config.ActiveInterpreterAPIConfig.WlanConnectionId)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring up wifi connection: %w", err)
	}

	// Wait a bit for the connection to establish
	time.Sleep(5 * time.Second)

	// Check connection again
	ok, _, err = ping(config.ActiveInterpreterAPIConfig.SolarInverterIp)
	if err != nil {
		return err
	}
	if !ok {
		return ErrModbusNotConnected
	}
	return nil
}

func ping(host string) (bool, time.Duration, error) {
	pinger, err := probing.NewPinger(host)
	if err != nil {
		return false, 0, err
	}

	pinger.Count = 1
	pinger.Timeout = 2 * time.Second
	pinger.SetPrivileged(false) // UDP-based, no root needed

	err = pinger.Run()
	if err != nil {
		return false, 0, err
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv > 0 {
		return true, stats.AvgRtt, nil
	}

	return false, 0, fmt.Errorf("no response")
}
