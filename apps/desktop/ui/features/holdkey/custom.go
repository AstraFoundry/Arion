package holdkey

import (
	"fmt"

	holdname "arion.local/desktop/ui/entities/holdkey"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// PickerState owns a field's temporary recording and validation message.
type PickerState struct {
	listening, focus bool
	owner            string
	sequence         uint64
	error            copy.Key
}

// Close discards capture and restores activation, including on navigation.
func (s *PickerState) Close(store runtime.Store) {
	if s.listening && store.State.Hold.Recording == s.owner {
		store.Actions.RecordShortcut("")
	}
	s.listening, s.focus, s.error = false, false, ""
}

// shortcutField is the recorder. A key it refuses is the field's error,
// attached below it; end aligns that callout with a field ending its row.
func shortcutField(c *ui.Context, store runtime.Store, state *PickerState, owner string, current runtime.HoldKey, label copy.Key, set func(runtime.HoldKey), validate func(runtime.HoldKey) copy.Key, end bool) {
	messages, tokens := store.Copy(), theme.Resolve(c.Theme().Dark)
	t := c.Theme()
	closeRecorder := func() { state.Close(store); c.Invalidate() }
	if state.owner == "" {
		state.owner = fmt.Sprintf("%s-%p", owner, state)
	}
	owner = state.owner
	if state.listening && store.State.Hold.Recording != owner {
		state.listening = false
		state.error = ""
	}
	ui.Column(c.Key(owner)).Gap(6).AlignItems(ui.End).Children(func() {
		fill, ink := t.Surface, t.Text
		if state.listening {
			fill, ink = tokens.RecordingSoft, tokens.RecordingInk
		}
		field := kit.ControlFrame(c, ui.Row(c.Key("field"))).Background(fill)
		if state.listening {
			field.Focusable().FocusRing(false).Label(messages.Text(copy.HoldKeyRecording)).HandleInput(func(ev ui.InputEvent) bool {
				switch ev.Kind {
				case ui.InputKeyDown:
					if ev.Key == ui.KeyEscape && ev.Mods == 0 {
						closeRecorder()
					}
					return true
				case ui.InputKeyUp, ui.InputText, ui.InputCompose, ui.InputCommand:
					return true
				}
				return false
			})
			if state.focus {
				field.Focus()
				state.focus = false
			}
		}
		controlFocused := false
		field.Children(func() {
			text := holdname.Name(messages, current)
			icon, actionLabel := icons.Pencil, messages.Text(label)
			if state.listening {
				text = messages.Text(copy.HoldKeyRecording)
				icon, actionLabel = icons.X, messages.Text(copy.HoldKeyCancel)
			}
			ui.Text(c, text).SingleLine().Grow(1).TextColor(ink)
			button := kit.IconButton(c.Key("edit"), tokens, icon, actionLabel).Size(t.Space(4), t.Space(4)).Radius(theme.RadiusSM).TextColor(ink).Background(ui.Transparent)
			controlFocused = button.Focused()
			if button.Hovered() {
				button.Background(ink.Alpha(0.08))
			}
			if button.Clicked() {
				if state.listening {
					closeRecorder()
				} else {
					state.listening, state.focus, state.error = true, true, ""
					state.sequence = store.State.Hold.RecordedSeq
					store.Actions.RecordShortcut(owner)
					c.Invalidate()
				}
			}
		})
		if state.listening && !state.focus {
			if !field.Focused() && !controlFocused {
				closeRecorder()
			} else if captured := store.State.Hold; captured.RecordedSeq != state.sequence {
				state.sequence = captured.RecordedSeq
				switch {
				case captured.RecordCancelled:
					closeRecorder()
				case captured.RecordInvalid:
					state.error = copy.HoldKeyNeedsModifier
				case validate(captured.RecordedKey) != "":
					state.error = validate(captured.RecordedKey)
					store.Actions.RecordShortcut("")
					store.Actions.RecordShortcut(owner)
				case captured.RecordedKey != "":
					set(captured.RecordedKey)
					closeRecorder()
				}
				if state.error != "" {
					c.Announce(messages.Text(state.error))
				}
			}
		}
		if state.error != "" {
			field.Border(1, tokens.Danger)
			kit.FieldError(c, tokens, field, messages.Text(state.error), end)
		}
	})
}
