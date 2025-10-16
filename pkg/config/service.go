package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/NotCoffee418/european_smart_meter/pkg/pathing"
)

var (
	ActiveInterpreterAPIConfig *InterpreterAPIConfig
	ActiveMeterCollectorConfig *MeterCollectorConfig
)

func LoadInterpreterAPIConfig() error {
	configPath := filepath.Join(pathing.GetConfigDir(), "interpreter_api.toml")

	// Create default if not exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := &InterpreterAPIConfig{
			SerialDevice:            "/dev/ttyUSB0",
			Baudrate:                115200,
			ListenAddress:           "0.0.0.0",
			ListenPort:              9039,
			SolarInverterIp:         "192.168.200.1",
			SolarInverterModbusPort: 502,
			WlanConnectionId:        "preconfigured", // Check with `nmcli device status`
		}
		// Create file
		cfgFile, err := os.Create(configPath)
		if err != nil {
			return err
		}
		defer cfgFile.Close()
		toml.NewEncoder(cfgFile).Encode(cfg)
		ActiveInterpreterAPIConfig = cfg
		return nil
	}

	// Load existing config
	var config InterpreterAPIConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		return err
	}
	ActiveInterpreterAPIConfig = &config
	return nil
}

func LoadMeterCollectorConfig() error {
	configPath := filepath.Join(pathing.GetConfigDir(), "meter_collector.toml")

	// Create default if not exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := &MeterCollectorConfig{
			InterpreterAPIHost: "localhost:9039",
			TLSEnabled:         false,
		}
		// Create file
		cfgFile, err := os.Create(configPath)
		if err != nil {
			return err
		}
		defer cfgFile.Close()
		toml.NewEncoder(cfgFile).Encode(cfg)
		ActiveMeterCollectorConfig = cfg
		return nil
	}

	// Load existing config
	var config MeterCollectorConfig
	_, err := toml.DecodeFile(configPath, &config)
	if err != nil {
		return err
	}
	ActiveMeterCollectorConfig = &config
	return nil
}
