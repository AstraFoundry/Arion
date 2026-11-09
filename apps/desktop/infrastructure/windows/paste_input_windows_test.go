//go:build windows

package windows

import (
	"testing"
	"unsafe"
)

func TestPasteInputsMatchThePhasedChord(t *testing.T) {
	if unsafe.Sizeof(keyboardInput{}) != 40 {
		t.Fatalf("INPUT size %d", unsafe.Sizeof(keyboardInput{}))
	}
	plan := planPaste([]uint16{vkRMenu})
	release := keyboardInputs(plan.Release)
	if len(release) != 1 || release[0].VK != vkRMenu || release[0].Scan == 0 || release[0].Flags&keyeventfKeyUp == 0 {
		t.Fatalf("release %+v", release)
	}
	chord := keyboardInputs(plan.Chord)
	if len(chord) != 3 || chord[0].VK != vkLControl || chord[0].Scan == 0 || chord[1].VK != vkV || chord[1].Scan == 0 {
		t.Fatalf("chord %+v", chord)
	}
	up := keyboardInputs(plan.CtrlUp)
	if len(up) != 1 || up[0].VK != vkLControl || up[0].Flags&keyeventfKeyUp == 0 || up[0].Scan == 0 {
		t.Fatalf("ctrl up %+v", up)
	}
	for _, input := range append(append(release, chord...), up...) {
		if input.Extra != pasteInputExtra {
			t.Fatalf("paste key lost its hold-hook mark: %+v", input)
		}
	}
}
