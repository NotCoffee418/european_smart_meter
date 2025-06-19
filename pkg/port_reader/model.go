package port_reader

import (
	"io"
	"regexp"
	"sync"

	"github.com/NotCoffee418/european_smart_meter/pkg/interpreter"
)

type P1Reader struct {
	port          string
	baudrate      uint
	serialPort    io.ReadWriteCloser
	latestReading *interpreter.RawMeterReading
	readingMutex  sync.RWMutex
	stopSignal    bool

	// Pre-compiled regex patterns
	obisPatterns    map[string]*regexp.Regexp
	specialPatterns map[string]*regexp.Regexp
}
