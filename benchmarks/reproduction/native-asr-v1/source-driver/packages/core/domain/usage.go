package domain

import (
	"sort"
	"time"
	"unicode"
)

// Usage is what one committed dictation adds to its day's counts. It holds
// numbers and a scene name only, never text (005).
type Usage struct {
	// SpeechMs is how long the microphone listened.
	SpeechMs int64
	// Words counts the committed text with CountWords.
	Words int
	// Scene is the scene of an accepted rewrite; empty when the committed
	// text is the recognized text.
	Scene Scene
}

// UsageDay sums the dictations committed on one local calendar day.
type UsageDay struct {
	// Date is the local day as 2006-01-02.
	Date     string `json:"date"`
	Sessions int    `json:"sessions"`
	SpeechMs int64  `json:"speechMs"`
	Words    int    `json:"words"`
	// Scenes counts words of rewritten dictations by scene.
	Scenes map[Scene]int `json:"scenes,omitempty"`
}

// PolishedWords counts the words a rewrite produced.
func (d UsageDay) PolishedWords() int {
	total := 0
	for _, words := range d.Scenes {
		total += words
	}
	return total
}

// WordsPerMinute is the average dictation speed, 0 before any speech.
func (d UsageDay) WordsPerMinute() int {
	if d.SpeechMs <= 0 {
		return 0
	}
	return int(float64(d.Words)*60000/float64(d.SpeechMs) + 0.5)
}

func (d *UsageDay) add(u UsageDay) {
	d.Sessions += u.Sessions
	d.SpeechMs += u.SpeechMs
	d.Words += u.Words
	for scene, words := range u.Scenes {
		if d.Scenes == nil {
			d.Scenes = map[Scene]int{}
		}
		d.Scenes[scene] += words
	}
}

// DayKey names the local calendar day of t.
func DayKey(t time.Time) string { return t.Format(time.DateOnly) }

// UsageLog is the per-day history, oldest first, one entry per day.
type UsageLog []UsageDay

// Add returns the log with u counted on the day of at.
func (l UsageLog) Add(at time.Time, u Usage) UsageLog {
	entry := UsageDay{Date: DayKey(at), Sessions: 1, SpeechMs: max(u.SpeechMs, 0), Words: u.Words}
	if u.Scene != "" {
		entry.Scenes = map[Scene]int{u.Scene: u.Words}
	}
	out := append(UsageLog(nil), l...)
	i := sort.Search(len(out), func(i int) bool { return out[i].Date >= entry.Date })
	if i < len(out) && out[i].Date == entry.Date {
		out[i] = out[i].clone()
		out[i].add(entry)
		return out
	}
	return append(out[:i], append(UsageLog{entry}, out[i:]...)...)
}

func (d UsageDay) clone() UsageDay {
	if d.Scenes != nil {
		scenes := make(map[Scene]int, len(d.Scenes))
		for scene, words := range d.Scenes {
			scenes[scene] = words
		}
		d.Scenes = scenes
	}
	return d
}

// Since sums the days from the day of from on; a zero from sums every day.
func (l UsageLog) Since(from time.Time) UsageDay {
	first := ""
	if !from.IsZero() {
		first = DayKey(from)
	}
	var total UsageDay
	for _, day := range l {
		if day.Date >= first {
			total.add(day)
		}
	}
	return total
}

// Days returns the n days ending on the day of end, oldest first, with
// zero counts for days without dictation.
func (l UsageLog) Days(end time.Time, n int) []UsageDay {
	byDate := make(map[string]UsageDay, len(l))
	for _, day := range l {
		byDate[day.Date] = day
	}
	out := make([]UsageDay, n)
	for i := range out {
		key := DayKey(end.AddDate(0, 0, i-n+1))
		day, ok := byDate[key]
		if !ok {
			day = UsageDay{Date: key}
		}
		out[i] = day
	}
	return out
}

// CountWords counts text as people count what they wrote: each Han, kana
// or Hangul character is one word, as are runs of other letters and
// digits joined by apostrophes, hyphens or decimal points.
func CountWords(text string) int {
	words, inWord := 0, false
	runes := []rune(text)
	for i, r := range runes {
		switch {
		case unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul):
			words++
			inWord = false
		case unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r):
			if !inWord {
				words++
			}
			inWord = true
		case inWord && (r == '\'' || r == '’' || r == '-' || r == '.') && i+1 < len(runes) &&
			(unicode.IsLetter(runes[i+1]) || unicode.IsDigit(runes[i+1])) && !unicode.In(runes[i+1], unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul):
			// Joined: don't, e-mail, 3.5
		default:
			inWord = false
		}
	}
	return words
}
