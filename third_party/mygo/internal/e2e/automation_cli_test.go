//go:build !mygo_noinspector && !js

package e2e

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/automationcli"
	"github.com/egoist/mygo/internal/wait"
)

// TestAutomationCLI drives automationApp with mygo-automation's commands,
// as an agent does: it opens a session, reads the window's elements with
// refs, acts on them and reads what changed once the window settled,
// types, chooses a menu item, saves a screenshot, sees a ref go stale and
// closes the session, which leaves the app running.
func TestAutomationCLI(t *testing.T) {
	app := startAutomationApp(t)
	app.dial(t).Close()
	dir := t.TempDir()
	// run runs a command line and returns its output and exit status.
	run := func(args ...string) (string, int) {
		var out bytes.Buffer
		args = append(args, "--session", "e2e")
		code := automationcli.Run(args, strings.NewReader(""), &out, &out, automationcli.Options{Dir: dir})
		return out.String(), code
	}
	// must runs a command line that succeeds, and checks that its output
	// has the lines want, in order.
	must := func(want []string, args ...string) string {
		t.Helper()
		out, code := run(args...)
		if code != 0 {
			t.Fatalf("mygo-automation %s: exit %d\n%s", strings.Join(args, " "), code, out)
		}
		rest := out
		for _, w := range want {
			i := strings.Index(rest, w+"\n")
			if i < 0 || (i > 0 && rest[i-1] != '\n') {
				t.Fatalf("mygo-automation %s:\n%s\nlacks the line, in order: %s", strings.Join(args, " "), out, w)
			}
			rest = rest[i+len(w)+1:]
		}
		return out
	}
	// --timeout 0: the window settles; how long it takes is the machine's.
	settle := func(want []string, args ...string) string {
		t.Helper()
		return must(want, append(args, "--settle", "--timeout", "0")...)
	}

	must([]string{"opened " + app.path + " (session e2e)"}, "open", "--socket", app.path)
	wait.Until(t, "the app's window", func() bool {
		out, _ := run("windows")
		return strings.HasPrefix(out, "w") && strings.Contains(out, `[content] "Automation"`)
	})
	must(nil, "wait", "--timeout", "0")
	must([]string{
		`@e1 [text] "Count: 0"`,
		`@e2 [button] "Add" key=add`,
		`@e3 [textField] "Name" key=name`,
	}, "snapshot")
	if out := must(nil, "snapshot", "-i"); strings.Contains(out, "Count") || !strings.Contains(out, `@e2 [button] "Add"`) {
		t.Errorf("snapshot -i:\n%s", out)
	}

	// Pressing by ref and by key, as assistive technology does, and the
	// counter's change once the window settled.
	settle([]string{`- @e1 [text] "Count: 0"`, `+ @e1 [text] "Count: 1"`}, "press", "@e2")
	settle([]string{`- @e1 [text] "Count: 1"`, `+ @e1 [text] "Count: 2"`}, "press", "key=add")

	// Filling the field shows the greeting, with a ref of its own.
	out := settle([]string{`- @e3 [textField] "Name" key=name`, `+ @e4 [text] "Hello, Ada"`}, "fill", "@e3", "Ada")
	if !strings.Contains(out, `+ @e3 [textField] "Name" value="Ada" key=name`) {
		t.Errorf("fill:\n%s", out)
	}

	// Clicking the field, then typing and pressing keys.
	settle(nil, "click", "@e3")
	settle([]string{`+ @e4 [text] "Hello, Ada L"`}, "type", " L")
	settle([]string{`+ @e4 [text] "Hello, Ada "`}, "key", "Backspace")

	// The greeting goes, and its ref with it: it never names what comes
	// back in its place.
	settle([]string{`- @e4 [text] "Hello, Ada "`}, "fill", "@e3", "")
	if out, code := run("click", "@e4"); code != 1 || !strings.Contains(out, "stale ref @e4") {
		t.Errorf("clicking a stale ref: exit %d\n%s", code, out)
	}
	settle([]string{`+ @e5 [text] "Hello, Bo"`}, "fill", "@e3", "Bo")

	// A page's script, in native UI.
	if out, code := run("eval", "1"); code != 1 || !strings.Contains(out, "unsupported") {
		t.Errorf("eval in native UI: exit %d\n%s", code, out)
	}

	// The window as an image, in a file.
	shot := filepath.Join(t.TempDir(), "shot.png")
	must(nil, "screenshot", shot)
	f, err := os.Open(shot)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if s := img.Bounds().Size(); s.X < 200 || s.Y < 150 {
		t.Errorf("the screenshot is %v", s)
	}

	// The application menu, where the platform lets the plugin choose.
	if out, code := run("menu", "Test", "Reset", "--settle", "--timeout", "0"); code == 0 {
		if !strings.Contains(out, `+ @e1 [text] "Count: 0"`) {
			t.Errorf("menu Test Reset:\n%s", out)
		}
	} else {
		t.Logf("menus cannot be chosen on this platform: %s", out)
	}

	// Closing the session leaves the app running.
	must([]string{"closed session e2e"}, "close")
	if out, code := run("snapshot"); code != 1 || !strings.Contains(out, `no session "e2e"`) {
		t.Errorf("snapshot after close: exit %d\n%s", code, out)
	}
	select {
	case err := <-app.exited:
		app.ended = true
		t.Fatalf("the app exited: %v", err)
	default:
	}
	app.dial(t).Close()
}
