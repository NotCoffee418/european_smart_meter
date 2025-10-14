package aggregator

import (
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/NotCoffee418/european_smart_meter/pkg/meterdb"
)

var (
	ErrTimeframeNotCompleted = errors.New("timeframe is not yet completed")
	ErrAggregateExists       = errors.New("aggregate already exists for this timeframe")
	ErrSnapshotExists        = errors.New("snapshot already exists for this timeframe")
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
	now := time.Now().UTC()
	currentHourStart := roundToHourStart(now)

	// Check if trying to aggregate current or future hour
	if hourStart >= currentHourStart {
		return ErrTimeframeNotCompleted
	}

	// Check if aggregate already exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM aggregate_live_power_hourly WHERE hour_start = ?)", hourStart).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrAggregateExists
	}

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

	// Store average watts directly (not watthours)
	consumptionDayWatt := uint32(aggregateData[meterdb.PowerConsumptionDay])
	consumptionNightWatt := uint32(aggregateData[meterdb.PowerConsumptionNight])
	productionDayWatt := uint32(aggregateData[meterdb.PowerProductionDay])
	productionNightWatt := uint32(aggregateData[meterdb.PowerProductionNight])

	// Insert or replace the aggregate
	insertQuery := `
		INSERT OR REPLACE INTO aggregate_live_power_hourly 
		(hour_start, consumption_day_watt, consumption_night_watt, production_day_watt, production_night_watt, sample_count)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(insertQuery, hourStart, consumptionDayWatt, consumptionNightWatt, productionDayWatt, productionNightWatt, totalSampleCount)
	return err
}

// aggregateLivePowerDaily aggregates live power readings for a specific day
func aggregateLivePowerDaily(dayStart int64) error {
	db := meterdb.GetDB()
	now := time.Now().UTC()
	currentDayStart := roundToDayStart(now)

	// Check if trying to aggregate current or future day
	if dayStart >= currentDayStart {
		return ErrTimeframeNotCompleted
	}

	// Check if aggregate already exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM aggregate_live_power_daily WHERE day_start = ?)", dayStart).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrAggregateExists
	}

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

	// Store average watts directly (not watthours)
	consumptionDayWatt := uint32(aggregateData[meterdb.PowerConsumptionDay])
	consumptionNightWatt := uint32(aggregateData[meterdb.PowerConsumptionNight])
	productionDayWatt := uint32(aggregateData[meterdb.PowerProductionDay])
	productionNightWatt := uint32(aggregateData[meterdb.PowerProductionNight])

	// Insert or replace the aggregate
	insertQuery := `
		INSERT OR REPLACE INTO aggregate_live_power_daily 
		(day_start, consumption_day_watt, consumption_night_watt, production_day_watt, production_night_watt, sample_count)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(insertQuery, dayStart, consumptionDayWatt, consumptionNightWatt, productionDayWatt, productionNightWatt, totalSampleCount)
	return err
}

