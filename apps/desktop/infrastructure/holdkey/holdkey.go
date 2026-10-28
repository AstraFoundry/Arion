// Package holdkey translates native keyboard events into dictation gestures.
// Interpretation is pure and tested for both platforms on every host.
package holdkey

import (
	"strconv"

	"arion.local/core/application"
)

type Event int

const (
	Down Event = iota
	Up
	Other
	Recorded
	RecordInvalid
	RecordCancelled
	// Pending defers a standalone modifier that can prefix a configured chord.
	Pending
)
const (
	MacKeyDown      = 10
	MacKeyUp        = 11
	MacFlagsChanged = 12
	WinKeyDown      = 0x0100
	WinKeyUp        = 0x0101
	WinSysKeyDown   = 0x0104
	WinSysKeyUp     = 0x0105
	vkLControl      = 0xA2
	altGrScan       = 0x21D
)

type keySpec struct {
	mac      int64
	win      uint32
	flag     uint64
	modifier application.KeyModifiers
}

// Native codes identify physical keys; layout-dependent typed characters
// do not affect shortcut identity.
var keySpecs = map[application.HoldKey]keySpec{
	application.HoldFn:           {mac: 0x3f, flag: 0x800000},
	application.HoldRightOption:  {mac: 0x3d, flag: 0x40, modifier: application.KeyAlt},
	application.HoldLeftOption:   {mac: 0x3a, flag: 0x20, modifier: application.KeyAlt},
	application.HoldRightCommand: {mac: 0x36, flag: 0x10, modifier: application.KeySuper},
	application.HoldLeftCommand:  {mac: 0x37, flag: 0x08, modifier: application.KeySuper},
	application.HoldRightCtrl:    {mac: 0x3e, win: 0xa3, flag: 0x2000, modifier: application.KeyCtrl},
	application.HoldLeftCtrl:     {mac: 0x3b, win: 0xa2, flag: 0x01, modifier: application.KeyCtrl},
	application.HoldRightShift:   {mac: 0x3c, win: 0xa1, flag: 0x04, modifier: application.KeyShift},
	application.HoldLeftShift:    {mac: 0x38, win: 0xa0, flag: 0x02, modifier: application.KeyShift},
	application.HoldRightAlt:     {win: 0xa5, modifier: application.KeyAlt},
	application.HoldLeftAlt:      {win: 0xa4, modifier: application.KeyAlt},
	application.HoldMenu:         {win: 0x5d},
	"minus":                      {mac: 0x1b, win: 0xbd}, "equal": {mac: 0x18, win: 0xbb}, "comma": {mac: 0x2b, win: 0xbc}, "period": {mac: 0x2f, win: 0xbe}, "slash": {mac: 0x2c, win: 0xbf},
	"semicolon": {mac: 0x29, win: 0xba}, "quote": {mac: 0x27, win: 0xde}, "bracket_left": {mac: 0x21, win: 0xdb}, "bracket_right": {mac: 0x1e, win: 0xdd}, "backslash": {mac: 0x2a, win: 0xdc}, "grave": {mac: 0x32, win: 0xc0},
	"space": {mac: 0x31, win: 0x20}, "enter": {mac: 0x24, win: 0x0d}, "tab": {mac: 0x30, win: 0x09},
	"backspace": {mac: 0x33, win: 0x08}, "delete": {mac: 0x75, win: 0x2e}, "home": {mac: 0x73, win: 0x24}, "end": {mac: 0x77, win: 0x23},
	"page_up": {mac: 0x74, win: 0x21}, "page_down": {mac: 0x79, win: 0x22},
	"left": {mac: 0x7b, win: 0x25}, "right": {mac: 0x7c, win: 0x27}, "up": {mac: 0x7e, win: 0x26}, "down": {mac: 0x7d, win: 0x28},
}

