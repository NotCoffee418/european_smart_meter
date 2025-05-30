package types

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
