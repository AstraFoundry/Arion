//go:build !js

package mygo

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// testRecent gives the test empty recent documents in its own user data,
// and the system's list in the fake backend.
func testRecent(t *testing.T) {
	App.SetPath(PathUserData, t.TempDir())
	reset := func() {
		recentDocs.Lock()
		recentDocs.loaded, recentDocs.paths = false, nil
		recentDocs.Unlock()
	}
	reset()
	t.Cleanup(func() { App.SetPath(PathUserData, ""); reset() })
	onMain(func() { fb.RecentDocuments = nil; fb.RecentCleared = 0 })
}

func TestAddRecentDocument(t *testing.T) {
	testRecent(t)
	system := func() []string { return onMainValue(func() []string { return slices.Clone(fb.RecentDocuments) }) }

	// A relative path is recorded as an absolute one, there and with the
	// system.
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("todo.txt", nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := App.AddRecentDocument("todo.txt"); err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs("todo.txt")
	if got := App.RecentDocuments(); !slices.Equal(got, []string{want}) {
		t.Errorf("RecentDocuments = %q, want [%q]", got, want)
	}
	if got := system(); !slices.Equal(got, []string{want}) {
		t.Errorf("the system's recent documents = %q, want [%q]", got, want)
	}
	// The newest comes first, once, and the list keeps ten.
	var paths []string
	for i := range 12 {
		p := filepath.Join(dir, fmt.Sprintf("%d.txt", i))
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
		if err := App.AddRecentDocument(p); err != nil {
			t.Fatal(err)
		}
	}
	if err := App.AddRecentDocument(paths[5]); err != nil {
		t.Fatal(err)
	}
	got := App.RecentDocuments()
	if len(got) != maxRecentDocuments || got[0] != paths[5] || got[1] != paths[11] || slices.Contains(got[1:], paths[5]) {
		t.Errorf("RecentDocuments = %q", got)
	}
	// Files that no longer exist are left out.
	if err := os.Remove(paths[11]); err != nil {
		t.Fatal(err)
	}
	if got := App.RecentDocuments(); slices.Contains(got, paths[11]) || len(got) != maxRecentDocuments-1 {
		t.Errorf("RecentDocuments = %q, want no %s", got, paths[11])
	}
	// The list outlives the run.
	recentDocs.Lock()
	recentDocs.loaded, recentDocs.paths = false, nil
	recentDocs.Unlock()
	if got := App.RecentDocuments(); len(got) != maxRecentDocuments-1 || got[0] != paths[5] {
		t.Errorf("RecentDocuments read again = %q", got)
	}
	// An empty path is not recorded.
	if err := App.AddRecentDocument(""); err == nil {
		t.Error("AddRecentDocument(\"\") succeeded")
	}
}

func TestClearRecentDocuments(t *testing.T) {
	testRecent(t)
	p := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := App.AddRecentDocument(p); err != nil {
		t.Fatal(err)
	}
	if err := App.ClearRecentDocuments(); err != nil {
		t.Fatal(err)
	}
	if got := App.RecentDocuments(); len(got) != 0 {
		t.Errorf("RecentDocuments after clear = %q, want none", got)
	}
	if got := onMainValue(func() []string { return fb.RecentDocuments }); got != nil {
		t.Errorf("the system's recent documents after clear = %q, want none", got)
	}
	if n := onMainValue(func() int { return fb.RecentCleared }); n != 1 {
		t.Errorf("RecentCleared = %d, want 1", n)
	}
}

// TestOpenRecentWithoutDocuments checks that the Open Recent menu of an
// app without documents lists what it recorded, and gives the file it
// chooses to OnOpenFile.
func TestOpenRecentWithoutDocuments(t *testing.T) {
	testRecent(t)
	docRegistry.Lock()
	kinds := docRegistry.kinds
	docRegistry.kinds = nil
	docRegistry.Unlock()
	t.Cleanup(func() {
		docRegistry.Lock()
		docRegistry.kinds = kinds
		docRegistry.Unlock()
	})
	p := filepath.Join(t.TempDir(), "folder")
	if err := os.Mkdir(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := App.AddRecentDocument(p); err != nil {
		t.Fatal(err)
	}
	var opened []string
	off := App.OnOpenFile(func(path string) { opened = append(opened, path) })
	defer off()
	items := NewMenu([]*MenuItem{{Role: RoleOpenRecent}}).Items()[0].Submenu
	if len(items) != 3 || items[0].Label != "folder" || items[2].Label != "Clear Menu" {
		t.Fatalf("Open Recent = %v", items)
	}
	onMain(func() { items[0].Click(items[0], nil) })
	if !slices.Equal(opened, []string{p}) {
		t.Errorf("OnOpenFile got %q, want [%q]", opened, p)
	}
}

func TestJumpListSetTasks(t *testing.T) {
	list := func() (platform.JumpList, int) {
		return onMainValue(func() platform.JumpList { return fb.JumpList }), onMainValue(func() int { return fb.JumpListCommits })
	}
	_, commits := list()
	tasks := []JumpListTask{
		{Title: "New note", Description: "Write a note", Args: []string{"myapp://new", "two words"}},
		{Title: "Open", Path: `C:\Windows\notepad.exe`, WorkingDir: `C:\`, IconPath: "/icon.ico", IconIndex: 3},
	}
	if err := App.JumpList.SetTasks(tasks); err != nil {
		t.Fatal(err)
	}
	tasks[0].Args[0] = "changed" // the jump list keeps its own
	got, n := list()
	want := []platform.JumpListTask{
		{Title: "New note", Description: "Write a note", Args: []string{"myapp://new", "two words"}},
		{Title: "Open", Path: `C:\Windows\notepad.exe`, WorkingDir: `C:\`, IconPath: "/icon.ico", IconIndex: 3},
	}
	if n != commits+1 || !reflect.DeepEqual(got.Tasks, want) {
		t.Fatalf("JumpList = %+v after %d commits, want %+v", got, n-commits, want)
	}
	if kept := App.JumpList.Tasks(); len(kept) != 2 || kept[0].Args[0] != "myapp://new" {
		t.Errorf("Tasks = %+v", kept)
	}
	// No tasks removes them.
	if err := App.JumpList.SetTasks(nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := list(); len(got.Tasks) != 0 {
		t.Errorf("JumpList after SetTasks(nil) = %+v, want no tasks", got)
	}
	// A task without a title sets nothing.
	if err := App.JumpList.SetTasks([]JumpListTask{{Title: "A"}, {Args: []string{"--x"}}}); err == nil {
		t.Error("SetTasks with an untitled task succeeded")
	}
	if got, n := list(); len(got.Tasks) != 0 || n != commits+2 {
		t.Errorf("JumpList after an untitled task = %+v, %d commits", got, n-commits)
	}
}
