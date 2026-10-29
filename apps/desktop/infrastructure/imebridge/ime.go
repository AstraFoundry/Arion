// Package imebridge binds local input-method messages to a connection,
// activation and dictation session. It never exposes native client pointers.
package imebridge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

type message struct {
	Capabilities       []string               `json:"capabilities,omitempty"`
	Request            uint64                 `json:"request,omitempty"`
	VocabularyRevision uint64                 `json:"vocabularyRevision,omitempty"`
	Terms              *[]domain.KeyboardTerm `json:"terms,omitempty"`
	Status             string                 `json:"status,omitempty"`
	Build              string                 `json:"build,omitempty"`
	Protocol           int                    `json:"protocol,omitempty"`
	Op                 string                 `json:"op"`
	Activation         string                 `json:"activation"`
	Session            string                 `json:"session,omitempty"`
	Client             string                 `json:"client,omitempty"`
	Text               string                 `json:"text"`
	Revision           uint64                 `json:"revision,omitempty"`
}

type IME struct {
	vocabularyCapable  bool
	vocabularyDelivery *vocabularyDelivery
	vocabularyRequest  uint64
	vocabulary         *vocabularyExchange
	path               string
	mu                 sync.Mutex
	conn               net.Conn
	listener           net.Listener
	releaseSocket      func()
	build              string
	verifyClient       ClientVerifier
	clientVerified     bool
	clientError        error
	peer               PeerIdentity
	admission          uint64
	publishedAdmission uint64
	verifying          int
	changed            chan struct{}
	updating           bool
	closed             bool
	activation, client string
	identity           contracts.Identity
	token              contracts.TargetToken
	revision           uint64
	committing         bool
	ack                chan bool
	begin              chan bool
	requireBegin       bool
	onLeave            func()
	onDictate          func()
}

// New serves the socket at path. onLeave runs when an owned session loses
// its client; onDictate runs when the current client chooses Voice Input
// from the input menu (IME-06). Both run on a connection goroutine.
func New(path string, onLeave, onDictate func()) *IME {
	return &IME{path: path, onLeave: onLeave, onDictate: onDictate, changed: make(chan struct{})}
}

// RequireBeginAcknowledgement makes capture wait for the host's successful
// edit-session acquisition. Windows TSF uses it to refuse unsafe text stores
// before recording starts; older IMK hosts retain their existing protocol.
func (m *IME) RequireBeginAcknowledgement() { m.mu.Lock(); m.requireBegin = true; m.mu.Unlock() }

// ActiveClient reports a live activation. Windows hosts announce only the
// currently focused TSF thread and invalidate it on source or focus changes.
func (m *IME) ActiveClient() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.conn != nil && m.activation != ""
}

// ConnectedBuild is the executable hash announced by the current native host.
func (m *IME) ConnectedBuild() string { m.mu.Lock(); defer m.mu.Unlock(); return m.build }

func (m *IME) Listen() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listener != nil || m.closed {
		return errors.New("input method listener already started or closed")
	}
	if runtime.GOOS != "windows" {
		if err := os.MkdirAll(filepath.Dir(m.path), 0700); err != nil {
			return err
		}
	}
	listener, release, err := listenSocket(m.path)
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(m.path, 0600); err != nil {
			listener.Close()
			release()
			return err
		}
	}
	m.listener, m.releaseSocket = listener, release
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go m.serve(conn)
		}
	}()
	return nil
}

