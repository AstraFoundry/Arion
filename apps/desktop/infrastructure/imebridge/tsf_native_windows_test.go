//go:build windows && arion_e2e

package imebridge

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"arion.local/core/contracts"
)

// This fixture loads the production COM DLL into an owned native editor.
// No machine registration, selected profile or existing document is changed.
func TestWindowsNativeTSFVoice(t *testing.T) {
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	dll := filepath.Join(root, "apps/desktop/resources/windows-amd64/ime/ArionTSF.dll")
	fixture := filepath.Join(root, ".workspace/windows-tsf/arion-tsf-fixture.exe")
	if override := os.Getenv("ARION_TSF_FIXTURE"); override != "" {
		fixture = override
	}
	data, err := os.ReadFile(dll)
	if err != nil {
		t.Fatal("build native TSF resources before this test: ", err)
	}
	sum := sha256.Sum256(data)
	path, err := WindowsPipePath()
	if err != nil {
		t.Fatal(err)
	}
	left := make(chan struct{}, 8)
	m := New(path, func() { left <- struct{}{} }, nil)
	m.RequireBeginAcknowledgement()
	if err := m.Listen(); err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	command := exec.Command(fixture, dll)
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		fmt.Fprintln(input, "quit")
		input.Close()
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			command.Process.Kill()
			<-done
		}
	}()
	lines := make(chan string, 32)
	go func() {
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	line := func() string {
		t.Helper()
		select {
		case value := <-lines:
			return value
		case <-time.After(3 * time.Second):
			t.Fatal("native fixture did not answer")
			return ""
		}
	}
	if got := line(); got != `{"ready":true}` {
		t.Fatalf("fixture startup %s", got)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := m.WaitReady(ctx, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal("native host did not connect: ", err)
	}
	var target contracts.TargetSnapshot
	for limit := time.Now().Add(3 * time.Second); ctx.Err() == nil && time.Now().Before(limit); {
		target, err = m.Capture(ctx, contracts.Identity{SessionID: "native-voice", Generation: 1})
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatal("native TSF has no editable context: ", err)
	}
	if err := m.Update(ctx, target.Target, contracts.Draft{Identity: target.Identity, Revision: 1, Text: "draft"}); err != nil {
		t.Fatal(err)
	}
	text := func() string {
		t.Helper()
		fmt.Fprintln(input, "text")
		var value struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(line()), &value); err != nil {
			t.Fatal(err)
		}
		return value.Text
	}
	waitText := func(want string) {
		t.Helper()
		for limit := time.Now().Add(3 * time.Second); time.Now().Before(limit); {
			if text() == want {
				return
			}
			time.Sleep(30 * time.Millisecond)
		}
		t.Fatalf("native text %q, want %q", text(), want)
	}
	waitText("A🙂draft中Z")
	request := contracts.CommitRequest{Identity: target.Identity, Target: target.Target, Text: "你好🙂e\u0301"}
	result, err := m.Commit(ctx, request)
	if err != nil || result.Outcome != contracts.Applied {
		t.Fatalf("native commit %+v: %v", result, err)
	}
	waitText("A🙂你好🙂e\u0301中Z")
	result, _ = m.Commit(ctx, request)
	if result.Outcome == contracts.Applied {
		t.Fatal("native duplicate commit applied")
	}
	control := func(command string) {
		t.Helper()
		fmt.Fprintln(input, command)
		if got := line(); got != `{"done":true}` {
			t.Fatalf("fixture command %s: %s", command, got)
		}
		time.Sleep(40 * time.Millisecond)
	}
	acquire := func(id string) contracts.TargetSnapshot {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for {
			target, err := m.Capture(ctx, contracts.Identity{SessionID: id, Generation: 1})
			if err == nil {
				return target
			}
			if time.Now().After(deadline) {
				t.Fatal("composition acquisition: ", err)
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	draft := func(target contracts.TargetSnapshot) {
		t.Helper()
		if err := m.Update(ctx, target.Target, contracts.Draft{Identity: target.Identity, Revision: 1, Text: "draft"}); err != nil {
			t.Fatal(err)
		}
		waitText("A🙂draftZ")
	}
	run := func(name string, check func()) { t.Log(name); check() }
	run("cancel restores selected Unicode text", func() {
		control("reset")
		target := acquire("cancel-selected")
		draft(target)
		if err := m.Cancel(ctx, target.Identity, target.Target); err != nil {
			t.Fatal(err)
		}
		waitText("A🙂中Z")
	})
	run("selection change revokes draft and stale delivery", func() {
		control("reset")
		target := acquire("move-selection")
		draft(target)
		control("select")
		waitText("A🙂中Z")
		if err := m.Update(ctx, target.Target, contracts.Draft{Identity: target.Identity, Revision: 2, Text: "stale"}); err == nil {
			t.Fatal("stale selection update accepted")
		}
		result, _ := m.Commit(ctx, contracts.CommitRequest{Identity: target.Identity, Target: target.Target, Text: "stale"})
		if result.Outcome == contracts.Applied {
			t.Fatal("stale selection commit applied")
		}
		waitText("A🙂中Z")
	})
	run("focus change cancels owned composition", func() {
		control("reset")
		target := acquire("focus-change")
		draft(target)
		control("password")
		waitText("A🙂中Z")
		if _, err := m.Capture(ctx, contracts.Identity{SessionID: "secure", Generation: 1}); err == nil {
			t.Fatal("password capture accepted")
		}
		control("edit")
	})
	run("read only field refuses acquisition", func() {
		control("reset")
		control("readonly")
		if _, err := m.Capture(ctx, contracts.Identity{SessionID: "readonly", Generation: 1}); err == nil {
			t.Fatal("read-only composition accepted")
		}
		waitText("A🙂中Z")
		control("writable")
	})
	if os.Getenv("ARION_TSF_FULL_STORE") == "1" {
		type storeState struct {
			Text           string
			Writes, Starts uint64
			Ends, Refused  uint64
			Deferred       uint64
		}
		state := func() storeState {
			t.Helper()
			fmt.Fprintln(input, "state")
			var value storeState
			if err := json.Unmarshal([]byte(line()), &value); err != nil {
				t.Fatal(err)
			}
			return value
		}
		type captureResult struct {
			target contracts.TargetSnapshot
			err    error
		}
		begin := func(captureCtx context.Context, id string) <-chan captureResult {
			done := make(chan captureResult, 1)
			go func() {
				target, err := m.Capture(captureCtx, contracts.Identity{SessionID: id, Generation: 1})
				done <- captureResult{target, err}
			}()
			return done
		}
		waitDeferred := func(before storeState) {
			t.Helper()
			for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
				if state().Deferred > before.Deferred {
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
			t.Fatal("native text store did not defer the requested TSF lock")
		}
		waitCapture := func(done <-chan captureResult) captureResult {
			t.Helper()
			select {
			case result := <-done:
				return result
			case <-time.After(3 * time.Second):
				t.Fatal("native acquisition did not finish")
				return captureResult{}
			}
		}
		run("native user replacement wins over late recognition", func() {
			control("reset")
			target := acquire("user-replacement")
			draft(target)
			control("replace")
			waitText("A🙂userZ")
			if err := m.Update(ctx, target.Target, contracts.Draft{Identity: target.Identity, Revision: 2, Text: "stale"}); err == nil {
				t.Fatal("user edit retained stale draft ownership")
			}
			result, _ := m.Commit(ctx, contracts.CommitRequest{Identity: target.Identity, Target: target.Target, Text: "stale"})
			if result.Outcome == contracts.Applied {
				t.Fatal("late native commit overwrote user replacement")
			}
			waitText("A🙂userZ")
		})
		run("context owner refusal returns no composition capability", func() {
			control("reset")
			before := state()
			control("refuse")
			result := waitCapture(begin(ctx, "host-refuses"))
			if result.err == nil || result.target.Target != "" {
				t.Fatalf("refused composition returned capability: %+v, %v", result.target, result.err)
			}
			after := state()
			if after.Refused != before.Refused+1 || after.Starts != before.Starts || after.Writes != before.Writes {
				t.Fatalf("owner refusal changed native document: before=%+v after=%+v", before, after)
			}
			waitText("A🙂中Z")
			control("accept")
		})
		run("selection change before deferred begin rejects capture", func() {
			control("reset")
			control("block")
			before := state()
			done := begin(ctx, "deferred-stale-selection")
			waitDeferred(before)
			control("select")
			control("unblock")
			result := waitCapture(done)
			if result.err == nil || result.target.Target != "" {
				t.Fatalf("stale deferred capture succeeded: %+v, %v", result.target, result.err)
			}
			waitText("A🙂中Z")
		})
		run("cancellation revokes deferred native acquisition", func() {
			control("reset")
			control("block")
			before := state()
			captureCtx, cancelCapture := context.WithCancel(ctx)
			done := begin(captureCtx, "deferred-cancel")
			waitDeferred(before)
			cancelCapture()
			result := waitCapture(done)
			if result.err == nil || result.target.Target != "" {
				t.Fatalf("cancelled acquisition returned ownership: %+v, %v", result.target, result.err)
			}
			control("unblock")
			waitText("A🙂中Z")
			if owns(m, result.target.Target) {
				t.Fatal("cancelled acquisition retained a usable target")
			}
		})
	}
}