func init() {
	for i, code := range []int64{0x00, 0x0b, 0x08, 0x02, 0x0e, 0x03, 0x05, 0x04, 0x22, 0x26, 0x28, 0x25, 0x2e, 0x2d, 0x1f, 0x23, 0x0c, 0x0f, 0x01, 0x11, 0x20, 0x09, 0x0d, 0x07, 0x10, 0x06} {
		keySpecs[application.HoldKey(string(rune('a'+i)))] = keySpec{mac: code, win: uint32(0x41 + i)}
	}
	for i, code := range []int64{0x1d, 0x12, 0x13, 0x14, 0x15, 0x17, 0x16, 0x1a, 0x1c, 0x19} {
		keySpecs[application.HoldKey(string(rune('0'+i)))] = keySpec{mac: code, win: uint32(0x30 + i)}
	}
	macFunctions := []int64{0x7a, 0x78, 0x63, 0x76, 0x60, 0x61, 0x62, 0x64, 0x65, 0x6d, 0x67, 0x6f, 0x69, 0x6b, 0x71, 0x6a, 0x40, 0x4f, 0x50, 0x5a}
	for n := 1; n <= 24; n++ {
		spec := keySpec{mac: -1, win: uint32(0x70 + n - 1)}
		if n <= len(macFunctions) {
			spec.mac = macFunctions[n-1]
		}
		keySpecs[application.HoldKey("f"+strconv.Itoa(n))] = spec
	}
}

type target struct {
	key      application.HoldKey
	code     int64
	mods     application.KeyModifiers
	deferred bool
}

type matcher struct {
	targets   []target
	down      map[int64]bool
	active    *target
	pending   bool
	recording *recording
}

func newMatcher(platform string, keys []application.HoldKey) (*matcher, bool) {
	m := &matcher{down: map[int64]bool{}}
	if len(keys) == 1 && keys[0] == RecordKeys {
		m.recording = newRecording(platform)
		return m, true
	}
	seen := map[application.HoldKey]bool{}
	for _, key := range keys {
		if key == "" || key == application.HoldOff || seen[key] {
			continue
		}
		s, ok := application.ParseHoldShortcut(platform, key)
		if !ok {
			return nil, false
		}
		spec, ok := keySpecs[s.Key]
		if !ok {
			return nil, false
		}
		code := spec.mac
		if platform == "windows" {
			code = int64(spec.win)
		}
		m.targets = append(m.targets, target{key: key, code: code, mods: s.Modifiers | spec.modifier})
		seen[key] = true
	}
	for i := range m.targets {
		t := &m.targets[i]
		shortcut, _ := application.ParseHoldShortcut(platform, t.key)
		family := keySpecs[shortcut.Key].modifier
		if shortcut.Modifiers != 0 || family == 0 {
			continue
		}
		for _, chord := range m.targets {
			if chord.code != t.code && chord.mods&family != 0 {
				parsed, _ := application.ParseHoldShortcut(platform, chord.key)
				if parsed.Modifiers != 0 {
					t.deferred = true
				}
			}
		}
	}
	return m, len(m.targets) > 0
}

func (m *matcher) event(code int64, pressed bool, mods application.KeyModifiers, repeat bool) (application.HoldKey, Event, bool) {
	if m.recording != nil {
		return m.recording.event(code, pressed, mods, repeat)
	}
	if repeat {
		return "", 0, false
	}
	wasDown := m.down[code]
	m.down[code] = pressed
	if wasDown == pressed {
		return "", 0, false
	}
	if active := m.active; active != nil {
		if m.pending && pressed {
			// Resolve a full chord before the pending standalone modifier.
			for i := range m.targets {
				t := &m.targets[i]
				if !t.deferred && t.code == code && t.mods == mods {
					m.active, m.pending = t, false
					return t.key, Down, true
				}
			}
		}
		if (!pressed && code == active.code) || mods&active.mods != active.mods {
			m.active, m.pending = nil, false
			return active.key, Up, true
		}
		if pressed {
			return active.key, Other, true
		}
		return "", 0, false
	}
	if pressed {
		for i := range m.targets {
			t := &m.targets[i]
			if t.code == code && t.mods == mods {
				m.active, m.pending = t, t.deferred
				if t.deferred {
					return t.key, Pending, true
				}
				return t.key, Down, true
			}
		}
	}
	return "", 0, false
}

// Mac consumes Quartz key-down, key-up and side-specific modifier changes.
type Mac struct{ matcher *matcher }

