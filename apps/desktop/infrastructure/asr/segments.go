package asr

import (
	"math"

	"arion.local/core/domain"
)

// speechFloor is -42 dBFS RMS. Endpointing and the quiet-window gate share it.
const speechFloor = 0.008

func JoinSegments(parts []string) string { return domain.JoinText(parts...) }

// classifyPCM reports whether any 20 ms span reaches the speech floor.
// A shorter buffer is one span. Non-finite samples are invalid PCM, not silence:
// Qwen3 reads system-turn hotwords back when a quiet window is decoded.
func classifyPCM(samples []float32) (speech, invalid bool) {
	const window = 16000 / 50
	if len(samples) == 0 {
		return false, false
	}
	step := window
	if len(samples) < step {
		step = len(samples)
	}
	for start := 0; start < len(samples); start += step {
		end := start + step
		if end > len(samples) {
			end = len(samples)
		}
		var energy float64
		for _, sample := range samples[start:end] {
			if math.IsNaN(float64(sample)) || math.IsInf(float64(sample), 0) {
				return false, true
			}
			energy += float64(sample) * float64(sample)
		}
		if math.Sqrt(energy/float64(end-start)) >= speechFloor {
			speech = true
		}
	}
	return speech, false
}

// SplitAtQuiet cuts audio into segments no longer than limit seconds,
// choosing the lowest-energy 20 ms window in the last third of each span.
func SplitAtQuiet(samples []float32, rate, limit int) [][]float32 {
	window := rate / 50
	max := rate * limit
	var out [][]float32
	for len(samples) > max {
		best, bestEnergy := max, float32(-1)
		for start := max * 2 / 3; start+window <= max; start += window {
			var energy float32
			for _, s := range samples[start : start+window] {
				energy += s * s
			}
			if bestEnergy < 0 || energy < bestEnergy {
				best, bestEnergy = start+window/2, energy
			}
		}
		out = append(out, samples[:best])
		samples = samples[best:]
	}
	if len(samples) > 0 {
		out = append(out, samples)
	}
	return out
}
