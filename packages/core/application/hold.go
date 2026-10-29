package application

import (
	"strconv"
	"time"
)

// HoldKey is the key that dictates while held, or starts and finishes a
// dictation when tapped (DSK-02). It is the only global activation, so a
// platform with an adapter always has one chosen.
type HoldKey string

const (
	HoldOff          HoldKey = "off"
	HoldFn           HoldKey = "fn"
	HoldRightOption  HoldKey = "right_option"
	HoldRightCommand HoldKey = "right_command"
	HoldRightAlt     HoldKey = "right_alt"
	HoldRightCtrl    HoldKey = "right_ctrl"
	HoldLeftCtrl     HoldKey = "left_ctrl"
	HoldLeftOption   HoldKey = "left_option"
	HoldLeftCommand  HoldKey = "left_command"
	HoldLeftAlt      HoldKey = "left_alt"
	HoldLeftShift    HoldKey = "left_shift"
	HoldRightShift   HoldKey = "right_shift"
	HoldMenu         HoldKey = "menu"
)

// HoldKeysFor lists the keys a platform's adapter can observe alone.
// Windows keyboards report no fn key to the OS.
func HoldKeysFor(platform string) []HoldKey {
	var keys []HoldKey
	switch platform {
	case "darwin":
		keys = []HoldKey{HoldFn, HoldRightOption, HoldRightCommand, HoldLeftOption, HoldLeftCommand, HoldLeftCtrl, HoldRightCtrl, HoldLeftShift, HoldRightShift}
	case "windows":
		keys = []HoldKey{HoldRightAlt, HoldRightCtrl, HoldLeftAlt, HoldLeftCtrl, HoldLeftShift, HoldRightShift, HoldMenu}
	default:
		return nil
	}
	n := 20
	if platform == "windows" {
		n = 24
	}
	for i := 1; i <= n; i++ {
		keys = append(keys, HoldKey("f"+strconv.Itoa(i)))
	}
	return keys
}

// TranslateKeysFor lists the translation hold-key choices for a platform,
// including HoldOff to disable the translation shortcut.
func TranslateKeysFor(platform string) []HoldKey {
	keys := HoldKeysFor(platform)
	if len(keys) == 0 {
		return nil
	}
	return append([]HoldKey{HoldOff}, keys...)
}

// DefaultHoldKey is the first-run key: Right Option on macOS and Right Alt
// on Windows. A platform without an adapter has none.
func DefaultHoldKey(platform string) HoldKey {
	switch platform {
	case "darwin":
		return HoldRightOption
	case "windows":
		return HoldRightAlt
	}
	return ""
}

// DefaultTranslateKey is the first-run translation key: Right Command on
// macOS and Right Ctrl on Windows. A platform without an adapter has none.
func DefaultTranslateKey(platform string) HoldKey {
	switch platform {
	case "darwin":
		return HoldRightCommand
	case "windows":
		return HoldRightCtrl
	}
	return ""
}

func knownHoldKey(platform string, key HoldKey) bool {
	if DefaultHoldKey(platform) == "" {
		return key == ""
	}
	_, ok := ParseHoldShortcut(platform, key)
	return ok
}

func knownTranslateKey(platform string, key HoldKey) bool {
	if DefaultTranslateKey(platform) == "" {
		return key == "" || key == HoldOff
	}
	return key == HoldOff || knownHoldKey(platform, key)
}

// ChooseHoldKey records a dictation key only when translation does not conflict.
func (p *Preferences) ChooseHoldKey(key HoldKey) {
	if HoldKeysConflict(p.TranslateKey, key) {
		return
	}
	p.HoldKey = key
}

// ChooseTranslateKey records a translation key only when dictation does not conflict.
func (p *Preferences) ChooseTranslateKey(platform string, key HoldKey) {
	if HoldKeysConflict(key, p.HoldKey) {
		return
	}
	if key == HoldOff && p.TranslateKey != "" && p.TranslateKey != HoldOff {
		p.PreviousTranslateKey = p.TranslateKey
	}
	p.TranslateKey = key
}

// EnableTranslationShortcut restores a disabled assignment if it remains
// available, otherwise chooses a nonconflicting key without moving dictation.
func (p *Preferences) EnableTranslationShortcut(platform string, enabled bool) {
	if !enabled {
		p.ChooseTranslateKey(platform, HoldOff)
		return
	}
	if p.TranslateKey != "" && p.TranslateKey != HoldOff {
		return
	}
	candidates := append([]HoldKey{p.PreviousTranslateKey, DefaultTranslateKey(platform)}, HoldKeysFor(platform)...)
	for _, key := range candidates {
		if knownHoldKey(platform, key) && !HoldKeysConflict(key, p.HoldKey) {
			p.TranslateKey = key
			return
		}
	}
}

// DefaultHoldThreshold separates a hold from a tap.
const DefaultHoldThreshold = 300 * time.Millisecond

// HoldSession is the dictation state a hold gesture acts on.
type HoldSession int

const (
	// HoldIdle: no live session; a press starts one.
	HoldIdle HoldSession = iota
	// HoldCapturing: a session is preparing or capturing; a press ends it.
	HoldCapturing
	// HoldBusy: a session is finalizing or committing; presses are ignored.
	HoldBusy
)

// HoldAction is what the gesture asks of the coordinator.
type HoldAction int

const (
	HoldNone HoldAction = iota
	HoldStart
	HoldStop
	HoldCancel
)

// HoldGesture interprets the hold key. Holding past the threshold dictates
// until release; a shorter tap keeps dictating hands-free until the next
// press. Another key during a hold that started the session is a chord
// (fn+arrow, Option+letter), so that session is cancelled and inserts
// nothing.
type HoldGesture struct {
	Threshold time.Duration
	down      bool
	started   bool
	pressedAt time.Time
}

// Press handles key-down. Auto-repeat presses while down are ignored.
func (g *HoldGesture) Press(now time.Time, session HoldSession) HoldAction {
	if g.down {
		return HoldNone
	}
	g.down, g.started, g.pressedAt = true, false, now
	switch session {
	case HoldIdle:
		g.started = true
		return HoldStart
	case HoldCapturing:
		return HoldStop
	}
	return HoldNone
}

// Release handles key-up.
func (g *HoldGesture) Release(now time.Time) HoldAction {
	if !g.down {
		return HoldNone
	}
	g.down = false
	started := g.started
	g.started = false
	if started && now.Sub(g.pressedAt) >= g.threshold() {
		return HoldStop
	}
	return HoldNone
}

// Other handles any other key going down.
func (g *HoldGesture) Other() HoldAction {
	if !g.down || !g.started {
		return HoldNone
	}
	g.started = false
	return HoldCancel
}

// Reset forgets a press, e.g. when the key listener stops.
func (g *HoldGesture) Reset() { g.down, g.started = false, false }

func (g *HoldGesture) threshold() time.Duration {
	if g.Threshold > 0 {
		return g.Threshold
	}
	return DefaultHoldThreshold
}
