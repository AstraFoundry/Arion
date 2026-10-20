package application

import "math"

// GainStop is one microphone gain setting, in decibels, and the speaking
// distance from the computer it suits.
type GainStop struct {
	DB         int
	DistanceCM int
}

// GainStops are the selectable gains, quietest first. Speech level falls
// 6 dB for each doubling of distance, so every 3 dB step suits a distance
// √2 times farther than the step before; 0 dB is the device's own level,
// suited to about 30 cm.
var GainStops = []GainStop{
	{DB: -6, DistanceCM: 15}, {DB: -3, DistanceCM: 21}, {DB: 0, DistanceCM: 30},
	{DB: 3, DistanceCM: 42}, {DB: 6, DistanceCM: 60}, {DB: 9, DistanceCM: 85}, {DB: 12, DistanceCM: 120},
}

// GainStep is the distance in decibels between neighboring stops.
const GainStep = 3

// GainFactor converts a gain in decibels to the factor samples are scaled by.
func GainFactor(db int) float32 { return float32(math.Pow(10, float64(db)/20)) }

func knownGain(db int) bool {
	for _, stop := range GainStops {
		if stop.DB == db {
			return true
		}
	}
	return false
}
