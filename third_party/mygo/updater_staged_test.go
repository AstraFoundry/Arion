//go:build !js

package mygo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPreparedUpdateStagesAndRecoversWithoutNetwork(t *testing.T) {
	for _, delta := range []string{"", "1.1.9"} {
		t.Run("delta="+delta, func(t *testing.T) {
			_, key, m, requests := updateServer(t, "1.2.0", "v2", delta)
			target, main := installedApp(t)
			p, err := prepareUpdate(context.Background(), m, key, "1.1.9", target, nil)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { p.Discard() })
			if got, _ := os.ReadFile(main); string(got) != "v1" {
				t.Fatalf("preparation replaced installation: %q", got)
			}
			if p.Version() != "1.2.0" {
				t.Fatal(p.Version())
			}
			recovered, err := recoverPrepared(p.Path(), key, target)
			if err != nil {
				t.Fatal(err)
			}
			before := requests("/app.tar.gz") + requests("/app.delta")
			if err := recovered.Apply(); err != nil {
				t.Fatal(err)
			}
			if got, _ := os.ReadFile(main); string(got) != "v2" {
				t.Fatalf("apply: %q", got)
			}
			if requests("/app.tar.gz")+requests("/app.delta") != before {
				t.Fatal("recovery/apply contacted the feed")
			}
			if !recovered.Applied() {
				t.Fatal("completed update not reported")
			}
			receipt, err := recoverPrepared(p.Path(), key, target)
			if err != nil || !receipt.Applied() {
				t.Fatalf("completed receipt not recovered: %v", err)
			}
			if err := p.Discard(); !errors.Is(err, ErrUpdateRecoveryRequired) {
				t.Fatalf("stale handle removed completed receipt: %v", err)
			}
			if err := receipt.Acknowledge(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(p.work); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("acknowledged stage remains: %v", err)
			}
			if err := recovered.Apply(); err == nil {
				t.Fatal("stage applied twice")
			}
		})
	}
}

func TestPreparedUpdateRejectsChangedState(t *testing.T) {
	for _, change := range []string{"target", "stage", "stage-and-record", "artifact", "key", "identity", "interrupted"} {
		t.Run(change, func(t *testing.T) {
			_, key, m, _ := updateServer(t, "1.2.0", "v2", "")
			target, main := installedApp(t)
			p, err := prepareUpdate(context.Background(), m, key, "1.1.9", target, nil)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.RemoveAll(p.work) })
			stage, err := updateStageRoot(filepath.Join(p.work, "app"))
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "target":
				err = os.WriteFile(main, []byte("external change"), 0o755)
			case "stage", "stage-and-record":
				err = os.WriteFile(filepath.Join(stage, "main"), []byte("tampered"), 0o755)
				if change == "stage-and-record" {
					p.record.StageDigest, err = updateTreeDigest(filepath.Join(p.work, "app"))
					if err == nil {
						err = p.save()
					}
				}
			case "artifact":
				err = os.WriteFile(filepath.Join(p.work, "update.tar.gz"), []byte("tampered"), 0o600)
			case "key":
				key = "invalid"
			case "identity":
				target = filepath.Join(filepath.Dir(target), "other-app")
			case "interrupted":
				p.record.Phase = "applying"
				err = p.save()
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = recoverPrepared(p.Path(), key, target)
			if err == nil {
				t.Fatal("recovered changed stage")
			}
			if change == "interrupted" && !errors.Is(err, ErrUpdateRecoveryRequired) {
				t.Fatalf("missing recovery diagnosis: %v", err)
			}
			if change != "target" {
				if got, _ := os.ReadFile(main); string(got) != "v1" {
					t.Fatalf("rejection changed installation: %q", got)
				}
			}
		})
	}
}

