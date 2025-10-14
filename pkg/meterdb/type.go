package meterdb

type MeterDbPowerReadingType uint8

const (
	PowerConsumptionDay   MeterDbPowerReadingType = 0
	PowerConsumptionNight MeterDbPowerReadingType = 1
	PowerProductionDay    MeterDbPowerReadingType = 2
	PowerProductionNight  MeterDbPowerReadingType = 3
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

// Aggregate models - computed consumption deltas
type AggregateLivePowerHourly struct {
	HourStart            int64  `db:"hour_start"`
	ConsumptionDayWatt   uint32 `db:"consumption_day_watt"`
	ConsumptionNightWatt uint32 `db:"consumption_night_watt"`
	ProductionDayWatt    uint32 `db:"production_day_watt"`
	ProductionNightWatt  uint32 `db:"production_night_watt"`
	SampleCount          uint32 `db:"sample_count"`
}

type AggregateLivePowerDaily struct {
	DayStart             int64  `db:"day_start"`
	ConsumptionDayWatt   uint32 `db:"consumption_day_watt"`
	ConsumptionNightWatt uint32 `db:"consumption_night_watt"`
	ProductionDayWatt    uint32 `db:"production_day_watt"`
	ProductionNightWatt  uint32 `db:"production_night_watt"`
	SampleCount          uint32 `db:"sample_count"`
}

type AggregateLivePowerMonthly struct {
	MonthStart           int64  `db:"month_start"`
	ConsumptionDayWatt   uint32 `db:"consumption_day_watt"`
	ConsumptionNightWatt uint32 `db:"consumption_night_watt"`
	ProductionDayWatt    uint32 `db:"production_day_watt"`
	ProductionNightWatt  uint32 `db:"production_night_watt"`
	SampleCount          uint32 `db:"sample_count"`
}

// Snapshot models - retained meter readings
type SnapshotTotalPowerHourly struct {
	Timestamp                int64  `db:"timestamp"`
	ConsumptionDayStanding   uint32 `db:"consumption_day_standing"`
	ConsumptionNightStanding uint32 `db:"consumption_night_standing"`
	ProductionDayStanding    uint32 `db:"production_day_standing"`
	ProductionNightStanding  uint32 `db:"production_night_standing"`
}

type SnapshotTotalGasHourly struct {
	Timestamp   int64  `db:"timestamp"`
	Dm3Standing uint32 `db:"dm3_standing"`
}
