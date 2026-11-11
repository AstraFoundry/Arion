package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

func TestShellIntegrationMarkedPromptResize(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), ShellIntegration: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Resize(100, 20)
	term.Feed([]byte("history\r\n\x1b]133;A\a\r\n\x1b[Aresize-prompt\x1b[90Gclock\r\n> pending-command\x1b]133;B\a"))
	// The rows stay, shrink by one, grow or shrink by more.
	for _, size := range [][2]int{{80, 20}, {100, 19}, {40, 18}, {120, 24}, {80, 23}, {60, 23}} {
		cols, rows := size[0], size[1]
		term.Resize(cols, rows)
		want := "history\nresize-prompt" + strings.Repeat(" ", cols-18) + "clock\n> pending-command"
		if got := term.Text(); got != "history" {
			t.Fatalf("at %dx%d, the prompt was left: %q", cols, rows, got)
		}
		term.Feed([]byte(fmt.Sprintf("\r\x1b[A\x1b]133;A\aresize-prompt\x1b[%dGclock\r\n> pending-command\x1b]133;B\a", cols-4)))
		if got := term.Text(); got != want {
			t.Fatalf("at %dx%d: %q", cols, rows, got)
		}
	}
	term.Feed([]byte("\x1b]133;C\a\r\n"))
	output := strings.Repeat("long command output ", 20)
	term.Feed([]byte(output))
	term.Resize(40, 10)
	term.Resize(120, 30)
	if got := term.Text(); !strings.Contains(got, strings.TrimSpace(output)) {
		t.Fatalf("resizing command output lost text: %q", got)
	}
}

func TestShellIntegrationPreservesAlternateScreen(t *testing.T) {
	loadLib(t)
	term, err := New(Options{Conn: newPipe(), ShellIntegration: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Resize(100, 20)
	term.Feed([]byte("\x1b]133;A\aprompt\x1b]133;B\a\x1b[?1049halt-program"))
	term.Resize(40, 20)
	if got := term.Text(); !strings.Contains(got, "alt-program") {
		t.Fatalf("prompt tracking cleared a full-screen program: %q", got)
	}
}

func TestShellIntegrationOtherShells(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("runs sleep")
	}
	loadLib(t)
	term, err := New(Options{Command: []string{"sleep", "60"}, ShellIntegration: true})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	if term.shellDir != "" {
		t.Fatal("a program other than zsh was configured")
	}
	// Its own marks don't make it redraw its prompt on a resize: the
	// prompt stays, as without ShellIntegration.
	term.Feed([]byte("\x1b]133;A\aprompt\x1b]133;B\a"))
	term.Resize(40, 24)
	if got := term.Text(); got != "prompt" {
		t.Fatalf("the prompt of a program other than zsh was cleared: %q", got)
	}
}

func TestShellIntegrationUserStartup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires zsh")
	}
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh is not installed")
	}
	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprint(custom), func(t *testing.T) {
			home := t.TempDir()
			config := home
			env := []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "TERM=xterm-256color"}
			if custom {
				config = t.TempDir()
				env = append(env, "ZDOTDIR="+config)
			}
			for _, name := range []string{".zshenv", ".zprofile", ".zshrc", ".zlogin"} {
				if err := os.WriteFile(filepath.Join(config, name), []byte("print -r -- startup-"+name+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			env, dir, err := zshEnvironment(env)
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dir)
			cmd := exec.CommandContext(wait.Context(t), zsh, "-lic", `print -r -- restored:${+ZDOTDIR}:${ZDOTDIR-unset}:${+MYGO_TERMINAL_ZDOTDIR}`)
			cmd.Env = env
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("zsh startup: %v: %s", err, output)
			}
			for _, name := range []string{".zshenv", ".zprofile", ".zshrc", ".zlogin"} {
				if strings.Count(string(output), "startup-"+name) != 1 {
					t.Fatalf("startup file %s did not run exactly once: %q", name, output)
				}
			}
			want := "restored:0:unset:0"
			if custom {
				want = "restored:1:" + config + ":0"
			}
			if !strings.Contains(string(output), want) {
				t.Fatalf("ZDOTDIR was not restored: %q", output)
			}
		})
	}
}

