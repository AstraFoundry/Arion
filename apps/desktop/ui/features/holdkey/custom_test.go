package holdkey

import (
	"image/png"
	"os"
	"path/filepath"
	platform "runtime"
	"testing"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

type shortcutActions struct {
	runtime.Actions
	state    *runtime.State
	recorded []string
}

func (a *shortcutActions) SetHoldKey(key runtime.HoldKey) { a.state.Preferences.ChooseHoldKey(key) }
func (a *shortcutActions) SetTranslateKey(key runtime.HoldKey) {
	a.state.Preferences.ChooseTranslateKey(platform.GOOS, key)
}
func (a *shortcutActions) RecordShortcut(owner string) {
	a.recorded = append(a.recorded, owner)
	a.state.Hold.Recording = owner
}

func saveRecorder(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	folder := os.Getenv("ARION_UI_SNAPSHOTS")
	if folder == "" {
		return
	}
	if err := os.MkdirAll(folder, 0o700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(folder, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}

func TestShortcutRecorder(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		for _, dark := range []bool{false, true} {
			t.Run(string(locale)+map[bool]string{false: "-light", true: "-dark"}[dark], func(t *testing.T) {
				state := &runtime.State{Preferences: runtime.Preferences{UILanguage: locale, HoldKey: runtime.DefaultHoldKey(platform.GOOS), TranslateKey: "f9"}}
				actions := &shortcutActions{state: state}
				store := runtime.Store{State: state, Actions: actions}
				editor := &PickerState{}
				tt := ui.NewTester(func(c *ui.Context) {
					theme.Apply(c)
					ui.Column(c).Padding(20).Children(func() { Picker(c, store, editor) })
				}, 440, 140)
				tt.SetDark(dark)
				messages := store.Copy()
				click := func(label string) {
					t.Helper()
					if err := tt.Click(label); err != nil {
						t.Fatal(err)
					}
					tt.Frame()
				}
				name := string(locale) + map[bool]string{false: "-light", true: "-dark"}[dark]
				saveRecorder(t, tt, "shortcut-idle-"+name)
				click(messages.Text(copy.HoldKeyEdit))
				tt.Frame()
				if !editor.listening || !tt.Focused(messages.Text(copy.HoldKeyRecording)) {
					t.Fatalf("recorder not focused: %q", tt.Texts())
				}
				saveRecorder(t, tt, "shortcut-recording-"+name)
				// Native capture publishes a whole chord only after every key is up.
				state.Hold.RecordedKey = "ctrl+shift+k"
				state.Hold.RecordedSeq++
				tt.Frame()
				if state.Preferences.HoldKey != "ctrl+shift+k" || state.Preferences.TranslateKey != "f9" || editor.listening || state.Hold.Recording != "" {
					t.Fatalf("not committed: %+v", state)
				}
				saveRecorder(t, tt, "shortcut-saved-"+name)
				click(messages.Text(copy.HoldKeyEdit))
				click(messages.Text(copy.HoldKeyCancel))
				if editor.listening || state.Hold.Recording != "" || state.Preferences.HoldKey != "ctrl+shift+k" {
					t.Fatal("cancel changed assignment or retained capture")
				}
				click(messages.Text(copy.HoldKeyEdit))
				tt.Key(0, ui.KeyEscape)
				if editor.listening {
					t.Fatal("Escape did not cancel")
				}
				click(messages.Text(copy.HoldKeyEdit))
				tt.SetFocused(false)
				if editor.listening || state.Hold.Recording != "" {
					t.Fatal("blur retained global capture")
				}
				tt.SetFocused(true)
				click(messages.Text(copy.HoldKeyEdit))
				editor.Close(store)
				if state.Hold.Recording != "" {
					t.Fatal("close retained global capture")
				}
			})
		}
	}
}

func TestShortcutInfoIconShowsOnlyBehavior(t *testing.T) {
	for _, translation := range []bool{false, true} {
		state := &runtime.State{Preferences: runtime.Preferences{HoldKey: "ctrl+f8", TranslateKey: "alt+f9"}}
		actions := &shortcutActions{state: state}
		store := runtime.Store{State: state, Actions: actions}
		editor := &PickerState{}
		tt := ui.NewTester(func(c *ui.Context) {
			theme.Apply(c)
			if translation {
				TranslateSetting(c, store, editor)
			} else {
				Setting(c, store, editor)
			}
		}, 500, 240)
		messages := store.Copy()
		if tt.HasText(messages.Text(copy.HoldKeySystemHint)) {
			t.Fatal("idle shortcut shows unsolicited guidance")
		}
		tt.Key(0, ui.KeyTab)
		detail := copy.HoldKeyDetail
		if translation {
			detail = copy.TranslateKeyDetail
		}
		if !tt.HasText(messages.Text(detail)) || tt.HasText(messages.Text(copy.HoldKeySystemHint)) {
			t.Fatal("information icon should expose behavior without recording guidance")
		}
	}
}

func TestOnboardingRecorderRejectsDictationConflictAndCanRetry(t *testing.T) {
	state := &runtime.State{Preferences: runtime.Preferences{UILanguage: runtime.LanguageEnglish, HoldKey: "ctrl+space", TranslateKey: "alt+enter"}}
	actions := &shortcutActions{state: state}
	store := runtime.Store{State: state, Actions: actions}
	editor := &PickerState{}
	tt := ui.NewTester(func(c *ui.Context) { theme.Apply(c); FreeTranslatePicker(c, store, editor) }, 500, 200)
	if err := tt.Click(store.Copy().Text(copy.TranslateKeyEdit)); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	state.Hold.RecordedKey = "ctrl+space"
	state.Hold.RecordedSeq++
	tt.Frame()
	if !editor.listening || state.Preferences.TranslateKey != "alt+enter" || editor.error != copy.TranslateKeyConflict {
		t.Fatal("conflict accepted")
	}
	state.Hold.RecordedKey = "shift+f8"
	state.Hold.RecordedSeq++
	tt.Frame()
	if state.Preferences.TranslateKey != "shift+f8" || state.Preferences.HoldKey != "ctrl+space" {
		t.Fatal("retry failed")
	}
}

func TestSettingsRecordersRejectConflictsAndCanRetry(t *testing.T) {
	for _, translate := range []bool{false, true} {
		for _, other := range []runtime.HoldKey{"ctrl+space", runtime.HoldRightCtrl} {
			state := &runtime.State{Preferences: runtime.Preferences{HoldKey: "f8", TranslateKey: other, PreviousTranslateKey: "f10"}}
			label, conflict := copy.HoldKeyEdit, copy.HoldKeyConflict
			if translate {
				state.Preferences.HoldKey, state.Preferences.TranslateKey = other, "f8"
				label, conflict = copy.TranslateKeyEdit, copy.TranslateKeyConflict
			}
			actions := &shortcutActions{state: state}
			store := runtime.Store{State: state, Actions: actions}
			editor := &PickerState{}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				if translate {
					TranslatePicker(c, store, editor)
				} else {
					Picker(c, store, editor)
				}
			}, 500, 200)
			if err := tt.Click(store.Copy().Text(label)); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			before := state.Preferences
			state.Hold.RecordedKey = other
			state.Hold.RecordedSeq++
			tt.Frame()
			if !editor.listening || editor.error != conflict || !tt.HasText(store.Copy().Text(conflict)) {
				t.Fatalf("translation=%v other=%s: missing conflict prompt", translate, other)
			}
			if state.Preferences.HoldKey != before.HoldKey || state.Preferences.TranslateKey != before.TranslateKey || state.Preferences.PreviousTranslateKey != before.PreviousTranslateKey {
				t.Fatal("rejected shortcut changed assignments")
			}
			state.Hold.RecordedKey = "shift+f8"
			state.Hold.RecordedSeq++
			tt.Frame()
			if editor.listening || state.Hold.Recording != "" {
				t.Fatal("valid retry retained capture")
			}
			if translate && (state.Preferences.TranslateKey != "shift+f8" || state.Preferences.HoldKey != other) ||
				!translate && (state.Preferences.HoldKey != "shift+f8" || state.Preferences.TranslateKey != other) {
				t.Fatal("valid retry changed the other shortcut")
			}
		}
	}
}

