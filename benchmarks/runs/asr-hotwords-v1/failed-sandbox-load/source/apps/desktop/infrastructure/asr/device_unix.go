//go:build !windows

package asr

import "runtime"

func workerName() string {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return "arion-asr"
	}
	return "speech"
}
