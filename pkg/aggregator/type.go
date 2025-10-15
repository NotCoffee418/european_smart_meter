package aggregator

import "github.com/NotCoffee418/european_smart_meter/pkg/meterdb"

type AggregateData struct {
	Timeframe          Timeframe
	EndTime            int64
	IsInDb             bool
	IsCurrentTimeframe bool
	Aggregate          meterdb.AggregateLivePowerTable
}
