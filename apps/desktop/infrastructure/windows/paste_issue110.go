package windows

import "time"

// One Windows paste is Left Ctrl+V (ADR 003). Shift, Alt and the Windows
// key are released first: SendInput does not clear keys that are already
// down, and a held Right Alt turns the chord into a literal V or no paste.
// V is sent while Left Ctrl is still down. Left Ctrl is released in a
// following call so a target that samples the keyboard after a batch does
// not see V with Control already up. Scan codes travel with the virtual
// keys. This is still one paste: no Unicode input and no WM_PASTE.
const (
	inputKeyboard        = 1
	keyeventfExtendedKey = 0x0001
	keyeventfKeyUp       = 0x0002
	vkLControl           = 0xA2
	vkRControl           = 0xA3
	vkControl            = 0x11
	vkLShift             = 0xA0
	vkRShift             = 0xA1
	vkShift              = 0x10
	vkLMenu              = 0xA4
	vkRMenu              = 0xA5
	vkMenu               = 0x12
	vkLWin               = 0x5B
	vkRWin               = 0x5C
	vkV                  = 0x56
	// pastePhaseGap lets the focused app observe a phase before the next
	// one is queued. It is not the clipboard hold and not another paste.
	pastePhaseGap = 30 * time.Millisecond
	// pasteInputExtra marks every key this paste injects. The hold hook
	// skips that mark, so releasing Right Alt is not a user key-up.
	pasteInputExtra uintptr = 0xA710110
)

type pasteKey struct {
	VK    uint16
	Scan  uint16
	Flags uint32
}

// pastePlan is the single Ctrl+V post, split so Control is still down when
// V is sampled.
type pastePlan struct {
	Release []pasteKey
	Chord   []pasteKey
	CtrlUp  []pasteKey
}

func interferingModifier(vk uint16) bool {
	switch vk {
	case vkMenu, vkLMenu, vkRMenu, vkShift, vkLShift, vkRShift, vkLWin, vkRWin:
		return true
	default:
		return false
	}
}

func keyEvent(vk uint16, up bool) pasteKey {
	key := pasteKey{VK: vk, Scan: scanCode(vk)}
	if extendedKey(vk) {
		key.Flags |= keyeventfExtendedKey
	}
	if up {
		key.Flags |= keyeventfKeyUp
	}
	return key
}

func extendedKey(vk uint16) bool {
	switch vk {
	case vkRMenu, vkRControl, vkLWin, vkRWin:
		return true
	default:
		return false
	}
}

func scanCode(vk uint16) uint16 {
	switch vk {
	case vkLControl, vkRControl, vkControl:
		return 0x1D
	case vkLShift, vkShift:
		return 0x2A
	case vkRShift:
		return 0x36
	case vkLMenu, vkRMenu, vkMenu:
		return 0x38
	case vkLWin:
		return 0x5B
	case vkRWin:
		return 0x5C
	case vkV:
		return 0x2F
	default:
		return 0
	}
}

// planPaste builds the one paste from the modifiers that are down. Control
// is not released here; a held Right Ctrl stays a Control chord. Keys that
// do not change Ctrl+V are ignored.
func planPaste(held []uint16) pastePlan {
	var release []pasteKey
	seen := map[uint16]bool{}
	for _, vk := range held {
		if !interferingModifier(vk) || seen[vk] {
			continue
		}
		seen[vk] = true
		release = append(release, keyEvent(vk, true))
	}
	return pastePlan{
		Release: release,
		Chord:   []pasteKey{keyEvent(vkLControl, false), keyEvent(vkV, false), keyEvent(vkV, true)},
		CtrlUp:  []pasteKey{keyEvent(vkLControl, true)},
	}
}

// issue110Host is how a focused control consumes the phased chord.
type issue110Host int

const (
	// issue110Edit pastes inside the V key message when the chord is Ctrl+V
	// and Alt is up. A standard EDIT control does this before the hold.
	issue110Edit issue110Host = iota
	// issue110Composer inserts V when the chord is not exact Ctrl+V.
	issue110Composer
	// issue110AsyncSample reads modifier state after the V phase and before
	// Left Ctrl is released.
	issue110AsyncSample
	// issue110DeferredRead accepts Ctrl+V but reads the clipboard at ReadAfter.
	issue110DeferredRead
	// issue110VoiceSwallow drops a posted Ctrl+V. ADR 003 does not follow
	// that with another transport.
	issue110VoiceSwallow
)

// issue110Held is the physical keyboard when the paste starts. Right Alt is
// the default dictation key. Right Ctrl is the default translation key.
type issue110Held struct {
	RightAlt  bool
	RightCtrl bool
	LeftShift bool
	LeftWin   bool
}

func (h issue110Held) keys() []uint16 {
	var held []uint16
	if h.RightAlt {
		held = append(held, vkRMenu)
	}
	if h.RightCtrl {
		held = append(held, vkRControl)
	}
	if h.LeftShift {
		held = append(held, vkLShift)
	}
	if h.LeftWin {
		held = append(held, vkLWin)
	}
	return held
}

