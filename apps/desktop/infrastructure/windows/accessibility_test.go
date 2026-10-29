//go:build windows

package windows

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"arion.local/core/domain"
)

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
	source = strings.Replace(source, "Add-Type -AssemblyName UIAutomationClient\n",
		"[Console]::Error.WriteLine('ARION_INIT apartment-' + [System.Threading.Thread]::CurrentThread.GetApartmentState())\nAdd-Type -AssemblyName UIAutomationClient\n", 1)
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
	if !strings.Contains(diagnostics.String(), "ARION_INIT apartment-MTA") {
		t.Fatal("real UI Automation reader did not initialize on an MTA thread")
	}
}

func TestContextReaderRejectsSTABeforeAuthorization(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), contextStartupTimeout+time.Second)
	defer cancel()
	cmd := windowTextCommand(ctx, 0, accessibilityScript)
	for i, arg := range cmd.Args {
		if arg == "-Mta" {
			cmd.Args[i] = "-Sta"
		}
	}
	var diagnostics bytes.Buffer
	cmd.Stderr = &diagnostics
	authorized := false
	text, err := runContextProcess(ctx, cmd, func() error { authorized = true; return nil }, contextStartupTimeout, screenTimeout)
	if text != "" || authorized || domain.CodeOf(err) != domain.CodeTargetUnsupported {
		t.Fatalf("STA reader reached target authorization: authorized=%v text=%q error=%v", authorized, text, err)
	}
	if !strings.Contains(diagnostics.String(), "UI Automation requires a multithreaded apartment") {
		t.Fatalf("STA did not fail at the apartment contract: %s", diagnostics.String())
	}
	requireContextProcessReaped(t, cmd)
}
