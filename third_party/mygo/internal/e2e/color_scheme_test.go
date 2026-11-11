//go:build !js

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egoist/mygo"
)

// A page follows the appearance the app sets, as its prefers-color-scheme,
// also when it changes, and so does Theme.IsDark.
func TestPageColorScheme(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Title: "Color scheme", Width: 300, Height: 200})
	w.Page().LoadHTML(`<meta name="color-scheme" content="light dark"><p>scheme</p>`, "")
	waitFor(t, w, "document.readyState === 'complete'")
	if os.Getenv("MYGO_E2E_DARK_THEME") != "" && !mygo.Theme.IsDark() {
		t.Fatal("GTK's theme is not Adwaita-dark")
	}
	defer mygo.Theme.SetSource(mygo.Theme.Source())
	for _, source := range []mygo.ThemeSource{mygo.ThemeLight, mygo.ThemeDark, mygo.ThemeLight} {
		mygo.Theme.SetSource(source)
		dark := source == mygo.ThemeDark
		waitFor(t, w, fmt.Sprintf("matchMedia('(prefers-color-scheme: dark)').matches === %v", dark))
		if got := mygo.Theme.IsDark(); got != dark {
			t.Errorf("%s: Theme.IsDark() = %v", source, got)
		}
	}
}

// On a desktop whose GTK theme is named for its dark variant, as Ubuntu's
// Yaru-dark in dark mode, ThemeLight makes pages light too (#224):
// WebKitGTK takes such a theme for dark whatever
// gtk-application-prefer-dark-theme says.
func TestPageColorSchemeDarkTheme(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("GTK's theme is Linux's")
	}
	if os.Getenv("MYGO_E2E_DARK_THEME") != "" {
		t.Skip("in the helper process")
	}
	config := t.TempDir()
	if err := os.MkdirAll(filepath.Join(config, "gtk-3.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "gtk-3.0", "settings.ini"), []byte("[Settings]\ngtk-theme-name=Adwaita-dark\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The helper's waits have their guard, after which it fails and says
	// what it waited for.
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestPageColorScheme$", "-test.v")
	cmd.Env = append(os.Environ(), "MYGO_E2E_DARK_THEME=1", "XDG_CONFIG_HOME="+config)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("%v:\n%s", err, out)
	}
}
