package asr

import "arion.local/core/domain"

func JoinSegments(parts []string) string { return domain.JoinText(parts...) }

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
