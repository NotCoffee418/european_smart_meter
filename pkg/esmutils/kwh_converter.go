package esmutils

import "math"

// No negative values
func KwToW(kw float64) uint32 {
	if kw < 0 {
		return 0
	}
	return uint32(math.Round(kw * 1000))
}

func WToKw(w uint32) float64 {
	return float64(w) / 1000
}

// Convert m3 to dm3 for storage - No negative values
func M3ToDM3(m3 float64) uint32 {
	if m3 < 0 {
		return 0
	}
	return uint32(math.Round(m3 * 1000)) // 1 m³ = 1000 dm³
}

// Convert dm3 to m3 from storage - No negative values
func DM3ToM3(dm3 uint32) float64 {
	return float64(dm3) / 1000
}
