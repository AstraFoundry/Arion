package macos

/*
#include "native.h"
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// IMESetupError identifies the failed observable setup stage without UI copy.
type IMESetupError struct {
	Stage string
	Err   error
}

func (e *IMESetupError) Error() string                 { return fmt.Sprintf("input method %s: %v", e.Stage, e.Err) }
func (e *IMESetupError) Unwrap() error                 { return e.Err }
func (e *IMESetupError) InputMethodSetupStage() string { return e.Stage }

type activationOperations struct {
	register     func() error
	state        func() IMEState
	enable       func() error
	selectSource func() error
	ready        func(context.Context) error
	wait         func(context.Context) error
}

// activateInputSource never replaces files. A notification wakes observation;
// bounded polling also observes missed/coalesced notifications and cancellation.
func activateInputSource(ctx context.Context, op activationOperations) error {
	stage := "discovery"
	fail := func(err error) error { return &IMESetupError{Stage: stage, Err: err} }
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := op.register(); err != nil {
		return fail(err)
	}
	requested := false
	var selectionError error
	for {
		if err := ctx.Err(); err != nil {
			return fail(errors.Join(err, selectionError))
		}
		switch op.state() {
		case IMEAbsent, IMECopied:
			stage = "discovery"
		case IMERegistered:
			stage = "enable"
			if !requested {
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := op.enable(); err != nil {
					return fail(err)
				}
				requested = true
				continue
			}
		case IMEEnabled:
			stage = "selection"
			if err := ctx.Err(); err != nil {
				return err
			}
			selectionError = op.selectSource()
			// Even noErr is only a request result; query the actual selection.
			if op.state() == IMESelected {
				continue
			}
		case IMESelected:
			stage = "connection"
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := op.ready(ctx); err != nil {
				return fail(err)
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if op.state() != IMESelected {
				return fail(errors.New("input source changed during host verification"))
			}
			return nil
		default:
			return fail(errors.New("invalid input source state"))
		}
		if err := op.wait(ctx); err != nil {
			return fail(errors.Join(err, selectionError))
		}
	}
}

// ActivateIME continues an explicit setup attempt through actual selection and
// a build-matching host connection. The caller bounds and cancels the attempt.
func ActivateIME(ctx context.Context, ready func(context.Context, string) error) error {
	imeInstallation.Lock()
	defer imeInstallation.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	bundle := InstalledIMEPath()
	if err := ValidateIMEPayload(bundle); err != nil {
		return err
	}
	if ready == nil {
		return errors.New("input method activation requires host verification")
	}
	build, err := inputBuildIdentity(bundle)
	if err != nil {
		return err
	}
	watch := C.arion_ime_watch_create()
	if watch == nil {
		return errors.New("input source observation could not start")
	}
	// Keep the cancellation observer alive until native callbacks have drained.
	finished, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-ctx.Done():
			C.arion_ime_watch_cancel(watch)
		case <-finished:
		}
	}()
	defer func() { close(finished); <-stopped; C.arion_ime_watch_close(watch) }()
	path, freePath := cstring(bundle)
	defer freePath()
	id, freeID := cstring(IMESourceID)
	defer freeID()
	return activateInputSource(ctx, activationOperations{
		register: func() error {
			if status := C.arion_ime_register(path); status != 0 {
				return fmt.Errorf("registration returned %d", status)
			}
			return nil
		},
		state: func() IMEState { return QueryIME() },
		enable: func() error {
			if status := C.arion_ime_enable(id); status != 0 {
				return fmt.Errorf("enable returned %d", status)
			}
			return nil
		},
		selectSource: func() error {
			if status := C.arion_ime_watch_select(watch, id); status != 0 {
				return fmt.Errorf("selection returned %d", status)
			}
			return nil
		},
		ready: func(ctx context.Context) error { return launchInputMethodContext(ctx, bundle, build, ready) },
		wait: func(ctx context.Context) error {
			delay := 250 * time.Millisecond
			if deadline, ok := ctx.Deadline(); ok {
				delay = min(delay, time.Until(deadline))
			}
			if delay > 0 {
				C.arion_ime_watch_wait(watch, C.int(delay.Milliseconds()))
			}
			return ctx.Err()
		},
	})
}
