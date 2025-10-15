package meterdb

type MeterDbPowerReadingType uint8

const (
	PowerConsumptionDay   MeterDbPowerReadingType = 0
	PowerConsumptionNight                         = 1
	PowerProductionDay                            = 2
	PowerProductionNight                          = 3
)

type MeterDbLivePowerReading struct {
	Timestamp   int64                   `db:"timestamp"`
	Watt        uint32                  `db:"watt"`
	ReadingType MeterDbPowerReadingType `db:"reading_type"`
}

type MeterDbTotalPowerReading struct {
	Timestamp   int64                   `db:"timestamp"`
	Watthour    uint32                  `db:"watthour"`
	ReadingType MeterDbPowerReadingType `db:"reading_type"`
}

type MeterDbTotalGasReading struct {
	Timestamp           int64  `db:"timestamp"`
	TotalConsumptionDM3 uint32 `db:"consumption_dm3"`
}
