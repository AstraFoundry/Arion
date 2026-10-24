//go:build windows

package windows

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"

	"arion.local/core/domain"
)

// The fixed UI Automation script reads visible controls in the captured window.
// Text providers expose visible ranges; password and offscreen branches are skipped.
const accessibilityScript = `$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName WindowsBase
Add-Type -AssemblyName UIAutomationClientsideProviders
# PowerShell's dynamic call frames break the managed proxy loader's stack walk.
# Reflection keeps provider registration on a typed framework call stack.
$providers = [UIAutomationClientsideProviders.UIAutomationClientSideProviders].Assembly.GetName()
$register = [System.Windows.Automation.ClientSettings].GetMethod('RegisterClientSideProviderAssembly')
$register.Invoke($null, @($providers))
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
[Console]::Out.WriteLine('ARION_CONTEXT_READY')
[Console]::Out.Flush()
if ([Console]::In.ReadLine() -ne 'GO') { throw 'Traversal not authorized' }
$root = [System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]::new([long]$env:ARION_CONTEXT_HWND))
if ($null -eq $root) { throw 'Window unavailable' }
$walker = [System.Windows.Automation.TreeWalker]::ControlViewWalker
$script:count = 0
$script:length = 0
$lines = [System.Collections.Generic.List[string]]::new()
function Read-Visible($node, [int]$depth) {
 $script:count += 1
 if ($depth -gt 32 -or $script:count -gt 512) { throw 'Traversal limit' }
 $current = $node.Current
 if ($current.IsPassword -or $current.IsOffscreen) { return }
 $text = ''
 $pattern = $null
 if ($node.TryGetCurrentPattern([System.Windows.Automation.TextPattern]::Pattern, [ref]$pattern)) {
  foreach ($range in $pattern.GetVisibleRanges()) {
   $text += $range.GetText(32769)
   if ($script:length + $text.Length -gt 32768) { throw 'Text limit' }
  }
 } elseif ($current.ControlType -eq [System.Windows.Automation.ControlType]::Text) {
  $text = $current.Name
 }
 if ($text.Length -gt 0) {
  $script:length += $text.Length
  if ($script:length -gt 32768) { throw 'Text limit' }
  $lines.Add($text)
  return
 }
 $child = $walker.GetFirstChild($node)
 while ($null -ne $child) { Read-Visible $child ($depth + 1); $child = $walker.GetNextSibling($child) }
}
Read-Visible $root 0
if ($lines.Count -eq 0) { throw 'Visible text unavailable' }
[Console]::Write(($lines -join "` + "`n" + `"))`

func readWindowText(ctx context.Context, hwnd uintptr, authorize func() error) (string, error) {
	encoded := utf16.Encode([]rune(accessibilityScript))
	script := make([]byte, len(encoded)*2)
	for i, u := range encoded {
		script[2*i], script[2*i+1] = byte(u), byte(u>>8)
	}
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(script))
	cmd.Env = append(cmd.Environ(), "ARION_CONTEXT_HWND="+strconv.FormatUint(uint64(hwnd), 10))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return runContextProcess(ctx, cmd, authorize, contextStartupTimeout, screenTimeout)
}

const (
	contextStartupTimeout = 5 * time.Second
	screenTimeout         = 1500 * time.Millisecond
	contextReadyFrame     = "ARION_CONTEXT_READY\n"
	maxContextOutput      = 32768 * 4 // Maximum UTF-8 bytes for the UTF-16 text limit.
)

// runContextProcess separates runtime initialization from target traversal.
// The command must use CommandContext with ctx so caller cancellation also
// interrupts initialization. READY authorizes no target access: only GO does.
func runContextProcess(ctx context.Context, cmd *exec.Cmd, authorize func() error, startup, traversal time.Duration) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	startupDeadline := time.Now().Add(startup)
	startupTimer := time.NewTimer(startup)
	defer startupTimer.Stop()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", domain.Fail(domain.CodeTargetUnsupported, "UI Automation input unavailable")
	}
	defer stdin.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if child, ok := cmd.Stdin.(io.Closer); ok {
			_ = child.Close()
		}
		return "", domain.Fail(domain.CodeTargetUnsupported, "UI Automation output unavailable")
	}
	defer stdout.Close()
	cmd.Stderr = io.Discard
	cmd.WaitDelay = 100 * time.Millisecond
	if err := cmd.Start(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", domain.Fail(domain.CodeTargetUnsupported, "UI Automation initialization failed")
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
			_ = cmd.Wait()
			return
		}
		ready <- nil
		data, readErr := io.ReadAll(io.LimitReader(reader, maxContextOutput+1))
		if readErr != nil || len(data) > maxContextOutput {
			_ = cmd.Process.Kill()
			readErr = errors.New("invalid UI Automation output")
		}
		waitErr := cmd.Wait() // Read completed; this is the only Wait owner.
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
			return "", ctx.Err()
		}
		if err != nil {
			return "", domain.Fail(domain.CodeTargetUnsupported, "UI Automation initialization protocol failed")
		}
	case <-ctx.Done():
		return "", ctx.Err()
	case <-startupTimer.C:
		return "", domain.Fail(domain.CodeTargetUnsupported, "UI Automation initialization timed out")
	}
	startupTimer.Stop()
	if !time.Now().Before(startupDeadline) {
		return "", domain.Fail(domain.CodeTargetUnsupported, "UI Automation initialization timed out")
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if authorize != nil {
		if err := authorize(); err != nil {
			return "", err
		}
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if !time.Now().Before(startupDeadline) {
		return "", domain.Fail(domain.CodeTargetUnsupported, "UI Automation initialization timed out")
	}
	// Start the unchanged traversal budget before releasing the child into UIA.
	traversalDeadline := time.Now().Add(traversal)
	traversalTimer := time.NewTimer(traversal)
	defer traversalTimer.Stop()
	if _, err := io.WriteString(stdin, "GO\n"); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if !time.Now().Before(traversalDeadline) {
			return "", domain.Fail(domain.CodeTargetUnsupported, "UI Automation traversal timed out")
		}
		return "", domain.Fail(domain.CodeTargetUnsupported, "UI Automation traversal unavailable")
	}
	_ = stdin.Close()
	select {
	case out := <-result:
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if !time.Now().Before(traversalDeadline) {
			return "", domain.Fail(domain.CodeTargetUnsupported, "UI Automation traversal timed out")
		}
		if out.err != nil || strings.TrimSpace(out.text) == "" {
			return "", domain.Fail(domain.CodeTargetUnsupported, "visible UI Automation text unavailable")
		}
		return strings.TrimSpace(strings.ReplaceAll(out.text, "\r\n", "\n")), nil
	case <-ctx.Done():
		return "", ctx.Err()
	case <-traversalTimer.C:
		return "", domain.Fail(domain.CodeTargetUnsupported, "UI Automation traversal timed out")
	}
}
