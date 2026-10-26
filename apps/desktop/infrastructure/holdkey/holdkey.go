// Package holdkey turns raw keyboard events from the macOS event tap and the
// Windows low-level keyboard hook into hold-key transitions. It is pure so
// the interpretation is tested on every OS; the adapters only forward raw
// fields.
package holdkey

import "arion.local/core/application"

// Event is a hold-key transition.
type Event int

const (
	Down Event = iota
	Up
	// Other is another key or modifier going down.
	Other
)

// Quartz event types and fields the macOS adapter forwards.
const (
	MacKeyDown       = 10 // kCGEventKeyDown
	MacFlagsChanged  = 12 // kCGEventFlagsChanged
	MacSystemDefined = 14 // kCGEventSystemDefined: volume and media keys. The tap does not subscribe to this type.
)

type macSpec struct {
	keycode int64
	flag    uint64 // device-dependent bit, so the other side's key does not count
}

// macSpecs: kVK_Function with kCGEventFlagMaskSecondaryFn; right Option and
// right Command with their NX_DEVICER*KEYMASK bits.
var macSpecs = map[application.HoldKey]macSpec{
	application.HoldFn:           {keycode: 0x3F, flag: 0x800000},
	application.HoldRightOption:  {keycode: 0x3D, flag: 0x40},
	application.HoldRightCommand: {keycode: 0x36, flag: 0x10},
}

type macTarget struct {
	key  application.HoldKey
	spec macSpec
	down bool
}

// Mac interprets Quartz keyboard events for one or more hold keys.
type Mac struct {
	targets []macTarget
}

// NewMac returns false when no supported key is given or any key is
// unavailable on macOS. HoldOff and empty entries are ignored when paired
// with a primary key.
func NewMac(keys ...application.HoldKey) (*Mac, bool) {
	var targets []macTarget
	for _, key := range keys {
		if key == "" || key == application.HoldOff {
			continue
		}
		spec, ok := macSpecs[key]
		if !ok {
			return nil, false
		}
		duplicate := false
		for _, existing := range targets {
			if existing.key == key {
				duplicate = true
				break
			}
		}
		if !duplicate {
			targets = append(targets, macTarget{key: key, spec: spec})
		}
	}
	if len(targets) == 0 {
		return nil, false
	}
	return &Mac{targets: targets}, true
}

func (m *Mac) held() *macTarget {
	for i := range m.targets {
		if m.targets[i].down {
			return &m.targets[i]
		}
	}
	return nil
}

func (m *Mac) targetFor(keycode int64) *macTarget {
	for i := range m.targets {
		if m.targets[i].spec.keycode == keycode {
			return &m.targets[i]
		}
	}
	return nil
}

// InterpretKey maps one event and identifies which hold key it applies to;
// ok is false when it is not a transition.
func (m *Mac) InterpretKey(eventType int, keycode int64, flags uint64, autorepeat bool) (application.HoldKey, Event, bool) {
	switch eventType {
	case MacKeyDown:
		if autorepeat {
			return "", 0, false
		}
		if held := m.held(); held != nil {
			return held.key, Other, true
		}
		return m.targets[0].key, Other, true
	case MacFlagsChanged:
		held := m.held()
		target := m.targetFor(keycode)
		if target == nil {
			if held != nil {
				return held.key, Other, true
			}
			return "", 0, false
		}
		if held != nil && held != target {
			if flags&target.spec.flag != 0 {
				return held.key, Other, true
			}
			return "", 0, false
		}
		down := flags&target.spec.flag != 0
		if down == target.down {
			return "", 0, false
		}
		target.down = down
		if down {
			return target.key, Down, true
		}
		return target.key, Up, true
	}
	return "", 0, false
}

// Interpret maps one event; ok is false when it is not a transition.
func (m *Mac) Interpret(eventType int, keycode int64, flags uint64, autorepeat bool) (Event, bool) {
	_, event, ok := m.InterpretKey(eventType, keycode, flags, autorepeat)
	return event, ok
}

