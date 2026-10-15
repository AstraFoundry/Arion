//go:build darwin

package macos

/*
#include <stdlib.h>
#include "native.h"
*/
import "C"

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

// IMESourceID is the input source Arion's IMK bundle declares.
const IMESourceID = "si.astrafoundry.arion.inputmethod.voice"

// IMEState is the registration state of Arion's input source.
type IMEState int

const (
	IMEAbsent IMEState = iota
	IMERegistered
	IMEEnabled
	IMESelected
)

func cstring(s string) (*C.char, func()) {
	c := C.CString(s)
	return c, func() { C.free(unsafe.Pointer(c)) }
}

// InstalledIMEPath is where a user's input methods live.
func InstalledIMEPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Input Methods", "ArionInput.app")
}

// QueryIME reports the input source state.
func QueryIME() IMEState {
	id, free := cstring(IMESourceID)
	defer free()
	return IMEState(C.arion_ime_state(id))
}

// InstallIME copies the bundled input method into ~/Library/Input Methods,
// registers and enables it. The user selects it in the input menu (IME-01).
func InstallIME(bundled string) error {
	destination := InstalledIMEPath()
	if err := os.RemoveAll(destination); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if err := copyTree(bundled, destination); err != nil {
		return err
	}
	path, free := cstring(destination)
	defer free()
	if status := C.arion_ime_register(path); status != 0 {
		return domain.Fail(domain.CodeTargetUnsupported, "input source registration failed")
	}
	id, freeID := cstring(IMESourceID)
	defer freeID()
	if status := C.arion_ime_enable(id); status != 0 {
		return domain.Fail(domain.CodeTargetUnsupported, "input source enable failed")
	}
	return nil
}

// RemoveIME disables the input source and deletes the bundle so no stale
// registration remains.
func RemoveIME() error {
	id, free := cstring(IMESourceID)
	defer free()
	C.arion_ime_disable(id)
	return os.RemoveAll(InstalledIMEPath())
}

func copyTree(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(source, path)
		target := filepath.Join(destination, relative)
		switch {
		case info.IsDir():
			return os.MkdirAll(target, info.Mode().Perm())
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		default:
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, info.Mode().Perm())
		}
	})
}

// IME serves Arion's input method over a per-user socket. It implements
// ports.TargetProvider, ports.InputTarget and ports.CompositionTarget for
// system_ime sessions: the IME owns marked text in the focused client.
type IME struct {
	path string

	mu       sync.Mutex
	conn     net.Conn
	active   bool
	client   string
	session  string
	acks     map[string]chan struct{}
	onLeave  func()
	listener net.Listener
}

// NewIME prepares the server; OnDeactivate is called when the IME loses the
// client or the user presses Escape during composition.
func NewIME(path string, onLeave func()) *IME {
	return &IME{path: path, acks: map[string]chan struct{}{}, onLeave: onLeave}
}

// DefaultIMESocket matches the path the IMK bundle dials.
func DefaultIMESocket() string {
	base, _ := os.UserConfigDir()
	return filepath.Join(base, "Arion", "ime.sock")
}

func (m *IME) Listen() error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return err
	}
	_ = os.Remove(m.path)
	listener, err := net.Listen("unix", m.path)
	if err != nil {
		return err
	}
	_ = os.Chmod(m.path, 0o600)
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

func (m *IME) Close() {
	if m.listener != nil {
		_ = m.listener.Close()
	}
}

