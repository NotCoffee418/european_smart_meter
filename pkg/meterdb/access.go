package meterdb

import "database/sql"

func InsertLivePowerReading(reading *MeterDbLivePowerReading) error {
	db := GetDB()

	_, err := db.Exec(
		"INSERT INTO live_power_readings (timestamp, watt, reading_type) "+
			"VALUES (?, ?, ?)",
		reading.Timestamp,
		reading.Watt,
		reading.ReadingType,
	)
	if err != nil {
		return err
	}
	return nil
}

func InsertTotalPowerReading(reading *MeterDbTotalPowerReading) error {
	db := GetDB()

	_, err := db.Exec(
		"INSERT INTO total_power_readings "+
			"(timestamp, watthour, reading_type) "+
			"VALUES (?, ?, ?)",
		reading.Timestamp,
		reading.Watthour,
		reading.ReadingType,
	)
	if err != nil {
		return err
	}
	return nil
}

func InsertTotalGasReading(reading *MeterDbTotalGasReading) error {
	db := GetDB()

	_, err := db.Exec(
		"INSERT INTO total_gas_readings "+
			"(timestamp, consumption_dm3) "+
			"VALUES (?, ?)",
		reading.Timestamp,
		reading.TotalConsumptionDM3,
	)
	if err != nil {
		return err
	}
	return nil
}

func GetLastTotalPowerReading(readingType MeterDbPowerReadingType) (*MeterDbTotalPowerReading, error) {
	db := GetDB()

	var reading MeterDbTotalPowerReading
	err := db.QueryRow("SELECT timestamp, watthour, reading_type "+
		"FROM total_power_readings WHERE reading_type = ? ORDER BY timestamp DESC LIMIT 1",
		readingType,
	).Scan(&reading.Timestamp, &reading.Watthour, &reading.ReadingType)
	if err != nil {
		return nil, err
	}
	return &reading, nil
}

// Returns nil if not in DB
func GetAggregatePowerRow(timeframeStr string, startTime int64) (*AggregateLivePowerTable, error) {
	db := GetDB()

	var row AggregateLivePowerTable
	err := db.QueryRow("SELECT * FROM aggregate_live_power_"+timeframeStr+" WHERE start_time = ?", startTime).Scan(&row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &row, err
}

// Returns map of reading type to slice of watt readings
// All types are guaranteed
func GetLivePowerMapInTimeframe(startTime int64, endTime int64) (map[MeterDbPowerReadingType][]uint32, error) {
	db := GetDB()

	rows, err := db.Query("SELECT * FROM live_power_readings WHERE timestamp >= ? AND timestamp < ?", startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	powerMap := make(map[MeterDbPowerReadingType][]uint32)
	powerMap[PowerConsumptionDay] = []uint32{}
	powerMap[PowerConsumptionNight] = []uint32{}
	powerMap[PowerProductionDay] = []uint32{}
	powerMap[PowerProductionNight] = []uint32{}
	for rows.Next() {
		var reading MeterDbLivePowerReading
		err := rows.Scan(&reading.Timestamp, &reading.Watt, &reading.ReadingType)
		if err != nil {
			return nil, err
		}
		powerMap[reading.ReadingType] = append(powerMap[reading.ReadingType], reading.Watt)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return powerMap, nil

}

func GetLastAggregatePowerRow(timeframeStr string) (*AggregateLivePowerTable, error) {
	db := GetDB()
	var row AggregateLivePowerTable
	err := db.QueryRow("SELECT start_time FROM aggregate_live_power_" + timeframeStr + " ORDER BY start_time DESC LIMIT 1").Scan(&row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func GetEarliestLivePowerTime() (int64, error) {
	db := GetDB()
	var ts int64
	err := db.QueryRow("SELECT timestamp FROM live_power_readings ORDER BY timestamp ASC LIMIT 1").Scan(&ts)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return ts, nil
}

func InsertLivePowerAggregate(timeframeStr string, aggregate *AggregateLivePowerTable) error {
	db := GetDB()

	_, err := db.Exec(
		"INSERT INTO aggregate_live_power_"+timeframeStr+" "+
			"(start_time, consumption_day_wh, consumption_day_sample_count, consumption_night_wh, consumption_night_sample_count, "+
			"production_day_wh, production_day_sample_count, production_night_wh, production_night_sample_count) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		aggregate.StartTime,
		aggregate.ConsumptionDayWh,
		aggregate.ConsumptionDaySampleCount,
		aggregate.ConsumptionNightWh,
		aggregate.ConsumptionNightSampleCount,
		aggregate.ProductionDayWh,
		aggregate.ProductionDaySampleCount,
		aggregate.ProductionNightWh,
		aggregate.ProductionNightSampleCount,
	)
	if err != nil {
		return err
	}
	return nil
}

func DeleteLivePowerReadingsBefore(cutoff int64) error {
	db := GetDB()
	_, err := db.Exec("DELETE FROM live_power_readings WHERE timestamp < ?;", cutoff)
	return err
}

func Vacuum() error {
	db := GetDB()
	_, err := db.Exec("VACUUM")
	return err
}