func TestSettingsRecordersAllowModifierPrefixesInBothDirections(t *testing.T) {
	for _, translate := range []bool{false, true} {
		for _, chord := range []bool{false, true} {
			other, recorded := runtime.HoldRightCommand, runtime.HoldKey("super+1")
			if !chord {
				other, recorded = recorded, other
			}
			state := &runtime.State{Preferences: runtime.Preferences{HoldKey: "f8", TranslateKey: other}}
			label := copy.HoldKeyEdit
			if translate {
				state.Preferences.HoldKey, state.Preferences.TranslateKey = other, "f8"
				label = copy.TranslateKeyEdit
			}
			actions := &shortcutActions{state: state}
			store := runtime.Store{State: state, Actions: actions}
			editor := &PickerState{}
			tt := ui.NewTester(func(c *ui.Context) {
				theme.Apply(c)
				if translate {
					TranslatePicker(c, store, editor)
				} else {
					Picker(c, store, editor)
				}
			}, 500, 200)
			if err := tt.Click(store.Copy().Text(label)); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			state.Hold.RecordedKey = recorded
			state.Hold.RecordedSeq++
			tt.Frame()
			if editor.listening || editor.error != "" || state.Hold.Recording != "" {
				t.Fatal("modifier prefix was not saved")
			}
			if translate && (state.Preferences.TranslateKey != recorded || state.Preferences.HoldKey != other) ||
				!translate && (state.Preferences.HoldKey != recorded || state.Preferences.TranslateKey != other) {
				t.Fatal("saving modifier prefix changed the other assignment")
			}
		}
	}
}

func TestRecordersTransferOwnershipWithoutRestoringTheOldListener(t *testing.T) {
	state := &runtime.State{Preferences: runtime.Preferences{UILanguage: runtime.LanguageEnglish, HoldKey: "f8", TranslateKey: "f9"}}
	actions := &shortcutActions{state: state}
	store := runtime.Store{State: state, Actions: actions}
	dictation, translation := &PickerState{}, &PickerState{}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.Apply(c)
		ui.Column(c).Children(func() { Picker(c, store, dictation); TranslatePicker(c, store, translation) })
	}, 500, 200)
	if err := tt.Click(store.Copy().Text(copy.TranslateKeyEdit)); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click(store.Copy().Text(copy.HoldKeyEdit)); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !dictation.listening || translation.listening || state.Hold.Recording != dictation.owner {
		t.Fatal("old recorder retired the new listener")
	}
	state.Hold.RecordedKey = ""
	state.Hold.RecordInvalid = true
	state.Hold.RecordedSeq++
	tt.Frame()
	if !dictation.listening || dictation.error != copy.HoldKeyNeedsModifier {
		t.Fatal("invalid key was not rejected")
	}
	state.Hold.RecordInvalid = false
	state.Hold.RecordedKey = "ctrl+space"
	state.Hold.RecordedSeq++
	tt.Frame()
	if state.Preferences.HoldKey != "ctrl+space" || state.Hold.Recording != "" {
		t.Fatal("valid retry not committed")
	}
}