// invalidateLocked wakes any pending commit and revokes its capability.
func (m *IME) invalidateLocked() bool {
	m.invalidateVocabularyLocked()
	had := m.token != ""
	m.identity = contracts.Identity{}
	m.token = ""
	if had {
		m.notifyLocked()
	}
	m.revision = 0
	m.committing = false
	if m.ack != nil {
		m.ack <- false
		m.ack = nil
	}
	if m.begin != nil {
		m.begin <- false
		m.begin = nil
	}
	return had
}
func (m *IME) leave(had bool) {
	if had && m.onLeave != nil {
		m.onLeave()
	}
}
func (m *IME) Close() {
	m.mu.Lock()
	m.closed = true
	if m.vocabularyDelivery != nil {
		m.vocabularyDelivery.stop()
	}
	if m.listener != nil {
		m.listener.Close()
		m.listener = nil
	}
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}
	if m.releaseSocket != nil {
		m.releaseSocket()
		m.releaseSocket = nil
	}
	m.activation, m.build = "", ""
	m.notifyLocked()
	had := m.invalidateLocked()
	m.mu.Unlock()
	m.leave(had)
}
func (m *IME) serve(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	if !scanner.Scan() {
		conn.Close()
		return
	}
	var hello message
	if json.Unmarshal(scanner.Bytes(), &hello) != nil ||
		!((hello.Op == "hello" && hello.Build != "") || (hello.Op == "activate" && hello.Activation != "")) {
		conn.Close()
		return
	}
	m.mu.Lock()
	verify := m.verifyClient
	m.admission++
	ticket := m.admission
	if verify != nil {
		m.verifying++
	}
	m.mu.Unlock()
	verificationPending := verify != nil
	defer func() {
		if verificationPending {
			m.mu.Lock()
			m.verifying--
			m.notifyLocked()
			m.mu.Unlock()
		}
	}()
	var peer PeerIdentity
	if verify != nil {
		var release func()
		var err error
		peer, release, err = inspectPeer(conn)
		defer release()
		if err == nil {
			err = verify(peer, ClientHello{Build: hello.Build, Protocol: hello.Protocol})
		}
		if err != nil {
			m.rejectClient(err)
			conn.Close()
			return
		}
	}
	_ = conn.SetReadDeadline(time.Time{})
	m.mu.Lock()
	if m.closed || ticket < m.publishedAdmission {
		m.mu.Unlock()
		conn.Close()
		return
	}
	if m.conn != nil {
		m.conn.Close()
	}
	had := m.invalidateLocked()
	m.activation, m.client, m.build = "", "", hello.Build
	if hello.Op == "activate" {
		m.activation, m.client = hello.Activation, hello.Client
	}
	m.conn = conn
	m.vocabularyCapable = false
	for _, capability := range hello.Capabilities {
		if capability == vocabularyCapability {
			m.vocabularyCapable = true
		}
	}
	m.publishedAdmission = ticket
	if verificationPending {
		m.verifying--
		verificationPending = false
	}
	m.peer, m.clientVerified, m.clientError = peer, verify != nil, nil
	m.notifyLocked()
	m.mu.Unlock()
	m.leave(had)
	defer conn.Close()
	for scanner.Scan() {
		var msg message
		if json.Unmarshal(scanner.Bytes(), &msg) != nil {
			continue
		}
		m.mu.Lock()
		had, dictate := false, false
		if m.conn != conn {
			m.mu.Unlock()
			break
		}
		switch msg.Op {
		case "vocabulary_state", "vocabulary_published":
			m.receiveVocabularyLocked(msg)
		case "begun":
			if msg.Activation == m.activation && msg.Session == m.identity.SessionID && m.begin != nil {
				m.begin <- true
				m.begin = nil
			}
		case "activate":
			if msg.Activation != "" && msg.Activation != m.activation {
				had = m.invalidateLocked()
				m.activation = msg.Activation
				m.client = msg.Client
				m.notifyLocked()
			}
		case "deactivate", "escape":
			if msg.Activation != "" && msg.Activation == m.activation && (msg.Op == "deactivate" || (m.token != "" && msg.Session == m.identity.SessionID)) {
				had = m.invalidateLocked()
				if msg.Op == "deactivate" {
					m.activation = ""
				}
			}
		case "committed":
			if msg.Activation == m.activation && msg.Session == m.identity.SessionID && m.ack != nil {
				m.ack <- true
				m.ack = nil
			}
		case "dictate":
			dictate = msg.Activation != "" && msg.Activation == m.activation
		}
		m.mu.Unlock()
		m.leave(had)
		if dictate && m.onDictate != nil {
			m.onDictate()
		}
	}
	m.mu.Lock()
	had = false
	if m.conn == conn {
		m.conn = nil
		m.clientVerified = false
		m.activation, m.build = "", ""
		m.notifyLocked()
		had = m.invalidateLocked()
	}
	m.mu.Unlock()
	m.leave(had)
}
func changed() error { return domain.Fail(domain.CodeTargetChanged, "input method ownership changed") }
func (m *IME) ownsLocked(token contracts.TargetToken) bool {
	return m.conn != nil && m.activation != "" && m.token != "" && token == m.token
}
func (m *IME) sendLocked(ctx context.Context, msg message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.conn == nil {
		return changed()
	}
	msg.Activation = m.activation
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := m.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	cancelled := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { _ = m.conn.SetWriteDeadline(time.Now()); close(cancelled) })
	defer func() {
		if !stop() {
			<-cancelled
		}
	}()
	if (msg.Op == "vocabulary_publish" || msg.Op == "vocabulary_query") && len(data) >= (1<<20)-1 {
		return domain.ErrInvalidKeyboardVocabulary
	}
	data = append(data, '\n')
	for len(data) > 0 {
		n, err := m.conn.Write(data)
		if err != nil {
			_ = m.conn.Close()
			return changed()
		}
		if n == 0 {
			_ = m.conn.Close()
			return changed()
		}
		data = data[n:]
	}
	return nil
}
func (m *IME) Capture(ctx context.Context, id contracts.Identity) (contracts.TargetSnapshot, error) {
	if err := m.waitForClientVerification(ctx); err != nil {
		return contracts.TargetSnapshot{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updating {
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "input method update in progress")
	}
	if m.conn == nil {
		if m.clientError != nil {
			return contracts.TargetSnapshot{}, m.clientError
		}
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "input method connection unavailable")
	}
	// The proof and capability are checked under the same lock as the begin
	// write. A concurrent replacement cannot substitute another connection.
	if m.verifyClient != nil && (!m.clientVerified || !m.peer.Alive()) {
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeArtifactUnverified, domain.DetailInputMethodArtifactUnverified)
	}
	if m.activation == "" {
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "input method client unavailable")
	}
	if m.token != "" {
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetChanged, "input method already owns a session")
	}
	if id.SessionID == "" {
		return contracts.TargetSnapshot{}, changed()
	}
	var acknowledged chan bool
	if m.requireBegin {
		acknowledged = make(chan bool, 1)
		m.begin = acknowledged
	}
	if err := m.sendLocked(ctx, message{Op: "begin", Session: id.SessionID}); err != nil {
		m.begin = nil
		return contracts.TargetSnapshot{}, err
	}
	m.invalidateVocabularyLocked()
	m.identity = id
	m.token = contracts.TargetToken("ime:" + m.activation + ":" + id.SessionID)
	if acknowledged != nil {
		token := m.token
		m.mu.Unlock()
		timer := time.NewTimer(time.Second)
		accepted := false
		select {
		case accepted = <-acknowledged:
		case <-ctx.Done():
		case <-timer.C:
		}
		timer.Stop()
		m.mu.Lock()
		if !accepted || ctx.Err() != nil || m.token != token || !m.ownsLocked(token) {
			if m.token == token {
				_ = m.sendLocked(context.Background(), message{Op: "cancel", Session: id.SessionID})
				m.invalidateLocked()
			}
			if ctx.Err() != nil {
				return contracts.TargetSnapshot{}, ctx.Err()
			}
			return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "input method refused composition acquisition")
		}
	}
	return contracts.TargetSnapshot{Identity: id, Target: m.token, Application: m.client}, nil
}
func (m *IME) Update(ctx context.Context, token contracts.TargetToken, draft contracts.Draft) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.ownsLocked(token) || draft.Identity != m.identity || m.committing || draft.Revision <= m.revision {
		return changed()
	}
	if err := m.sendLocked(ctx, message{Op: "mark", Session: draft.SessionID, Revision: draft.Revision, Text: draft.Text}); err != nil {
		return err
	}
	m.revision = draft.Revision
	return nil
}
func (m *IME) Cancel(ctx context.Context, id contracts.Identity, token contracts.TargetToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.ownsLocked(token) || id != m.identity {
		return nil
	}
	err := m.sendLocked(ctx, message{Op: "cancel", Session: id.SessionID})
	if err != nil && m.conn != nil {
		_ = m.conn.Close()
	}
	m.invalidateLocked()
	return err
}
func (m *IME) ReadContext(context.Context, contracts.Identity, contracts.TargetToken) (contracts.ContextSnapshot, error) {
	return contracts.ContextSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "context read not offered")
}
func (m *IME) Commit(ctx context.Context, request contracts.CommitRequest) (contracts.CommitResult, error) {
	result := contracts.CommitResult{Identity: request.Identity, Outcome: contracts.NotApplied}
	m.mu.Lock()
	if !m.ownsLocked(request.Target) || request.Identity != m.identity || m.committing {
		m.mu.Unlock()
		result.Failure = domain.CodeTargetChanged
		return result, nil
	}
	if err := ctx.Err(); err != nil {
		m.mu.Unlock()
		return result, err
	}
	m.committing = true
	ack := make(chan bool, 1)
	m.ack = ack
	err := m.sendLocked(ctx, message{Op: "commit", Session: request.SessionID, Text: request.Text})
	if err != nil {
		m.ack = nil
		m.invalidateLocked()
		m.mu.Unlock()
		// A transport error after a write starts cannot prove non-insertion.
		result.Outcome = contracts.Unknown
		result.Failure = domain.CodeOf(err)
		return result, nil
	}
	m.mu.Unlock()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	result.Outcome = contracts.Unknown
	select {
	case applied := <-ack:
		if applied {
			result.Outcome = contracts.Applied
		}
	case <-ctx.Done():
	case <-timer.C:
	}
	m.mu.Lock()
	if m.token == request.Target {
		m.ack = nil
		m.invalidateLocked()
	}
	m.mu.Unlock()
	return result, nil
}

