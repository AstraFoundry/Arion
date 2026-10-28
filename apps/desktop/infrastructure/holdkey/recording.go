package holdkey

import "arion.local/core/application"

// RecordKeys selects capture instead of activation in the native adapters.
// It is never a persisted shortcut.
const RecordKeys application.HoldKey = "record_shortcut"

// A recording commits only after every constituent is released, so restoring
// the activation listener cannot interpret a held key's repeat as dictation.
type recording struct {
	platform      string
	keys          map[int64]application.HoldKey
	down          map[int64]bool
	solo, chord   application.HoldKey
	invalid, done bool
	arming        bool
}

func newRecording(platform string) *recording {
	r := &recording{platform: platform, keys: map[int64]application.HoldKey{}, down: map[int64]bool{}}
	keys := append(application.HoldKeysFor(platform), application.ShortcutKeysFor(platform)...)
	for _, key := range keys {
		spec := keySpecs[key]
		code := spec.mac
		if platform == "windows" {
			code = int64(spec.win)
		}
		r.keys[code] = key
	}
	return r
}

func (r *recording) event(code int64, pressed bool, mods application.KeyModifiers, repeat bool) (application.HoldKey, Event, bool) {
	if r.done || repeat || pressed && r.down[code] {
		return "", 0, false
	}
	r.down[code] = pressed
	escape := int64(0x35)
	if r.platform == "windows" {
		escape = 0x1b
	}
	if pressed && code == escape {
		r.done = true
		return "", RecordCancelled, true
	}
	// Discard keys held before the recorder was installed. In particular,
	// sampled AltGr's synthetic Ctrl cannot become a recorded modifier.
	if r.arming {
		for _, held := range r.down {
			if held {
				return "", 0, false
			}
		}
		if mods != 0 {
			return "", 0, false
		}
		r.arming = false
		return "", 0, false
	}
	if pressed && r.chord == "" {
		key, known := r.keys[code]
		switch {
		case !known:
			r.invalid = true
			r.solo = ""
		case keySpecs[key].modifier != 0 || key == application.HoldFn:
			count := 0
			for _, held := range r.down {
				if held {
					count++
				}
			}
			if count == 1 && mods == keySpecs[key].modifier {
				r.solo = key
			} else {
				r.solo = ""
			}
		default:
			r.solo = ""
			chord, valid := application.HoldShortcut(r.platform, mods, key)
			if valid {
				r.chord = chord
			} else {
				r.invalid = true
			}
		}
	}
	for _, held := range r.down {
		if held {
			return "", 0, false
		}
	}
	if mods != 0 {
		return "", 0, false
	}
	if r.invalid {
		r.solo, r.chord, r.invalid = "", "", false
		return "", RecordInvalid, true
	}
	key := r.chord
	if key == "" {
		key = r.solo
	}
	if key == "" {
		return "", 0, false
	}
	r.done = true
	return key, Recorded, true
}