func (m *IME) serve(conn net.Conn) {
	m.mu.Lock()
	if m.conn != nil {
		_ = m.conn.Close()
	}
	m.conn = conn
	m.mu.Unlock()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var message struct {
			Op      string `json:"op"`
			Client  string `json:"client"`
			Session string `json:"session"`
		}
		if json.Unmarshal(scanner.Bytes(), &message) != nil {
			continue
		}
		m.mu.Lock()
		leave := false
		switch message.Op {
		case "activate":
			m.active, m.client = true, message.Client
		case "deactivate":
			m.active = false
			leave = true
		case "escape":
			leave = true
		case "committed":
			if ack := m.acks[message.Session]; ack != nil {
				close(ack)
				delete(m.acks, message.Session)
			}
		}
		onLeave := m.onLeave
		m.mu.Unlock()
		if leave && onLeave != nil {
			onLeave()
		}
	}
	m.mu.Lock()
	if m.conn == conn {
		m.conn, m.active = nil, false
	}
	m.mu.Unlock()
}

func (m *IME) send(message map[string]any) error {
	m.mu.Lock()
	conn := m.conn
	m.mu.Unlock()
	if conn == nil {
		return domain.Fail(domain.CodeTargetChanged, "input method disconnected")
	}
	data, _ := json.Marshal(message)
	_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return domain.Fail(domain.CodeTargetChanged, "input method write failed")
	}
	return nil
}

// Capture requires Arion's input source to be active in the focused client.
func (m *IME) Capture(_ context.Context, id contracts.Identity) (contracts.TargetSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.active || m.conn == nil {
		return contracts.TargetSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "input method not active")
	}
	m.session = id.SessionID
	return contracts.TargetSnapshot{Identity: id, Target: contracts.TargetToken("ime:" + id.SessionID), Application: m.client,
		Capabilities: contracts.Capabilities{OwnedComposition: true, SingleInsertion: true}}, nil
}

func (m *IME) owns(token contracts.TargetToken) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.active && token == contracts.TargetToken("ime:"+m.session)
}

// Update replaces the owned marked text with the newer draft.
func (m *IME) Update(_ context.Context, token contracts.TargetToken, draft contracts.Draft) error {
	if !m.owns(token) {
		return domain.Fail(domain.CodeTargetChanged, "composition no longer owned")
	}
	return m.send(map[string]any{"op": "mark", "text": draft.Text, "revision": draft.Revision, "session": draft.SessionID})
}

// Cancel clears the owned marked text.
func (m *IME) Cancel(_ context.Context, id contracts.Identity, token contracts.TargetToken) error {
	if token != contracts.TargetToken("ime:"+id.SessionID) {
		return nil
	}
	return m.send(map[string]any{"op": "cancel", "session": id.SessionID})
}

func (m *IME) Capabilities(_ context.Context, token contracts.TargetToken) (contracts.Capabilities, error) {
	if !m.owns(token) {
		return contracts.Capabilities{}, domain.Fail(domain.CodeTargetChanged, "composition no longer owned")
	}
	return contracts.Capabilities{OwnedComposition: true, SingleInsertion: true}, nil
}

func (m *IME) ReadContext(context.Context, contracts.Identity, contracts.TargetToken) (contracts.ContextSnapshot, error) {
	return contracts.ContextSnapshot{}, domain.Fail(domain.CodeTargetUnsupported, "context read not offered")
}

// Commit asks the IME to insert the reviewed text through IMK once and
// waits for its acknowledgement; without one the outcome is unknown.
func (m *IME) Commit(_ context.Context, request contracts.CommitRequest) (contracts.CommitResult, error) {
	result := contracts.CommitResult{Identity: request.Identity, Outcome: contracts.NotApplied}
	if !m.owns(request.Target) {
		result.Failure = domain.CodeTargetChanged
		return result, nil
	}
	ack := make(chan struct{})
	m.mu.Lock()
	m.acks[request.SessionID] = ack
	m.mu.Unlock()
	if err := m.send(map[string]any{"op": "commit", "text": request.Text, "session": request.SessionID}); err != nil {
		result.Failure = domain.CodeOf(err)
		return result, nil
	}
	select {
	case <-ack:
		result.Outcome = contracts.Applied
	case <-time.After(time.Second):
		result.Outcome = contracts.Unknown
	}
	return result, nil
}
