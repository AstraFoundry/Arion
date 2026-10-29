package imebridge

import (
	"context"
	"net"
	"time"
)

// ClientHello describes a wire contract, not authenticated process identity.
type ClientHello struct {
	Build    string
	Protocol int
}

// PeerIdentity is supplied by the OS transport. Its process lease remains held
// for the connection lifetime; Alive also prevents accepting an exited peer.
type PeerIdentity struct {
	ProcessID              uint32
	SessionID              uint32
	SID                    string
	Integrity              uint32
	alive                  func() bool
	matchesCodeRequirement func(string) bool
	bindPublisher          func(string) bool
}

func (p PeerIdentity) Alive() bool { return p.alive != nil && p.alive() }

// MatchesCodeRequirement verifies live macOS code against a trusted caller's
// signing requirement. A hello field must never supply the requirement.
func (p PeerIdentity) MatchesCodeRequirement(requirement string) bool {
	return p.matchesCodeRequirement != nil && p.matchesCodeRequirement(requirement)
}

// BindPublisher requires the named peer to share this running application's
// Developer ID publisher, then pins that requirement for subsequent Alive checks.
func (p PeerIdentity) BindPublisher(identifier string) bool {
	return p.bindPublisher != nil && p.bindPublisher(identifier)
}

type ClientVerifier func(PeerIdentity, ClientHello) error

type peerConnection interface {
	peerIdentity() (PeerIdentity, func(), error)
}

func inspectPeer(conn net.Conn) (PeerIdentity, func(), error) {
	if transport, ok := conn.(peerConnection); ok {
		return transport.peerIdentity()
	}
	return inspectPlatformPeer(conn)
}

// RequireClientVerification is configured before Listen. Unconfigured bridges
// retain the macOS protocol-0 behavior; Windows installs an explicit verifier.
func (m *IME) RequireClientVerification(verify ClientVerifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.verifyClient = verify
	m.clientVerified = false
}

func (m *IME) rejectClient(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// A forged/unsupported connection cannot evict or poison a valid client.
	if m.conn == nil {
		m.clientError = err
		m.notifyLocked()
	}
}

// A focus connection may still be authenticating when dictation is pressed.
// Wait only for an already pending admission, with a fixed bound and caller
// cancellation. The capability is still checked atomically inside Capture.
func (m *IME) waitForClientVerification(ctx context.Context) error {
	m.mu.Lock()
	pending := m.verifyClient != nil && m.verifying > 0 && (m.conn == nil || m.activation == "")
	m.mu.Unlock()
	if !pending {
		return nil
	}
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		m.mu.Lock()
		ready := m.conn != nil && m.activation != ""
		failed := m.closed || (m.conn == nil && m.verifying == 0)
		changed := m.changed
		m.mu.Unlock()
		if ready || failed {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		case <-changed:
		}
	}
}
