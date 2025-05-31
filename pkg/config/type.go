package config

type MeterCollectorConfig struct {
	InterpreterAPIHost string `toml:"interpreter_api_host"`
	TLSEnabled         bool   `toml:"tls_enabled"`
}

type InterpreterAPIConfig struct {
	SerialDevice  string `toml:"serial_device"`
	Baudrate      uint   `toml:"baudrate"`
	ListenAddress string `toml:"listen_address"`
	ListenPort    int    `toml:"listen_port"`
}