func TestShellIntegrationPromptTracker(t *testing.T) {
	var p promptTracker
	for _, part := range []string{"\x1b]13", "3;A;redraw=1\x1b", "\\prompt\x1b]133;B\a"} {
		p.feed([]byte(part))
	}
	if !p.atPrompt {
		t.Fatal("fragmented prompt was missed")
	}
	for _, data := range []string{"\x1b]2;title\a", "\x1b]133;Command\a", "\x1bP\x1b]133;C\a\x1b\\"} {
		p.feed([]byte(data))
		if !p.atPrompt {
			t.Fatalf("unrelated or opaque control data changed prompt state: %q", data)
		}
	}
	p.feed([]byte("\x1b]133;C\a"))
	if p.atPrompt {
		t.Fatal("command output was marked as a prompt")
	}
	p.feed([]byte("\x1b]133;A;" + strings.Repeat("x", 1024) + "\a"))
	if p.atPrompt || len(p.body) > 64 {
		t.Fatal("oversized OSC was retained or accepted")
	}
	p.feed([]byte("\x1b]133;A\x18ignored\a"))
	if p.atPrompt {
		t.Fatal("canceled OSC was accepted")
	}
	p.feed([]byte("\x1b]133;A\a\x1bc"))
	if p.atPrompt {
		t.Fatal("terminal reset retained prompt state")
	}
	p.feed([]byte("\x1b]133;N\a"))
	if !p.atPrompt {
		t.Fatal("a new command's prompt was missed")
	}
}

func TestShellIntegrationZshResize(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires a Unix zsh PTY")
	}
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh is not installed")
	}
	loadLib(t)
	config := filepath.Join(t.TempDir(), "user config")
	if err := os.MkdirAll(config, 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		".zshenv": "export STARTUP_ENV=loaded\nprint -r -- startup-env:$ZDOTDIR:$NO_COLOR\n",
		".zshrc":  "setopt promptsubst\nPROMPT=$'resize-prompt:${COLUMNS}${(l:COLUMNS-24:)}clock\\n> '\nRPROMPT=right-prompt\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(config, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("SHELL", zsh)
	term, err := New(Options{ShellIntegration: true, Env: []string{"ZDOTDIR=" + config, "NO_COLOR=user-preference"}})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	term.Resize(100, 24)
	wait.Until(t, "initial zsh prompt", func() bool { return strings.Contains(term.Text(), "right-prompt") })
	if got := term.Text(); strings.Count(got, "startup-env:") != 1 || !strings.Contains(got, config+":user-preference") {
		t.Fatalf("user startup/environment not preserved: %q", got)
	}
	term.Send([]byte("echo pending-command"))
	wait.Until(t, "pending input", func() bool { return strings.Contains(term.Text(), "echo pending-command") })
	for _, cols := range []int{40, 120, 60, 100, 40} {
		term.Resize(cols, 24)
		wait.Until(t, fmt.Sprintf("zsh redraw at %d columns", cols), func() bool {
			text := term.Text()
			return strings.Contains(text, fmt.Sprintf("resize-prompt:%d", cols)) && strings.Contains(text, "echo pending-command")
		})
		if got := term.Text(); strings.Count(got, "resize-prompt:") != 1 {
			t.Fatalf("prompt duplicated at %d columns: %q", cols, got)
		}
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(config, name))
		if err != nil || string(got) != want {
			t.Fatal("shell integration modified user startup files")
		}
	}
	dir := term.shellDir
	term.Close()
	wait.For(t, "zsh exit", term.Done())
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("temporary shell configuration was not removed: %v", err)
	}
}

func TestShellIntegrationCloseBeforeStart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires zsh")
	}
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh is not installed")
	}
	loadLib(t)
	t.Setenv("SHELL", zsh)
	t.Setenv("HOME", t.TempDir())
	term, err := New(Options{ShellIntegration: true})
	if err != nil {
		t.Fatal(err)
	}
	dir := term.shellDir
	term.Close()
	wait.For(t, "unstarted terminal close", term.Done())
	if dir == "" {
		t.Fatal("zsh integration was not configured")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("temporary configuration leaked before start: %v", err)
	}
}
