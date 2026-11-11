//go:build js

package web

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"syscall/js"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestKeyOf(t *testing.T) {
	for _, tc := range []struct {
		key, code string
		want      platform.Key
	}{
		{"a", "KeyA", platform.KeyA},
		{"A", "KeyA", platform.KeyA},
		// Letters follow the layout: the key typing a z on a German
		// keyboard is Z, where QWERTY has Y.
		{"z", "KeyY", platform.KeyZ},
		// Shifted and composed characters go by the key.
		{"!", "Digit1", platform.Key1},
		{"å", "KeyA", platform.KeyA},
		{"ß", "Minus", platform.KeyMinus},
		{" ", "Space", platform.KeySpace},
		{"Enter", "Enter", platform.KeyEnter},
		{"ArrowLeft", "ArrowLeft", platform.KeyLeft},
		{"F12", "F12", platform.KeyF12},
		{"BrowserBack", "BrowserBack", platform.KeyBack},
		{"Unidentified", "", platform.KeyUnknown},
	} {
		if got := keyOf(tc.key, tc.code); got != tc.want {
			t.Errorf("keyOf(%q, %q) = %v, want %v", tc.key, tc.code, got, tc.want)
		}
	}
}

func TestCodeKey(t *testing.T) {
	// The key by position, whatever the layout types: ы is on S.
	for code, want := range map[string]platform.Key{
		"KeyS": platform.KeyS, "Digit1": platform.Key1, "Semicolon": platform.KeySemicolon,
		"IntlBackslash": platform.KeyUnknown, "": platform.KeyUnknown,
	} {
		if got := codeKey(code); got != want {
			t.Errorf("codeKey(%q) = %v, want %v", code, got, want)
		}
	}
}

func TestAccelText(t *testing.T) {
	for _, tc := range []struct {
		accel string
		mac   bool
		want  string
	}{
		{"CmdOrCtrl+C", true, "⌘C"},
		{"CmdOrCtrl+C", false, "Ctrl+C"},
		{"CmdOrCtrl+Shift+z", true, "⌘⇧Z"},
		{"Alt+F4", false, "Alt+F4"},
		{"", true, ""},
	} {
		if got := accelText(tc.accel, tc.mac); got != tc.want {
			t.Errorf("accelText(%q, %v) = %q, want %q", tc.accel, tc.mac, got, tc.want)
		}
	}
}

func TestButtonOf(t *testing.T) {
	// DOM: 0 primary, 1 middle, 2 secondary; MyGo: 0 primary, 1
	// secondary, 2 middle.
	for dom, want := range []int{0, 2, 1} {
		if got := buttonOf(dom); got != want {
			t.Errorf("buttonOf(%d) = %d, want %d", dom, got, want)
		}
	}
}

func TestUnpremul(t *testing.T) {
	for _, tc := range []struct{ c, a, want byte }{{128, 128, 255}, {64, 128, 128}, {0, 10, 0}, {200, 255, 200}} {
		if got := unpremul(tc.c, tc.a); got != tc.want {
			t.Errorf("unpremul(%d, %d) = %d, want %d", tc.c, tc.a, got, tc.want)
		}
	}
}