// BeginUpdate prevents new owned sessions while the host is replaced.
func (m *IME) BeginUpdate() {
	m.mu.Lock()
	m.updating = true
	m.invalidateVocabularyLocked()
	m.mu.Unlock()
}
func (m *IME) EndUpdate() { m.mu.Lock(); m.updating = false; m.notifyLocked(); m.mu.Unlock() }

func (m *IME) notifyLocked() { close(m.changed); m.changed = make(chan struct{}) }

// WaitReady requires a connection from the expected executable, independently
// of whether the current window has an editable IMK client.
func (m *IME) WaitReady(ctx context.Context, build string) error {
	if build == "" {
		return errors.New("input method build identity is empty")
	}
	for {
		m.mu.Lock()
		ready, closed, changed := m.conn != nil && m.build == build, m.closed, m.changed
		m.mu.Unlock()
		if closed {
			return errors.New("input method transport closed")
		}
		if ready {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}

// EnsureListening also repairs a previous failed startup attempt before an
// update can replace the host. Closed transports cannot be restarted.
func (m *IME) EnsureListening() error {
	m.mu.Lock()
	listening := m.listener != nil
	m.mu.Unlock()
	if listening {
		return nil
	}
	return m.Listen()
}

// UpdateBusy is conservative for hosts without a keyboard activity protocol:
// an activated editor may own preedit even with no voice session.
func (m *IME) UpdateBusy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updating || m.activation != "" || m.token != "" || m.committing
}

// TryMaintenance excludes new app-owned sessions while the main app replaces
// itself. The independently installed keyboard host stays alive throughout.
func (m *IME) TryMaintenance() (func(), bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updating || m.activation != "" || m.token != "" || m.committing {
		return nil, false
	}
	m.updating = true
	var once sync.Once
	return func() { once.Do(m.EndUpdate) }, true
}
