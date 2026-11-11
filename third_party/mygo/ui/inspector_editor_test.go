//go:build !mygo_noinspector

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/wait"
)

func TestSplitSource(t *testing.T) {
	file := filepath.Join("dir", "main.go")
	for _, tt := range []struct {
		source, file string
		line         int
	}{
		{"dir/main.go:42", file, 42},
		{"C:/dir/main.go:7", filepath.FromSlash("C:/dir/main.go"), 7},
		{"main.go", "main.go", 0},
		{"C:/dir", "C:/dir", 0},
	} {
		if f, l := splitSource(tt.source); f != tt.file || l != tt.line {
			t.Errorf("splitSource(%q) = %q, %d, want %q, %d", tt.source, f, l, tt.file, tt.line)
		}
	}
}

func TestFileURL(t *testing.T) {
	if got, want := fileURL(filepath.FromSlash("/a b/main.go")), "file:///a%20b/main.go"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := fileURL("C:/dir/main.go"), "file:///C:/dir/main.go"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSplitCommand(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want []string
	}{
		{"code", []string{"code"}},
		{"  code   --wait ", []string{"code", "--wait"}},
		{`"/Applications/Sublime Text.app/subl" -w`, []string{"/Applications/Sublime Text.app/subl", "-w"}},
		{`'C:\Program Files\Editor\e.exe' --new`, []string{`C:\Program Files\Editor\e.exe`, "--new"}},
		{`e ""`, []string{"e", ""}},
		{"", nil},
	} {
		if got := splitCommand(tt.in); !slices.Equal(got, tt.want) {
			t.Errorf("splitCommand(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEditorCommand(t *testing.T) {
	for _, tt := range []struct {
		setting string
		want    []string
	}{
		{"code", []string{"code", "-r", "-g", "/a/m.go:42:1"}},
		{"code --wait", []string{"code", "--wait", "-r", "-g", "/a/m.go:42:1"}},
		{"cursor", []string{"cursor", "-r", "-g", "/a/m.go:42:1"}},
		{"/usr/local/bin/zed", []string{"/usr/local/bin/zed", "/a/m.go:42:1"}},
		{"subl -w", []string{"subl", "-w", "/a/m.go:42:1"}},
		{"goland", []string{"goland", "--line", "42", "--column", "1", "/a/m.go"}},
		{"mate", []string{"mate", "--line", "42", "/a/m.go"}},
		{"gvim", []string{"gvim", "+42", "/a/m.go"}},
		{"emacsclient -n", []string{"emacsclient", "-n", "+42:1", "/a/m.go"}},
		{"notepad++", []string{"notepad++", "-n42", "-c1", "/a/m.go"}},
		// What isn't known gets no line, which could open a wrong file.
		{"kwrite", []string{"kwrite", "/a/m.go"}},
		// A command says where they go.
		{"tmux new-window nvim +{line} {file}", []string{"tmux", "new-window", "nvim", "+42", "/a/m.go"}},
		{"open-at {file}:{line}", []string{"open-at", "/a/m.go:42"}},
	} {
		name, args := editorCommand(tt.setting, "/a/m.go", 42)
		if got := append([]string{name}, args...); !slices.Equal(got, tt.want) {
			t.Errorf("editorCommand(%q) = %q, want %q", tt.setting, got, tt.want)
		}
	}
	// The executables and scripts of Windows, whatever their case.
	for _, program := range []string{"code.cmd", "Code.exe", `"C:/Program Files/VS Code/bin/code.CMD"`} {
		if _, args := editorCommand(program, "m.go", 1); !slices.Equal(args, []string{"-r", "-g", "m.go:1:1"}) {
			t.Errorf("%s: %q", program, args)
		}
	}
}

func TestChooseEditor(t *testing.T) {
	env := func(mygo, visual, editor string) func(string) string {
		return func(name string) string {
			return map[string]string{"MYGO_EDITOR": mygo, "VISUAL": visual, "EDITOR": editor}[name]
		}
	}
	for _, tt := range []struct {
		mygo, visual, editor, want string
	}{
		{"zed", "code", "subl", "zed"},
		{"", "code", "subl", "code"},
		{"", "", "subl", "subl"},
		{"", "", "", ""},
		{"  ", "", "subl", "subl"},
		// A terminal's editors can't open from an app.
		{"", "", "vim", ""},
		{"", "nvim", "subl", "subl"},
		{"vim", "", "code --wait", "code --wait"},
		{"", "/usr/bin/nano", "", ""},
		{"", "emacs -nw", "", ""},
		{"", "emacs", "", "emacs"},
		// Unless the command says how.
		{"kitty nvim +{line} {file}", "code", "", "kitty nvim +{line} {file}"},
	} {
		if got := chooseEditor(env(tt.mygo, tt.visual, tt.editor)); got != tt.want {
			t.Errorf("MYGO_EDITOR=%q VISUAL=%q EDITOR=%q: got %q, want %q", tt.mygo, tt.visual, tt.editor, got, tt.want)
		}
	}
}

// sourced returns an open inspector with an element chosen whose source is
// the location given: callSite skips package ui, so tests can't have one.
func sourced(t *testing.T, source string) *Tester {
	t.Helper()
	var tt *Tester
	tt, _ = inspected(func(c *context) {
		coreColumn(c).Key("box").Children(func() { coreText(c, "Inside") })
		c.rt.insp.source = source
	})
	openInspector(tt)
	chooseKeyed(t, tt, "box")
	tt.Frame()
	return tt
}

// sourceFile returns a file to open, and its location at a line.
func sourceFile(t *testing.T, name string) (file, source string) {
	t.Helper()
	file = filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return file, filepath.ToSlash(file) + ":42"
}

func noEditors(t *testing.T) {
	for _, name := range []string{"MYGO_EDITOR", "VISUAL", "EDITOR"} {
		t.Setenv(name, "")
	}
}

// TestEditorHelper stands in for an editor, as TestInspectorOpensTheEditor
// starts this test's program: it notes its arguments, whole once it renames
// them into place.
func TestEditorHelper(t *testing.T) {
	out := os.Getenv("MYGO_TEST_EDITOR_OUT")
	if out == "" {
		t.Skip("run by TestInspectorOpensTheEditor")
	}
	args := os.Args[slices.Index(os.Args, "--")+1:]
	if err := os.WriteFile(out+".tmp", []byte(strings.Join(args, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(out+".tmp", out); err != nil {
		t.Fatal(err)
	}
}

func TestInspectorOpensTheEditor(t *testing.T) {
	if !editorsRun {
		t.Skip("this system doesn't start programs")
	}
	noEditors(t)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "args")
	t.Setenv("MYGO_TEST_EDITOR_OUT", out)
	t.Setenv("MYGO_EDITOR", fmt.Sprintf(`"%s" -test.run=^TestEditorHelper$ -- {file} {line}`, exe))
	file, source := sourceFile(t, "main.go")
	tt := sourced(t, source)

	if err := tt.Click("main.go:42"); err != nil {
		t.Fatal(err)
	}
	wait.Until(t, "the editor to run", func() bool {
		_, err := os.Stat(out)
		return err == nil
	})
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if want := file + "\n42"; string(got) != want {
		t.Errorf("the editor got %q, want %q", got, want)
	}
	if len(tt.OpenedURLs()) != 0 {
		t.Errorf("opened %q as well", tt.OpenedURLs())
	}
}

// Without an editor set, the file opens in the system's application for it.
func TestInspectorOpensTheDefaultApplication(t *testing.T) {
	if !editorsRun {
		t.Skip("this system doesn't start programs")
	}
	noEditors(t)
	t.Setenv("EDITOR", "vim")
	file, source := sourceFile(t, "main.go")
	tt := sourced(t, source)
	if err := tt.Click("main.go:42"); err != nil {
		t.Fatal(err)
	}
	if got, want := tt.OpenedURLs(), []string{fileURL(file)}; !slices.Equal(got, want) {
		t.Errorf("opened %q, want %q", got, want)
	}
	if n := len(tt.rt.warnings); n != 0 {
		t.Errorf("warned: %q", tt.rt.warnings)
	}

	tt.FailOpenURL(fmt.Errorf("no application"))
	if err := tt.Click("main.go:42"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if n := len(tt.rt.warnings); n != 1 || !strings.Contains(tt.rt.warnings[0], "no application") {
		t.Errorf("warnings: %q", tt.rt.warnings)
	}
}

// What can't start is an issue, not a click that does nothing.
func TestInspectorWarnsOfEditors(t *testing.T) {
	if !editorsRun {
		t.Skip("this system doesn't start programs")
	}
	noEditors(t)
	t.Setenv("MYGO_EDITOR", "mygo-no-such-editor")
	_, source := sourceFile(t, "main.go")
	tt := sourced(t, source)
	if err := tt.Click("main.go:42"); err != nil {
		t.Fatal(err)
	}
	if w := tt.rt.warnings; len(w) != 1 || !strings.Contains(w[0], "mygo-no-such-editor") {
		t.Errorf("warnings: %q", w)
	}

	t.Setenv("MYGO_EDITOR", "vim")
	if err := tt.Click("main.go:42"); err != nil {
		t.Fatal(err)
	}
	if w := tt.rt.warnings; len(w) != 2 || !strings.Contains(w[1], "needs a terminal") {
		t.Errorf("warnings: %q", w)
	}
	if got := tt.OpenedURLs(); len(got) != 1 {
		t.Errorf("opened %q, want the file in the default application", got)
	}
}

// A path that isn't a file here, as -trimpath gives, doesn't look like a link.
func TestInspectorSourceWithoutAFile(t *testing.T) {
	noEditors(t)
	tt := sourced(t, "example.com/app/main.go:42")
	var link *node
	var walk func(e *node)
	walk = func(e *node) {
		if e.kind == kindText && e.text == "main.go:42" {
			link = e
		}
		for c := e.first; c != nil; c = c.next {
			walk(c)
		}
	}
	walk(tt.rt.c.root)
	if link == nil {
		t.Fatalf("no location: %q", tt.Texts())
	}
	if link.role == RoleLink {
		t.Error("a location that isn't a file is a link")
	}
	if err := tt.Click("main.go:42"); err != nil {
		t.Fatal(err)
	}
	if got := tt.OpenedURLs(); len(got) != 0 {
		t.Errorf("opened %q", got)
	}
	if w := tt.rt.warnings; len(w) != 0 {
		t.Errorf("warnings: %q", w)
	}
}