// TestDroppedFiles reads files dropped through os, as apps do with the
// paths of FileDrop, and leaves other paths to the fs object (Node's, in
// tests).
func TestDroppedFiles(t *testing.T) {
	installFiles()
	paths := drops.add([]string{"notes.txt", "notes.txt", "a/b.png"}, [][]byte{[]byte("hello"), []byte("again"), {1, 2, 3}})
	dir := path.Dir(paths[0])
	want := []string{dir + "/notes.txt", dir + "/2-notes.txt", dir + "/a_b.png"}
	if !slices.Equal(paths, want) {
		t.Fatalf("paths %v, want %v", paths, want)
	}
	for i, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil || len(b) == 0 {
			t.Fatalf("ReadFile(%s) = %q, %v", p, b, err)
		}
		if i == 0 && string(b) != "hello" {
			t.Errorf("ReadFile(%s) = %q", p, b)
		}
	}
	if st, err := os.Stat(paths[2]); err != nil || st.Size() != 3 || st.IsDir() {
		t.Errorf("Stat(%s) = %v, %v", paths[2], st, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 3 {
		t.Fatalf("ReadDir(%s) = %v, %v", dir, entries, err)
	}
	f, err := os.Open(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 3)
	if n, err := f.ReadAt(buf, 2); err != nil || string(buf[:n]) != "llo" {
		t.Errorf("ReadAt = %q, %v", buf[:n], err)
	}
	f.Close()
	if err := os.WriteFile(paths[0], []byte("x"), 0o644); err == nil {
		t.Error("a file dropped was written")
	}
	if _, err := os.ReadFile(dir + "/missing"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a missing file: %v", err)
	}
	// Other paths are the fs object's.
	other := filepath.Join(t.TempDir(), "x")
	if err := os.WriteFile(other, []byte("node"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(other); err != nil || string(b) != "node" {
		t.Errorf("ReadFile(%s) = %q, %v", other, b, err)
	}
	// The oldest drops go.
	for range keptDrops {
		drops.add([]string{"x"}, [][]byte{{0}})
	}
	if _, err := os.Stat(paths[0]); err == nil {
		t.Error("the files of the oldest drop stay")
	}
}

// TestWebViewAvailable: browsers have no system webview, so the call says
// so, and an app opens a window of native UI instead of one showing a page
// it could not show.
func TestWebViewAvailable(t *testing.T) {
	if New().WebViewAvailable() {
		t.Error("WebViewAvailable = true in a browser, which has no webview")
	}
}

func TestChord(t *testing.T) {
	// A DOM pointermove with button (DOM numbering) going down or up while
	// another is held, and buttons held after it.
	for _, tc := range []struct {
		button, buttons int
		kind            platform.SurfaceEventKind
		want            int
		ok              bool
	}{
		{-1, 1, 0, 0, false},                      // a move
		{2, 1 | 2, platform.PointerDown, 1, true}, // the secondary goes down while the primary is held
		{2, 1, platform.PointerUp, 1, true},       // and up
		{1, 2 | 4, platform.PointerDown, 2, true}, // the middle while the secondary is held
		{0, 4, platform.PointerUp, 0, true},       // the primary up while the middle is held
		{3, 1 | 8, 0, 0, false},                   // a side button
	} {
		kind, b, ok := chord(tc.button, tc.buttons)
		if ok != tc.ok || ok && (kind != tc.kind || b != tc.want) {
			t.Errorf("chord(%d, %d) = %v, %d, %v; want %v, %d, %v", tc.button, tc.buttons, kind, b, ok, tc.kind, tc.want, tc.ok)
		}
	}
}

func TestDevice(t *testing.T) {
	// A PointerEvent's pointerType and pressure, which browsers give a
	// mouse too, and an event of a browser without them.
	for _, tc := range []struct {
		props    map[string]any
		want     platform.PointerKind
		pressure float64
	}{
		{map[string]any{"pointerType": "mouse", "pressure": 0.5}, platform.PointerMouse, 0.5},
		{map[string]any{"pointerType": "pen", "pressure": 0.75}, platform.PointerPen, 0.75},
		{map[string]any{"pointerType": "touch", "pressure": 0}, platform.PointerTouch, 0},
		{map[string]any{}, platform.PointerMouse, 0},
	} {
		e := js.Global().Get("Object").New()
		for k, v := range tc.props {
			e.Set(k, v)
		}
		if k, p := device(e); k != tc.want || p != tc.pressure {
			t.Errorf("device(%v) = %d, %v; want %d, %v", tc.props, k, p, tc.want, tc.pressure)
		}
	}
}

func TestCursorNames(t *testing.T) {
	// Every cursor has its CSS name.
	for c := platform.CursorDefault; c <= platform.CursorNoDrop; c++ {
		if int(c) >= len(cursors) || cursors[c] == "" {
			t.Errorf("cursor %d has no CSS name", c)
		}
	}
	if cursors[platform.CursorProgress] != "progress" || cursors[platform.CursorNoDrop] != "no-drop" {
		t.Errorf("progress is %q, no-drop %q", cursors[platform.CursorProgress], cursors[platform.CursorNoDrop])
	}
}
