package windows

import "time"

// pasteWindow is one verify window, measured from the accepted chord or
// from the check that stretched it. The field is checked at Check; the
// board is released at Release.
type pasteWindow struct {
	Check   time.Duration
	Release time.Duration
}

// The clipboard is held until the host has had its delay. A host may read
// the board long after SendInput returns; taking it back earlier leaves the
// field empty and the board empty. When the first check finds the field
// unchanged, the window is stretched from that check.
var (
	pasteVerify          = pasteWindow{Check: 300 * time.Millisecond, Release: 600 * time.Millisecond}
	pasteVerifyStretched = pasteWindow{Check: 500 * time.Millisecond, Release: 1100 * time.Millisecond}
)

// pastePoll bounds how late a newer clipboard owner is noticed during the
// hold. It is not another paste.
const pastePoll = 50 * time.Millisecond

// pasteBoard is the clipboard as seen during the hold.
type pasteBoard int

const (
	// boardOwned is the board this paste wrote; its sequence is unchanged.
	boardOwned pasteBoard = iota
	// boardTouched still carries Arion's marker after its sequence moved:
	// the host has touched the board this paste wrote.
	boardTouched
	// boardReplaced belongs to a newer owner (DSK-A05).
	boardReplaced
)

// pasteHoldResult is what the hold leaves on the clipboard.
type pasteHoldResult int

const (
	// holdRestore puts the previous clipboard back: the host read the board.
	holdRestore pasteHoldResult = iota
	// holdYield leaves a newer owner's clipboard alone.
	holdYield
	// holdKeep leaves the dictated text: the host never showed that it read
	// the board, so a manual Ctrl+V still has the text.
	holdKeep
)

// pasteHoldProbe observes one posted paste. landed reports whether the
// focused field changed length since before the chord; it is false when the
// field cannot be measured. Neither probe posts input.
type pasteHoldProbe struct {
	board  func() pasteBoard
	landed func() bool
	sleep  func(time.Duration)
}

// holdPaste waits for evidence that the host read the board. The previous
// clipboard is never restored while the board is still this paste's and
// the verify window is open. A field that never changes keeps the dictated
// text on the clipboard instead of an earlier, possibly empty, board.
func holdPaste(p pasteHoldProbe) pasteHoldResult {
	var elapsed time.Duration
	// wait sleeps until the deadline and reports a board that left this
	// paste's ownership before then.
	wait := func(until time.Duration) (pasteBoard, bool) {
		for elapsed < until {
			step := min(pastePoll, until-elapsed)
			p.sleep(step)
			elapsed += step
			if board := p.board(); board != boardOwned {
				return board, true
			}
		}
		return boardOwned, false
	}
	settle := func(board pasteBoard) pasteHoldResult {
		if board == boardReplaced {
			return holdYield
		}
		return holdRestore
	}
	if board, moved := wait(pasteVerify.Check); moved {
		return settle(board)
	}
	window, base := pasteVerify, time.Duration(0)
	seen := p.landed()
	if !seen {
		window, base = pasteVerifyStretched, elapsed
		if board, moved := wait(base + window.Check); moved {
			return settle(board)
		}
		seen = p.landed()
	}
	if board, moved := wait(base + window.Release); moved {
		return settle(board)
	}
	if seen || p.landed() {
		return holdRestore
	}
	return holdKeep
}
