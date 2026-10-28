//go:build windows

package windows

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"arion.local/core/domain"
)

// This real subprocess fixture exercises phase ownership, not UIA behavior.
func TestContextProcessHelper(t *testing.T) {
	mode := os.Getenv("ARION_CONTEXT_PROCESS_HELPER")
	if mode == "" {
		return
	}
	if mode == "startup-timeout" || mode == "startup-cancel" {
		if signal := os.Getenv("ARION_CONTEXT_PROCESS_SIGNAL"); signal != "" {
			_ = os.WriteFile(signal, []byte("initialization"), 0600)
		}
		time.Sleep(time.Minute)
		os.Exit(10)
	}
	if mode == "bad-ready" {
		fmt.Println("unexpected frame")
		os.Exit(0)
	}
	if mode == "eof-before-ready" {
		os.Exit(0)
	}
	if mode == "slow-start" {
		time.Sleep(250 * time.Millisecond)
	}
	fmt.Print(contextReadyFrame)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil || line != "GO\n" {
		os.Exit(11)
	}
	if signal := os.Getenv("ARION_CONTEXT_PROCESS_SIGNAL"); signal != "" {
		_ = os.WriteFile(signal, []byte("traversal"), 0600)
	}
	switch mode {
	case "traversal-timeout", "traversal-cancel":
		fmt.Print("partial private context")
		time.Sleep(time.Minute)
	case "empty":
	case "failed-output":
		fmt.Print("partial private context")
		os.Exit(12)
	case "oversized":
		fmt.Print(strings.Repeat("x", maxContextOutput+1))
	default:
		fmt.Print("visible fixture context")
	}
	os.Exit(0)
}

func contextProcessCommand(t *testing.T, ctx context.Context, mode, signal string) *exec.Cmd {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestContextProcessHelper$")
	// The race detector's default exit sleep is unrelated to protocol work.
	cmd.Env = append(os.Environ(), "ARION_CONTEXT_PROCESS_HELPER="+mode, "ARION_CONTEXT_PROCESS_SIGNAL="+signal, "GORACE=atexit_sleep_ms=0")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return cmd
}

func requireContextProcessReaped(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd.ProcessState == nil {
		t.Fatal("context subprocess was not reaped")
	}
}

// Compile and register the real managed reader, then deny GO. An invalid HWND
// must never be inspected before the parent's ownership authorization succeeds.
func TestContextReaderInitializesBeforeAuthorization(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), contextStartupTimeout+time.Second)
	defer cancel()
	denied := errors.New("synthetic target ownership refusal")
	// Record only static phase names and elapsed milliseconds. The invalid
	// fixture HWND is never authorized, so no foreign UI text is collected.
	mark := func(phase string) string {
		return "\n[Console]::Error.WriteLine('ARION_INIT " + phase + " ' + $arionInitClock.ElapsedMilliseconds)\n"
	}
	source := "$arionInitClock = [System.Diagnostics.Stopwatch]::StartNew()" + mark("started") + accessibilityScript
	for _, assembly := range []string{"UIAutomationClient", "UIAutomationTypes", "WindowsBase", "UIAutomationClientsideProviders"} {
		load := "Add-Type -AssemblyName " + assembly + "\n"
		source = strings.Replace(source, load, load+mark(assembly), 1)
	}
	register := "$register.Invoke($null, @($providers))"
	source = strings.Replace(source, register, register+mark("providers-registered"), 1)
	source = strings.Replace(source, "'@\n", "'@\n"+mark("reader-compiled"), 1)
	cmd := windowTextCommand(ctx, 0, source)
	var diagnostics bytes.Buffer
	cmd.Stderr = &diagnostics
	started := time.Now()
	text, err := runContextProcess(ctx, cmd, func() error { return denied }, contextStartupTimeout, screenTimeout)
	t.Logf("managed reader initialization: %s", time.Since(started))
	t.Logf("managed reader initialization phases:\n%s", diagnostics.String())
	if !errors.Is(err, denied) || text != "" {
		t.Fatalf("reader initialization/authorization: text=%q error=%v", text, err)
	}
	if !strings.Contains(diagnostics.String(), "ARION_INIT reader-compiled ") {
		t.Fatal("initialization completed without its compilation diagnostic")
	}
}

