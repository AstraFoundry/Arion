//go:build windows

package windows

import (
	"bytes"
	"context"
	"encoding/base64"
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
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
[Console]::Write(($lines -join "` + "`n" + `"))`

func readWindowText(ctx context.Context, hwnd uintptr) (string, error) {
	encoded := utf16.Encode([]rune(accessibilityScript))
	script := make([]byte, len(encoded)*2)
	for i, u := range encoded {
		script[2*i], script[2*i+1] = byte(u), byte(u>>8)
	}
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(script))
	cmd.Env = append(cmd.Environ(), "ARION_CONTEXT_HWND="+strconv.FormatUint(uint64(hwnd), 10))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", domain.Fail(domain.CodeTargetUnsupported, "visible UI Automation text unavailable")
	}
	return strings.TrimSpace(strings.ReplaceAll(out.String(), "\r\n", "\n")), nil
}

const screenTimeout = 1500 * time.Millisecond