// aggregateLivePowerMonthly aggregates live power readings for a specific month
func aggregateLivePowerMonthly(monthStart int64) error {
	db := meterdb.GetDB()
	now := time.Now().UTC()
	currentMonthStart := roundToMonthStart(now)

	// Check if trying to aggregate current or future month
	if monthStart >= currentMonthStart {
		return ErrTimeframeNotCompleted
	}

	// Check if aggregate already exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM aggregate_live_power_monthly WHERE month_start = ?)", monthStart).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrAggregateExists
	}

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

	// Store average watts directly (not watthours)
	consumptionDayWatt := uint32(aggregateData[meterdb.PowerConsumptionDay])
	consumptionNightWatt := uint32(aggregateData[meterdb.PowerConsumptionNight])
	productionDayWatt := uint32(aggregateData[meterdb.PowerProductionDay])
	productionNightWatt := uint32(aggregateData[meterdb.PowerProductionNight])

	// Insert or replace the aggregate
	insertQuery := `
		INSERT OR REPLACE INTO aggregate_live_power_monthly 
		(month_start, consumption_day_watt, consumption_night_watt, production_day_watt, production_night_watt, sample_count)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(insertQuery, monthStart, consumptionDayWatt, consumptionNightWatt, productionDayWatt, productionNightWatt, totalSampleCount)
	return err
}

// snapshotTotalGasHourly creates a snapshot of gas readings for a specific hour
func snapshotTotalGasHourly(hourStart int64) error {
	db := meterdb.GetDB()
	now := time.Now().UTC()
	currentHourStart := roundToHourStart(now)

	// Check if trying to snapshot current or future hour
	if hourStart >= currentHourStart {
		return ErrTimeframeNotCompleted
	}

	// Check if snapshot already exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM snapshot_total_gas_hourly WHERE timestamp = ?)", hourStart).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrSnapshotExists
	}

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
	err = db.QueryRow(query, hourStart, hourEnd).Scan(&dm3Standing)
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
	now := time.Now().UTC()
	currentHourStart := roundToHourStart(now)

	// Check if trying to snapshot current or future hour
	if hourStart >= currentHourStart {
		return ErrTimeframeNotCompleted
	}

	// Check if snapshot already exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM snapshot_total_power_hourly WHERE timestamp = ?)", hourStart).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrSnapshotExists
	}

	hourEnd := getHourEnd(hourStart)

	// For power, we look 24 hours before the hour end (not just within the hour)
	lookbackStart := hourEnd - (24 * 3600)

	// Helper function to get the last known reading for a specific type
	getLastReading := func(readingType meterdb.MeterDbPowerReadingType) (uint32, bool) {
		query := `
			SELECT watthour
			FROM total_power_readings
			WHERE reading_type = ? AND timestamp <= ? AND timestamp >= ?
			ORDER BY timestamp DESC
			LIMIT 1
		`

		var watthour uint32
		err := db.QueryRow(query, readingType, hourEnd, lookbackStart).Scan(&watthour)
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

	_, err = db.Exec(insertQuery, hourStart, consumptionDayStanding, consumptionNightStanding, productionDayStanding, productionNightStanding)
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
	var lastAggregateHour sql.NullInt64
	err := db.QueryRow("SELECT MAX(hour_start) FROM aggregate_live_power_hourly").Scan(&lastAggregateHour)
	if err != nil {
		return err
	}

	// If no aggregates exist yet, don't clean up
	if !lastAggregateHour.Valid {
		return nil
	}

	// Only clean up if we have aggregated data up to the cutoff point
	if lastAggregateHour.Int64 < cutoffTimestamp {
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
	db := meterdb.GetDB()

	// Process hourly aggregates - backfill if needed
	if err := processHourlyAggregates(db, now); err != nil {
		return err
	}

	// Process daily aggregates - backfill if needed
	if err := processDailyAggregates(db, now); err != nil {
		return err
	}

	// Process monthly aggregates - backfill if needed
	if err := processMonthlyAggregates(db, now); err != nil {
		return err
	}

	// Run cleanup
	if err := cleanupOldData(); err != nil {
		log.Printf("Error cleaning up old data: %v", err)
		return err
	}

	log.Println("Aggregation and cleanup completed successfully")
	return nil
}

// processHourlyAggregates creates hourly aggregates and snapshots, backfilling from earliest data if needed
func processHourlyAggregates(db *sql.DB, now time.Time) error {
	currentHourStart := roundToHourStart(now)

	// Find the last hourly aggregate
	var lastAggregateHour sql.NullInt64
	err := db.QueryRow("SELECT MAX(hour_start) FROM aggregate_live_power_hourly").Scan(&lastAggregateHour)
	if err != nil {
		return err
	}

	var startHour int64
	if !lastAggregateHour.Valid {
		// No aggregates exist yet, find earliest data point
		var earliestReading sql.NullInt64
		err := db.QueryRow("SELECT MIN(timestamp) FROM live_power_readings").Scan(&earliestReading)
		if err != nil {
			return err
		}

		if !earliestReading.Valid {
			// No data to aggregate
			log.Println("No live power readings found, skipping hourly aggregation")
			return nil
		}

		startHour = roundToHourStart(time.Unix(earliestReading.Int64, 0))
		log.Printf("Starting hourly aggregation from earliest data: %s", time.Unix(startHour, 0).Format(time.RFC3339))
	} else {
		// Start from the next hour after the last aggregate
		startHour = lastAggregateHour.Int64 + 3600
	}

	// Aggregate all hours from startHour up to (but not including) current hour
	hoursProcessed := 0
	for hourStart := startHour; hourStart < currentHourStart; hourStart += 3600 {
		if err := aggregateLivePowerHourly(hourStart); err != nil {
			if err == ErrAggregateExists {
				continue // Skip if already exists
			}
			if err == ErrTimeframeNotCompleted {
				break // Stop if we hit current timeframe
			}
			log.Printf("Error aggregating hourly live power for %s: %v", time.Unix(hourStart, 0).Format(time.RFC3339), err)
			return err
		}

		if err := snapshotTotalGasHourly(hourStart); err != nil {
			if err != ErrSnapshotExists {
				log.Printf("Error creating gas snapshot for %s: %v", time.Unix(hourStart, 0).Format(time.RFC3339), err)
				// Continue even if snapshot fails - it's not critical
			}
		}

		if err := snapshotTotalPowerHourly(hourStart); err != nil {
			if err != ErrSnapshotExists {
				log.Printf("Error creating power snapshot for %s: %v", time.Unix(hourStart, 0).Format(time.RFC3339), err)
				// Continue even if snapshot fails - it's not critical
			}
		}

		hoursProcessed++
	}

	if hoursProcessed > 0 {
		log.Printf("Processed %d hourly aggregates/snapshots", hoursProcessed)
	}

	return nil
}

// processDailyAggregates creates daily aggregates, backfilling from earliest data if needed
func processDailyAggregates(db *sql.DB, now time.Time) error {
	currentDayStart := roundToDayStart(now)

	// Find the last daily aggregate
	var lastAggregateDay sql.NullInt64
	err := db.QueryRow("SELECT MAX(day_start) FROM aggregate_live_power_daily").Scan(&lastAggregateDay)
	if err != nil {
		return err
	}

	var startDay int64
	if !lastAggregateDay.Valid {
		// No aggregates exist yet, find earliest data point
		var earliestReading sql.NullInt64
		err := db.QueryRow("SELECT MIN(timestamp) FROM live_power_readings").Scan(&earliestReading)
		if err != nil {
			return err
		}

		if !earliestReading.Valid {
			// No data to aggregate
			return nil
		}

		startDay = roundToDayStart(time.Unix(earliestReading.Int64, 0))
		log.Printf("Starting daily aggregation from earliest data: %s", time.Unix(startDay, 0).Format(time.RFC3339))
	} else {
		// Start from the next day after the last aggregate
		startDay = lastAggregateDay.Int64 + 86400
	}

	// Aggregate all days from startDay up to (but not including) current day
	daysProcessed := 0
	for dayStart := startDay; dayStart < currentDayStart; dayStart += 86400 {
		if err := aggregateLivePowerDaily(dayStart); err != nil {
			if err == ErrAggregateExists {
				continue // Skip if already exists
			}
			if err == ErrTimeframeNotCompleted {
				break // Stop if we hit current timeframe
			}
			log.Printf("Error aggregating daily live power for %s: %v", time.Unix(dayStart, 0).Format(time.RFC3339), err)
			return err
		}
		daysProcessed++
	}

	if daysProcessed > 0 {
		log.Printf("Processed %d daily aggregates", daysProcessed)
	}

	return nil
}

// processMonthlyAggregates creates monthly aggregates, backfilling from earliest data if needed
func processMonthlyAggregates(db *sql.DB, now time.Time) error {
	currentMonthStart := roundToMonthStart(now)

	// Find the last monthly aggregate
	var lastAggregateMonth sql.NullInt64
	err := db.QueryRow("SELECT MAX(month_start) FROM aggregate_live_power_monthly").Scan(&lastAggregateMonth)
	if err != nil {
		return err
	}

	var startMonth int64
	if !lastAggregateMonth.Valid {
		// No aggregates exist yet, find earliest data point
		var earliestReading sql.NullInt64
		err := db.QueryRow("SELECT MIN(timestamp) FROM live_power_readings").Scan(&earliestReading)
		if err != nil {
			return err
		}

		if !earliestReading.Valid {
			// No data to aggregate
			return nil
		}

		startMonth = roundToMonthStart(time.Unix(earliestReading.Int64, 0))
		log.Printf("Starting monthly aggregation from earliest data: %s", time.Unix(startMonth, 0).Format(time.RFC3339))
	} else {
		// Start from the next month after the last aggregate
		t := time.Unix(lastAggregateMonth.Int64, 0)
		startMonth = roundToMonthStart(t.AddDate(0, 1, 0))
	}

	// Aggregate all months from startMonth up to (but not including) current month
	monthsProcessed := 0
	for monthStart := startMonth; monthStart < currentMonthStart; {
		if err := aggregateLivePowerMonthly(monthStart); err != nil {
			if err == ErrAggregateExists {
				// Move to next month
				t := time.Unix(monthStart, 0)
				monthStart = roundToMonthStart(t.AddDate(0, 1, 0))
				continue
			}
			if err == ErrTimeframeNotCompleted {
				break // Stop if we hit current timeframe
			}
			log.Printf("Error aggregating monthly live power for %s: %v", time.Unix(monthStart, 0).Format(time.RFC3339), err)
			return err
		}
		monthsProcessed++

		// Move to next month
		t := time.Unix(monthStart, 0)
		monthStart = roundToMonthStart(t.AddDate(0, 1, 0))
	}

	if monthsProcessed > 0 {
		log.Printf("Processed %d monthly aggregates", monthsProcessed)
	}

	return nil
}