func TestContextProcessSeparatesStartupFromTraversal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := contextProcessCommand(t, ctx, "slow-start", "")
	started := time.Now()
	text, err := runContextProcess(ctx, cmd, nil, 3*time.Second, 150*time.Millisecond)
	if err != nil || text != "visible fixture context" || time.Since(started) < 250*time.Millisecond {
		t.Fatalf("separate startup/traversal: text=%q error=%v elapsed=%s", text, err, time.Since(started))
	}
	requireContextProcessReaped(t, cmd)
}

func TestContextProcessPhaseTimeoutsDiscardOutput(t *testing.T) {
	for _, mode := range []string{"startup-timeout", "traversal-timeout"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := contextProcessCommand(t, ctx, mode, "")
			startup := 3 * time.Second
			if mode == "startup-timeout" {
				startup = 150 * time.Millisecond
			}
			text, err := runContextProcess(ctx, cmd, nil, startup, 150*time.Millisecond)
			if text != "" || domain.CodeOf(err) != domain.CodeTargetUnsupported || !strings.Contains(err.Error(), "timed out") {
				t.Fatalf("phase timeout leaked output or wrong error: text=%q error=%v", text, err)
			}
			requireContextProcessReaped(t, cmd)
		})
	}
}

func TestContextProcessRejectsMalformedAndFailedOutput(t *testing.T) {
	for _, mode := range []string{"bad-ready", "eof-before-ready", "empty", "failed-output", "oversized"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := contextProcessCommand(t, ctx, mode, "")
			text, err := runContextProcess(ctx, cmd, nil, 3*time.Second, time.Second)
			if text != "" || domain.CodeOf(err) != domain.CodeTargetUnsupported {
				t.Fatalf("invalid output accepted: text=%q error=%v", text, err)
			}
			requireContextProcessReaped(t, cmd)
		})
	}
}

func TestContextProcessCancellationReapsEitherPhase(t *testing.T) {
	for _, mode := range []string{"startup-cancel", "traversal-cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			signal := filepath.Join(t.TempDir(), "traversal")
			cmd := contextProcessCommand(t, ctx, mode, signal)
			done := make(chan struct{})
			var text string
			var err error
			go func() { text, err = runContextProcess(ctx, cmd, nil, 3*time.Second, 2*time.Second); close(done) }()
			{
				deadline := time.Now().Add(3 * time.Second)
				for {
					if _, statErr := os.Stat(signal); statErr == nil {
						break
					}
					if time.Now().After(deadline) {
						cancel()
						<-done
						t.Fatal("fixture never entered traversal")
					}
					time.Sleep(time.Millisecond)
				}
			}
			cancel()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("caller cancellation did not reap context process promptly")
			}
			if text != "" || !errors.Is(err, context.Canceled) {
				t.Fatalf("caller cancellation: text=%q error=%v", text, err)
			}
			requireContextProcessReaped(t, cmd)
		})
	}
}

func TestContextProcessRevalidatesBeforeGO(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	signal := filepath.Join(t.TempDir(), "must-not-traverse")
	cmd := contextProcessCommand(t, ctx, "success", signal)
	text, err := runContextProcess(ctx, cmd, func() error { return domain.Fail(domain.CodeTargetChanged, "owned target changed") }, 3*time.Second, time.Second)
	if text != "" || domain.CodeOf(err) != domain.CodeTargetChanged {
		t.Fatalf("changed target: text=%q error=%v", text, err)
	}
	if _, err := os.Stat(signal); !os.IsNotExist(err) {
		t.Fatalf("unauthorized target traversal started: %v", err)
	}
	requireContextProcessReaped(t, cmd)
}

func TestContextProcessAuthorizationCannotReleaseExpiredWork(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			signal := filepath.Join(t.TempDir(), "must-not-traverse")
			cmd := contextProcessCommand(t, ctx, "success", signal)
			authorize := func() error {
				if mode == "cancel" {
					cancel()
				} else {
					time.Sleep(1100 * time.Millisecond)
				}
				return nil
			}
			text, err := runContextProcess(ctx, cmd, authorize, time.Second, time.Second)
			if text != "" || mode == "cancel" && !errors.Is(err, context.Canceled) || mode == "deadline" && domain.CodeOf(err) != domain.CodeTargetUnsupported {
				t.Fatalf("expired authorization released work: text=%q error=%v", text, err)
			}
			if _, err := os.Stat(signal); !os.IsNotExist(err) {
				t.Fatalf("expired authorization sent GO: %v", err)
			}
			requireContextProcessReaped(t, cmd)
		})
	}
}
