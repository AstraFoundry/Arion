//go:build !windows

package asr

func workerName() string {
	return "speech"
}
