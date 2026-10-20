package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// lockedBuffer is a buffer that the spinner's goroutine writes to.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// liveConsole is a console that takes buf for a terminal.
func liveConsole(buf *lockedBuffer) *console {
	return &console{w: buf, live: true, sym: unicodeSymbols}
}

// screen replays what a console wrote to a terminal and returns the lines
// it shows: carriage returns, line feeds (which return too, as the
// terminal driver makes them), erasing to the end of the line, moving the
// cursor up and saving it; colors and other modes aside.
func screen(out string) []string {
	lines := [][]rune{nil}
	row, col := 0, 0
	var saved [2]int
	csi := regexp.MustCompile(`^\x1b\[([?0-9;]*)([A-Za-z])`)
	for out != "" {
		if m := csi.FindStringSubmatch(out); m != nil {
			switch m[2] {
			case "K":
				lines[row] = lines[row][:min(col, len(lines[row]))]
			case "A":
				n := 1
				if m[1] != "" {
					n = int(m[1][0] - '0')
				}
				row = max(0, row-n)
			}
			out = out[len(m[0]):]
			continue
		}
		switch {
		case strings.HasPrefix(out, "\x1b7"):
			saved = [2]int{row, col}
			out = out[2:]
			continue
		case strings.HasPrefix(out, "\x1b8"):
			row, col = saved[0], saved[1]
			out = out[2:]
			continue
		}
		r := []rune(out[:len(string([]rune(out)[0]))])[0]
		out = out[len(string(r)):]
		switch r {
		case '\r':
			col = 0
		case '\n':
			row, col = row+1, 0
			if row == len(lines) {
				lines = append(lines, nil)
			}
		default:
			for len(lines[row]) < col {
				lines[row] = append(lines[row], ' ')
			}
			if col < len(lines[row]) {
				lines[row][col] = r
			} else {
				lines[row] = append(lines[row], r)
			}
			col++
		}
	}
	var s []string
	for _, l := range lines {
		s = append(s, string(l))
	}
	return s
}

// durations matches how long steps took, which vary.
var durations = regexp.MustCompile(`\d+(\.\d)?(ms|s)\b`)

