//go:build darwin && cgo

package imebridge

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestDarwinSocketPeerChild(t *testing.T) {
	path := os.Getenv("ARION_TEST_SOCKET_PEER")
	if path == "" {
		return
	}
	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if os.Getenv("ARION_TEST_SOCKET_HELLO") == "1" {
		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(message{Op: "hello", Build: "real-socket-helper", Protocol: 1}); err != nil {
			t.Fatal(err)
		}
		if err := encoder.Encode(message{Op: "activate", Activation: "real-editor"}); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func startDarwinSocketPeer(t *testing.T, path string, hello bool) (*exec.Cmd, io.WriteCloser) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestDarwinSocketPeerChild$")
	cmd.Env = append(os.Environ(), "ARION_TEST_SOCKET_PEER="+path)
	if hello {
		cmd.Env = append(cmd.Env, "ARION_TEST_SOCKET_HELLO=1")
	}
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		input.Close()
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	})
	return cmd, input
}

func TestDarwinBridgeVerifiesActualSocketPeer(t *testing.T) {
	directory, err := os.MkdirTemp("/tmp", "arion-admission-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(directory) })
	path := filepath.Join(directory, "ime.sock")
	m := New(path, nil, nil)
	defer m.Close()
	seen := make(chan uint32, 1)
	m.RequireClientVerification(func(peer PeerIdentity, hello ClientHello) error {
		// "always" is a test requirement, not the product publisher policy.
		if hello.Protocol != 1 || !peer.MatchesCodeRequirement("always") {
			return ErrVocabularyRejected
		}
		seen <- peer.ProcessID
		return nil
	})
	if err := m.Listen(); err != nil {
		t.Fatal(err)
	}
	cmd, input := startDarwinSocketPeer(t, path, true)
	select {
	case pid := <-seen:
		if pid != uint32(cmd.Process.Pid) {
			t.Fatal("bridge verified a different process")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("real socket authentication did not complete")
	}
	wait(t, m.ActiveClient)
	m.mu.Lock()
	verified := m.clientVerified && m.peer.Alive()
	m.mu.Unlock()
	if !verified {
		t.Fatal("verified admission did not retain the native proof")
	}
	input.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	wait(t, func() bool { return !m.ActiveClient() })
}

func TestDarwinSocketAuditTokenIdentityAndLifetime(t *testing.T) {
	// Darwin Unix paths have a small fixed limit; the ordinary test temp path
	// plus a descriptive test name can exceed that limit before authentication.
	directory, err := os.MkdirTemp("/tmp", "arion-peer-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(directory) })
	path := filepath.Join(directory, "peer.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	listener.SetDeadline(time.Now().Add(10 * time.Second))
	cmd, input := startDarwinSocketPeer(t, path, false)
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	peer, release, err := inspectPeer(conn)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if peer.ProcessID != uint32(cmd.Process.Pid) || peer.SID != strconv.Itoa(os.Geteuid()) || !peer.Alive() {
		t.Fatalf("incorrect kernel peer identity: pid=%d uid=%s alive=%v", peer.ProcessID, peer.SID, peer.Alive())
	}
	if !peer.MatchesCodeRequirement("always") {
		t.Fatal("running signed helper did not satisfy valid requirement")
	}
	// Go's test executable has an ad-hoc signature, not Developer ID publisher
	// authority. Neither a real product identifier nor a forged one can elevate it.
	for _, identifier := range []string{BundleID, "", "bad\"identifier", "app\x00hidden"} {
		if peer.BindPublisher(identifier) {
			t.Fatalf("test process acquired product publisher authority for %q", identifier)
		}
	}
	for _, rule := range []string{"", "never", "identifier \"not.arion.test\"", "invalid requirement syntax!", "always\x00never"} {
		if peer.MatchesCodeRequirement(rule) {
			t.Fatalf("invalid or mismatched requirement accepted: %q", rule)
		}
	}
	input.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if peer.Alive() || peer.MatchesCodeRequirement("always") {
		t.Fatal("exited process retained live code proof")
	}
	release()
	if peer.Alive() || peer.MatchesCodeRequirement("always") {
		t.Fatal("released proof remained usable")
	}
}
