package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo"
)

func TestDevelopmentDoesNotLoadReleaseProfileOrInstalledInputMethod(t *testing.T) {
	if !mygo.IsDev() {
		t.Fatal("this source test must run as a development build")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	production := filepath.Join(base, "Arion", "preferences.json")
	if err := os.MkdirAll(filepath.Dir(production), 0700); err != nil {
		t.Fatal(err)
	}
	// A corrupted release profile must not prevent development from starting.
	const original = "invalid production preferences"
	if err := os.WriteFile(production, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	a, err := New(Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if a.opts.DataDir != filepath.Join(base, "Arion Dev") || a.ime != nil {
		t.Fatalf("development owns release state: directory=%q ime=%T", a.opts.DataDir, a.ime)
	}
	got, err := os.ReadFile(production)
	if err != nil || string(got) != original {
		t.Fatalf("release profile changed: %q %v", got, err)
	}
}
