package client

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/llm/protocol"
)

// The test binary doubles as a fake worker speaking the real protocol:
// "slow" blocks until cancelled, "crash" exits, anything else is upper-cased.
func TestMain(m *testing.M) {
	if os.Getenv("ARION_FAKE_LLM") == "1" {
		fakeWorker()
		return
	}
	os.Exit(m.Run())
}

func fakeWorker() {
	in := bufio.NewScanner(os.Stdin)
	out := json.NewEncoder(os.Stdout)
	cancelled := make(chan uint64, 8)
	requests := make(chan protocol.Request, 8)
	go func() {
		for in.Scan() {
			var req protocol.Request
			_ = json.Unmarshal(in.Bytes(), &req)
			if req.Op == protocol.OpCancel {
				cancelled <- req.Target
				continue
			}
			requests <- req
		}
		close(requests)
	}()
	for req := range requests {
		switch {
		case req.Op == protocol.OpLoad:
			_ = out.Encode(protocol.Response{ID: req.ID, OK: true, PromptID: "test", PrefixTokens: 3})
		case req.Op == protocol.OpClose:
			_ = out.Encode(protocol.Response{ID: req.ID, OK: true})
			return
		case req.Text == "crash":
			os.Exit(3)
		case req.Text == "slow":
			<-cancelled
			_ = out.Encode(protocol.Response{ID: req.ID, OK: true, Cancelled: true})
		default:
			_ = out.Encode(protocol.Response{ID: req.ID, OK: true, Text: strings.ToUpper(req.Text)})
		}
	}
}

func start(t *testing.T) *Client {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARION_FAKE_LLM", "1")
	c, info, err := Start(context.Background(), Config{Worker: exe, Threads: 1, MaxLength: 64, MaxNewTokens: 8}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if info.PromptID != "test" || info.PrefixTokens != 3 {
		t.Fatalf("unexpected load info %+v", info)
	}
	t.Cleanup(c.Close)
	return c
}

var id = contracts.Identity{SessionID: "s", Generation: 1}

func TestRewriteRoundTrip(t *testing.T) {
	c := start(t)
	result, err := c.Rewrite(context.Background(), ports.RewriteRequest{Identity: id, Text: "hello"})
	if err != nil || result.Text != "HELLO" {
		t.Fatalf("got %q, %v", result.Text, err)
	}
}

// RWT-A02: a cancelled request returns promptly and the worker keeps serving.
func TestCancelStopsRequest(t *testing.T) {
	c := start(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	began := time.Now()
	_, err := c.Rewrite(ctx, ports.RewriteRequest{Identity: id, Text: "slow"})
	if domain.CodeOf(err) != domain.CodeCancelled || time.Since(began) > time.Second {
		t.Fatalf("cancel not prompt: %v after %v", err, time.Since(began))
	}
	result, err := c.Rewrite(context.Background(), ports.RewriteRequest{Identity: id, Text: "next"})
	if err != nil || result.Text != "NEXT" {
		t.Fatalf("worker unusable after cancel: %q, %v", result.Text, err)
	}
}

// RWT-A06: a dead worker is an explicit failure, never a silent success.
func TestWorkerExitIsExplicit(t *testing.T) {
	c := start(t)
	if _, err := c.Rewrite(context.Background(), ports.RewriteRequest{Identity: id, Text: "crash"}); domain.CodeOf(err) != domain.CodeModelUnsupported {
		t.Fatalf("crash not reported: %v", err)
	}
	if c.Alive() {
		t.Fatal("dead worker reported alive")
	}
	if _, err := c.Rewrite(context.Background(), ports.RewriteRequest{Identity: id, Text: "again"}); domain.CodeOf(err) != domain.CodeModelUnsupported {
		t.Fatalf("request after exit not failed: %v", err)
	}
}
