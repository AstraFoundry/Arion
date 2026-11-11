//go:build linux && (amd64 || arm64)

package linux

import "testing"

func TestKeysDownRepeat(t *testing.T) {
	var d keysDown
	if d.press(38) {
		t.Fatal("the first press is no repeat")
	}
	if !d.press(38) || !d.press(38) {
		t.Fatal("presses of a key held with no release between are repeats")
	}
	if d.press(39) {
		t.Fatal("another key's press is no repeat")
	}
	d.release(38)
	if d.press(38) {
		t.Fatal("a press after the release is no repeat")
	}
	d.clear()
	if d.press(38) || d.press(39) {
		t.Fatal("presses after losing the keyboard are no repeats")
	}
	if d.press(0) || d.press(0) {
		t.Fatal("a key with no keycode is never a repeat")
	}
}

func TestKeysDownOtherSideHeld(t *testing.T) {
	const leftShift, rightShift, leftSuper, rightSuper, a = 42 + 8, 54 + 8, 125 + 8, 126 + 8, 30 + 8
	var d keysDown
	if d.otherSideHeld(leftShift) {
		t.Error("the other Shift is held with nothing held")
	}
	d.press(rightShift)
	d.press(a)
	if !d.otherSideHeld(leftShift) {
		t.Error("the right Shift is held")
	}
	if d.otherSideHeld(rightShift) {
		t.Error("the left Shift is not held")
	}
	if d.otherSideHeld(leftSuper) || d.otherSideHeld(rightSuper) {
		t.Error("no Super is held")
	}
	if d.otherSideHeld(a) {
		t.Error("A has no other side")
	}
}
