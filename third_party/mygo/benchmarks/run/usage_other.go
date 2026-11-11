//go:build !darwin

package main

import "errors"

// usage is what a process has used up to now.
type usage struct {
	cpu             int64
	footprint, peak uint64
}

// usageOf is macOS's only, for now: the comparison runs on macOS.
func usageOf(int) (usage, error) { return usage{}, errors.New("the comparison runs on macOS") }

func windowOnScreen(int) bool { return false }
