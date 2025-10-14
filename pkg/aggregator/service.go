package aggregator

import (
	"database/sql"
	"log"
	"time"

	"github.com/NotCoffee418/european_smart_meter/pkg/meterdb"
)

// roundToHourStart returns the Unix timestamp of the start of the hour for the given time
func roundToHourStart(t time.Time) int64 {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC).Unix()
}

// roundToDayStart returns the Unix timestamp of the start of the day for the given time
func roundToDayStart(t time.Time) int64 {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
}

// roundToMonthStart returns the Unix timestamp of the start of the month for the given time
func roundToMonthStart(t time.Time) int64 {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()
}

// getHourEnd returns the Unix timestamp of the last second of the hour (next hour start - 1)
func getHourEnd(hourStart int64) int64 {
	return time.Unix(hourStart, 0).Add(time.Hour).Unix() - 1
}

// getDayEnd returns the Unix timestamp of the last second of the day (next day start - 1)
func getDayEnd(dayStart int64) int64 {
	return time.Unix(dayStart, 0).AddDate(0, 0, 1).Unix() - 1
}

// getMonthEnd returns the Unix timestamp of the last second of the month (next month start - 1)
func getMonthEnd(monthStart int64) int64 {
	t := time.Unix(monthStart, 0)
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC).Unix() - 1
}