func TestConsoleSteps(t *testing.T) {
	var buf lockedBuffer
	c := liveConsole(&buf)
	step := c.start("Compiling windows/amd64")
	step.set("1 package · demo")
	out := c.output("web", &buf)
	_, _ = out.Write([]byte("$ vite\r\nbuilding"))
	_, _ = out.Write([]byte(" for production...\n\n"))
	time.Sleep(200 * time.Millisecond) // a frame or two of the spinner
	_, _ = out.Write([]byte("no newline"))
	out.flush()
	c.println("  • a line of the CLI's own")
	step.done("Compiled windows/amd64")
	inner := c.start("Signing Todo.exe")
	outer := c.start("Creating Todo Setup 1.0.0.exe")
	outer.fail()
	inner.stop()
	got := screen(durations.ReplaceAllString(buf.String(), "D"))
	want := []string{
		"  web │ $ vite",
		"  web │ building for production...",
		"  web │",
		"  web │ no newline",
		"  • a line of the CLI's own",
		"  ✓ Compiled windows/amd64  D",
		"  ✗ Creating Todo Setup 1.0.0.exe  D",
		"",
	}
	if !slices.Equal(got, want) {
		t.Errorf("screen:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if len(c.tasks) != 0 {
		t.Errorf("steps left: %d", len(c.tasks))
	}
}

func TestConsoleSpinnerLine(t *testing.T) {
	var buf lockedBuffer
	c := liveConsole(&buf)
	c.mu.Lock() // no frames from the spinner's goroutine meanwhile
	c.tasks = []*task{{c: c, title: "Compiling windows/amd64", detail: strings.Repeat("x", 200), start: time.Now().Add(-3 * time.Second)}}
	c.draw()
	c.tasks = nil
	c.mu.Unlock()
	got := screen(buf.String())
	if len(got) != 1 || !strings.HasPrefix(got[0], "  ⠋ Compiling windows/amd64 3s  xxx") || !strings.HasSuffix(got[0], "x…") {
		t.Errorf("spinner line: %q", got)
	}
	if n := len([]rune(got[0])); n != 79 { // a column to spare on 80
		t.Errorf("spinner line of %d columns, want 79", n)
	}
}

func TestConsoleNotLive(t *testing.T) {
	var buf lockedBuffer
	c := &console{w: &buf, sym: plainSymbols}
	step := c.start("Compiling windows/amd64")
	step.set("1 package · demo") // not shown
	step.done("Compiled windows/amd64")
	got := durations.ReplaceAllString(buf.String(), "D")
	if want := "  o Compiling windows/amd64\n  √ Compiled windows/amd64  D\n"; got != want {
		t.Errorf("output:\n%q\nwant:\n%q", got, want)
	}
	if strings.Contains(got, "\x1b") {
		t.Error("escape sequences without a terminal")
	}
}

func TestLabeledOutput(t *testing.T) {
	var buf lockedBuffer
	c := &console{w: &buf, sym: unicodeSymbols}
	out := c.output("app", &buf)
	for _, s := range []string{"a\r", "\nb\rc\n", "d"} {
		_, _ = out.Write([]byte(s))
	}
	if got, want := buf.String(), "  app │ a\n  app │ b\n  app │ c\n"; got != want {
		t.Errorf("before flush: %q, want %q", got, want)
	}
	out.flush()
	out.close()
	_, _ = out.Write([]byte("dropped\n"))
	out.flush()
	if got, want := buf.String(), "  app │ a\n  app │ b\n  app │ c\n  app │ d\n"; got != want {
		t.Errorf("after close: %q, want %q", got, want)
	}
}

func TestBannerSettles(t *testing.T) {
	var buf lockedBuffer
	c := liveConsole(&buf)
	c.color = true
	c.startBanner("dev")
	c.println("next")
	time.Sleep(3 * bannerFrame) // no frame after the line
	out := buf.String()
	last := strings.LastIndex(out, "\x1b7\x1b[2A")
	next := strings.Index(out, "next")
	if last < 0 || next < last {
		t.Fatalf("the banner was not settled before the next line:\n%q", out)
	}
	for _, col := range memberColors {
		if !strings.Contains(out[last:next], c.fg(col)+"m!") {
			t.Errorf("settled banner without the member color %v:\n%q", col, out[last:next])
		}
	}
	if got := screen(out); !strings.HasPrefix(got[1], "  MyGo!!!!! v") || got[3] != "next" {
		t.Errorf("screen: %q", got)
	}
}

func TestGoProgress(t *testing.T) {
	var buf lockedBuffer
	c := &console{w: &buf, sym: unicodeSymbols}
	step := c.start("Compiling")
	p := &goProgress{t: step}
	_, _ = p.Write([]byte("go: downloading github.com/ebitengine/purego v0.11.1\ninternal/goarch\r\nruntime\n# demo\n.\\main.go:42:2: undefined: thing\ngithub.com/egoist/mygo/ui"))
	p.flush()
	if p.packages != 3 || step.detail != "3 packages · github.com/egoist/mygo/ui" {
		t.Errorf("progress: %d packages, detail %q", p.packages, step.detail)
	}
	if want := "# demo\n.\\main.go:42:2: undefined: thing\n"; p.diag.String() != want {
		t.Errorf("errors: %q, want %q", p.diag.String(), want)
	}
	err := &goBuildError{out: p.diag.String(), err: errors.New("exit status 1")}
	if got, want := err.Error(), "go build failed:\n# demo\n.\\main.go:42:2: undefined: thing"; got != want {
		t.Errorf("error: %q, want %q", got, want)
	}
	step.stop()
}

func TestChangeSummary(t *testing.T) {
	root := filepath.FromSlash("/work/app")
	paths := func(names ...string) []string {
		var p []string
		for _, n := range names {
			p = append(p, filepath.Join(root, filepath.FromSlash(n)))
		}
		return p
	}
	for _, tt := range []struct {
		paths []string
		want  string
	}{
		{paths("main.go"), "main.go changed"},
		{paths("a.go", "ui/b.go"), "a.go and ui/b.go changed"},
		{paths("a.go", "b.go", "c.go"), "a.go, b.go and c.go changed"},
		{paths("a.go", "b.go", "c.go", "d.go", "e.go"), "a.go, b.go, c.go and 2 more changed"},
		{paths("../lib/x.go"), "../lib/x.go changed"},
	} {
		if got := changeSummary(root, tt.paths); got != tt.want {
			t.Errorf("changeSummary(%v) = %q, want %q", tt.paths, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	for d, want := range map[time.Duration]string{
		340 * time.Millisecond:  "340ms",
		2340 * time.Millisecond: "2.3s",
		65 * time.Second:        "1m 05s",
	} {
		if got := formatDuration(d); got != want {
			t.Errorf("formatDuration(%v) = %q, want %q", d, got, want)
		}
	}
	if got := formatElapsed(3900 * time.Millisecond); got != "3s" {
		t.Errorf("formatElapsed = %q, want 3s", got)
	}
}

func TestIsImportPath(t *testing.T) {
	for s, want := range map[string]bool{
		"runtime":                           true,
		"github.com/egoist/mygo/ui":         true,
		"command-line-arguments":            true,
		"# demo":                            false,
		".\\main.go:42:2: undefined: thing": false,
		"go: downloading x v1":              false,
		"":                                  false,
	} {
		if got := isImportPath(s); got != want {
			t.Errorf("isImportPath(%q) = %v", s, got)
		}
	}
}

func TestWaited(t *testing.T) {
	if waited(exec.ErrWaitDelay) != nil || waited(nil) != nil || waited(errors.New("x")) == nil {
		t.Error("waited")
	}
}