// issue110Request is one dictation paste. Injected is false when SendInput
// does not accept the chord. ReadAfter is when issue110DeferredRead reads
// the board. Unmeasured is a field whose length cannot be read. Touch moves
// the clipboard sequence while Arion's marker stays; Replace is a newer
// owner writing Replacement. Zero leaves the board alone.
type issue110Request struct {
	Host        issue110Host
	Held        issue110Held
	Injected    bool
	ReadAfter   time.Duration
	Unmeasured  bool
	Touch       time.Duration
	Replace     time.Duration
	Replacement string
	Previous    string
	Dictated    string
}

// issue110Result is the field, the clipboard after the paste returns, and
// the commit outcome. Elapsed is when the hold released the board.
type issue110Result struct {
	Field         string
	ClipboardLeft string
	Outcome       string
	Elapsed       time.Duration
	UnicodeEvents int
	PostedChord   bool
}

type issue110Keys struct {
	down map[uint16]bool
}

func newIssue110Keys(held []uint16) *issue110Keys {
	k := &issue110Keys{down: map[uint16]bool{}}
	for _, vk := range held {
		k.down[vk] = true
	}
	return k
}

func (k *issue110Keys) apply(e pasteKey) {
	k.down[e.VK] = e.Flags&keyeventfKeyUp == 0
}

func (k *issue110Keys) control() bool {
	return k.down[vkLControl] || k.down[vkRControl] || k.down[vkControl]
}

func (k *issue110Keys) alt() bool {
	return k.down[vkMenu] || k.down[vkLMenu] || k.down[vkRMenu]
}

func (k *issue110Keys) shiftOrWin() bool {
	return k.down[vkShift] || k.down[vkLShift] || k.down[vkRShift] || k.down[vkLWin] || k.down[vkRWin]
}

// applyPhase applies key-downs before the V sample and key-ups after it.
// sample reports whether V was seen and whether that sample was a clean
// Ctrl+V (Control down, Alt/Shift/Win up).
func applyPhase(k *issue110Keys, phase []pasteKey, sample *bool, clean *bool) {
	for _, e := range phase {
		up := e.Flags&keyeventfKeyUp != 0
		if !up {
			k.apply(e)
		}
		if e.VK == vkV && !up {
			*sample = true
			*clean = k.control() && !k.alt() && !k.shiftOrWin()
		}
		if up {
			k.apply(e)
		}
	}
}

// simulateIssue110 runs the production plan and hold against one host. A
// rejected injection leaves the dictated text and does not restore. A
// posted paste is held by holdPaste on a simulated clock. A swallowed chord
// is still one posted paste and is not retried.
func simulateIssue110(req issue110Request) issue110Result {
	plan := planPaste(req.Held.keys())
	if !req.Injected {
		return issue110Result{Outcome: "copied", ClipboardLeft: req.Dictated}
	}
	keys := newIssue110Keys(req.Held.keys())
	var saw, clean bool
	applyPhase(keys, plan.Release, &saw, &clean)
	// The V phase is sampled before Left Ctrl goes up.
	applyPhase(keys, plan.Chord, &saw, &clean)
	if req.Host == issue110AsyncSample {
		// Async state after the V phase, still before Ctrl up.
		clean = keys.control() && !keys.alt() && !keys.shiftOrWin()
		saw = true
	}
	applyPhase(keys, plan.CtrlUp, &saw, &clean)

	// insert is what the field gains and when; read is a board read.
	insert, at, read := "", time.Duration(0), false
	switch {
	case req.Host == issue110VoiceSwallow:
	case saw && clean && req.Host == issue110DeferredRead:
		read, at = true, req.ReadAfter
	case saw && clean:
		read = true
	case saw && req.Host == issue110Composer:
		insert = "V"
	}
	var now time.Duration
	board := func() pasteBoard {
		switch {
		case req.Replace > 0 && now >= req.Replace:
			return boardReplaced
		case req.Touch > 0 && now >= req.Touch:
			return boardTouched
		}
		return boardOwned
	}
	if read && req.Replace > 0 && at >= req.Replace {
		insert = req.Replacement
	} else if read {
		insert = req.Dictated
	}
	result := holdPaste(pasteHoldProbe{
		board:  board,
		landed: func() bool { return !req.Unmeasured && insert != "" && now >= at },
		sleep:  func(d time.Duration) { now += d },
	})
	res := issue110Result{Outcome: "unknown", Elapsed: now, PostedChord: true}
	// A read after the hold returned sees whatever the hold left.
	switch result {
	case holdRestore:
		res.ClipboardLeft = req.Previous
	case holdYield:
		res.ClipboardLeft = req.Replacement
	case holdKeep:
		res.ClipboardLeft, res.Outcome = req.Dictated, "copied"
	}
	if read && at > now {
		insert = res.ClipboardLeft
	}
	res.Field = insert
	return res
}

// holdObservesKey reports whether a low-level key event belongs to the
// user. Paste injection carries pasteInputExtra and is not a hold gesture.
func holdObservesKey(extra uintptr) bool {
	return extra != pasteInputExtra
}
