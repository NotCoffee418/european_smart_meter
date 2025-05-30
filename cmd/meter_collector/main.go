// Responsible for storing the data collected from the smart meter
// Depends on the interpreter API being online.
package main

import (
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
	lastLivePowerInsertedValueW uint32    = 0
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
		meterdb.InsertTotalGasReading(&meterdb.MeterDbTotalGasReading{
			Timestamp:           unixTimestampInt,
			TotalConsumptionDM3: currentGasValueDM3,
		})
		lastGasInsertedUtcTimestamp = utcTime
		lastGasValueDM3 = currentGasValueDM3
	}

	// Store live power reading always
	meterdb.InsertLivePowerReading(&meterdb.MeterDbLivePowerReading{
		Timestamp:   unixTimestampInt,
		Watt:        esmutils.KwToW(liveKw),
		ReadingType: readingType,
	})

	// Store total power reading if value any value has changed
	currentTotalConsumptionDayWh := esmutils.KwToW(reading.TotalConsumptionDayKWH)
	currentTotalProductionDayWh := esmutils.KwToW(reading.TotalProductionDayKWH)
	currentTotalConsumptionNightWh := esmutils.KwToW(reading.TotalConsumptionNightKWH)
	currentTotalProductionNightWh := esmutils.KwToW(reading.TotalProductionNightKWH)
	if currentTotalConsumptionDayWh != lastTotalConsumptionDayWh ||
		currentTotalProductionDayWh != lastTotalProductionDayWh ||
		currentTotalConsumptionNightWh != lastTotalConsumptionNightWh ||
		currentTotalProductionNightWh != lastTotalProductionNightWh {
		meterdb.InsertTotalPowerReading(&meterdb.MeterDbTotalPowerReading{
			Timestamp:               unixTimestampInt,
			TotalConsumptionDayWh:   currentTotalConsumptionDayWh,
			TotalProductionDayWh:    currentTotalProductionDayWh,
			TotalConsumptionNightWh: currentTotalConsumptionNightWh,
			TotalProductionNightWh:  currentTotalProductionNightWh,
		})
		lastTotalConsumptionDayWh = currentTotalConsumptionDayWh
		lastTotalProductionDayWh = currentTotalProductionDayWh
		lastTotalConsumptionNightWh = currentTotalConsumptionNightWh
		lastTotalProductionNightWh = currentTotalProductionNightWh
	}
}
