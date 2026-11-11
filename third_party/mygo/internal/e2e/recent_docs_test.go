//go:build !js

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

// TestRecentDocuments records a document the user opened and clears it
// again. On Linux the entry goes into the user's recently-used.xbel (GLib
// reads XDG_DATA_HOME once, so a temporary one would not be seen): the test
// removes its own entries, the only ones of its executable. Windows lists
// only files of the types the app opens, so the test registers one.
func TestRecentDocuments(t *testing.T) {
	if _, ok := recentDocumentRecorded(""); !ok {
		t.Skip("recent documents cannot be read back on " + runtime.GOOS)
	}
	if runtime.GOOS == "windows" {
		registerFileType(t, ".mygorecent")
	}
	// The systems keep paths with symbolic links (macOS: /var is
	// /private/var) and short names (Windows: RUNNER~1) resolved.
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "recent document.mygorecent")
	if err := os.WriteFile(path, []byte("recent"), 0o644); err != nil {
		t.Fatal(err)
	}
	recorded := func() bool { r, _ := recentDocumentRecorded(path); return r }
	defer mygo.App.ClearRecentDocuments()

	if err := mygo.App.AddRecentDocument(path); err != nil {
		t.Fatalf("AddRecentDocument: %v", err)
	}
	eventually(t, "the document to be recorded", recorded)
	if err := mygo.App.ClearRecentDocuments(); err != nil {
		t.Fatalf("ClearRecentDocuments: %v", err)
	}
	eventually(t, "the recent documents to be cleared", func() bool { return !recorded() })
}

// TestJumpListTasks commits the tasks of the jump list (Windows), then
// removes them. Elsewhere SetTasks does nothing.
func TestJumpListTasks(t *testing.T) {
	tasks := []mygo.JumpListTask{
		{Title: "New window", Description: "Open a new window", Args: []string{"mygo-e2e://new", "two words"}, WorkingDir: os.TempDir()},
		{Title: "No arguments"},
	}
	if err := mygo.App.JumpList.SetTasks(tasks); err != nil {
		t.Errorf("SetTasks: %v", err)
	}
	if err := mygo.App.JumpList.SetTasks(nil); err != nil {
		t.Errorf("SetTasks(nil): %v", err)
	}
}

// registerFileType makes the test executable the handler of the files with
// extension ext for the current user (Windows), as the installer does for
// the app's file associations, until the test ends.
func registerFileType(t *testing.T, ext string) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	reg := func(args ...string) {
		if out, err := exec.Command("reg", args...).CombinedOutput(); err != nil {
			t.Fatalf("reg %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	const progID = "MyGo.E2E.RecentDocument"
	reg("add", `HKCU\Software\Classes\`+ext+`\OpenWithProgids`, "/v", progID, "/t", "REG_NONE", "/f")
	reg("add", `HKCU\Software\Classes\`+progID+`\shell\open\command`, "/ve", "/d", `"`+exe+`" "%1"`, "/f")
	t.Cleanup(func() {
		exec.Command("reg", "delete", `HKCU\Software\Classes\`+ext, "/f").Run()
		exec.Command("reg", "delete", `HKCU\Software\Classes\`+progID, "/f").Run()
	})
}
