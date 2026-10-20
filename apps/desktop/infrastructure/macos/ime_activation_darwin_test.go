package macos

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestActivationWaitsForDiscoveryEnablementAndVerifiedSelection(t *testing.T) {
	state := IMECopied
	var events []string
	waits := 0
	op := activationOperations{
		register:     func() error { events = append(events, "register"); return nil },
		state:        func() IMEState { return state },
		enable:       func() error { events = append(events, "enable"); return nil },
		selectSource: func() error { events = append(events, "select"); return nil },
		ready:        func(context.Context) error { events = append(events, "ready"); return nil },
		wait: func(context.Context) error {
			waits++
			switch waits {
			case 1:
				state = IMERegistered
			case 2:
				state = IMEEnabled
			case 3:
				state = IMESelected
			}
			return nil
		},
	}
	if err := activateInputSource(context.Background(), op); err != nil {
		t.Fatal(err)
	}
	if want := []string{"register", "enable", "select", "ready"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("stage order %v", events)
	}
	if waits != 3 {
		t.Fatalf("request return was mistaken for observed completion: %d", waits)
	}
}

func TestActivationReportsStageAndPreservesSelectionFailure(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state IMEState
		stage string
	}{
		{"not discovered", IMECopied, "discovery"}, {"not enabled", IMERegistered, "enable"}, {"cannot select", IMEEnabled, "selection"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failure := errors.New("selection API refused")
			enabled := 0
			op := activationOperations{
				register: func() error { return nil }, state: func() IMEState { return tc.state },
				enable: func() error { enabled++; return nil }, selectSource: func() error { return failure },
				ready: func(context.Context) error { t.Fatal("unselected source reported operational"); return nil },
				wait:  func(context.Context) error { return context.DeadlineExceeded },
			}
			err := activateInputSource(context.Background(), op)
			var stage *IMESetupError
			if !errors.As(err, &stage) || stage.Stage != tc.stage || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("wrong stage/error: %v", err)
			}
			if tc.state == IMEEnabled && !errors.Is(err, failure) {
				t.Fatalf("selection refusal lost: %v", err)
			}
			if enabled > 1 {
				t.Fatal("enable retried implicitly")
			}
		})
	}
}

func TestActivationCancellationPreventsSelectionAndHostLaunch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	state := IMERegistered
	op := activationOperations{
		register: func() error { return nil }, state: func() IMEState { return state },
		enable:       func() error { state = IMEEnabled; cancel(); return nil },
		selectSource: func() error { t.Fatal("cancelled attempt selected a source"); return nil },
		ready:        func(context.Context) error { t.Fatal("cancelled attempt launched host"); return nil },
		wait:         func(context.Context) error { t.Fatal("cancelled attempt kept waiting"); return nil },
	}
	if err := activateInputSource(ctx, op); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel lost: %v", err)
	}
}

func TestActivationDoesNotReselectCurrentSourceAndVerifiesHost(t *testing.T) {
	failure := errors.New("wrong build handshake")
	op := activationOperations{
		register: func() error { return nil }, state: func() IMEState { return IMESelected },
		enable:       func() error { t.Fatal("already selected source enabled again"); return nil },
		selectSource: func() error { t.Fatal("already selected source selected again"); return nil },
		ready:        func(context.Context) error { return failure },
	}
	err := activateInputSource(context.Background(), op)
	var stage *IMESetupError
	if !errors.As(err, &stage) || stage.Stage != "connection" || !errors.Is(err, failure) {
		t.Fatalf("host failure hidden: %v", err)
	}
}

func TestActivationSelectionRetriesUntilObserved(t *testing.T) {
	state := IMEEnabled
	calls := 0
	op := activationOperations{
		register: func() error { return nil }, state: func() IMEState { return state },
		selectSource: func() error {
			calls++
			if calls == 1 {
				return errors.New("temporary refusal")
			}
			state = IMESelected
			return nil
		},
		ready: func(context.Context) error { return nil }, wait: func(context.Context) error { return nil },
	}
	if err := activateInputSource(context.Background(), op); err != nil || calls != 2 {
		t.Fatalf("retry failed: %v calls=%d", err, calls)
	}
}

func TestActivationReturnsRegisterAndEnableFailures(t *testing.T) {
	failure := errors.New("native API failed")
	for _, stage := range []string{"discovery", "enable"} {
		t.Run(stage, func(t *testing.T) {
			op := activationOperations{register: func() error {
				if stage == "discovery" {
					return failure
				}
				return nil
			}, state: func() IMEState { return IMERegistered }, enable: func() error { return failure }}
			err := activateInputSource(context.Background(), op)
			var actual *IMESetupError
			if !errors.As(err, &actual) || actual.Stage != stage || !errors.Is(err, failure) {
				t.Fatalf("native failure hidden: %v", err)
			}
		})
	}
}
