//go:build !windows

package main

import (
	"runtime"
	"syscall"
)

// peakRSS is the worker's lifetime resident high-water mark in bytes.
func peakRSS() int64 {
	var usage syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &usage) != nil {
		return 0
	}
	if runtime.GOOS == "darwin" {
		return usage.Maxrss
	}
	return usage.Maxrss * 1024
}
