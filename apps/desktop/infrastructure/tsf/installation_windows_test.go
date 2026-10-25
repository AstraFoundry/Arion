//go:build windows

package tsf

import (
	"context"
	"errors"
	"testing"
	"time"

	"arion.local/core/domain"
)

func TestRegistrationObservationWaitsForIndependentDiscovery(t *testing.T) {
	calls := 0
	err := observeRegistration(context.Background(), "expected", func(context.Context) (State, error) {
		calls++
		return State{Registered: calls == 3, Build: "expected"}, nil
	}, time.Millisecond)
	if err != nil || calls != 3 {
		t.Fatalf("delayed native discovery: calls=%d error=%v", calls, err)
	}
}

func TestRegistrationObservationDoesNotHideNativeFailure(t *testing.T) {
	failure := errors.New("native profile enumeration denied")
	calls := 0
	err := observeRegistration(context.Background(), "expected", func(context.Context) (State, error) { calls++; return State{}, failure }, time.Millisecond)
	if !errors.Is(err, failure) || calls != 1 {
		t.Fatalf("native error was retried or hidden: calls=%d error=%v", calls, err)
	}
}

func TestRegistrationObservationRefusesAnotherBuild(t *testing.T) {
	err := observeRegistration(context.Background(), "expected", func(context.Context) (State, error) { return State{Registered: true, Build: "different"}, nil }, time.Millisecond)
	if domain.CodeOf(err) != domain.CodeArtifactUnverified {
		t.Fatalf("wrong installed build accepted: %v", err)
	}
}

func TestRegistrationObservationDeadlineIdentifiesDiscoveryStage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	err := observeRegistration(ctx, "expected", func(context.Context) (State, error) { return State{}, nil }, time.Millisecond)
	var stage interface{ InputMethodSetupStage() string }
	if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &stage) || stage.InputMethodSetupStage() != "discovery" {
		t.Fatalf("unbounded or unclassified discovery: %v", err)
	}
}

func TestEnablementWaitsForObservedUserState(t *testing.T) {
	calls := 0
	err := observeEnablement(context.Background(), "expected", func(context.Context) (State, error) {
		calls++
		return State{Registered: true, Enabled: calls == 3, Build: "expected"}, nil
	}, time.Millisecond)
	if err != nil || calls != 3 {
		t.Fatalf("enable request was mistaken for observed enablement: calls=%d error=%v", calls, err)
	}
}

func TestEnablementRejectsChangedRegistration(t *testing.T) {
	err := observeEnablement(context.Background(), "expected", func(context.Context) (State, error) {
		return State{Registered: true, Enabled: true, Build: "different"}, nil
	}, time.Millisecond)
	if domain.CodeOf(err) != domain.CodeArtifactUnverified {
		t.Fatalf("changed registration accepted: %v", err)
	}
}

func TestEnablementDeadlineKeepsItsStage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	err := observeEnablement(ctx, "expected", func(context.Context) (State, error) {
		return State{Registered: true, Build: "expected"}, nil
	}, time.Millisecond)
	var stage interface{ InputMethodSetupStage() string }
	if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &stage) || stage.InputMethodSetupStage() != "enable" {
		t.Fatalf("enablement deadline reported as another stage: %v", err)
	}
}

func TestCancelledEnablementDoesNotQuery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := observeEnablement(ctx, "expected", func(context.Context) (State, error) {
		t.Fatal("cancelled setup continued into native observation")
		return State{}, nil
	}, time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}

func TestRegistrationRepairOnlyForOwnedDisabledProfile(t *testing.T) {
	for _, tc := range []struct {
		name       string
		state      State
		wantRepair bool
		wantError  bool
	}{
		{"enabled", State{Registered: true, Enabled: true, Build: "expected"}, false, false},
		{"legacy disabled registration", State{Registered: true, Build: "expected"}, true, false},
		{"different executable", State{Registered: true, Build: "other"}, false, true},
		{"legacy language with matching executable", State{Registered: true, Legacy: true, Build: "expected"}, false, true},
		{"migration required with matching executable", State{Registered: true, UpgradeRequired: true, Build: "expected"}, false, true},
		{"removed", State{}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := repairDisabledRegistration(context.Background(), "expected", func(context.Context) (State, error) { return tc.state, nil }, func() error { calls++; return nil })
			if (err != nil) != tc.wantError || (calls == 1) != tc.wantRepair || calls > 1 {
				t.Fatalf("repair calls=%d error=%v", calls, err)
			}
		})
	}
}

func TestLegacyProfileCannotSatisfyInstallationOrEnablement(t *testing.T) {
	for _, state := range []State{
		{Registered: true, Enabled: true, Selected: true, Build: "expected", Legacy: true},
		{Registered: true, Enabled: true, Selected: true, Build: "expected", UpgradeRequired: true},
	} {
		query := func(context.Context) (State, error) { return state, nil }
		if err := observeRegistration(context.Background(), "expected", query, time.Millisecond); err == nil {
			t.Fatal("legacy profile satisfied new installation")
		}
		if err := observeEnablement(context.Background(), "expected", query, time.Millisecond); err == nil {
			t.Fatal("legacy profile satisfied new enablement")
		}
	}
}

func TestRegistrationRepairPreservesDenialAndCancellation(t *testing.T) {
	denied := errors.New("administrator approval declined")
	err := repairDisabledRegistration(context.Background(), "expected", func(context.Context) (State, error) {
		return State{Registered: true, Build: "expected"}, nil
	}, func() error { return denied })
	if !errors.Is(err, denied) {
		t.Fatalf("approval denial hidden: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	err = repairDisabledRegistration(ctx, "expected", func(context.Context) (State, error) {
		cancel()
		return State{Registered: true, Build: "expected"}, nil
	}, func() error { t.Fatal("repair ran after cancellation"); return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}
