// Responsible for storing the data collected from the smart meter
// Depends on the interpreter API being online.
package main

import (
	"github.com/NotCoffee418/european_smart_meter/pkg/meterdb"
)

func main() {
	// Initialize database
	meterdb.InitializeDatabase()

	// Start the collector

}
