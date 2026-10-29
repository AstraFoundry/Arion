//go:build darwin && cgo

package imebridge

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Run only through the explicit Developer ID fixture runner. Ordinary Go test
// binaries remain ad-hoc and continue exercising rejection in peer_darwin_test.
func TestDarwinSignedPeerChild(t *testing.T) {
	socket := os.Getenv("ARION_SIGNED_PEER_SOCKET")
	if socket == "" {
		t.Skip("requires an explicit signed-process fixture")
	}
	conn, err := net.DialTimeout("unix", socket, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	peer, release, err := inspectPeer(conn)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	accepted := peer.BindPublisher(os.Getenv("ARION_SIGNED_PEER_APP_ID"))
	encoder := json.NewEncoder(conn)
	if os.Getenv("ARION_SIGNED_PEER_BRIDGE") == "1" {
		if os.Getenv("ARION_SIGNED_PEER_REQUIRE_PARENT") == "1" && !accepted {
			t.Fatal("signed helper rejected its expected application publisher")
		}
		for _, frame := range []message{
			{Op: "hello", Build: "signed-process-fixture", Protocol: 1, Capabilities: []string{vocabularyCapability}},
			{Op: "activate", Activation: "owned-editor"},
		} {
			// Rejected peers may be disconnected immediately after hello.
			if err := encoder.Encode(frame); err != nil && os.Getenv("ARION_SIGNED_PEER_REQUIRE_PARENT") == "1" {
				t.Fatal(err)
			}
		}
	} else if err := encoder.Encode(struct {
		Authenticated bool `json:"authenticated"`
	}{accepted}); err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestDarwinSignedPublisherPair(t *testing.T) {
	fixtures := os.Getenv("ARION_SIGNED_PEER_FIXTURES")
	if fixtures == "" {
		t.Skip("requires an explicit Developer ID fixture runner")
	}
	for _, scenario := range []struct {
		name         string
		peer, parent bool
	}{
		{"ime-peer", true, true}, {"wrong-peer", false, true}, {"adhoc-peer", false, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			directory, err := os.MkdirTemp("/tmp", "arion-signed-peer-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(directory)
			socket := filepath.Join(directory, "peer.sock")
			listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			if err = listener.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
				t.Fatal(err)
			}
			cmd, input := startSignedPeer(t, fixtures, scenario.name, socket, false)
			conn, err := listener.Accept()
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if err = conn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
				t.Fatal(err)
			}
			peer, release, err := inspectPeer(conn)
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			if peer.ProcessID != uint32(cmd.Process.Pid) || !peer.Alive() {
				t.Fatal("socket audit identity differs from live helper")
			}
			if peer.BindPublisher(BundleID + ".not-the-peer") {
				t.Fatal("wrong identifier acquired publisher authority")
			}
			if got := peer.BindPublisher(BundleID); got != scenario.peer {
				t.Fatalf("peer publisher admission: got %v, want %v", got, scenario.peer)
			}
			var response struct {
				Authenticated bool `json:"authenticated"`
			}
			if err = json.NewDecoder(conn).Decode(&response); err != nil {
				t.Fatal(err)
			}
			if response.Authenticated != scenario.parent {
				t.Fatalf("reverse publisher admission: got %v, want %v", response.Authenticated, scenario.parent)
			}
			if !peer.Alive() {
				t.Fatal("identity lease unexpectedly expired while peer was running")
			}
			input.Close()
			if err = cmd.Wait(); err != nil {
				t.Fatal(err)
			}
			if peer.Alive() || peer.BindPublisher(BundleID) {
				t.Fatal("exited signed peer retained publisher authority")
			}
			release()
			if peer.Alive() || peer.BindPublisher(BundleID) {
				t.Fatal("released signed peer retained publisher authority")
			}
		})
	}
}

func TestDarwinSignedBridgeAdmission(t *testing.T) {
	fixtures := os.Getenv("ARION_SIGNED_PEER_FIXTURES")
	if fixtures == "" {
		t.Skip("requires an explicit Developer ID fixture runner")
	}
	for _, name := range []string{"ime-peer", "wrong-peer", "adhoc-peer"} {
		t.Run(name, func(t *testing.T) {
			directory, err := os.MkdirTemp("/tmp", "arion-signed-bridge-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(directory)
			socket := filepath.Join(directory, "bridge.sock")
			bridge := New(socket, nil, nil)
			defer bridge.Close()
			admitted := make(chan bool, 1)
			bridge.RequireClientVerification(func(peer PeerIdentity, hello ClientHello) error {
				accepted := peer.BindPublisher(BundleID)
				admitted <- accepted
				if !accepted {
					return ErrVocabularyRejected
				}
				return nil
			})
			if err = bridge.Listen(); err != nil {
				t.Fatal(err)
			}
			cmd, input := startSignedPeer(t, fixtures, name, socket, true)
			select {
			case accepted := <-admitted:
				if accepted != (name == "ime-peer") {
					t.Fatal("bridge admitted a wrong publisher or rejected the signed host")
				}
			case <-time.After(15 * time.Second):
				t.Fatal("signed bridge admission timed out")
			}
			if name == "ime-peer" {
				wait(t, bridge.ActiveClient)
				bridge.mu.Lock()
				verified := bridge.clientVerified && bridge.peer.Alive() && bridge.peer.ProcessID == uint32(cmd.Process.Pid)
				bridge.mu.Unlock()
				if !verified {
					t.Fatal("bridge did not bind activation to actual signed peer")
				}
			} else if bridge.ActiveClient() {
				t.Fatal("unverified claimed capability activated a client")
			}
			input.Close()
			if err = cmd.Wait(); err != nil {
				t.Fatal(err)
			}
			wait(t, func() bool { return !bridge.ActiveClient() })
		})
	}
}

func startSignedPeer(t *testing.T, fixtures, name, socket string, bridge bool) (*exec.Cmd, io.WriteCloser) {
	t.Helper()
	cmd := exec.Command(filepath.Join(fixtures, name), "-test.run=^TestDarwinSignedPeerChild$", "-test.timeout=30s")
	mode, requireParent := "0", "0"
	if bridge {
		mode = "1"
		if name == "ime-peer" {
			requireParent = "1"
		}
	}
	cmd.Env = append(os.Environ(), "ARION_SIGNED_PEER_SOCKET="+socket, "ARION_SIGNED_PEER_BRIDGE="+mode, "ARION_SIGNED_PEER_REQUIRE_PARENT="+requireParent)
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
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
