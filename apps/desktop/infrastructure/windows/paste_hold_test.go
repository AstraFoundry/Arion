package windows

import (
	"testing"
	"time"
)

func TestIssue110VerifyWindows(t *testing.T) {
	if pasteVerify != (pasteWindow{Check: 300 * time.Millisecond, Release: 600 * time.Millisecond}) {
		t.Fatalf("verify %+v", pasteVerify)
	}
	if pasteVerifyStretched != (pasteWindow{Check: 500 * time.Millisecond, Release: 1100 * time.Millisecond}) {
		t.Fatalf("stretched %+v", pasteVerifyStretched)
	}
}

// The reported miss: the previous clipboard is empty and the host reads
// the board late. Every read inside the verify window must see the
// dictated text, and the board is never released before that read.
func TestIssue110HoldOutlastsSlowHost(t *testing.T) {
	last := pasteVerifyStretched.Release + pasteVerify.Check
	for read := time.Duration(0); read <= last; read += 10 * time.Millisecond {
		got := simulateIssue110(issue110Request{
			Host: issue110DeferredRead, Held: issue110Held{RightAlt: true}, Injected: true,
			Dictated: "你好", Previous: "", ReadAfter: read,
		})
		if got.Field != "你好" || got.Elapsed < read || got.Outcome != "unknown" || got.ClipboardLeft != "" {
			t.Fatalf("read at %s: %+v", read, got)
		}
	}
	early := simulateIssue110(issue110Request{Host: issue110Edit, Injected: true, Dictated: "你好", Previous: ""})
	if early.Elapsed != pasteVerify.Release || early.Field != "你好" || early.ClipboardLeft != "" {
		t.Fatalf("in-message paste %+v", early)
	}
}

// While the sequence is unchanged and the field shows nothing, the
// previous clipboard is not put back: not inside the window, and not after
// it. The dictated text stays for a manual Ctrl+V and the commit is copied.
func TestIssue110UnreadBoardIsNotRestored(t *testing.T) {
	for _, req := range []issue110Request{
		{Host: issue110VoiceSwallow, Injected: true, Dictated: "你好", Previous: ""},
		{Host: issue110Edit, Injected: true, Unmeasured: true, Dictated: "你好", Previous: "old"},
		{Host: issue110DeferredRead, Injected: true, Dictated: "你好", Previous: "old", ReadAfter: 5 * time.Second},
	} {
		var now time.Duration
		var checks []time.Duration
		result := holdPaste(pasteHoldProbe{
			board:  func() pasteBoard { return boardOwned },
			landed: func() bool { checks = append(checks, now); return false },
			sleep:  func(d time.Duration) { now += d },
		})
		end := pasteVerify.Check + pasteVerifyStretched.Release
		if result != holdKeep || now != end {
			t.Fatalf("result %d after %s", result, now)
		}
		want := []time.Duration{pasteVerify.Check, pasteVerify.Check + pasteVerifyStretched.Check, end}
		if len(checks) != len(want) {
			t.Fatalf("checks %v", checks)
		}
		for i := range want {
			if checks[i] != want[i] {
				t.Fatalf("checks %v", checks)
			}
		}
		got := simulateIssue110(req)
		if got.Outcome != "copied" || got.ClipboardLeft != "你好" || !got.PostedChord {
			t.Fatalf("%+v: %+v", req, got)
		}
	}
}

// A moved sequence that still carries Arion's marker means the host
// touched the board: the previous clipboard goes back. A newer owner is
// left alone (DSK-A05). Neither waits for the rest of the window.
func TestIssue110MovedSequence(t *testing.T) {
	touched := simulateIssue110(issue110Request{
		Host: issue110DeferredRead, Injected: true, Unmeasured: true,
		Dictated: "你好", Previous: "old", ReadAfter: 100 * time.Millisecond, Touch: 120 * time.Millisecond,
	})
	if touched.ClipboardLeft != "old" || touched.Field != "你好" || touched.Outcome != "unknown" || touched.Elapsed != 150*time.Millisecond {
		t.Fatalf("touched %+v", touched)
	}
	replaced := simulateIssue110(issue110Request{
		Host: issue110VoiceSwallow, Injected: true,
		Dictated: "你好", Previous: "old", Replace: 700 * time.Millisecond, Replacement: "copied by user",
	})
	if replaced.ClipboardLeft != "copied by user" || replaced.Outcome != "unknown" || replaced.Elapsed != 700*time.Millisecond {
		t.Fatalf("replaced %+v", replaced)
	}
}
