package config

type MeterCollectorConfig struct {
	InterpreterAPIHost string `toml:"interpreter_api_host"`
	TLSEnabled         bool   `toml:"tls_enabled"`
}

type InterpreterAPIConfig struct {
	SerialDevice            string `toml:"serial_device"`
	Baudrate                uint   `toml:"baudrate"`
	ListenAddress           string `toml:"listen_address"`
	ListenPort              int    `toml:"listen_port"`
	SolarInverterIp         string `toml:"solar_inverter_ip"`
	SolarInverterModbusPort int    `toml:"solar_inverter_modbus_port"`
	// Should be named `preconfigured`
	// Check with `nmcli device status`
	WlanConnectionId string `toml:"wlan_connection_id"`
}
