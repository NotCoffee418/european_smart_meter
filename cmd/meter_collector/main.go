// Responsible for storing the data collected from the smart meter
// Depends on the interpreter API being online.
package main

import (
	"fmt"
	"os"

	"github.com/NotCoffee418/european_smart_meter/pkg/interpreter_listener"
	"github.com/NotCoffee418/european_smart_meter/pkg/meterdb"
	"github.com/NotCoffee418/european_smart_meter/pkg/types"
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
	interpreter_listener.StartListener(host, handleMeterReading)

}

// Handle meter reading data
func handleMeterReading(reading *types.RawMeterReading) {
	fmt.Println(string(reading.ToJsonBytes()))
}
