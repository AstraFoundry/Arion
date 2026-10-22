package imebridge

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func socketFixture(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "arion-recovery-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "ime.sock")
}

func TestOrphanedSocketRecoversAndAcceptsHost(t *testing.T) {
	path := socketFixture(t)
	old, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	old.SetUnlinkOnClose(false)
	old.Close()
	m := New(path, nil, nil)
	if err := m.Listen(); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.Write([]byte("{\"op\":\"hello\",\"build\":\"new-build\"}\n"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.WaitReady(ctx, "new-build"); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryNeverRemovesUnrelatedOrLivePaths(t *testing.T) {
	for _, kind := range []string{"file", "symlink", "live-socket"} {
		t.Run(kind, func(t *testing.T) {
			path := socketFixture(t)
			switch kind {
			case "file":
				if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink("missing", path); err != nil {
					t.Fatal(err)
				}
			case "live-socket":
				l, err := net.Listen("unix", path)
				if err != nil {
					t.Fatal(err)
				}
				defer l.Close()
			}
			before, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			m := New(path, nil, nil)
			if err := m.Listen(); err == nil {
				m.Close()
				t.Fatal("replaced protected path")
			}
			after, err := os.Lstat(path)
			if err != nil || !os.SameFile(before, after) {
				t.Fatal("protected path changed", err)
			}
		})
	}
}

func TestSocketProbeDoesNotRevokeActiveComposition(t *testing.T) {
	m := New(socketFixture(t), nil, nil)
	if err := m.Listen(); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	h := attach(t, m, "active")
	s := capture(t, m, h, "session")
	if l, err := bindSocket(m.path); err == nil {
		l.Close()
		t.Fatal("replaced live listener")
	}
	// The probe's EOF must be processed without replacing the active connection.
	time.Sleep(20 * time.Millisecond)
	if _, err := m.Capabilities(context.Background(), s.Target); err != nil {
		t.Fatal("probe revoked ownership", err)
	}
}
