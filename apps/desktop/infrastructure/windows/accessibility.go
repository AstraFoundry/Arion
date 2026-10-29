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
if ([System.Threading.Thread]::CurrentThread.GetApartmentState() -ne [System.Threading.ApartmentState]::MTA) {
 throw 'UI Automation requires a multithreaded apartment'
}
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
Add-Type -AssemblyName WindowsBase
Add-Type -AssemblyName UIAutomationClientsideProviders
$providers = [UIAutomationClientsideProviders.UIAutomationClientSideProviders].Assembly.GetName()
$register = [System.Windows.Automation.ClientSettings].GetMethod('RegisterClientSideProviderAssembly')
$register.Invoke($null, @($providers))
# Compile traversal before READY to avoid dynamic per-node PowerShell binding
# inside the target traversal budget. Provider registration needs a typed
# framework call stack for the managed proxy loader.
$references = @([System.Windows.Automation.AutomationElement].Assembly.Location,
 [System.Windows.Automation.ControlType].Assembly.Location,
 [System.Windows.Rect].Assembly.Location,
 [UIAutomationClientsideProviders.UIAutomationClientSideProviders].Assembly.Location)
Add-Type -ReferencedAssemblies $references -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Windows.Automation;
public sealed class ArionContextReader {
 private int count, length;
 private readonly List<string> lines = new List<string>();
 public static string Read(long handle) {
  var root = AutomationElement.FromHandle(new IntPtr(handle));
  if (root == null) throw new InvalidOperationException("Window unavailable");
  var reader = new ArionContextReader();
  reader.Visit(root, 0);
  if (reader.lines.Count == 0) throw new InvalidOperationException("Visible text unavailable");
  return string.Join("\n", reader.lines);
 }
 private void Visit(AutomationElement node, int depth) {
  if (depth > 32 || ++count > 512) throw new InvalidOperationException("Traversal limit");
  var current = node.Current;
  if (current.IsPassword || current.IsOffscreen) return;
  string text = "";
  object pattern;
  if (node.TryGetCurrentPattern(TextPattern.Pattern, out pattern)) {
   foreach (var range in ((TextPattern)pattern).GetVisibleRanges()) {
    text += range.GetText(32769);
    if (length + text.Length > 32768) throw new InvalidOperationException("Text limit");
   }
  } else if (current.ControlType == ControlType.Text) {
   text = current.Name;
  }
  if (text.Length > 0) {
   length += text.Length;
   if (length > 32768) throw new InvalidOperationException("Text limit");
   lines.Add(text);
   return;
  }
  var walker = TreeWalker.ControlViewWalker;
  for (var child = walker.GetFirstChild(node); child != null; child = walker.GetNextSibling(child)) Visit(child, depth + 1);
 }
}
'@
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
[Console]::Out.WriteLine('ARION_CONTEXT_READY')
[Console]::Out.Flush()
if ([Console]::In.ReadLine() -ne 'GO') { throw 'Traversal not authorized' }
[Console]::Write([ArionContextReader]::Read([long]$env:ARION_CONTEXT_HWND))`

func readWindowText(ctx context.Context, hwnd uintptr, authorize func() error) (string, error) {
	return runContextProcess(ctx, windowTextCommand(ctx, hwnd, accessibilityScript), authorize, contextStartupTimeout, screenTimeout)
}

func windowTextCommand(ctx context.Context, hwnd uintptr, source string) *exec.Cmd {
	encoded := utf16.Encode([]rune(source))
	script := make([]byte, len(encoded)*2)
	for i, u := range encoded {
		script[2*i], script[2*i+1] = byte(u), byte(u>>8)
	}
	// UI Automation clients require a non-UI MTA thread; PowerShell defaults to STA.
	cmd := exec.CommandContext(ctx, "powershell.exe", "-Mta", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(script))
	cmd.Env = append(cmd.Environ(), "ARION_CONTEXT_HWND="+strconv.FormatUint(uint64(hwnd), 10))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return cmd
}

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
	if cmd.Stderr == nil {
		cmd.Stderr = io.Discard
	}
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
