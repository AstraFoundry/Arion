package imebridge

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
)

type host struct {
	conn  net.Conn
	input *bufio.Reader
}

func attach(t *testing.T, m *IME, activation string) *host {
	t.Helper()
	server, client := net.Pipe()
	go m.serve(server)
	h := &host{client, bufio.NewReader(client)}
	t.Cleanup(func() { client.Close() })
	h.send(t, message{Op: "activate", Activation: activation, Client: "test.editor"})
	wait(t, func() bool { m.mu.Lock(); defer m.mu.Unlock(); return m.activation == activation })
	return h
}
func (h *host) send(t *testing.T, msg message) {
	t.Helper()
	h.conn.SetWriteDeadline(time.Now().Add(time.Second))
	if err := json.NewEncoder(h.conn).Encode(msg); err != nil {
		t.Fatal(err)
	}
}
func (h *host) read(t *testing.T) message {
	t.Helper()
	h.conn.SetReadDeadline(time.Now().Add(time.Second))
	line, err := h.input.ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	var msg message
	if err := json.Unmarshal(line, &msg); err != nil {
		t.Fatal(err)
	}
	return msg
}
func wait(t *testing.T, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatal("state transition timed out")
		}
		time.Sleep(time.Millisecond)
	}
}
func capture(t *testing.T, m *IME, h *host, id string) contracts.TargetSnapshot {
	t.Helper()
	type response struct {
		s   contracts.TargetSnapshot
		err error
	}
	done := make(chan response, 1)
	go func() {
		s, err := m.Capture(context.Background(), contracts.Identity{SessionID: id, Generation: 1})
		done <- response{s, err}
	}()
	msg := h.read(t)
	if msg.Op != "begin" || msg.Session != id || msg.Activation == "" {
		t.Fatalf("bad binding: %+v", msg)
	}
	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	return result.s
}
func TestActivationRevokesOldSessionAndMessages(t *testing.T) {
	var leaves atomic.Int32
	m := New("", func() { leaves.Add(1) }, nil)
	defer m.Close()
	h := attach(t, m, "first")
	old := capture(t, m, h, "s1")
	h.send(t, message{Op: "activate", Activation: "second"})
	wait(t, func() bool { return leaves.Load() == 1 })
	next := capture(t, m, h, "s2")
	h.send(t, message{Op: "deactivate", Activation: "first"})
	if _, err := m.Capabilities(context.Background(), old.Target); err == nil {
		t.Fatal("old capability survived activation")
	}
	if _, err := m.Capabilities(context.Background(), next.Target); err != nil {
		t.Fatal(err)
	}
	if err := m.Cancel(context.Background(), old.Identity, old.Target); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Capabilities(context.Background(), next.Target); err != nil {
		t.Fatal("stale cancel revoked current session")
	}
}
func TestDisconnectInvalidatesAndNotifies(t *testing.T) {
	var leaves atomic.Int32
	m := New("", func() { leaves.Add(1) }, nil)
	defer m.Close()
	h := attach(t, m, "a")
	s := capture(t, m, h, "s")
	h.conn.Close()
	wait(t, func() bool { return leaves.Load() == 1 })
	if _, err := m.Capabilities(context.Background(), s.Target); err == nil {
		t.Fatal("disconnected capability survived")
	}
}
func TestCommitAcknowledgementScopedAndExactlyOnce(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	h := attach(t, m, "a")
	s := capture(t, m, h, "s")
	done := make(chan contracts.CommitResult, 1)
	go func() {
		result, _ := m.Commit(context.Background(), contracts.CommitRequest{Identity: s.Identity, Target: s.Target, Text: "hello"})
		done <- result
	}()
	if msg := h.read(t); msg.Op != "commit" || msg.Activation != "a" || msg.Session != "s" {
		t.Fatalf("bad commit: %+v", msg)
	}
	h.send(t, message{Op: "committed", Activation: "old", Session: "s"})
	select {
	case <-done:
		t.Fatal("stale activation acknowledged")
	case <-time.After(10 * time.Millisecond):
	}
	h.send(t, message{Op: "committed", Activation: "a", Session: "s"})
	if result := <-done; result.Outcome != contracts.Applied {
		t.Fatal(result)
	}
	if result, _ := m.Commit(context.Background(), contracts.CommitRequest{Identity: s.Identity, Target: s.Target, Text: "again"}); result.Outcome != contracts.NotApplied {
		t.Fatal("duplicate commit permitted")
	}
	capture(t, m, h, "next")
}
func TestCancelledCommitRetiresCapability(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	h := attach(t, m, "a")
	s := capture(t, m, h, "s")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan contracts.CommitResult, 1)
	go func() {
		r, _ := m.Commit(ctx, contracts.CommitRequest{Identity: s.Identity, Target: s.Target, Text: "hello"})
		done <- r
	}()
	h.read(t)
	cancel()
	if result := <-done; result.Outcome != contracts.Unknown {
		t.Fatal(result)
	}
	m.mu.Lock()
	pending := m.ack != nil
	m.mu.Unlock()
	if pending {
		t.Fatal("pending acknowledgement leaked")
	}
	capture(t, m, h, "next")
}
func TestRevisionAndIdentityValidation(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	h := attach(t, m, "a")
	s := capture(t, m, h, "s")
	done := make(chan error, 1)
	draft := contracts.Draft{Identity: s.Identity, Revision: 2, Text: "😀中文"}
	go func() { done <- m.Update(context.Background(), s.Target, draft) }()
	if msg := h.read(t); msg.Text != draft.Text || msg.Revision != 2 {
		t.Fatal(msg)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	draft.Revision = 1
	if err := m.Update(context.Background(), s.Target, draft); err == nil {
		t.Fatal("stale revision accepted")
	}
	draft.Revision = 3
	draft.Generation++
	if err := m.Update(context.Background(), s.Target, draft); err == nil {
		t.Fatal("wrong generation accepted")
	}
}
func TestOldConnectionCannotDeactivateReplacement(t *testing.T) {
	var leaves atomic.Int32
	m := New("", func() { leaves.Add(1) }, nil)
	defer m.Close()
	first := attach(t, m, "a")
	capture(t, m, first, "old")
	second := attach(t, m, "b")
	next := capture(t, m, second, "new")
	wait(t, func() bool { return leaves.Load() == 1 })
	if _, err := m.Capabilities(context.Background(), next.Target); err != nil {
		t.Fatal(err)
	}
}

// The input menu's Voice Input item toggles dictation only for the client
// that is active now on the current connection (IME-06).
func TestDictateIsActivationScoped(t *testing.T) {
	var dictations atomic.Int32
	m := New("", nil, func() { dictations.Add(1) })
	defer m.Close()
	first := attach(t, m, "a")
	first.send(t, message{Op: "dictate", Activation: "stale"})
	first.send(t, message{Op: "dictate"})
	first.send(t, message{Op: "dictate", Activation: "a"})
	wait(t, func() bool { return dictations.Load() == 1 })
	first.send(t, message{Op: "deactivate", Activation: "a"})
	first.send(t, message{Op: "dictate", Activation: "a"})
	second := attach(t, m, "b")
	// The replaced connection is closed; its late message never arrives.
	_ = json.NewEncoder(first.conn).Encode(message{Op: "dictate", Activation: "b"})
	second.send(t, message{Op: "dictate", Activation: "b"})
	wait(t, func() bool { return dictations.Load() == 2 })
	// Pipes are synchronous: this activation is read after the dictate.
	second.send(t, message{Op: "activate", Activation: "c"})
	wait(t, func() bool { m.mu.Lock(); defer m.mu.Unlock(); return m.activation == "c" })
	if n := dictations.Load(); n != 2 {
		t.Fatalf("stale or deactivated dictate toggled dictation: %d", n)
	}
}

func TestCancelledContextBeforeCommitDoesNotConsumeSession(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	h := attach(t, m, "a")
	s := capture(t, m, h, "s")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result, err := m.Commit(ctx, contracts.CommitRequest{Identity: s.Identity, Target: s.Target}); err == nil || result.Outcome != contracts.NotApplied {
		t.Fatal(result, err)
	}
	if _, err := m.Capabilities(context.Background(), s.Target); err != nil {
		t.Fatal(err)
	}
}
func TestCloseRevokesAndStopsTransport(t *testing.T) {
	var leaves atomic.Int32
	m := New("", func() { leaves.Add(1) }, nil)
	h := attach(t, m, "a")
	s := capture(t, m, h, "s")
	m.Close()
	if leaves.Load() != 1 {
		t.Fatal("missing close notification")
	}
	if _, err := m.Capabilities(context.Background(), s.Target); err == nil {
		t.Fatal("closed capability survived")
	}
	m.Close()
	if leaves.Load() != 1 {
		t.Fatal("duplicate close notification")
	}
}
func TestSocketLifecycleDoesNotReplaceExistingPath(t *testing.T) {
	directory, err := os.MkdirTemp("", "arion-ime-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "ime.sock")
	if runtime.GOOS == "windows" {
		path = `\\.\pipe\Arion.Voice.TSF.Test.` + filepath.Base(directory)
	}
	m := New(path, nil, nil)
	if err := m.Listen(); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	other := New(path, nil, nil)
	if err := other.Listen(); err == nil {
		other.Close()
		t.Fatal("existing socket was replaced")
	}
	m.Close()
	replacement := New(path, nil, nil)
	if err := replacement.Listen(); err != nil {
		t.Fatal(err)
	}
	replacement.Close()
}

func TestBlockedCommitHonorsContextAndReturnsUncertain(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	h := attach(t, m, "a")
	s := capture(t, m, h, "s")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result, err := m.Commit(ctx, contracts.CommitRequest{Identity: s.Identity, Target: s.Target, Text: "hello"})
	if err != nil || result.Outcome != contracts.Unknown {
		t.Fatal(result, err)
	}
	m.mu.Lock()
	pending := m.ack != nil
	m.mu.Unlock()
	if pending {
		t.Fatal("failed send retained pending acknowledgement")
	}
}

func TestEscapeIsSessionScopedAndLeavesSourceActive(t *testing.T) {
	var leaves atomic.Int32
	m := New("", func() { leaves.Add(1) }, nil)
	defer m.Close()
	h := attach(t, m, "a")
	s := capture(t, m, h, "s")
	h.send(t, message{Op: "escape", Activation: "a", Session: "old"})
	if _, err := m.Capabilities(context.Background(), s.Target); err != nil {
		t.Fatal("old escape revoked current session")
	}
	h.send(t, message{Op: "escape", Activation: "a", Session: "s"})
	wait(t, func() bool { return leaves.Load() == 1 })
	if _, err := m.Capabilities(context.Background(), s.Target); err == nil {
		t.Fatal("escaped capability survived")
	}
	capture(t, m, h, "next")
}

func TestEmptyMarkAndCommitPreserveStringField(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	h := attach(t, m, "a")
	s := capture(t, m, h, "s")
	done := make(chan error, 1)
	go func() {
		done <- m.Update(context.Background(), s.Target, contracts.Draft{Identity: s.Identity, Revision: 1})
	}()
	h.conn.SetReadDeadline(time.Now().Add(time.Second))
	line, err := h.input.ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(line, &fields); err != nil {
		t.Fatal(err)
	}
	if text, ok := fields["text"].(string); !ok || text != "" {
		t.Fatal("empty draft text omitted", fields)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	result := make(chan contracts.CommitResult, 1)
	go func() {
		r, _ := m.Commit(context.Background(), contracts.CommitRequest{Identity: s.Identity, Target: s.Target})
		result <- r
	}()
	line, err = h.input.ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	fields = nil
	if err := json.Unmarshal(line, &fields); err != nil {
		t.Fatal(err)
	}
	if text, ok := fields["text"].(string); !ok || text != "" {
		t.Fatal("empty commit text omitted", fields)
	}
	h.send(t, message{Op: "committed", Activation: "a", Session: "s"})
	if r := <-result; r.Outcome != contracts.Applied {
		t.Fatal(r)
	}
}
func TestCancelledCancelDisconnectsToClearNativeOwnership(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	h := attach(t, m, "a")
	s := capture(t, m, h, "s")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Cancel(ctx, s.Identity, s.Target); err == nil {
		t.Fatal("cancelled context should report failed delivery")
	}
	h.conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := h.input.ReadByte(); err == nil {
		t.Fatal("transport stayed connected with uncleared native ownership")
	}
	if _, err := m.Capabilities(context.Background(), s.Target); err == nil {
		t.Fatal("cancelled capability survived")
	}
}

func TestWriteKeyboardSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Arion", "keyboard.json")
	want := application.KeyboardPreferences{PageSize: 6, CandidateSize: application.CandidateLarge, Appearance: application.AppearanceDark, Accent: application.AccentBlue}
	if err := WriteKeyboardSettings(path, want, application.HoldRightCommand); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// The input method reads exactly these keys.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil || raw["pageSize"] != 6.0 || raw["candidateSize"] != "large" ||
		raw["appearance"] != "dark" || raw["accent"] != "blue" || raw["contextRanking"] != false ||
		raw["dictationKey"] != "right_command" {
		t.Fatalf("unexpected settings file %s (%v)", data, err)
	}
	// The highlight arrives resolved for each appearance.
	highlight, _ := raw["highlight"].(map[string]any)
	light, _ := highlight["light"].(map[string]any)
	dark, _ := highlight["dark"].(map[string]any)
	if light["fill"] != "#0088ff" || light["text"] != "#ffffff" || dark["fill"] != "#0099f2" || dark["text"] != "#ffffff" {
		t.Fatalf("unexpected highlight %s", data)
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Fatalf("temporary file left behind: %v", entries)
	}
}

func TestReadinessRequiresExpectedBuildAndReconnectsWithoutClient(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	connect := func(build string) net.Conn {
		server, client := net.Pipe()
		go m.serve(server)
		if err := json.NewEncoder(client).Encode(message{Op: "hello", Build: build}); err != nil {
			t.Fatal(err)
		}
		return client
	}
	old := connect("old-build")
	defer old.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	if err := m.WaitReady(ctx, "new-build"); err == nil {
		t.Fatal("old build satisfied new readiness")
	}
	cancel()
	current := connect("new-build")
	defer current.Close()
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.WaitReady(ctx, "new-build"); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	active := m.activation
	m.mu.Unlock()
	if active != "" {
		t.Fatal("readiness invented an editable client")
	}
	current.Close()
	wait(t, func() bool { m.mu.Lock(); defer m.mu.Unlock(); return m.conn == nil })
	ctx2, cancel2 := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel2()
	if err := m.WaitReady(ctx2, "new-build"); err == nil {
		t.Fatal("disconnected host remained ready")
	}
}

func TestUpdateRejectsNewOwnedCapture(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	h := attach(t, m, "a")
	m.BeginUpdate()
	if _, err := m.Capture(context.Background(), contracts.Identity{SessionID: "blocked"}); domain.CodeOf(err) != domain.CodeTargetUnsupported {
		t.Fatal("update allowed capture", err)
	}
	m.EndUpdate()
	capture(t, m, h, "allowed")
}