func TestPreparedUpdateCancellationDiscardsStage(t *testing.T) {
	_, key, m, _ := updateServer(t, "1.2.0", "v2", "")
	target, main := installedApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	_, err := prepareUpdate(ctx, m, key, "1.1.9", target, func(n, total int64) {
		if n == total {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	stages, err := filepath.Glob(filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".update-*"))
	if err != nil || len(stages) != 0 {
		t.Fatalf("unfinished stage retained: %v, %v", stages, err)
	}
	if got, _ := os.ReadFile(main); string(got) != "v1" {
		t.Fatalf("cancellation replaced app: %q", got)
	}
}

func TestPreparedUpdateSwapRollbackAndFailure(t *testing.T) {
	for _, failRollback := range []bool{false, true} {
		name := "restores"
		if failRollback {
			name = "preserves-failed-rollback"
		}
		t.Run(name, func(t *testing.T) {
			_, key, m, _ := updateServer(t, "1.2.0", "v2", "")
			target, main := installedApp(t)
			p, err := prepareUpdate(context.Background(), m, key, "1.1.9", target, nil)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				os.RemoveAll(p.work)
				os.RemoveAll(filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".update-lock"))
			})
			calls := 0
			failure := errors.New("injected rename failure")
			p.renameFn = func(from, to string) error {
				calls++
				if calls == 2 || failRollback && calls == 3 {
					return failure
				}
				return rename(from, to)
			}
			err = p.Apply()
			if !errors.Is(err, failure) {
				t.Fatalf("apply error: %v", err)
			}
			if failRollback {
				if !errors.Is(err, ErrUpdateRecoveryRequired) || !strings.Contains(err.Error(), p.Path()) {
					t.Fatalf("missing rollback diagnosis: %v", err)
				}
				backup := filepath.Join(p.work, "backup")
				entries, err := os.ReadDir(backup)
				if err != nil || len(entries) == 0 {
					t.Fatalf("recovery bytes lost: %v, %v", entries, err)
				}
				if err := p.Discard(); !errors.Is(err, ErrUpdateRecoveryRequired) {
					t.Fatalf("discard lost recovery bytes: %v", err)
				}
				if _, err := recoverPrepared(p.Path(), key, target); !errors.Is(err, ErrUpdateRecoveryRequired) {
					t.Fatalf("interrupted stage reused: %v", err)
				}
			} else {
				if got, _ := os.ReadFile(main); string(got) != "v1" {
					t.Fatalf("rollback lost original: %q", got)
				}
				if err := p.Discard(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPreparedUpdateSerializesRecoveredHandles(t *testing.T) {
	_, key, m, _ := updateServer(t, "1.2.0", "v2", "")
	target, _ := installedApp(t)
	p, err := prepareUpdate(context.Background(), m, key, "1.1.9", target, nil)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := recoverPrepared(p.Path(), key, target)
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	p.renameFn = func(from, to string) error {
		select {
		case <-entered:
		default:
			close(entered)
			<-release
		}
		return rename(from, to)
	}
	done := make(chan error, 1)
	go func() { done <- p.Apply() }()
	<-entered
	if err := recovered.Apply(); err == nil {
		t.Error("concurrent apply acquired ownership")
	}
	if err := recovered.Discard(); err == nil {
		t.Error("concurrent discard removed applying stage")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := recovered.Discard(); !errors.Is(err, ErrUpdateRecoveryRequired) {
		t.Fatalf("stale handle discarded completed receipt: %v", err)
	}
	if err := p.Acknowledge(); err != nil {
		t.Fatal(err)
	}
}

func TestPreparedFilesRollbackRemovesNewEntries(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("multi-file swap applies to Windows and Linux")
	}
	_, key, m, _ := updateServer(t, "1.2.0", "v2", "")
	target, _ := installedApp(t)
	if err := os.Remove(filepath.Join(target, "code")); err != nil {
		t.Fatal(err)
	}
	p, err := prepareUpdate(context.Background(), m, key, "1.1.9", target, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Discard()
	calls := 0
	p.renameFn = func(from, to string) error {
		calls++
		if calls == 3 {
			return errors.New("second entry failed")
		}
		return rename(from, to)
	}
	if err := p.Apply(); err == nil {
		t.Fatal("swap unexpectedly completed")
	}
	if _, err := os.Stat(filepath.Join(target, "code")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("new entry survived rollback: %v", err)
	}
}

func TestAppliedReceiptRejectsChangedInstalledBytes(t *testing.T) {
	for _, delta := range []string{"", "1.1.9"} {
		t.Run("delta="+delta, func(t *testing.T) {
			_, key, m, _ := updateServer(t, "1.2.0", "v2", delta)
			target, main := installedApp(t)
			prepared, err := prepareUpdate(context.Background(), m, key, "1.1.9", target, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(prepared.work)
			if err := prepared.Apply(); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(main, []byte("altered installed bytes"), 0o755); err != nil {
				t.Fatal(err)
			}
			if _, err := recoverPrepared(prepared.Path(), key, target); err == nil {
				t.Fatal("completed receipt blessed altered installed bytes")
			}
			if _, err := os.Stat(prepared.Path()); err != nil {
				t.Fatalf("failed validation deleted recovery receipt: %v", err)
			}
		})
	}
}

func TestAcknowledgedCleanupClearsRetainedOwnershipJournal(t *testing.T) {
	_, key, m, _ := updateServer(t, "1.2.0", "v2", "")
	target, _ := installedApp(t)
	prepared, err := prepareUpdate(context.Background(), m, key, "1.1.9", target, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(prepared.work)
	if err := prepared.Apply(); err != nil {
		t.Fatal(err)
	}
	// Simulate termination after recording acknowledgement but before clearing
	// the stable ownership pointer. Startup cleanup must clear that pointer
	// before it removes the receipt bytes it refers to.
	prepared.record.Phase = "acknowledged"
	if err := prepared.save(); err != nil {
		t.Fatal(err)
	}
	oldExe := startExe
	defer func() { startExe = oldExe }()
	startExe = filepath.Join(target, "main")
	if runtime.GOOS == "darwin" {
		startExe = filepath.Join(target, "Contents", "MacOS", "main")
	}
	cleanPreparedUpdateLeftovers()
	if _, err := os.Stat(prepared.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("acknowledged receipt survived cleanup: %v", err)
	}
	ownership, err := acquireUpdateOwnership(target)
	if err != nil {
		t.Fatal(err)
	}
	defer ownership.close()
	if err := ownership.allow("", false); err != nil {
		t.Fatalf("cleanup stranded ownership pointer: %v", err)
	}
}
