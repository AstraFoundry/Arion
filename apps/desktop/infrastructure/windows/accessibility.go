//go:build windows

package windows

import (
	"context"
	"encoding/base64"
	"os/exec"
	"strconv"
	"syscall"
	"unicode/utf16"
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
using System.Reflection;
using System.Runtime.CompilerServices;
using System.Windows.Automation;
public sealed class ArionContextReader {
 private int count, length;
 private readonly List<string> lines = new List<string>();
 // Moves first-use costs out of the traversal budget without inspecting the
 // target: JIT for the traversal methods, the UI Automation client connection
 // and the pattern, walker and control-type statics that Visit uses.
 public static void Warm() {
  RuntimeHelpers.PrepareMethod(typeof(ArionContextReader).GetMethod("Read").MethodHandle);
  RuntimeHelpers.PrepareMethod(typeof(ArionContextReader).GetMethod("Visit", BindingFlags.NonPublic | BindingFlags.Instance).MethodHandle);
  GC.KeepAlive(AutomationElement.RootElement);
  GC.KeepAlive(TreeWalker.ControlViewWalker);
  GC.KeepAlive(TextPattern.Pattern);
  GC.KeepAlive(ControlType.Text);
 }
 public static string Read(long handle) {
  Console.Error.WriteLine("` + contextTraceReadEntered + `");
  Console.Error.WriteLine("` + contextTraceRootStarting + `");
  var root = AutomationElement.FromHandle(new IntPtr(handle));
  Console.Error.WriteLine("` + contextTraceRootReady + `");
  if (root == null) throw new InvalidOperationException("Window unavailable");
  var reader = new ArionContextReader();
  reader.Visit(root, 0);
  if (reader.lines.Count == 0) throw new InvalidOperationException("Visible text unavailable");
  var result = string.Join("\n", reader.lines);
  Console.Error.WriteLine("` + contextTraceWalkFinished + `");
  return result;
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
[ArionContextReader]::Warm()
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
[Console]::Out.WriteLine('ARION_CONTEXT_READY')
[Console]::Out.Flush()
if ([Console]::In.ReadLine() -ne 'GO') { throw 'Traversal not authorized' }
[Console]::Error.WriteLine('` + contextTraceGoReceived + `')
[Console]::Write([ArionContextReader]::Read([long]$env:ARION_CONTEXT_HWND))
[Console]::Out.Flush()
[Console]::Error.WriteLine('` + contextTraceOutputFlushed + `')`

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
