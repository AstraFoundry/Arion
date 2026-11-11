//go:build !js

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

// TestInstalledApps lists the installed apps, sorted, with names, IDs and
// paths whose icons the shell draws, and launches one where that is
// harmless: on Linux an app of a desktop entry the test adds, which leaves
// a file behind, and on Windows CI Notepad.
func TestInstalledApps(t *testing.T) {
	var marker, entryID string
	if runtime.GOOS == "linux" {
		// GLib reads the XDG directories once: the test adds an entry only
		// where the runner points them, as CI does.
		data := os.Getenv("XDG_DATA_HOME")
		if data == "" {
			t.Log("XDG_DATA_HOME unset: not adding a desktop entry")
		} else {
			marker = filepath.Join(t.TempDir(), "launched")
			entryID = "mygo-e2e-launcher-test.desktop"
			entry := filepath.Join(data, "applications", entryID)
			os.MkdirAll(filepath.Dir(entry), 0o755)
			os.WriteFile(entry, []byte("[Desktop Entry]\nType=Application\nName=MyGo Launcher Test\nName[fr]=Test du lanceur\nExec=touch "+marker+"\nIcon=folder\n"), 0o644)
			t.Cleanup(func() { os.Remove(entry) })
		}
	}

	var apps []mygo.InstalledApp
	eventually(t, "the installed apps", func() bool {
		var err error
		if apps, err = mygo.Shell.InstalledApps(); err != nil {
			t.Fatal(err)
		}
		return len(apps) > 0 && (entryID == "" || slices.ContainsFunc(apps, func(a mygo.InstalledApp) bool { return a.ID == entryID }))
	})
	t.Logf("%d apps, the first %+v", len(apps), apps[0])
	if !slices.IsSortedFunc(apps, func(a, b mygo.InstalledApp) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	}) {
		t.Error("the apps are not sorted by name")
	}
	for i, a := range apps {
		if a.Name == "" || a.ID == "" || a.Path == "" {
			t.Errorf("app %d lacks a name, an ID or a path: %+v", i, a)
		}
		if runtime.GOOS != "windows" {
			if _, err := os.Stat(a.Path); err != nil {
				t.Errorf("%s: %v", a.Name, err)
			}
		}
	}
	for _, a := range apps[:min(len(apps), 10)] {
		if _, err := mygo.Shell.FileIcon(a.Path, 32); err != nil {
			t.Errorf("the icon of %s (%s): %v", a.Name, a.Path, err)
		}
	}

	switch runtime.GOOS {
	case "darwin":
		i := slices.IndexFunc(apps, func(a mygo.InstalledApp) bool { return a.ID == "com.apple.finder" })
		if i < 0 || apps[i].Path != "/System/Library/CoreServices/Finder.app" {
			t.Errorf("Finder is not listed as itself: %v", i)
		}
		if slices.ContainsFunc(apps, func(a mygo.InstalledApp) bool { return strings.HasSuffix(a.Name, ".app") }) {
			t.Error("a name ends with .app")
		}
	case "linux":
		if entryID == "" {
			return
		}
		i := slices.IndexFunc(apps, func(a mygo.InstalledApp) bool { return a.ID == entryID })
		if apps[i].Name != "MyGo Launcher Test" {
			t.Errorf("the entry's name is %q", apps[i].Name)
		}
		if err := mygo.Shell.LaunchApp(apps[i]); err != nil {
			t.Fatal(err)
		}
		eventually(t, "the app launched", func() bool { _, err := os.Stat(marker); return err == nil })
	case "windows":
		if os.Getenv("GITHUB_ACTIONS") == "" {
			return // not on someone's desktop
		}
		i := slices.IndexFunc(apps, func(a mygo.InstalledApp) bool { return strings.HasSuffix(strings.ToLower(a.ID), `\notepad.exe`) })
		if i < 0 {
			t.Log("no Notepad to launch")
			return
		}
		if err := mygo.Shell.LaunchApp(apps[i]); err != nil {
			t.Fatal(err)
		}
		eventually(t, "Notepad running", func() bool {
			out, _ := exec.Command("tasklist", "/FI", "IMAGENAME eq notepad.exe").Output()
			return strings.Contains(strings.ToLower(string(out)), "notepad.exe")
		})
		exec.Command("taskkill", "/IM", "notepad.exe", "/F").Run()
		time.Sleep(100 * time.Millisecond)
	}
}
