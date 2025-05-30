package types

type MeterDbPowerReading struct {
	Timestamp                 uint64                  `db:"timestamp"`
	Watt                      uint32                  `db:"watt"`
	ReadingType               MeterDbPowerReadingType `db:"reading_type"`
	TotalConsumptionWattHours uint32                  `db:"total_consumption_watt_hours"`
	TotalProductionWattHours  uint32                  `db:"total_production_watt_hours"`
}

type MeterDbGasReading struct {
	Timestamp     uint64 `db:"timestamp"`
	CubicMeter    uint32 `db:"cubic_meter"`
}

type MeterDbPowerReadingType uint8

const (
	PowerConsumptionDay MeterDbPowerReadingType = iota
	PowerConsumptionNight
	PowerProductionDay
	PowerProductionNight
)
