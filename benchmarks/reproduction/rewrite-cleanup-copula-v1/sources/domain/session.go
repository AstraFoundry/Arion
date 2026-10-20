// Package domain contains platform-independent voice input invariants.
package domain

import "fmt"

type InputMode string

const (
	ShortcutMode  InputMode = "shortcut"
	SystemIMEMode InputMode = "system_ime"
)

type SessionState string

const (
	Idle       SessionState = "idle"
	Preparing  SessionState = "preparing"
	Capturing  SessionState = "capturing"
	Finalizing SessionState = "finalizing"
	Reviewing  SessionState = "reviewing"
	Committing SessionState = "committing"
	Completed  SessionState = "completed"
	Cancelled  SessionState = "cancelled"
	Failed     SessionState = "failed"
)

// CanTransition defines final target commit, not provisional IME composition.
func CanTransition(from, to SessionState) bool {
	if to == Cancelled || to == Failed {
		return from == Preparing || from == Capturing || from == Finalizing || from == Reviewing || from == Committing
	}
	switch from {
	case Idle:
		return to == Preparing
	case Preparing:
		return to == Capturing
	case Capturing:
		return to == Finalizing
	case Finalizing:
		return to == Reviewing
	case Reviewing:
		return to == Committing
	case Committing:
		return to == Completed
	default:
		return false
	}
}

func Transition(from, to SessionState) (SessionState, error) {
	if !CanTransition(from, to) {
		return from, fmt.Errorf("invalid session transition: %s -> %s", from, to)
	}
	return to, nil
}
