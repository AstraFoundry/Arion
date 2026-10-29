package windows

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"time"

	"arion.local/core/domain"
)

const (
	// Cold PowerShell/.NET provider loading and compilation can exceed five
	// seconds on hosted Windows. No target is inspected during this phase.
	contextStartupTimeout = 15 * time.Second
	screenTimeout         = 1500 * time.Millisecond
	contextReadyFrame     = "ARION_CONTEXT_READY\n"
	maxContextOutput      = 32768 * 4 // Maximum UTF-8 bytes for the UTF-16 text limit.
)

// runContextProcess separates runtime initialization from target traversal.
// The command must use CommandContext with ctx so caller cancellation also
// interrupts initialization. READY authorizes no target access: only GO does.
func runContextProcess(ctx context.Context, cmd *exec.Cmd, authorize func() error, startup, traversal time.Duration) (string, error) {
	trace := newContextTrace()
	fail := func(cause error, cutoff time.Time) (string, error) {
		return "", &contextReadFailure{cause: cause, observation: trace.snapshot(cutoff)}
	}
	if ctx.Err() != nil {
		return fail(ctx.Err(), time.Now())
	}
	startupDeadline := time.Now().Add(startup)
	startupTimer := time.NewTimer(startup)
	defer startupTimer.Stop()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fail(domain.Fail(domain.CodeTargetUnsupported, "UI Automation input unavailable"), time.Now())
	}
	defer stdin.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if child, ok := cmd.Stdin.(io.Closer); ok {
			_ = child.Close()
		}
		return fail(domain.Fail(domain.CodeTargetUnsupported, "UI Automation output unavailable"), time.Now())
	}
	defer stdout.Close()
	if cmd.Stderr == nil {
		cmd.Stderr = trace
	} else {
		cmd.Stderr = io.MultiWriter(cmd.Stderr, trace)
	}
	if cancelCommand := cmd.Cancel; cancelCommand != nil {
		cmd.Cancel = func() error {
			// CommandContext can kill concurrently with this function's select.
			// Record that boundary before its watcher creates cleanup progress.
			trace.cancelling()
			return cancelCommand()
		}
	}
	cmd.WaitDelay = 100 * time.Millisecond
	if err := cmd.Start(); err != nil {
		if ctx.Err() != nil {
			return fail(ctx.Err(), time.Now())
		}
		return fail(domain.Fail(domain.CodeTargetUnsupported, "UI Automation initialization failed"), time.Now())
	}
	type output struct {
		text string
		err  error
	}
	ready := make(chan error, 1)
	result := make(chan output, 1)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		reader := bufio.NewReaderSize(stdout, 1024)
		frame, readErr := reader.ReadSlice('\n')
		if readErr != nil || strings.ReplaceAll(string(frame), "\r\n", "\n") != contextReadyFrame {
			ready <- errors.New("invalid UI Automation ready frame")
			_ = cmd.Process.Kill()
			waitErr := cmd.Wait()
			trace.exited(waitErr)
			return
		}
		ready <- nil
		data, readErr := io.ReadAll(io.LimitReader(observedContextReader{reader: reader, trace: trace}, maxContextOutput+1))
		if readErr != nil || len(data) > maxContextOutput {
			_ = cmd.Process.Kill()
			readErr = errors.New("invalid UI Automation output")
		}
		waitErr := cmd.Wait() // Read completed; this is the only Wait owner.
		trace.exited(waitErr)
		if readErr == nil {
			readErr = waitErr
		}
		result <- output{text: string(data), err: readErr}
	}()
	defer func() {
		_ = cmd.Process.Kill()
		_ = stdin.Close()
		_ = stdout.Close()
		<-finished
	}()
	select {
	case err := <-ready:
		if ctx.Err() != nil {
			return fail(ctx.Err(), time.Now())
		}
		if err != nil {
			return fail(domain.Fail(domain.CodeTargetUnsupported, "UI Automation initialization protocol failed"), time.Now())
		}
	case <-ctx.Done():
		return fail(ctx.Err(), time.Now())
	case <-startupTimer.C:
		return fail(domain.Fail(domain.CodeTargetUnsupported, "UI Automation initialization timed out"), startupDeadline)
	}
	startupTimer.Stop()
	if !time.Now().Before(startupDeadline) {
		return fail(domain.Fail(domain.CodeTargetUnsupported, "UI Automation initialization timed out"), startupDeadline)
	}
	if ctx.Err() != nil {
		return fail(ctx.Err(), time.Now())
	}
	if authorize != nil {
		if err := authorize(); err != nil {
			return fail(err, time.Now())
		}
	}
	if ctx.Err() != nil {
		return fail(ctx.Err(), time.Now())
	}
	if !time.Now().Before(startupDeadline) {
		return fail(domain.Fail(domain.CodeTargetUnsupported, "UI Automation initialization timed out"), startupDeadline)
	}
	// Start the unchanged traversal budget before releasing the child into UIA.
	traversalDeadline := time.Now().Add(traversal)
	traversalTimer := time.NewTimer(traversal)
	defer traversalTimer.Stop()
	if _, err := io.WriteString(stdin, "GO\n"); err != nil {
		if ctx.Err() != nil {
			return fail(ctx.Err(), time.Now())
		}
		if !time.Now().Before(traversalDeadline) {
			return fail(domain.Fail(domain.CodeTargetUnsupported, "UI Automation traversal timed out"), traversalDeadline)
		}
		return fail(domain.Fail(domain.CodeTargetUnsupported, "UI Automation traversal unavailable"), time.Now())
	}
	trace.record(stageGoWritten)
	_ = stdin.Close()
	select {
	case out := <-result:
		if ctx.Err() != nil {
			return fail(ctx.Err(), time.Now())
		}
		if !time.Now().Before(traversalDeadline) {
			return fail(domain.Fail(domain.CodeTargetUnsupported, "UI Automation traversal timed out"), traversalDeadline)
		}
		if out.err != nil || strings.TrimSpace(out.text) == "" {
			return fail(domain.Fail(domain.CodeTargetUnsupported, "visible UI Automation text unavailable"), time.Now())
		}
		return strings.TrimSpace(strings.ReplaceAll(out.text, "\r\n", "\n")), nil
	case <-ctx.Done():
		return fail(ctx.Err(), time.Now())
	case <-traversalTimer.C:
		return fail(domain.Fail(domain.CodeTargetUnsupported, "UI Automation traversal timed out"), traversalDeadline)
	}
}