func NewMac(keys ...application.HoldKey) (*Mac, bool) {
	m, ok := newMatcher("darwin", keys)
	return &Mac{m}, ok
}
func (m *Mac) InterpretKey(eventType int, keycode int64, flags uint64, autorepeat bool) (application.HoldKey, Event, bool) {
	var mods application.KeyModifiers
	for _, s := range keySpecs {
		if s.modifier != 0 && s.flag != 0 && flags&s.flag != 0 {
			mods |= s.modifier
		}
	}
	// Shared flags also cover modifiers already held when the listener starts.
	for _, f := range []struct {
		flag uint64
		mod  application.KeyModifiers
	}{{0x40000, application.KeyCtrl}, {0x80000, application.KeyAlt}, {0x20000, application.KeyShift}, {0x100000, application.KeySuper}} {
		if flags&f.flag != 0 {
			mods |= f.mod
		}
	}
	pressed := eventType == MacKeyDown
	switch eventType {
	case MacFlagsChanged:
		found := false
		for _, s := range keySpecs {
			if s.flag != 0 && s.mac == keycode {
				pressed = flags&s.flag != 0
				// Shared modifier flags can be present without device-side bits.
				// They still identify a press when neither side is marked.
				for _, family := range []struct {
					mod           application.KeyModifiers
					sides, shared uint64
				}{
					{application.KeyCtrl, 0x2001, 0x40000}, {application.KeyAlt, 0x60, 0x80000}, {application.KeyShift, 0x06, 0x20000}, {application.KeySuper, 0x18, 0x100000},
				} {
					if s.modifier == family.mod && flags&family.sides == 0 && flags&family.shared != 0 {
						pressed = true
					}
				}
				found = true
				break
			}
		}
		if !found {
			if m.matcher.recording != nil && !m.matcher.recording.done {
				return "", RecordInvalid, true
			}
			return "", 0, false
		}
	case MacKeyDown, MacKeyUp:
	default:
		return "", 0, false
	}
	return m.matcher.event(keycode, pressed, mods, autorepeat)
}

// Windows consumes low-level hook events, retaining both modifier sides.
// The fake left Ctrl emitted by AltGr is excluded from shortcut matching.
type Windows struct {
	matcher   *matcher
	modifiers map[uint32]bool
}

func NewWindows(keys ...application.HoldKey) (*Windows, bool) {
	m, ok := newMatcher("windows", keys)
	return &Windows{matcher: m, modifiers: map[uint32]bool{}}, ok
}

// SetModifier initializes a modifier held before hook registration.
func (w *Windows) SetModifier(vk uint32, down bool) { w.SetKeyDown(vk, down) }

// KeyCodes includes the triggers and modifiers whose initial state matters.
func (w *Windows) KeyCodes() []uint32 {
	codes := []uint32{0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0x5b, 0x5c}
	for _, target := range w.matcher.targets {
		codes = append(codes, uint32(target.code))
	}
	if r := w.matcher.recording; r != nil {
		for code := range r.keys {
			codes = append(codes, uint32(code))
		}
	}
	return codes
}

// SetKeyDown prevents a key held across listener replacement from activating
// on its next repeat, and seeds modifiers for exact matching.
func (w *Windows) SetKeyDown(vk uint32, down bool) {
	if winModifier(vk) != 0 {
		w.modifiers[vk] = down
	}
	w.matcher.down[int64(vk)] = down
	if r := w.matcher.recording; r != nil {
		r.down[int64(vk)] = down
		if down {
			r.arming = true
		}
	}
}
func winModifier(vk uint32) application.KeyModifiers {
	switch vk {
	case 0xa2, 0xa3:
		return application.KeyCtrl
	case 0xa4, 0xa5:
		return application.KeyAlt
	case 0xa0, 0xa1:
		return application.KeyShift
	case 0x5b, 0x5c:
		return application.KeySuper
	}
	return 0
}
func (w *Windows) InterpretKey(message uintptr, vk, scan uint32) (application.HoldKey, Event, bool) {
	if vk == vkLControl && scan == altGrScan {
		// A hook installed while AltGr is held may have sampled its fake Ctrl.
		// Its synthetic release must clear that seed without becoming a gesture.
		if message == WinKeyUp || message == WinSysKeyUp {
			w.SetKeyDown(vkLControl, false)
			if w.matcher.recording != nil {
				return w.matcher.event(int64(vkLControl), false, w.modifiersMask(), false)
			}
		}
		return "", 0, false
	}
	pressed := message == WinKeyDown || message == WinSysKeyDown
	if !pressed && message != WinKeyUp && message != WinSysKeyUp {
		return "", 0, false
	}
	if winModifier(vk) != 0 {
		w.modifiers[vk] = pressed
	}
	mods := w.modifiersMask()
	return w.matcher.event(int64(vk), pressed, mods, false)
}

func (w *Windows) modifiersMask() application.KeyModifiers {
	var mods application.KeyModifiers
	for key, down := range w.modifiers {
		if down {
			mods |= winModifier(key)
		}
	}
	return mods
}