// aggregateLivePowerHourly aggregates live power readings for a specific hour
func aggregateLivePowerHourly(hourStart int64) error {
	db := meterdb.GetDB()
	hourEnd := getHourEnd(hourStart)

	// Query to calculate averages grouped by reading_type
	query := `
		SELECT 
			reading_type,
			AVG(watt) as avg_watt,
			COUNT(*) as count
		FROM live_power_readings
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY reading_type
	`

	rows, err := db.Query(query, hourStart, hourEnd)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Collect data for each reading type
	aggregateData := make(map[meterdb.MeterDbPowerReadingType]float64)
	var totalSampleCount uint32 = 0

	for rows.Next() {
		var readingType meterdb.MeterDbPowerReadingType
		var avgWatt float64
		var count uint32

		if err := rows.Scan(&readingType, &avgWatt, &count); err != nil {
			return err
		}

		aggregateData[readingType] = avgWatt
		totalSampleCount += count
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Only insert if we have data
	if totalSampleCount == 0 {
		return nil
	}

	// Convert watts to watthours (average watt for 1 hour = watthours)
	consumptionDayWh := uint32(aggregateData[meterdb.PowerConsumptionDay])
	consumptionNightWh := uint32(aggregateData[meterdb.PowerConsumptionNight])
	productionDayWh := uint32(aggregateData[meterdb.PowerProductionDay])
	productionNightWh := uint32(aggregateData[meterdb.PowerProductionNight])

	// Insert or replace the aggregate
	insertQuery := `
		INSERT OR REPLACE INTO aggregate_live_power_hourly 
		(hour_start, consumption_day_wh, consumption_night_wh, production_day_wh, production_night_wh, sample_count)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(insertQuery, hourStart, consumptionDayWh, consumptionNightWh, productionDayWh, productionNightWh, totalSampleCount)
	return err
}

// aggregateLivePowerDaily aggregates live power readings for a specific day
func aggregateLivePowerDaily(dayStart int64) error {
	db := meterdb.GetDB()
	dayEnd := getDayEnd(dayStart)

	// Query to calculate averages grouped by reading_type
	query := `
		SELECT 
			reading_type,
			AVG(watt) as avg_watt,
			COUNT(*) as count
		FROM live_power_readings
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY reading_type
	`

	rows, err := db.Query(query, dayStart, dayEnd)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Collect data for each reading type
	aggregateData := make(map[meterdb.MeterDbPowerReadingType]float64)
	var totalSampleCount uint32 = 0

	for rows.Next() {
		var readingType meterdb.MeterDbPowerReadingType
		var avgWatt float64
		var count uint32

		if err := rows.Scan(&readingType, &avgWatt, &count); err != nil {
			return err
		}

		aggregateData[readingType] = avgWatt
		totalSampleCount += count
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Only insert if we have data
	if totalSampleCount == 0 {
		return nil
	}

	// For daily aggregates, we need to multiply by 24 hours
	consumptionDayWh := uint32(aggregateData[meterdb.PowerConsumptionDay] * 24)
	consumptionNightWh := uint32(aggregateData[meterdb.PowerConsumptionNight] * 24)
	productionDayWh := uint32(aggregateData[meterdb.PowerProductionDay] * 24)
	productionNightWh := uint32(aggregateData[meterdb.PowerProductionNight] * 24)

	// Insert or replace the aggregate
	insertQuery := `
		INSERT OR REPLACE INTO aggregate_live_power_daily 
		(day_start, consumption_day_wh, consumption_night_wh, production_day_wh, production_night_wh, sample_count)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(insertQuery, dayStart, consumptionDayWh, consumptionNightWh, productionDayWh, productionNightWh, totalSampleCount)
	return err
}

// aggregateLivePowerMonthly aggregates live power readings for a specific month
func aggregateLivePowerMonthly(monthStart int64) error {
	db := meterdb.GetDB()
	monthEnd := getMonthEnd(monthStart)

	// Query to calculate averages grouped by reading_type
	query := `
		SELECT 
			reading_type,
			AVG(watt) as avg_watt,
			COUNT(*) as count
		FROM live_power_readings
		WHERE timestamp >= ? AND timestamp <= ?
		GROUP BY reading_type
	`

	rows, err := db.Query(query, monthStart, monthEnd)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Collect data for each reading type
	aggregateData := make(map[meterdb.MeterDbPowerReadingType]float64)
	var totalSampleCount uint32 = 0

	for rows.Next() {
		var readingType meterdb.MeterDbPowerReadingType
		var avgWatt float64
		var count uint32

		if err := rows.Scan(&readingType, &avgWatt, &count); err != nil {
			return err
		}

		aggregateData[readingType] = avgWatt
		totalSampleCount += count
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Only insert if we have data
	if totalSampleCount == 0 {
		return nil
	}

	// For monthly aggregates, calculate hours in the month
	t := time.Unix(monthStart, 0)
	daysInMonth := time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	hoursInMonth := float64(daysInMonth * 24)

	consumptionDayWh := uint32(aggregateData[meterdb.PowerConsumptionDay] * hoursInMonth)
	consumptionNightWh := uint32(aggregateData[meterdb.PowerConsumptionNight] * hoursInMonth)
	productionDayWh := uint32(aggregateData[meterdb.PowerProductionDay] * hoursInMonth)
	productionNightWh := uint32(aggregateData[meterdb.PowerProductionNight] * hoursInMonth)

	// Insert or replace the aggregate
	insertQuery := `
		INSERT OR REPLACE INTO aggregate_live_power_monthly 
		(month_start, consumption_day_wh, consumption_night_wh, production_day_wh, production_night_wh, sample_count)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(insertQuery, monthStart, consumptionDayWh, consumptionNightWh, productionDayWh, productionNightWh, totalSampleCount)
	return err
}

// snapshotTotalGasHourly creates a snapshot of gas readings for a specific hour
func snapshotTotalGasHourly(hourStart int64) error {
	db := meterdb.GetDB()
	hourEnd := getHourEnd(hourStart)

	// Get the last known reading within the timespan
	query := `
		SELECT consumption_dm3
		FROM total_gas_readings
		WHERE timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var dm3Standing uint32
	err := db.QueryRow(query, hourStart, hourEnd).Scan(&dm3Standing)
	if err != nil {
		if err == sql.ErrNoRows {
			// No entry within timeframe, that's okay
			return nil
		}
		return err
	}

	// Insert or replace the snapshot
	insertQuery := `
		INSERT OR REPLACE INTO snapshot_total_gas_hourly 
		(timestamp, dm3_standing)
		VALUES (?, ?)
	`

	_, err = db.Exec(insertQuery, hourStart, dm3Standing)
	return err
}

// snapshotTotalPowerHourly creates a snapshot of power readings for a specific hour
func snapshotTotalPowerHourly(hourStart int64) error {
	db := meterdb.GetDB()
	hourEnd := getHourEnd(hourStart)
	
	// For power, we look 24 hours before the end of the timeframe
	lookbackStart := hourEnd - (24 * 3600)

	// Helper function to get the last known reading for a specific type
	getLastReading := func(readingType meterdb.MeterDbPowerReadingType) (uint32, bool) {
		query := `
			SELECT watthour
			FROM total_power_readings
			WHERE reading_type = ? AND timestamp >= ? AND timestamp <= ?
			ORDER BY timestamp DESC
			LIMIT 1
		`

		var watthour uint32
		err := db.QueryRow(query, readingType, lookbackStart, hourEnd).Scan(&watthour)
		if err != nil {
			if err == sql.ErrNoRows {
				return 0, false
			}
			log.Printf("Error querying power reading type %d: %v", readingType, err)
			return 0, false
		}
		return watthour, true
	}

	// Get readings for each type
	consumptionDayStanding, hasConsumptionDay := getLastReading(meterdb.PowerConsumptionDay)
	consumptionNightStanding, hasConsumptionNight := getLastReading(meterdb.PowerConsumptionNight)
	productionDayStanding, hasProductionDay := getLastReading(meterdb.PowerProductionDay)
	productionNightStanding, hasProductionNight := getLastReading(meterdb.PowerProductionNight)

	// Only create snapshot if we have at least one reading
	if !hasConsumptionDay && !hasConsumptionNight && !hasProductionDay && !hasProductionNight {
		return nil
	}

	// Insert or replace the snapshot
	insertQuery := `
		INSERT OR REPLACE INTO snapshot_total_power_hourly 
		(timestamp, consumption_day_standing, consumption_night_standing, production_day_standing, production_night_standing)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := db.Exec(insertQuery, hourStart, consumptionDayStanding, consumptionNightStanding, productionDayStanding, productionNightStanding)
	return err
}

// cleanupOldData removes raw data older than 3 months if we have aggregated it
func cleanupOldData() error {
	db := meterdb.GetDB()
	
	// Calculate the cutoff timestamp (3 months ago)
	threeMonthsAgo := time.Now().UTC().AddDate(0, -3, 0)
	cutoffTimestamp := threeMonthsAgo.Unix()

	// Check if we have aggregated data up to the cutoff point
	// We check the last hourly aggregate to see if we've aggregated recent enough data
	var lastAggregateHour int64
	err := db.QueryRow("SELECT MAX(hour_start) FROM aggregate_live_power_hourly").Scan(&lastAggregateHour)
	if err != nil {
		if err == sql.ErrNoRows {
			// No aggregates yet, don't clean up
			return nil
		}
		return err
	}

	// Only clean up if we have aggregated data up to the cutoff point
	if lastAggregateHour < cutoffTimestamp {
		// We haven't aggregated enough data yet, don't clean up
		return nil
	}

	// Delete old live power readings
	_, err = db.Exec("DELETE FROM live_power_readings WHERE timestamp < ?", cutoffTimestamp)
	if err != nil {
		return err
	}

	// Delete old total power readings
	_, err = db.Exec("DELETE FROM total_power_readings WHERE timestamp < ?", cutoffTimestamp)
	if err != nil {
		return err
	}

	// Delete old total gas readings
	_, err = db.Exec("DELETE FROM total_gas_readings WHERE timestamp < ?", cutoffTimestamp)
	if err != nil {
		return err
	}

	log.Printf("Cleaned up data older than %s", threeMonthsAgo.Format(time.RFC3339))
	return nil
}

// AggregateAndCleanup performs all aggregation and cleanup tasks
// This is the main function to call for data aggregation
func AggregateAndCleanup() error {
	now := time.Now().UTC()

	// Aggregate the previous hour (current hour is still ongoing)
	previousHour := now.Add(-time.Hour)
	hourStart := roundToHourStart(previousHour)
	
	log.Printf("Aggregating data for hour starting at %s", time.Unix(hourStart, 0).Format(time.RFC3339))
	
	if err := aggregateLivePowerHourly(hourStart); err != nil {
		log.Printf("Error aggregating hourly live power: %v", err)
		return err
	}

	if err := snapshotTotalGasHourly(hourStart); err != nil {
		log.Printf("Error creating gas snapshot: %v", err)
		return err
	}

	if err := snapshotTotalPowerHourly(hourStart); err != nil {
		log.Printf("Error creating power snapshot: %v", err)
		return err
	}

	// Aggregate the previous day if it's a new day
	if now.Hour() == 0 {
		previousDay := now.AddDate(0, 0, -1)
		dayStart := roundToDayStart(previousDay)
		
		log.Printf("Aggregating data for day starting at %s", time.Unix(dayStart, 0).Format(time.RFC3339))
		
		if err := aggregateLivePowerDaily(dayStart); err != nil {
			log.Printf("Error aggregating daily live power: %v", err)
			return err
		}
	}

	// Aggregate the previous month if it's a new month
	if now.Hour() == 0 && now.Day() == 1 {
		previousMonth := now.AddDate(0, -1, 0)
		monthStart := roundToMonthStart(previousMonth)
		
		log.Printf("Aggregating data for month starting at %s", time.Unix(monthStart, 0).Format(time.RFC3339))
		
		if err := aggregateLivePowerMonthly(monthStart); err != nil {
			log.Printf("Error aggregating monthly live power: %v", err)
			return err
		}
	}

	// Run cleanup
	if err := cleanupOldData(); err != nil {
		log.Printf("Error cleaning up old data: %v", err)
		return err
	}

	log.Println("Aggregation and cleanup completed successfully")
	return nil
}
