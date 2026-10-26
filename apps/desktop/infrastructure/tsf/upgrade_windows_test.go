//go:build windows

package tsf

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAutomaticUpgradePreservesUserChoiceAndCurrentInstallation(t *testing.T) {
	old, next := strings.Repeat("a", 64), strings.Repeat("b", 64)
	for _, tc := range []struct {
		name     string
		state    State
		active   bool
		deferred bool
	}{
		{"absent", State{}, false, true},
		{"disabled", State{Registered: true, Build: old}, false, true},
		{"selected", State{Registered: true, Enabled: true, Selected: true, Build: old}, false, true},
		{"editable client", State{Registered: true, Enabled: true, Build: old}, true, true},
		{"current", State{Registered: true, Enabled: true, Build: next}, false, false},
		{"disabled current", State{Registered: true, Build: next}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := maintainRegistration(context.Background(), next, "bundle.dll", true,
				func(context.Context) (State, error) { return tc.state, nil }, func() bool { return tc.active },
				func(context.Context, []string) error {
					t.Fatal("background operation requested administrator approval")
					return nil
				},
				func(context.Context, string) error { t.Fatal("background operation enabled/changed a TIP"); return nil }, time.Millisecond)
			var deferred interface{ InputMethodUpdateDeferred() bool }
			if errors.As(err, &deferred) != tc.deferred || (err != nil && !tc.deferred) {
				t.Fatalf("deferred=%v error=%v", tc.deferred, err)
			}
		})
	}
}

func TestUpgradeUsesOneMachineTransactionThenOrdinaryUserMigration(t *testing.T) {
	old, next := strings.Repeat("a", 64), strings.Repeat("b", 64)
	for _, legacy := range []bool{false, true} {
		state := State{Registered: true, Enabled: true, Build: old, Legacy: legacy, UpgradeRequired: legacy}
		var calls []string
		err := maintainRegistration(context.Background(), next, "bundled.dll", true,
			func(context.Context) (State, error) { return state, nil }, func() bool { return false },
			func(_ context.Context, args []string) error {
				if !reflect.DeepEqual(args, []string{"update", "bundled.dll", next, old}) {
					t.Fatalf("unbounded/unverified update args: %q", args)
				}
				calls = append(calls, "machine-update")
				state = State{Registered: true, Build: next}
				return nil
			}, func(_ context.Context, action string) error {
				calls = append(calls, action)
				if action == "enable" {
					state.Enabled = true
				}
				return nil
			}, time.Millisecond)
		want := []string{"machine-update", "enable"}
		if legacy {
			want = []string{"machine-update", "disable-legacy", "enable"}
		}
		if err != nil || !reflect.DeepEqual(calls, want) {
			t.Fatalf("legacy=%v calls=%v error=%v", legacy, calls, err)
		}
	}
}

func TestStableLoaderPublishesCoreWhileSelectedWithoutTogglingUserProfile(t *testing.T) {
	old, next := strings.Repeat("a", 64), strings.Repeat("b", 64)
	state := State{Registered: true, Enabled: true, Selected: true, Build: old, Loader: true}
	calls := 0
	err := maintainRegistration(context.Background(), next, "bundled.dll", true,
		func(context.Context) (State, error) { return state, nil },
		func() bool { return true },
		func(_ context.Context, args []string) error {
			if !reflect.DeepEqual(args, []string{"update", "bundled.dll", next, old}) {
				t.Fatalf("unexpected core publication: %q", args)
			}
			calls++
			state.Build = next
			return nil
		}, func(context.Context, string) error {
			t.Fatal("stable core publication changed user profile")
			return nil
		}, time.Millisecond)
	if err != nil || calls != 1 || !state.Selected || !state.Enabled {
		t.Fatalf("publication interrupted selection: calls=%d state=%+v err=%v", calls, state, err)
	}
}

func TestUpgradeApprovalFailureLeavesUserConfigurationUntouched(t *testing.T) {
	denied := errors.New("OS administrator approval cancelled")
	old, next := strings.Repeat("a", 64), strings.Repeat("b", 64)
	var calls int
	err := maintainRegistration(context.Background(), next, "bundle.dll", true,
		func(context.Context) (State, error) {
			return State{Registered: true, Enabled: true, Legacy: true, Build: old}, nil
		}, func() bool { return false },
		func(context.Context, []string) error { calls++; return denied },
		func(context.Context, string) error { t.Fatal("denied machine upgrade changed user's TIPs"); return nil }, time.Millisecond)
	if !errors.Is(err, denied) || calls != 1 {
		t.Fatalf("approval failure retried/hidden: calls=%d err=%v", calls, err)
	}
}

func TestExplicitRetryCompletesUserEnablementWithoutAnotherUAC(t *testing.T) {
	build := strings.Repeat("b", 64)
	state := State{Registered: true, Build: build}
	calls := 0
	err := maintainRegistration(context.Background(), build, "bundle.dll", false,
		func(context.Context) (State, error) { return state, nil }, func() bool { return false },
		func(context.Context, []string) error { t.Fatal("same build requested repeated UAC"); return nil },
		func(_ context.Context, action string) error {
			if action != "enable" {
				t.Fatalf("unexpected user action %s", action)
			}
			calls++
			state.Enabled = true
			return nil
		}, time.Millisecond)
	if err != nil || calls != 1 {
		t.Fatalf("user completion: calls=%d err=%v", calls, err)
	}
}

func TestUpgradeRejectsInvalidOldBuildObservationAndCancellation(t *testing.T) {
	for _, scenario := range []string{"invalid-build", "native-query-failed", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if scenario == "cancelled" {
				cancel()
			}
			err := maintainRegistration(ctx, strings.Repeat("b", 64), "bundle.dll", true,
				func(context.Context) (State, error) {
					if scenario == "native-query-failed" {
						return State{}, errors.New("profile query denied")
					}
					build := "wrong"
					if scenario == "cancelled" {
						build = strings.Repeat("a", 64)
					}
					return State{Registered: true, Enabled: true, Build: build}, nil
				}, func() bool { return false },
				func(context.Context, []string) error {
					t.Fatal("invalid/cancelled observation requested UAC")
					return nil
				}, func(context.Context, string) error {
					t.Fatal("invalid/cancelled observation changed user config")
					return nil
				}, time.Millisecond)
			if err == nil {
				t.Fatal("invalid observation accepted")
			}
		})
	}
}

func TestNativeSelectionRaceRefreshesObservedStateBeforeDeferral(t *testing.T) {
	queries := 0
	err := maintainRegistration(context.Background(), strings.Repeat("b", 64), "bundle.dll", true,
		func(context.Context) (State, error) {
			queries++
			return State{Registered: true, Enabled: true, Selected: queries > 1, Build: strings.Repeat("a", 64)}, nil
		}, func() bool { return false },
		func(context.Context, []string) error { return updateDeferred{} }, func(context.Context, string) error {
			t.Fatal("deferred machine transaction changed user state")
			return nil
		}, time.Millisecond)
	var deferred interface{ InputMethodUpdateDeferred() bool }
	if queries != 2 || !errors.As(err, &deferred) {
		t.Fatalf("selection cache not refreshed: queries=%d err=%v", queries, err)
	}
}
