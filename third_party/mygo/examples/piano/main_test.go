package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// Tests don't press keys, which would sound on the machine running them.
func TestView(t *testing.T) {
	p := newPiano()
	tt := ui.NewTester(p.view, int(keyboardWidth)+48, 330)
	for _, s := range songs {
		if !tt.HasText(s.name) {
			t.Errorf("no %q button in %q", s.name, tt.Texts())
		}
	}
}

func TestKeys(t *testing.T) {
	if n := whites(); n != 22 {
		t.Errorf("%d white keys, want 22", n)
	}
	// The left half of C♯3's black key, then C3 below it, then D3.
	if got := keyAt(whiteWidth-2, 10); got != 49 {
		t.Errorf("keyAt over C#3 = %d", got)
	}
	if got := keyAt(whiteWidth-2, blackHeight+10); got != 48 {
		t.Errorf("keyAt below C#3 = %d", got)
	}
	if got := keyAt(whiteWidth+blackWidth, 10); got != 50 {
		t.Errorf("keyAt right of C#3 = %d", got)
	}
	if got := keyAt(keyboardWidth+5, 10); got != -1 {
		t.Errorf("keyAt past the keyboard = %d", got)
	}
}

func TestSongs(t *testing.T) {
	for _, s := range songs {
		for _, ev := range s.events() {
			for _, n := range ev.notes {
				if n < lowest || n > highest {
					t.Errorf("%s plays %d, off the keyboard", s.name, n)
				}
			}
		}
	}
	if noteNumber("A4") != 69 || noteNumber("C#4") != 61 {
		t.Error("noteNumber")
	}
}

// A4 sounds at 440 Hz, after its hammer.
func TestSynth(t *testing.T) {
	s := synthSamples(69)
	if d := float64(len(s)) / synthRate; d < 1.4 || d > 4 {
		t.Errorf("A4 lasts %v s", d)
	}
	// The period is the lag the sound best correlates with itself at,
	// between 300 and 600 Hz.
	best, lag := 0.0, 0
	for l := synthRate / 600; l <= synthRate/300; l++ {
		c := 0.0
		for i := synthRate / 10; i < synthRate*6/10; i++ {
			c += float64(s[i]) * float64(s[i+l])
		}
		if c > best {
			best, lag = c, l
		}
	}
	if f := float64(synthRate) / float64(lag); f < 430 || f > 450 {
		t.Errorf("A4 at %v Hz", f)
	}
}
