// Responsible for storing the data collected from the smart meter
// Depends on the interpreter API being online.
package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/NotCoffee418/european_smart_meter/pkg/esmutils"
	"github.com/NotCoffee418/european_smart_meter/pkg/interpreter"
	"github.com/NotCoffee418/european_smart_meter/pkg/meterdb"
)

var (
	lastGasValueDM3             uint32    = 0
	lastGasInsertedUtcTimestamp time.Time = time.Time{}
	lastTotalConsumptionDayWh   uint32    = 0
	lastTotalProductionDayWh    uint32    = 0
	lastTotalConsumptionNightWh uint32    = 0
	lastTotalProductionNightWh  uint32    = 0
)

const (
	minGasSaveInsertInterval = 10 * time.Minute
)

func main() {
	// Initialize database
	meterdb.InitializeDatabase()

	// Set the host:port from env var INTERPRETER_API_HOST
	host := os.Getenv("INTERPRETER_API_HOST")
	if host == "" {
		host = "raspberrypi.local:9039"
	}

	// We only need total power on fresh start because:
	// Gas: Rarely changes regadless but still inserts every 10 minutes
	// Live power: Value changes constantly and is dependent on type
	// Total power: Type changes throughout the day, we don't need to update night data during the day
	loadLastTotalPowerReadings()

	// Subscribe to websocket with revive
	interpreter.StartListener(host, handleMeterReading)
}

// Handle meter reading data
func handleMeterReading(reading *interpreter.RawMeterReading) {
	// Parse as local time, but convert to UTC for storage
	utcTime, err := time.Parse(time.RFC3339, reading.Timestamp)
	if err != nil {
		log.Printf("Failed to parse timestamp: %v", err)
		return
	}
	unixTimestampInt := utcTime.Unix()

	// Interpret type and live power reading
	var liveKw float64 = 0
	var readingType meterdb.MeterDbPowerReadingType = meterdb.PowerConsumptionDay
	if reading.CurrentConsumptionKW > 0 {
		liveKw = reading.CurrentConsumptionKW
		if reading.CurrentTariff == 1 {
			readingType = meterdb.PowerConsumptionDay
		} else {
			readingType = meterdb.PowerConsumptionNight
		}
	} else if reading.CurrentProductionKW > 0 {
		liveKw = reading.CurrentProductionKW
		if reading.CurrentTariff == 1 {
			readingType = meterdb.PowerProductionDay
		} else {
			readingType = meterdb.PowerProductionNight
		}
	}

	// Store gas if reading has changed or if interval has passed
	currentGasValueDM3 := esmutils.M3ToDM3(reading.GasConsumptionM3)
	if currentGasValueDM3 != lastGasValueDM3 || time.Since(lastGasInsertedUtcTimestamp) > minGasSaveInsertInterval {
		err = meterdb.InsertTotalGasReading(&meterdb.MeterDbTotalGasReading{
			Timestamp:           unixTimestampInt,
			TotalConsumptionDM3: currentGasValueDM3,
		})
		if err != nil {
			log.Printf("Failed to insert gas reading: %v", err)
		} else {
			lastGasInsertedUtcTimestamp = utcTime
			lastGasValueDM3 = currentGasValueDM3
		}
	}

	// Store live power reading always
	err = meterdb.InsertLivePowerReading(&meterdb.MeterDbLivePowerReading{
		Timestamp:   unixTimestampInt,
		Watt:        esmutils.KwToW(liveKw),
		ReadingType: readingType,
	})
	if err != nil {
		log.Printf("Failed to insert live power reading: %v", err)
	}

	// Store total power if value changed for current type
	var currentTotalPowerWh uint32 = 0
	var totalPowerChanged bool = false
	switch readingType {
	case meterdb.PowerConsumptionDay:
		currentTotalPowerWh = esmutils.KwToW(reading.TotalConsumptionDayKWH)
		totalPowerChanged = currentTotalPowerWh != lastTotalConsumptionDayWh
		lastTotalConsumptionDayWh = currentTotalPowerWh
	case meterdb.PowerProductionDay:
		currentTotalPowerWh = esmutils.KwToW(reading.TotalProductionDayKWH)
		totalPowerChanged = currentTotalPowerWh != lastTotalProductionDayWh
		lastTotalProductionDayWh = currentTotalPowerWh
	case meterdb.PowerConsumptionNight:
		currentTotalPowerWh = esmutils.KwToW(reading.TotalConsumptionNightKWH)
		totalPowerChanged = currentTotalPowerWh != lastTotalConsumptionNightWh
		lastTotalConsumptionNightWh = currentTotalPowerWh
	case meterdb.PowerProductionNight:
		currentTotalPowerWh = esmutils.KwToW(reading.TotalProductionNightKWH)
		totalPowerChanged = currentTotalPowerWh != lastTotalProductionNightWh
		lastTotalProductionNightWh = currentTotalPowerWh
	}
	if totalPowerChanged {
		err = meterdb.InsertTotalPowerReading(&meterdb.MeterDbTotalPowerReading{
			Timestamp:   unixTimestampInt,
			Watthour:    currentTotalPowerWh,
			ReadingType: readingType,
		})
		if err != nil {
			log.Printf("Failed to insert total power reading: %v", err)
		}
	}
}

// Load last total power readings from database
func loadLastTotalPowerReadings() {
	// Shortcut function to handle no rows (is valid for total power readings)
	getLastTotalPower := func(meterDbReadingType meterdb.MeterDbPowerReadingType) uint32 {
		lastTotalPowerReading, err := meterdb.GetLastTotalPowerReading(meterDbReadingType)
		if err != nil {
			// No rows is valid, fresh DB
			if err == sql.ErrNoRows {
				return 0
			} else {
				log.Fatalf("Failed to get last live power reading: %v", err)
			}
		}
		return lastTotalPowerReading.Watthour
	}

	// Set last total consumption of each type
	lastTotalConsumptionDayWh = getLastTotalPower(meterdb.PowerConsumptionDay)
	lastTotalProductionDayWh = getLastTotalPower(meterdb.PowerProductionDay)
	lastTotalConsumptionNightWh = getLastTotalPower(meterdb.PowerConsumptionNight)
	lastTotalProductionNightWh = getLastTotalPower(meterdb.PowerProductionNight)
}
