package macos

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRemovalRetainsFilesUntilIndependentVerificationAndHostExit(t *testing.T) {
	for _, stage := range []string{"switch", "disable", "verify", "stop", "reselected", "restarted", "changed", "cancelled", "success"} {
		t.Run(stage, func(t *testing.T) {
			bundle := bundleFixture(t, t.TempDir(), "ArionInput.app", "previous")
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			failure := errors.New("refused")
			checks := 0
			op := removalOperations{
				switchAway: func() error {
					if stage == "switch" {
						return failure
					}
					return nil
				},
				disable: func() error {
					if stage == "disable" {
						return failure
					}
					return nil
				},
				verify: func() error {
					checks++
					if stage == "verify" || stage == "reselected" && checks > 1 {
						return ErrIMERemovalPending
					}
					return nil
				},
				stop: func() error {
					if stage == "stop" {
						return failure
					}
					if stage == "cancelled" {
						cancel()
					}
					if stage == "changed" {
						return os.WriteFile(filepath.Join(bundle, "Contents", "MacOS", "ArionInput"), []byte("another installer"), 0755)
					}
					return nil
				},
				stopped: func() error {
					if stage == "restarted" {
						return failure
					}
					return nil
				},
			}
			err := completeInputRemoval(ctx, bundle, op)
			_, statErr := os.Stat(bundle)
			if stage == "success" {
				if err != nil || !os.IsNotExist(statErr) || checks != 2 {
					t.Fatal(err, statErr, checks)
				}
			} else if err == nil || statErr != nil {
				t.Fatalf("unsafe deletion at %s: %v %v", stage, err, statErr)
			}
		})
	}
}

func TestMaintenanceLockExcludesAnotherOpenAndReleases(t *testing.T) {
	root := t.TempDir()
	unlock, err := lockInputMaintenance(root)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := lockInputMaintenance(root); err == nil {
		other()
		t.Fatal("parallel maintenance acquired lock")
	}
	unlock()
	other, err := lockInputMaintenance(root)
	if err != nil {
		t.Fatal(err)
	}
	other()
}

func TestInputBundleStageValidationPrecedesHostStop(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	source := bundleFixture(t, root, "source.app", "new")
	failure := errors.New("staged signature rejected")
	err := replaceInputBundleWithLifecycle(source, old, func() error { t.Fatal("activated unverified stage"); return nil }, func() error { t.Fatal("interrupted before validation"); return nil }, inputBundleLifecycle{
		validate: func(path string) error {
			if bundleContents(t, path) != "new" {
				t.Fatal("missing stage")
			}
			return failure
		},
		beforeReplace: func() error { t.Fatal("stopped before validation"); return nil },
	})
	if !errors.Is(err, failure) || bundleContents(t, old) != "previous" {
		t.Fatal(err)
	}
}

func TestInterruptedTransactionRepairRetainsCopiesUntilCommit(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	source := bundleFixture(t, root, "source.app", "new")
	failure := errors.New("host would not stop")
	err := replaceInputBundleWithLifecycle(source, old, func() error { return failure }, func() error { return nil }, inputBundleLifecycle{beforeRollback: func() error { return failure }})
	if err == nil {
		t.Fatal("missing failure")
	}
	records, _ := filepath.Glob(filepath.Join(root, ".arion-transactions", "install-*"))
	if len(records) != 1 {
		t.Fatal(records)
	}
	err = replaceInputBundle(source, old, func() error {
		if _, err := os.Stat(records[0]); err != nil {
			t.Fatal("recovery copy removed before readiness")
		}
		return nil
	}, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	requireNoTransaction(t, root)
}

func TestInterruptedTransactionRejectsExternalChanges(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	source := bundleFixture(t, root, "source.app", "new")
	_ = replaceInputBundleWithLifecycle(source, old, func() error { return errors.New("failed") }, func() error { return nil }, inputBundleLifecycle{beforeRollback: func() error { return errors.New("running") }})
	if err := os.WriteFile(filepath.Join(old, "Contents", "MacOS", "ArionInput"), []byte("external"), 0755); err != nil {
		t.Fatal(err)
	}
	err := replaceInputBundle(source, old, func() error { t.Fatal("ignored recovery conflict"); return nil }, func() error { return nil })
	if err == nil || bundleContents(t, old) != "external" {
		t.Fatal(err)
	}
}
