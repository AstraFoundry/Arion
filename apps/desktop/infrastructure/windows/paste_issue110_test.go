package windows

import (
	"testing"
	"time"

	"arion.local/desktop/infrastructure/focusclass"
)

func TestIssue110PasteKeysAreNotHoldEvents(t *testing.T) {
	if holdObservesKey(pasteInputExtra) {
		t.Fatal("paste injection would release the hold key")
	}
	if !holdObservesKey(0) {
		t.Fatal("a user key was ignored")
	}
}

func TestIssue110PlanSeparatesControlUpAndSetsScanCodes(t *testing.T) {
	plan := planPaste([]uint16{vkRMenu, vkLShift, vkLWin, vkRControl, vkV})
	if pastePhaseGap != 30*time.Millisecond || pasteInputExtra != 0xA710110 {
		t.Fatalf("gap %s extra %#x", pastePhaseGap, pasteInputExtra)
	}
	if len(plan.Release) != 3 {
		t.Fatalf("release %+v", plan.Release)
	}
	for _, key := range plan.Release {
		if key.Flags&keyeventfKeyUp == 0 || key.Scan == 0 || key.VK == vkRControl || key.VK == vkV {
			t.Fatalf("release event %+v", key)
		}
	}
	if len(plan.Chord) != 3 || plan.Chord[0].VK != vkLControl || plan.Chord[0].Flags&keyeventfKeyUp != 0 || plan.Chord[0].Scan == 0 {
		t.Fatalf("chord %+v", plan.Chord)
	}
	if plan.Chord[1].VK != vkV || plan.Chord[1].Scan == 0 || plan.Chord[2].VK != vkV || plan.Chord[2].Flags&keyeventfKeyUp == 0 {
		t.Fatalf("V %+v", plan.Chord)
	}
	if len(plan.CtrlUp) != 1 || plan.CtrlUp[0].VK != vkLControl || plan.CtrlUp[0].Flags&keyeventfKeyUp == 0 || plan.CtrlUp[0].Scan == 0 {
		t.Fatalf("ctrl up %+v", plan.CtrlUp)
	}
	for _, phase := range [][]pasteKey{plan.Release, plan.Chord, plan.CtrlUp} {
		for _, key := range phase {
			if key.Flags&0x0004 != 0 || key.VK == 0 {
				t.Fatalf("unicode or empty key %+v", key)
			}
		}
	}
}

func TestIssue110HeldModifiersStillPaste(t *testing.T) {
	held := issue110Held{RightAlt: true, RightCtrl: true, LeftShift: true, LeftWin: true}
	for _, host := range []issue110Host{issue110Edit, issue110Composer, issue110AsyncSample} {
		got := simulateIssue110(issue110Request{Host: host, Held: held, Injected: true, Dictated: "你好", Previous: "old"})
		if got.Field != "你好" || got.Outcome != "unknown" || !got.PostedChord || got.ClipboardLeft != "old" || got.UnicodeEvents != 0 {
			t.Fatalf("host %d: %+v", host, got)
		}
	}
}

func TestIssue110RejectedSendLeavesDictatedText(t *testing.T) {
	got := simulateIssue110(issue110Request{Host: issue110Edit, Injected: false, Dictated: "你好", Previous: "old"})
	if got.Field != "" || got.Outcome != "copied" || got.ClipboardLeft != "你好" || got.PostedChord || got.Elapsed != 0 {
		t.Fatalf("%+v", got)
	}
}

// A read after the former 400 ms hold still gets the dictated text and is
// not followed by another paste.
func TestIssue110LateReadIsNotAnotherPaste(t *testing.T) {
	late := simulateIssue110(issue110Request{
		Host: issue110DeferredRead, Held: issue110Held{RightAlt: true}, Injected: true,
		Dictated: "你好", Previous: "old", ReadAfter: 401 * time.Millisecond,
	})
	if late.Field != "你好" || late.Outcome != "unknown" || late.ClipboardLeft != "old" || !late.PostedChord || late.UnicodeEvents != 0 {
		t.Fatalf("late %+v", late)
	}
	inTime := simulateIssue110(issue110Request{
		Host: issue110DeferredRead, Held: issue110Held{RightAlt: true}, Injected: true,
		Dictated: "你好", Previous: "old", ReadAfter: 100 * time.Millisecond,
	})
	if inTime.Field != "你好" || inTime.Outcome != "unknown" || inTime.ClipboardLeft != "old" {
		t.Fatalf("in time %+v", inTime)
	}
}

// A swallowed chord is still one paste. The text stays on the clipboard
// for the user's own Ctrl+V, and the commit is copied, not pasted.
func TestIssue110SwallowedChordIsNotRetried(t *testing.T) {
	got := simulateIssue110(issue110Request{
		Host: issue110VoiceSwallow, Held: issue110Held{RightAlt: true}, Injected: true,
		Dictated: "你好", Previous: "old",
	})
	if got.Field != "" || got.Outcome != "copied" || !got.PostedChord || got.ClipboardLeft != "你好" || got.UnicodeEvents != 0 {
		t.Fatalf("%+v", got)
	}
	decision := focusclass.DecideWindows(focusclass.WinObservation{ControlType: focusclass.UIAEdit, ValuePattern: true, TextPattern: true, TextReadOnlyKnown: true})
	if !decision.Editable || decision.Secure {
		t.Fatalf("classifier %+v", decision)
	}
}
