package windows

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"arion.local/core/domain"
)

const contextPrivateCanary = "ARION_PRIVATE_CONTEXT_CANARY"

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
	case "slow-exit":
		fmt.Fprintln(os.Stderr, contextTraceGoReceived)
		fmt.Fprintln(os.Stderr, contextTraceReadEntered)
		fmt.Fprintln(os.Stderr, contextTraceRootStarting)
		fmt.Fprintln(os.Stderr, contextTraceRootReady)
		fmt.Fprintln(os.Stderr, contextTraceWalkFinished)
		fmt.Print("visible fixture context")
		fmt.Fprintln(os.Stderr, contextTraceOutputFlushed)
		time.Sleep(2 * time.Second)
	case "complete-failed":
		fmt.Print("visible fixture context")
		fmt.Fprintln(os.Stderr, contextTraceOutputFlushed)
		os.Exit(12)
	case "stderr-noise":
		fmt.Fprintln(os.Stderr, strings.Repeat(contextPrivateCanary, 50000))
		fmt.Fprintln(os.Stderr, contextTraceReadEntered+" "+contextPrivateCanary)
		fmt.Fprintln(os.Stderr, contextTraceRootReady+"\x00")
		fmt.Fprintln(os.Stderr, contextPrivateCanary+contextTraceWalkFinished)
		fmt.Fprintln(os.Stderr, contextTraceGoReceived)
		fmt.Print(contextPrivateCanary)
		time.Sleep(2 * time.Second)
	case "stdout-lookalike", "stdout-lookalike-slow-exit":
		fmt.Println(contextTraceReadEntered)
		fmt.Print("visible fixture context")
		if mode == "stdout-lookalike-slow-exit" {
			time.Sleep(2 * time.Second)
		}
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
	return cmd
}

func requireContextProcessReaped(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd.ProcessState == nil {
		t.Fatal("context subprocess was not reaped")
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
			var failure *contextReadFailure
			if !errors.As(err, &failure) || !failure.observation.events[stageStdoutEOF].IsZero() || !failure.observation.events[stageProcessExited].IsZero() {
				t.Fatalf("cancellation observation includes induced cleanup: %v", err)
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

func TestContextProcessReportsSlowExit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := contextProcessCommand(t, ctx, "slow-exit", "")
	text, err := runContextProcess(ctx, cmd, nil, 3*time.Second, 150*time.Millisecond)
	if text != "" || domain.CodeOf(err) != domain.CodeTargetUnsupported || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("slow exit must still fail closed: text=%q error=%v", text, err)
	}
	requireContextProcessReaped(t, cmd)
	if !strings.Contains(err.Error(), "output-flushed=") || !strings.Contains(err.Error(), "payload-seen=") {
		t.Fatalf("completed output was not distinguished from exit: %v", err)
	}
	if strings.Contains(err.Error(), "stdout-eof=") || strings.Contains(err.Error(), "process-exited=") {
		t.Fatalf("cleanup progress was included in the timeout observation: %v", err)
	}
	t.Log(err)
}

func TestContextProcessRejectsCompleteOutputFromFailedExit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := contextProcessCommand(t, ctx, "complete-failed", "")
	text, err := runContextProcess(ctx, cmd, nil, 3*time.Second, time.Second)
	if text != "" || domain.CodeOf(err) != domain.CodeTargetUnsupported {
		t.Fatalf("complete bytes from a failed child were accepted: text=%q error=%v", text, err)
	}
	if !strings.Contains(err.Error(), "process-exited=") || !strings.Contains(err.Error(), "exit-failed=true") {
		t.Fatalf("failed exit was not observed: %v", err)
	}
	requireContextProcessReaped(t, cmd)
}

func TestContextProcessNeverUsesStdoutAsDiagnosticInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := contextProcessCommand(t, ctx, "stdout-lookalike", "")
	text, err := runContextProcess(ctx, cmd, nil, 3*time.Second, time.Second)
	expected := contextTraceReadEntered + "\nvisible fixture context"
	if err != nil || text != expected {
		t.Fatalf("visible text resembling a marker was altered: text=%q error=%v", text, err)
	}
	requireContextProcessReaped(t, cmd)
	cmd = contextProcessCommand(t, ctx, "stdout-lookalike-slow-exit", "")
	text, err = runContextProcess(ctx, cmd, nil, 3*time.Second, 150*time.Millisecond)
	var failure *contextReadFailure
	if text != "" || !errors.As(err, &failure) || !failure.observation.events[stageReadEntered].IsZero() {
		t.Fatalf("stdout spoofed a producer diagnostic: text=%q error=%v", text, err)
	}
	requireContextProcessReaped(t, cmd)
}

