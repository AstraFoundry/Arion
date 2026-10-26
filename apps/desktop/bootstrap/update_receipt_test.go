package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

type failingAcknowledgement struct {
	*fakePrepared
	ackErr           error
	acknowledgements int
}

func (p *failingAcknowledgement) Acknowledge() error { p.acknowledgements++; return p.ackErr }

func TestFailedAcknowledgementRestoresDurableReceiptBeforeRestart(t *testing.T) {
	for _, failRestore := range []bool{false, true} {
		name := "restored"
		if failRestore {
			name = "restore-failed"
		}
		t.Run(name, func(t *testing.T) {
			feed := &fakeFeed{enabled: true}
			stage := &failingAcknowledgement{fakePrepared: &fakePrepared{version: "1.2.3"}, ackErr: errors.New("receipt storage locked")}
			feed.found = &release{Version: "1.2.3", Prepare: func(_ context.Context, _ func(int64, int64)) (preparedRelease, error) { return stage, nil }}
			u, state, drain := harnessUpdates(feed)
			u.check(time.Now(), true)
			drain()
			u.install()
			drain()
			var operations []preparedRelease
			durable := preparedRelease(stage)
			u.persist = func(next preparedRelease, _ bool) error {
				operations = append(operations, next)
				if next != nil && failRestore {
					return errors.New("cannot restore stage handle")
				}
				durable = next
				return nil
			}
			var result error
			u.apply(func(err error) { result = err })
			drain()
			if len(operations) != 2 || operations[0] != nil || operations[1] != stage {
				t.Fatalf("receipt clear/restore order: %v", operations)
			}
			if u.prepared != stage || stage.acknowledgements != 1 || !state.StateError {
				t.Fatalf("failed acknowledgement lost receipt: %+v", state)
			}
			if failRestore {
				if !errors.Is(result, mygo.ErrUpdateRecoveryRequired) || !u.blocked || state.Failed != arion.UpdateStepRecovery {
					t.Fatalf("failed restore not blocked: %v %+v", result, state)
				}
			} else {
				if result != nil || durable != stage || state.Phase != arion.UpdateReady || u.blocked {
					t.Fatalf("durable receipt cannot restart: %v %+v", result, state)
				}
			}
		})
	}
}

func TestFailedReceiptClearKeepsOriginalHandleWithoutAcknowledgement(t *testing.T) {
	u, state, _ := harnessUpdates(&fakeFeed{enabled: true})
	stage := &failingAcknowledgement{fakePrepared: &fakePrepared{version: "1.2.3", applied: 1}}
	failure := errors.New("state directory read-only")
	u.persist = func(preparedRelease, bool) error { return failure }
	if err := u.acknowledgeApplied(stage, false); !errors.Is(err, failure) {
		t.Fatalf("clear failure lost: %v", err)
	}
	if u.prepared != stage || stage.acknowledgements != 0 || !state.StateError {
		t.Fatalf("unpersisted clear deleted completion receipt: %+v", state)
	}
}

func TestVerifiedCompletedReceiptErrorBlocksOnlyUpdates(t *testing.T) {
	a := &App{updates: &updates{blocked: true}, updatePolicy: &updatePolicy{verifiedInstalled: true}}
	if a.updateApplying() {
		t.Fatal("verified coherent host cannot load resources after receipt storage error")
	}
	a.updatePolicy.verifiedInstalled = false
	if !a.updateApplying() {
		t.Fatal("unverified recovery permitted resource loads")
	}
	a.updatePolicy.verifiedInstalled = true
	a.updatePolicy.applying = true
	if !a.updateApplying() {
		t.Fatal("active replacement permitted resource loads")
	}
}
