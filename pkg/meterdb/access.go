package meterdb

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