// Windows low-level keyboard hook messages and virtual keys.
const (
	WinKeyDown    = 0x0100
	WinKeyUp      = 0x0101
	WinSysKeyDown = 0x0104
	WinSysKeyUp   = 0x0105

	vkLControl = 0xA2
	// altGrScan is the scan code of the left Ctrl that AltGr layouts
	// synthesize before right Alt; it is part of the hold key, not a chord.
	altGrScan = 0x21D

	vkVolumeMute        = 0xAD
	vkVolumeDown        = 0xAE
	vkVolumeUp          = 0xAF
	vkMediaNextTrack    = 0xB0
	vkMediaPrevTrack    = 0xB1
	vkMediaStop         = 0xB2
	vkMediaPlayPause    = 0xB3
	vkLaunchMediaSelect = 0xB5
)

var winKeys = map[application.HoldKey]uint32{
	application.HoldRightAlt:  0xA5, // VK_RMENU
	application.HoldRightCtrl: 0xA3, // VK_RCONTROL
}

type winTarget struct {
	key  application.HoldKey
	vk   uint32
	down bool
}

// Windows interprets low-level hook events for one or more hold keys.
type Windows struct {
	targets []winTarget
}

// NewWindows returns false when no supported key is given or any key is
// unavailable on Windows. HoldOff and empty entries are ignored when paired
// with a primary key.
func NewWindows(keys ...application.HoldKey) (*Windows, bool) {
	var targets []winTarget
	for _, key := range keys {
		if key == "" || key == application.HoldOff {
			continue
		}
		vk, ok := winKeys[key]
		if !ok {
			return nil, false
		}
		duplicate := false
		for _, existing := range targets {
			if existing.key == key {
				duplicate = true
				break
			}
		}
		if !duplicate {
			targets = append(targets, winTarget{key: key, vk: vk})
		}
	}
	if len(targets) == 0 {
		return nil, false
	}
	return &Windows{targets: targets}, true
}

func (w *Windows) held() *winTarget {
	for i := range w.targets {
		if w.targets[i].down {
			return &w.targets[i]
		}
	}
	return nil
}

func (w *Windows) targetFor(vk uint32) *winTarget {
	for i := range w.targets {
		if w.targets[i].vk == vk {
			return &w.targets[i]
		}
	}
	return nil
}

// windowsMediaKey reports a volume or media virtual key. Those keys are
// not a dictation chord: the hook still forwards them, and the OS shows
// its own volume flyout when the user pressed one.
func windowsMediaKey(vk uint32) bool {
	switch vk {
	case vkVolumeMute, vkVolumeDown, vkVolumeUp,
		vkMediaNextTrack, vkMediaPrevTrack, vkMediaStop, vkMediaPlayPause,
		vkLaunchMediaSelect:
		return true
	default:
		return false
	}
}

// InterpretKey maps one event and identifies which hold key it applies to.
// ok is false when the event is not a transition. Volume and media keys
// are ignored here; the low-level hook still forwards them to Windows.
func (w *Windows) InterpretKey(message uintptr, vk, scan uint32) (application.HoldKey, Event, bool) {
	if windowsMediaKey(vk) {
		return "", 0, false
	}
	pressed := message == WinKeyDown || message == WinSysKeyDown
	released := message == WinKeyUp || message == WinSysKeyUp
	if vk == vkLControl && scan == altGrScan {
		return "", 0, false
	}
	held := w.held()
	target := w.targetFor(vk)
	if target == nil {
		if pressed && held != nil {
			return held.key, Other, true
		}
		return "", 0, false
	}
	if held != nil && held != target {
		if pressed {
			return held.key, Other, true
		}
		return "", 0, false
	}
	switch {
	case pressed && !target.down:
		target.down = true
		return target.key, Down, true
	case released && target.down:
		target.down = false
		return target.key, Up, true
	}
	return "", 0, false
}

// Interpret maps one event; ok is false when it is not a transition.
func (w *Windows) Interpret(message uintptr, vk, scan uint32) (Event, bool) {
	_, event, ok := w.InterpretKey(message, vk, scan)
	return event, ok
}
