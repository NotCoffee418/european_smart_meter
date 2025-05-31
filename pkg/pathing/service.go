package pathing

import (
	"log"
	"os"
	"path/filepath"
)

// Ensure directories exist on startup
func init() {
	// Directories that must exist:
	dirs := []string{
		GetDataDir(),
	}

	// Create all directories
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func GetMeterDbPath() string {
	// Join path
	return filepath.Join(GetDataDir(), "esm-meter.db")
}

func GetDataDir() string {
	return "/var/lib/european_smart_meter"
}

func GetConfigDir() string {
	return "/etc/european_smart_meter"
}
