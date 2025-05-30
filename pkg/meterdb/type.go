package meterdb

type MeterDbPowerReadingType uint8

const (
	PowerConsumptionDay MeterDbPowerReadingType = iota
	PowerConsumptionNight
	PowerProductionDay
	PowerProductionNight
)

type MeterDbLivePowerReading struct {
	Timestamp   int64                   `db:"timestamp"`
	Watt        uint32                  `db:"watt"`
	ReadingType MeterDbPowerReadingType `db:"reading_type"`
}

type MeterDbTotalPowerReading struct {
	Timestamp               int64  `db:"timestamp"`
	TotalConsumptionDayWh   uint32 `db:"consumption_day_wh"`
	TotalProductionDayWh    uint32 `db:"production_day_wh"`
	TotalConsumptionNightWh uint32 `db:"consumption_night_wh"`
	TotalProductionNightWh  uint32 `db:"production_night_wh"`
}

type MeterDbTotalGasReading struct {
	Timestamp           int64  `db:"timestamp"`
	TotalConsumptionDM3 uint32 `db:"consumption_dm3"`
}
