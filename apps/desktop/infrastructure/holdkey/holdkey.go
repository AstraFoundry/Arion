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
	MacKeyDown      = 10 // kCGEventKeyDown
	MacFlagsChanged = 12 // kCGEventFlagsChanged
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

// Mac interprets Quartz keyboard events for one hold key.
type Mac struct {
	spec macSpec
	down bool
}

// NewMac returns false for keys macOS cannot observe alone.
func NewMac(key application.HoldKey) (*Mac, bool) {
	spec, ok := macSpecs[key]
	return &Mac{spec: spec}, ok
}

// Interpret maps one event; ok is false when it is not a transition.
func (m *Mac) Interpret(eventType int, keycode int64, flags uint64, autorepeat bool) (Event, bool) {
	switch eventType {
	case MacKeyDown:
		if autorepeat {
			return 0, false
		}
		return Other, true
	case MacFlagsChanged:
		if keycode != m.spec.keycode {
			return Other, m.down
		}
		down := flags&m.spec.flag != 0
		if down == m.down {
			return 0, false
		}
		m.down = down
		if down {
			return Down, true
		}
		return Up, true
	}
	return 0, false
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
)

var winKeys = map[application.HoldKey]uint32{
	application.HoldRightAlt:  0xA5, // VK_RMENU
	application.HoldRightCtrl: 0xA3, // VK_RCONTROL
}

// Windows interprets low-level hook events for one hold key.
type Windows struct {
	vk   uint32
	down bool
}

// NewWindows returns false for keys Windows cannot observe alone.
func NewWindows(key application.HoldKey) (*Windows, bool) {
	vk, ok := winKeys[key]
	return &Windows{vk: vk}, ok
}

// Interpret maps one event; ok is false when it is not a transition.
func (w *Windows) Interpret(message uintptr, vk, scan uint32) (Event, bool) {
	pressed := message == WinKeyDown || message == WinSysKeyDown
	released := message == WinKeyUp || message == WinSysKeyUp
	if vk == vkLControl && scan == altGrScan {
		return 0, false
	}
	if vk != w.vk {
		return Other, pressed && w.down
	}
	switch {
	case pressed && !w.down:
		w.down = true
		return Down, true
	case released && w.down:
		w.down = false
		return Up, true
	}
	return 0, false
}
