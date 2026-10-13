package bootstrap

import (
	"math"

	"github.com/egoist/mygo/audio"
)

// cue synthesizes a short two-note chime: rising when listening starts,
// falling when it stops. Generated, so no audio asset ships.
func cue(rising bool) *audio.Sound {
	const rate = 44100
	notes := []float64{660, 880}
	if !rising {
		notes = []float64{880, 587}
	}
	noteSamples := rate * 75 / 1000
	samples := make([]float32, 0, 2*noteSamples)
	for _, frequency := range notes {
		for i := 0; i < noteSamples; i++ {
			t := float64(i) / rate
			attack := math.Min(1, t/0.006)
			release := math.Exp(-t * 28)
			v := 0.16 * attack * release * (math.Sin(2*math.Pi*frequency*t) + 0.25*math.Sin(4*math.Pi*frequency*t))
			samples = append(samples, float32(v))
		}
	}
	return audio.NewSound(rate, 1, samples)
}
