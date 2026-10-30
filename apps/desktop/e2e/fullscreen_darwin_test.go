//go:build arion_e2e && darwin

package e2e

import (
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	uiapp "arion.local/desktop/ui/app"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

type editorState struct {
	ClipboardOwned bool `json:"clipboardOwned"`
	Web            struct {
		Text    string `json:"text"`
		Edits   int    `json:"edits"`
		Focused bool   `json:"focused"`
	} `json:"web"`
	PID int `json:"pid"`
	A   struct {
		Text  string `json:"text"`
		Edits int    `json:"edits"`
	} `json:"A"`
	B struct {
		Text  string `json:"text"`
		Edits int    `json:"edits"`
	} `json:"B"`
	Active        bool   `json:"active"`
	Fullscreen    bool   `json:"fullscreen"`
	Transitioning bool   `json:"transitioning"`
	Focused       string `json:"focused"`
}

// DSK-A13 uses a separate, owned app: a fullscreen window in Arion's own
// process would not test joining another application's fullscreen Space.
func TestOverlayInForeignFullscreenSpace(t *testing.T) {
	request := startEditorFixture(t)
	state := request("focus", map[string]any{"target": "A"})
	request("fullscreen", map[string]any{"enabled": true})
	defer func() {
		request("fullscreen", map[string]any{"enabled": false})
		if !within(5*time.Second, func() bool {
			state := request("state", nil)
			return !state.Fullscreen && !state.Transitioning
		}) {
			t.Error("owned editor did not leave its fullscreen Space during cleanup")
		}
	}()
	eventually(t, "foreign editor entered fullscreen", func() bool {
		state = request("state", nil)
		return state.Fullscreen && !state.Transitioning && state.Active && state.Focused == "A"
	})
	var overlay *mygo.Window
	eventually(t, "production overlay created", func() bool { overlay = app.Windows()["overlay"]; return overlay != nil })
	defer overlay.Hide()
	app.Coordinator().Start()
	eventually(t, "production session showed overlay", overlay.IsVisible)
	var observation overlayObservation
	visible := func() bool {
		mygo.RunOnMain(func() { observation = observeOverlay(overlay.NativeHandle(), state.PID) })
		return observation.OnActiveSpace && observation.OnScreen && observation.AboveFixture && observation.OverlapsFixture && observation.FrontmostPID == state.PID
	}
	if !within(5*time.Second, visible) {
		t.Fatalf("overlay absent above foreign fullscreen editor: %+v", observation)
	}
	// Exercise both an already visible window and a hide/show in that Space.
	overlay.ShowInactive()
	if !visible() || overlay.IsFocused() {
		t.Fatalf("overlay changed fullscreen focus or visibility: %+v", observation)
	}
	overlay.Hide()
	overlay.ShowInactive()
	if !within(5*time.Second, visible) || overlay.IsFocused() {
		t.Fatalf("overlay failed to reappear without taking focus: %+v", observation)
	}
	state = request("state", nil)
	if !state.Fullscreen || !state.Active || state.Focused != "A" {
		t.Fatalf("editor focus changed: %+v", state)
	}
	capture(t, overlay, "overlay-fullscreen", uiapp.OverlayWidth, uiapp.OverlayHeight)
	t.Logf("foreign fullscreen compositor observation: %+v; editor: %+v", observation, state)
	if output := os.Getenv("ARION_E2E_OUTPUT"); output != "" {
		data, err := json.MarshalIndent(observation, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(output, "overlay-fullscreen.json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// startEditorFixture compiles and launches the owned AppKit editor and
// returns a function sending it one command; the fixture quits at cleanup.
func startEditorFixture(t *testing.T) func(op string, values map[string]any) editorState {
	t.Helper()
	directory := t.TempDir()
	bundle := filepath.Join(directory, "ArionEditorFixture.app", "Contents")
	if err := os.MkdirAll(filepath.Join(bundle, "MacOS"), 0o700); err != nil {
		t.Fatal(err)
	}
	plist, err := os.ReadFile("fixtures/editor/Info.plist")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "Info.plist"), plist, 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(bundle, "MacOS", "ArionEditorFixture")
	if out, err := exec.Command("clang", "-fobjc-arc", "-framework", "Cocoa", "-framework", "WebKit", "fixtures/editor/main.m", "-o", binary).CombinedOutput(); err != nil {
		t.Fatalf("compile editor fixture: %v\n%s", err, out)
	}
	// macOS Unix socket paths must fit sockaddr_un.sun_path.
	socketDirectory, err := os.MkdirTemp("/tmp", "arion-fs-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(socketDirectory) })
	socket := filepath.Join(socketDirectory, "editor.sock")
	command := exec.Command(binary)
	command.Env = append(os.Environ(), "ARION_FIXTURE_SOCKET="+socket)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = command.Process.Kill(); _ = command.Wait() })
	var connection net.Conn
	eventually(t, "fixture socket ready", func() bool {
		connection, err = net.DialTimeout("unix", socket, time.Second)
		return err == nil
	})
	t.Cleanup(func() { _ = connection.Close() })
	encoder, decoder := json.NewEncoder(connection), json.NewDecoder(connection)
	request := func(op string, values map[string]any) editorState {
		t.Helper()
		if values == nil {
			values = map[string]any{}
		}
		values["op"] = op
		if err := connection.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			t.Fatal(err)
		}
		if err := encoder.Encode(values); err != nil {
			t.Fatal(err)
		}
		var state editorState
		if err := decoder.Decode(&state); err != nil {
			t.Fatal(err)
		}
		return state
	}
	return request
}

// CLP-A07: the actual picker must join a foreign fullscreen Space without
// activating Arion or requiring an Accessibility grant just to browse history.
func TestClipboardInForeignFullscreenSpace(t *testing.T) {
	request := startEditorFixture(t)
	request("focus", map[string]any{"target": "A"})
	request("fullscreen", map[string]any{"enabled": true})
	defer func() {
		request("fullscreen", map[string]any{"enabled": false})
		eventually(t, "owned editor left fullscreen", func() bool { s := request("state", nil); return !s.Fullscreen && !s.Transitioning })
	}()
	var state editorState
	eventually(t, "owned editor entered fullscreen", func() bool {
		state = request("state", nil)
		return state.Fullscreen && !state.Transitioning && state.Active
	})
	var original arion.Clipboard
	app.Update(func(s *arion.State) {
		original = s.Clipboard
		p := arion.ClipboardText("Owned clipboard history fixture")
		s.Clipboard.Items = []arion.Clip{{ID: p.ID(), Payload: p}}
	})
	defer func() {
		mygo.RunOnMain(func() { app.Store().Actions.CloseClipboard() })
		app.Update(func(s *arion.State) { s.Clipboard = original })
	}()
	for cycle := 0; cycle < 2; cycle++ {
		mygo.RunOnMain(func() { app.Store().Actions.OpenClipboard() })
		var picker *mygo.Window
		eventually(t, "picker appeared", func() bool { picker = app.Windows()["clipboard"]; return picker != nil && picker.IsVisible() })
		var observation overlayObservation
		eventually(t, "picker joined foreign fullscreen space", func() bool {
			mygo.RunOnMain(func() { observation = observeOverlay(picker.NativeHandle(), state.PID) })
			return observation.OnActiveSpace && observation.OnScreen && observation.AboveFixture && observation.OverlapsFixture && observation.FrontmostPID == state.PID
		})
		if !picker.IsFocused() {
			t.Fatal("picker could not receive keyboard input without activating Arion")
		}
		if cycle == 0 {
			capture(t, picker, "clipboard-fullscreen", uiapp.ClipboardWidth, 100)
		}
		mygo.RunOnMain(func() { app.Store().Actions.CloseClipboard() })
		eventually(t, "picker closed", func() bool { return app.Windows()["clipboard"] == nil })
		state = request("state", nil)
		if !state.Fullscreen || !state.Active || state.Focused != "A" {
			t.Fatalf("picker changed editor space/focus: %+v", state)
		}
	}
}
