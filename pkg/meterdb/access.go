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
			"(timestamp, consumption_day_wh, production_day_wh, consumption_night_wh, production_night_wh) "+
			"VALUES (?, ?, ?, ?, ?)",
		reading.Timestamp,
		reading.TotalConsumptionDayWh,
		reading.TotalProductionDayWh,
		reading.TotalConsumptionNightWh,
		reading.TotalProductionNightWh,
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
