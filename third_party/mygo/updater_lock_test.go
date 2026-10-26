//go:build !js

package mygo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateOwnershipSubprocess(t *testing.T) {
	target := os.Getenv("MYGO_TEST_UPDATE_LOCK_TARGET")
	if target == "" {
		t.Skip("ownership subprocess only")
	}
	ownership, err := acquireUpdateOwnership(target)
	if err != nil {
		t.Fatal(err)
	}
	defer ownership.close()
	if err := os.WriteFile(os.Getenv("MYGO_TEST_UPDATE_LOCK_READY"), []byte("locked"), 0o600); err != nil {
		t.Fatal(err)
	}
	<-time.After(30 * time.Second)
	t.Fatal("ownership subprocess was not stopped by parent")
}

func TestAppliedReceiptRecoversAfterOwnershipProcessCrash(t *testing.T) {
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
	ready := filepath.Join(t.TempDir(), "ready")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestUpdateOwnershipSubprocess$")
	cmd.Env = append(os.Environ(), "MYGO_TEST_UPDATE_LOCK_TARGET="+target, "MYGO_TEST_UPDATE_LOCK_READY="+ready)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("ownership subprocess did not acquire its lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := prepared.Acknowledge(); err == nil {
		t.Fatal("acknowledged while another process owned the lock")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	recovered, err := recoverPrepared(prepared.Path(), key, target)
	if err != nil || !recovered.Applied() {
		t.Fatalf("completed receipt did not survive ownership crash: %v", err)
	}
	if err := recovered.Acknowledge(); err != nil {
		t.Fatalf("kernel did not release crashed ownership: %v", err)
	}
	if _, err := os.Stat(prepared.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("acknowledged receipt remains: %v", err)
	}
}
