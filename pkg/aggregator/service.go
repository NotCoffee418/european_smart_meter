package aggregator

import (
	"errors"
	"time"

	"github.com/NotCoffee418/european_smart_meter/pkg/meterdb"
	"github.com/labstack/gommon/log"
)

var (
	ErrTimeframeNotCompleted = errors.New("timeframe is not yet completed")
	ErrAggregateExists       = errors.New("aggregate already exists for this timeframe")
	ErrSnapshotExists        = errors.New("snapshot already exists for this timeframe")
	ErrInvalidTimeframe      = errors.New("invalid timeframe specified")
)

type Timeframe string

const (
	Hourly  Timeframe = "hourly"
	Daily             = "daily"
	Monthly           = "monthly"
)

var allTimeframes = []Timeframe{Hourly, Daily, Monthly}

// RunAggregationAndCleanup performs all aggregation and cleanup tasks
// This is the main function to call for data aggregation
func RunAggregationAndCleanup() error {
	// Retain `now` for duration of the function to avoid transitional shenanigans
	now := time.Now().UTC()

	// Ensure all data aggregates are up to date
	for _, tf := range allTimeframes {
		nowSpanStartTime := getTimeframeStart(now, tf)

		// Get last inserted aggregate row for this timeframe
		var lastProcessedStartTime int64 = 0
		lastAggregateInDb, err := meterdb.GetLastAggregatePowerRow(string(tf))
		if err != nil {
			return err
		}
		if lastAggregateInDb != nil {
			lastProcessedStartTime = lastAggregateInDb.StartTime
		}

		// If no previous aggregate, determine start time from earliest live power reading
		if lastAggregateInDb == nil {
			earliestLivePowerTime, err := meterdb.GetEarliestLivePowerTime()
			if err != nil {
				return err
			}
			if earliestLivePowerTime == 0 {
				// No data to process, first run.
				break
			}
			lastProcessedStartTime = getTimeframeStart(time.Unix(earliestLivePowerTime, 0), tf)
		}

		// Backfill aggregates up to but not including current timeframe
		nextSpanStart := getNextTimeframeStart(time.Unix(lastProcessedStartTime, 0), tf)
		for nextSpanStart < nowSpanStartTime {
			aggData, err := GetLivePowerAggregate(tf, nextSpanStart)
			if err != nil {
				return err
			}
			if aggData == nil {
				return errors.New("failed to generate aggregate data")
			}
			if aggData.IsInDb {
				// Should not happen, but just in case
				nextSpanStart = getNextTimeframeStart(time.Unix(nextSpanStart, 0), tf)
				log.Warn("skipping already existing aggregate for ", tf, " starting at ", nextSpanStart, ". This should not happen")
				continue
			}

			// Insert into DB
			err = meterdb.InsertLivePowerAggregate(string(tf), &aggData.Aggregate)
			if err != nil {
				return err
			}
			nextSpanStart = getNextTimeframeStart(time.Unix(nextSpanStart, 0), tf)
		}
	}

	// Aggregate storage complete for all timeframes without errors
	// Clean up data older start of previous month.
	cleanupBefore := getPrevTimeframeStart(now, Monthly)
	err := meterdb.DeleteLivePowerReadingsBefore(cleanupBefore)
	if err != nil {
		return err
	}

	// Vacuum the database to reclaim space
	err = meterdb.Vacuum()
	if err != nil {
		return err
	}

	return nil

}

// GetLivePowerAggregate retrieves an aggregate for the specified timeframe and start timestamp
// returns (aggregate, error)
func GetLivePowerAggregate(tf Timeframe, startTs int64) (*AggregateData, error) {
	// validate startTs
	startTime := time.Unix(startTs, 0)
	if getTimeframeStart(startTime, tf) != startTs {
		return nil, ErrInvalidTimeframe
	}

	res := AggregateData{
		Timeframe: tf,
		EndTime:   getTimeframeEnd(startTime, tf),
		IsInDb:    false,
	}

	// Check if timeframe is the current timeframe.
	// If it is, it implies the data is imcomplete and should not be added to db yet, and is only for preview.
	res.IsCurrentTimeframe = getTimeframeStart(time.Now().UTC(), tf) == startTs

	// Load data from DB if it exists
	row, err := meterdb.GetAggregatePowerRow(string(tf), startTs)
	if err != nil {
		return nil, err
	}
	if row != nil {
		res.IsInDb = true
		res.Aggregate = *row
		return &res, nil
	}

	// Calculate aggregate
	pMap, err := meterdb.GetLivePowerMapInTimeframe(startTs, res.EndTime)
	if err != nil {
		return nil, err
	}
	res.Aggregate = meterdb.AggregateLivePowerTable{
		StartTime:                   startTs,
		ConsumptionDayWh:            calcWattsecondsToWatthoursRounded(pMap[meterdb.PowerConsumptionDay]),
		ConsumptionDaySampleCount:   uint32(len(pMap[meterdb.PowerConsumptionDay])),
		ConsumptionNightWh:          calcWattsecondsToWatthoursRounded(pMap[meterdb.PowerConsumptionNight]),
		ConsumptionNightSampleCount: uint32(len(pMap[meterdb.PowerConsumptionNight])),
		ProductionDayWh:             calcWattsecondsToWatthoursRounded(pMap[meterdb.PowerProductionDay]),
		ProductionDaySampleCount:    uint32(len(pMap[meterdb.PowerProductionDay])),
		ProductionNightWh:           calcWattsecondsToWatthoursRounded(pMap[meterdb.PowerProductionNight]),
		ProductionNightSampleCount:  uint32(len(pMap[meterdb.PowerProductionNight])),
	}
	return &res, nil
}

// Assumes each entry represents a second
func calcWattsecondsToWatthoursRounded(watts []uint32) uint32 {
	var totalWattseconds uint64 = 0
	for _, w := range watts {
		totalWattseconds += uint64(w)
	}

	// Convert to watthours with rounding
	return uint32(float64(totalWattseconds) / 3600.0)
}

// roundToHourStart returns the Unix timestamp of the start of the timeframe for the given time
func getTimeframeStart(t time.Time, frame Timeframe) int64 {
	switch frame {
	case Hourly:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC).Unix()
	case Daily:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
	case Monthly:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()
	default:
		return 0 // should be impossible™
	}
}

// roundToHourStart returns the Unix timestamp of the end of the timeframe for the given time
func getTimeframeEnd(t time.Time, frame Timeframe) int64 {
	return getNextTimeframeStart(t, frame) - 1
}

// Get the start of the next timeframe after this one
func getNextTimeframeStart(t time.Time, frame Timeframe) int64 {
	switch frame {
	case Hourly:
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC).Add(time.Hour).Unix()
	case Daily:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1).Unix()
	case Monthly:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0).Unix()
	default:
		return 0 // should not happen™
	}
}

// Get the start of the timeframe before this one
func getPrevTimeframeStart(t time.Time, frame Timeframe) int64 {
	prevEndUnix := getTimeframeStart(t, frame) - 1
	return getTimeframeStart(time.Unix(prevEndUnix, 0), frame)
}
