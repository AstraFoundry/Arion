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
	"sync"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

type message struct {
	Op         string `json:"op"`
	Activation string `json:"activation"`
	Session    string `json:"session,omitempty"`
	Client     string `json:"client,omitempty"`
	Text       string `json:"text"`
	Revision   uint64 `json:"revision,omitempty"`
}

type IME struct {
	path               string
	mu                 sync.Mutex
	conn               net.Conn
	listener           net.Listener
	closed             bool
	activation, client string
	identity           contracts.Identity
	token              contracts.TargetToken
	revision           uint64
	committing         bool
	ack                chan bool
	onLeave            func()
}

func New(path string, onLeave func()) *IME { return &IME{path: path, onLeave: onLeave} }

func (m *IME) Listen() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listener != nil || m.closed {
		return errors.New("input method listener already started or closed")
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0700); err != nil {
		return err
	}
	// Never replace a live server or an unrelated filesystem object.
	if info, err := os.Lstat(m.path); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return errors.New("input method socket path is not a socket")
		}
		return errors.New("input method socket path already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	listener, err := net.Listen("unix", m.path)
	if err != nil {
		return err
	}
	if err := os.Chmod(m.path, 0600); err != nil {
		listener.Close()
		return err
	}
	m.listener = listener
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
	had := m.token != ""
	m.identity = contracts.Identity{}
	m.token = ""
	m.revision = 0
	m.committing = false
	if m.ack != nil {
		m.ack <- false
		m.ack = nil
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
	if m.listener != nil {
		m.listener.Close()
		m.listener = nil
	}
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}
	m.activation = ""
	had := m.invalidateLocked()
	m.mu.Unlock()
	m.leave(had)
}
func (m *IME) serve(conn net.Conn) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		conn.Close()
		return
	}
	if m.conn != nil {
		m.conn.Close()
	}
	had := m.invalidateLocked()
	m.activation = ""
	m.conn = conn
	m.mu.Unlock()
	m.leave(had)
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		var msg message
		if json.Unmarshal(scanner.Bytes(), &msg) != nil {
			continue
		}
		m.mu.Lock()
		had = false
		if m.conn != conn {
			m.mu.Unlock()
			break
		}
		switch msg.Op {
		case "activate":
			if msg.Activation != "" && msg.Activation != m.activation {
				had = m.invalidateLocked()
				m.activation = msg.Activation
				m.client = msg.Client
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
		}
		m.mu.Unlock()
		m.leave(had)
	}
	m.mu.Lock()
	had = false
	if m.conn == conn {
		m.conn = nil
		m.activation = ""
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
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil || m.activation == "" {
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "input method not active")
	}
	if m.token != "" {
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetChanged, "input method already owns a session")
	}
	if id.SessionID == "" {
		return contracts.TargetSnapshot{}, changed()
	}
	if err := m.sendLocked(ctx, message{Op: "begin", Session: id.SessionID}); err != nil {
		return contracts.TargetSnapshot{}, err
	}
	m.identity = id
	m.token = contracts.TargetToken("ime:" + m.activation + ":" + id.SessionID)
	return contracts.TargetSnapshot{Identity: id, Target: m.token, Application: m.client, Capabilities: capabilities()}, nil
}
func capabilities() contracts.Capabilities {
	return contracts.Capabilities{OwnedComposition: true, SingleInsertion: true}
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
func (m *IME) Capabilities(_ context.Context, token contracts.TargetToken) (contracts.Capabilities, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.ownsLocked(token) {
		return contracts.Capabilities{}, changed()
	}
	return capabilities(), nil
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
