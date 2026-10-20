// Package shortcut records a new global dictation shortcut.
package shortcut

import (
	platform "runtime"
	"strings"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Recorder is the recorder's own state, kept by the page.
type Recorder struct {
	Listening bool
	Error     copy.Key
}

var keyNames = map[ui.Key]string{
	ui.KeySpace: "Space", ui.KeyEnter: "Enter", ui.KeyTab: "Tab", ui.KeyBackspace: "Backspace",
	ui.KeyLeft: "Left", ui.KeyRight: "Right", ui.KeyUp: "Up", ui.KeyDown: "Down",
	ui.KeyMinus: "-", ui.KeyEqual: "=", ui.KeyComma: ",", ui.KeyPeriod: ".", ui.KeySlash: "/",
	ui.KeySemicolon: ";", ui.KeyQuote: "'", ui.KeyBracketLeft: "[", ui.KeyBracketRight: "]", ui.KeyBackslash: "\\", ui.KeyBackquote: "`",
	ui.KeyF1: "F1", ui.KeyF2: "F2", ui.KeyF3: "F3", ui.KeyF4: "F4", ui.KeyF5: "F5", ui.KeyF6: "F6",
	ui.KeyF7: "F7", ui.KeyF8: "F8", ui.KeyF9: "F9", ui.KeyF10: "F10", ui.KeyF11: "F11", ui.KeyF12: "F12",
}

func init() {
	for i, k := range []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9} {
		keyNames[k] = string(rune('0' + i))
	}
	for i, k := range []ui.Key{ui.KeyA, ui.KeyB, ui.KeyC, ui.KeyD, ui.KeyE, ui.KeyF, ui.KeyG, ui.KeyH, ui.KeyI, ui.KeyJ, ui.KeyK, ui.KeyL, ui.KeyM,
		ui.KeyN, ui.KeyO, ui.KeyP, ui.KeyQ, ui.KeyR, ui.KeyS, ui.KeyT, ui.KeyU, ui.KeyV, ui.KeyW, ui.KeyX, ui.KeyY, ui.KeyZ} {
		keyNames[k] = string(rune('A' + i))
	}
}

// Accelerator converts a key press to MyGo's accelerator syntax. A shortcut
// needs a modifier unless it is a function key, so typing is never stolen.
func Accelerator(mods ui.Modifiers, key ui.Key) (string, bool) {
	name, ok := keyNames[key]
	if !ok {
		return "", false
	}
	var parts []string
	if mods&ui.Super != 0 {
		parts = append(parts, "Cmd")
	}
	if mods&ui.Ctrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if mods&ui.Alt != 0 {
		parts = append(parts, "Alt")
	}
	if mods&ui.Shift != 0 {
		parts = append(parts, "Shift")
	}
	function := strings.HasPrefix(name, "F") && len(name) > 1
	if len(parts) == 0 && !function {
		return "", false
	}
	return strings.Join(append(parts, name), "+"), true
}

// Start begins listening for a new shortcut, releasing the current one so
// pressing it is recorded instead of starting a dictation.
func Start(store runtime.Store, recorder *Recorder) {
	recorder.Listening, recorder.Error = true, ""
	store.Actions.SuspendShortcut(true)
}

// Capture is the focused box that records the next key combination. Esc
// cancels and restores the current shortcut.
func Capture(c *ui.Context, store runtime.Store, tokens theme.Tokens, recorder *Recorder) ui.Element {
	messages := store.Copy()
	capture := ui.Row(c.Key("capture")).Height(28).Padding(0, 10).AlignItems(ui.Center).Radius(theme.RadiusControl).
		Border(1, tokens.AccentInk).Background(tokens.AccentSoft).Focusable().AutoFocus().FocusRing(false)
	capture.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind != ui.InputKeyDown || ev.Repeat {
			return false
		}
		if ev.Key == ui.KeyEscape && ev.Mods == 0 {
			recorder.Listening, recorder.Error = false, ""
			store.Actions.SuspendShortcut(false)
			return true
		}
		accelerator, ok := Accelerator(ev.Mods, ev.Key)
		if !ok {
			recorder.Error = copy.ShortcutNeedsModifier
			if platform.GOOS == "windows" {
				recorder.Error = copy.ShortcutNeedsModifierWindows
			}
			return true
		}
		if err := store.Actions.SetShortcut(accelerator); err != nil {
			recorder.Error = copy.ShortcutTaken
			return true
		}
		recorder.Listening, recorder.Error = false, ""
		return true
	})
	capture.Children(func() {
		ui.Text(c, messages.Text(copy.ShortcutRecording)).FontSize(theme.Caption).TextColor(tokens.Text)
	})
	if !capture.FocusWithin() && !capture.Focused() {
		capture.Focus()
	}
	return capture
}

// Error explains why the last recorded combination was refused.
func Error(c *ui.Context, store runtime.Store, tokens theme.Tokens, recorder *Recorder) {
	if recorder.Error != "" {
		ui.Text(c.Key("error"), store.Copy().Text(recorder.Error)).FontSize(theme.Caption).TextColor(tokens.Danger).
			Transition(ui.ElementTransition{Enter: &ui.Motion{Y: -4}})
	}
}

// Field shows the current shortcut and records a new one on request.
func Field(c *ui.Context, store runtime.Store, tokens theme.Tokens, recorder *Recorder) {
	messages := store.Copy()
	ui.Column(c).Gap(6).AlignItems(ui.End).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if recorder.Listening {
				Capture(c, store, tokens, recorder)
			} else {
				kit.Keycaps(c, tokens, store.State.Shortcut.Accelerator, false)
				if kit.Button(c.Key("change"), tokens, kit.Outline, messages.Text(copy.ShortcutRecord), true).Clicked() {
					Start(store, recorder)
				}
			}
		})
		Error(c, store, tokens, recorder)
	})
}
