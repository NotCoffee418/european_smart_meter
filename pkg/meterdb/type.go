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
// Use timeframe specified types instead of this directly
type AggregateLivePowerTable struct {
	StartTime                   int64  `db:"start_time"`
	ConsumptionDayWh            uint32 `db:"consumption_day_wh"`
	ConsumptionDaySampleCount   uint32 `db:"consumption_day_sample_count"`
	ConsumptionNightWh          uint32 `db:"consumption_night_wh"`
	ConsumptionNightSampleCount uint32 `db:"consumption_night_sample_count"`
	ProductionDayWh             uint32 `db:"production_day_wh"`
	ProductionDaySampleCount    uint32 `db:"production_day_sample_count"`
	ProductionNightWh           uint32 `db:"production_night_wh"`
	ProductionNightSampleCount  uint32 `db:"production_night_sample_count"`
}

type AggregateLivePowerHourly = AggregateLivePowerTable
type AggregateLivePowerDaily = AggregateLivePowerTable
type AggregateLivePowerMonthly = AggregateLivePowerTable

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