func TestContextProcessDiscardsPrivateAndMalformedDiagnostics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := contextProcessCommand(t, ctx, "stderr-noise", "")
	text, err := runContextProcess(ctx, cmd, nil, 3*time.Second, 500*time.Millisecond)
	var failure *contextReadFailure
	if text != "" || !errors.As(err, &failure) {
		t.Fatalf("noisy child must still fail closed: text=%q error=%v", text, err)
	}
	if strings.Contains(err.Error(), contextPrivateCanary) || strings.Contains(err.Error(), contextTracePrefix) {
		t.Fatalf("private raw diagnostics escaped: %v", err)
	}
	if failure.observation.events[stageGoReceived].IsZero() || !failure.observation.events[stageReadEntered].IsZero() || !failure.observation.events[stageRootReady].IsZero() || !failure.observation.events[stageWalkFinished].IsZero() {
		t.Fatalf("malformed records influenced phase observations: %v", err)
	}
	requireContextProcessReaped(t, cmd)
}

func TestContextTraceAcceptsOnlyCompleteFixedRecords(t *testing.T) {
	for _, record := range contextTraceRecords {
		t.Run(contextStageNames[record.stage]+fmt.Sprintf("-%d", len(record.wire)), func(t *testing.T) {
			trace := newContextTrace()
			for index := range record.wire {
				if count, err := trace.Write([]byte{record.wire[index]}); err != nil || count != 1 {
					t.Fatalf("fragmented record write: %d %v", count, err)
				}
			}
			observation := trace.snapshot(time.Now())
			for stage, event := range observation.events {
				if (contextStage(stage) == record.stage) == event.IsZero() {
					t.Fatalf("unexpected stage %s in %s", contextStageNames[stage], observation)
				}
			}
		})
	}
}

func TestContextTraceNeverRetainsUnknownInput(t *testing.T) {
	trace := newContextTrace()
	for _, line := range []string{
		contextPrivateCanary + "\n",
		contextTraceReadEntered + " " + contextPrivateCanary + "\n",
		strings.Repeat(contextPrivateCanary, 50000) + "\n",
		contextTraceRootReady + "\x00\n",
		contextTraceWalkFinished + "\r\r\n",
	} {
		trace.Write([]byte(line))
	}
	if got := trace.snapshot(time.Now()).String(); got != "none" {
		t.Fatalf("unknown input changed trace: %s", got)
	}
	if strings.Contains(fmt.Sprintf("%+v", trace), contextPrivateCanary) {
		t.Fatal("trace retained private input")
	}
	trace.Write([]byte(contextTraceGoReceived + "\n"))
	first := trace.snapshot(time.Now()).events[stageGoReceived]
	for range 1000 {
		trace.Write([]byte(contextTraceGoReceived + "\n"))
	}
	if last := trace.snapshot(time.Now()).events[stageGoReceived]; first.IsZero() || last != first {
		t.Fatal("duplicate records changed the first observation")
	}
}

func TestContextTraceSnapshotExcludesLateAndCleanupProgress(t *testing.T) {
	trace := newContextTrace()
	trace.record(stageGoWritten)
	cutoff := time.Now()
	time.Sleep(time.Millisecond)
	trace.Write([]byte(contextTraceReadEntered + "\n"))
	beforeCleanup := trace.snapshot(cutoff)
	trace.record(stageStdoutEOF)
	trace.exited(errors.New("owned cleanup"))
	if beforeCleanup.events[stageGoWritten].IsZero() || !beforeCleanup.events[stageReadEntered].IsZero() || !beforeCleanup.events[stageStdoutEOF].IsZero() || !beforeCleanup.events[stageProcessExited].IsZero() || beforeCleanup.exitFailed {
		t.Fatalf("late or cleanup progress was included: %s", beforeCleanup)
	}
}

func TestContextProcessCancellationCutoffPrecedesWatcherCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := contextProcessCommand(t, ctx, "success", "")
	text, err := runContextProcess(ctx, cmd, func() error {
		cancel()
		// Let CommandContext's existing watcher terminate/reap the child while
		// authorization still owns this goroutine; these are cleanup events.
		time.Sleep(100 * time.Millisecond)
		return nil
	}, 3*time.Second, time.Second)
	var failure *contextReadFailure
	if text != "" || !errors.Is(err, context.Canceled) || !errors.As(err, &failure) {
		t.Fatalf("authorization cancellation: text=%q error=%v", text, err)
	}
	requireContextProcessReaped(t, cmd)
	if !failure.observation.events[stageGoWritten].IsZero() || !failure.observation.events[stageStdoutEOF].IsZero() || !failure.observation.events[stageProcessExited].IsZero() {
		t.Fatalf("CommandContext watcher cleanup appeared as pre-cancellation progress: %v", err)
	}
}
