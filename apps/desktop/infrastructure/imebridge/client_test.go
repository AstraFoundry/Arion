package imebridge

import (
	"bufio"
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

type proofConnection struct {
	net.Conn
	peer     PeerIdentity
	released *atomic.Int32
}

func (c proofConnection) peerIdentity() (PeerIdentity, func(), error) {
	return c.peer, func() { c.released.Add(1) }, nil
}

func verifiedHost(t *testing.T, m *IME, hello message, alive *atomic.Bool) (*host, *atomic.Int32) {
	t.Helper()
	server, client := net.Pipe()
	released := new(atomic.Int32)
	peer := PeerIdentity{ProcessID: 42, SID: "test-user", Integrity: 8192, alive: alive.Load}
	go m.serve(&proofConnection{server, peer, released})
	h := &host{client, bufio.NewReader(client)}
	t.Cleanup(func() { client.Close() })
	h.send(t, hello)
	return h, released
}

func TestVerifiedClientReplacementIsAtomicWithCapture(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	m.RequireBeginAcknowledgement()
	m.RequireClientVerification(func(peer PeerIdentity, hello ClientHello) error {
		if !peer.Alive() {
			return domain.Fail(domain.CodeArtifactUnverified, domain.DetailInputMethodArtifactUnverified)
		}
		if hello.Protocol != 1 {
			return domain.Fail(domain.CodeContractUnsupported, domain.DetailInputMethodProtocolUnsupported)
		}
		return nil
	})
	alive := new(atomic.Bool)
	alive.Store(true)
	first, _ := verifiedHost(t, m, message{Op: "hello", Build: "reviewed-older-build", Protocol: 1}, alive)
	wait(t, func() bool { return m.ConnectedBuild() == "reviewed-older-build" })
	first.send(t, message{Op: "activate", Activation: "old-editor"})
	wait(t, func() bool { return m.ActiveClient() })
	type result struct {
		snapshot contracts.TargetSnapshot
		err      error
	}
	done := make(chan result, 1)
	go func() {
		snapshot, err := m.Capture(context.Background(), contracts.Identity{SessionID: "pending"})
		done <- result{snapshot, err}
	}()
	if msg := first.read(t); msg.Op != "begin" {
		t.Fatalf("missing owned begin: %+v", msg)
	}
	bad, released := verifiedHost(t, m, message{Op: "hello", Build: "pretend-current", Protocol: 99}, alive)
	bad.conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := bad.input.ReadByte(); err == nil {
		t.Fatal("rejected client was not closed")
	}
	wait(t, func() bool { return released.Load() == 1 })
	if m.ConnectedBuild() != "reviewed-older-build" {
		t.Fatal("invalid connection evicted verified old editor")
	}
	// A genuinely accepted replacement revokes the pending begin instead of
	// allowing a proof for one socket to authorize another socket's response.
	second, _ := verifiedHost(t, m, message{Op: "hello", Build: "compatible-new-build", Protocol: 1}, alive)
	wait(t, func() bool { return m.ConnectedBuild() == "compatible-new-build" })
	second.send(t, message{Op: "activate", Activation: "new-editor"})
	select {
	case got := <-done:
		if got.err == nil || got.snapshot.Target != "" {
			t.Fatal("old pending capture survived replacement")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("replacement did not revoke capture")
	}
}

func TestRejectedClientDiagnosticDoesNotBecomeMissingClient(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	m.RequireClientVerification(func(PeerIdentity, ClientHello) error {
		return domain.Fail(domain.CodeContractUnsupported, domain.DetailInputMethodProtocolUnsupported)
	})
	alive := new(atomic.Bool)
	alive.Store(true)
	h, released := verifiedHost(t, m, message{Op: "hello", Build: "unknown", Protocol: 0}, alive)
	h.conn.SetReadDeadline(time.Now().Add(time.Second))
	_, _ = h.input.ReadByte()
	wait(t, func() bool { return released.Load() == 1 })
	_, err := m.Capture(context.Background(), contracts.Identity{SessionID: "test"})
	var coded *domain.Error
	if !errors.As(err, &coded) || coded.Code != domain.CodeContractUnsupported || coded.Detail != domain.DetailInputMethodProtocolUnsupported {
		t.Fatalf("specific failure was hidden: %v", err)
	}
}

func TestExitedVerifiedPeerCannotAcquireComposition(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	m.RequireClientVerification(func(PeerIdentity, ClientHello) error { return nil })
	alive := new(atomic.Bool)
	alive.Store(true)
	h, _ := verifiedHost(t, m, message{Op: "hello", Build: "verified", Protocol: 1}, alive)
	wait(t, func() bool { return m.ConnectedBuild() == "verified" })
	h.send(t, message{Op: "activate", Activation: "editor"})
	wait(t, func() bool { return m.ActiveClient() })
	alive.Store(false)
	_, err := m.Capture(context.Background(), contracts.Identity{SessionID: "test"})
	if domain.CodeOf(err) != domain.CodeArtifactUnverified {
		t.Fatalf("dead process lease accepted: %v", err)
	}
}

func TestSlowEarlierVerificationCannotEvictNewerClient(t *testing.T) {
	for _, disconnect := range []bool{false, true} {
		t.Run(map[bool]string{false: "connected", true: "disconnected"}[disconnect], func(t *testing.T) {
			m := New("", nil, nil)
			defer m.Close()
			entered, finish := make(chan struct{}), make(chan struct{})
			m.RequireClientVerification(func(_ PeerIdentity, hello ClientHello) error {
				if hello.Build == "slow" {
					close(entered)
					<-finish
				}
				return nil
			})
			alive := new(atomic.Bool)
			alive.Store(true)
			first, released := verifiedHost(t, m, message{Op: "hello", Build: "slow", Protocol: 1}, alive)
			<-entered
			second, _ := verifiedHost(t, m, message{Op: "hello", Build: "newer", Protocol: 1}, alive)
			wait(t, func() bool { return m.ConnectedBuild() == "newer" })
			second.send(t, message{Op: "activate", Activation: "newer-editor"})
			wait(t, func() bool { return m.ActiveClient() })
			if disconnect {
				first.conn.Close()
			}
			close(finish)
			wait(t, func() bool { return released.Load() == 1 })
			if m.ConnectedBuild() != "newer" || !m.ActiveClient() {
				t.Fatal("slow earlier proof evicted newer accepted activation")
			}
		})
	}
}

func TestCaptureWaitsForPendingProofAndRespectsCancellation(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	entered, finish := make(chan struct{}), make(chan struct{})
	m.RequireClientVerification(func(PeerIdentity, ClientHello) error { close(entered); <-finish; return nil })
	alive := new(atomic.Bool)
	alive.Store(true)
	h, _ := verifiedHost(t, m, message{Op: "hello", Build: "pending", Protocol: 1}, alive)
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Capture(ctx, contracts.Identity{SessionID: "cancelled"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pending verification ignored cancellation: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := m.Capture(context.Background(), contracts.Identity{SessionID: "after-proof"})
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("capture returned before proof: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(finish)
	h.send(t, message{Op: "activate", Activation: "ready-editor"})
	if msg := h.read(t); msg.Op != "begin" {
		t.Fatalf("missing verified begin: %+v", msg)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("verified activation did not wake capture")
	}
}
