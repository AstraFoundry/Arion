// Piano is a piano in native UI that plays with package audio: click or
// drag across its keys, play it from the keyboard, or have it play a tune.
// Its sound is synthesized, a sound per key, decoded nowhere.
//
//	go run ./examples/piano
//	go run ./cmd/mygo dev -web examples/piano
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

// The keyboard spans three octaves, C3 to C6, in MIDI note numbers.
const (
	lowest  = 48
	highest = 84
)

type piano struct {
	win      *mygo.Window
	held     map[int]*audio.Player // keys down, by note
	sources  map[int]int           // what holds each: a count of keys, pointer and tune
	pointer  int                   // the note the pointer holds, or -1
	sustain  bool                  // the pedal, Space
	octave   int                   // where the keyboard's lower row starts
	song     *song                 // the tune playing, or nil
	songHeld []int                 // the notes it holds
	scale    float32               // of the keyboard drawn, to its full size
}

func newPiano() *piano {
	p := &piano{held: map[int]*audio.Player{}, sources: map[int]int{}, pointer: -1, octave: 4, scale: 1}
	// Synthesize every key's sound ahead, so that none waits for it.
	go func() {
		for n := lowest; n <= highest; n++ {
			sound(n)
		}
	}()
	return p
}

// down presses note, which sounds until the last that holds it lets go.
func (p *piano) down(note int) {
	if note < lowest || note > highest {
		return
	}
	p.sources[note]++
	if p.sources[note] > 1 {
		return
	}
	if old := p.held[note]; old != nil {
		damp(old)
	}
	p.held[note] = sound(note).Play()
}

// up lets go of note: the damper stops it, unless the pedal is down.
func (p *piano) up(note int) {
	if p.sources[note] == 0 {
		return
	}
	if p.sources[note]--; p.sources[note] > 0 {
		return
	}
	delete(p.sources, note)
	if !p.sustain {
		if pl := p.held[note]; pl != nil {
			damp(pl)
			delete(p.held, note)
		}
	}
}

// pedal presses or lifts the sustain pedal; lifting it damps the strings
// of the keys up.
func (p *piano) pedal(down bool) {
	p.sustain = down
	if down {
		return
	}
	for note, pl := range p.held {
		if p.sources[note] == 0 {
			damp(pl)
			delete(p.held, note)
		}
	}
}

// damp fades a string out over 150 ms, as a damper does, and stops it.
func damp(pl *audio.Player) {
	go func() {
		v := pl.Volume()
		for i := 9; i >= 0; i-- {
			pl.SetVolume(v * float64(i) / 10)
			time.Sleep(15 * time.Millisecond)
		}
		pl.Close()
	}()
}

// Keys of a computer keyboard by position, for the notes of two octaves:
// the lower row and the one above it, as trackers lay them out.
var computerKeys = []struct {
	key   ui.Key
	label string
}{
	{ui.KeyZ, "Z"}, {ui.KeyS, "S"}, {ui.KeyX, "X"}, {ui.KeyD, "D"}, {ui.KeyC, "C"}, {ui.KeyV, "V"},
	{ui.KeyG, "G"}, {ui.KeyB, "B"}, {ui.KeyH, "H"}, {ui.KeyN, "N"}, {ui.KeyJ, "J"}, {ui.KeyM, "M"},
	{ui.KeyQ, "Q"}, {ui.Key2, "2"}, {ui.KeyW, "W"}, {ui.Key3, "3"}, {ui.KeyE, "E"}, {ui.KeyR, "R"},
	{ui.Key5, "5"}, {ui.KeyT, "T"}, {ui.Key6, "6"}, {ui.KeyY, "Y"}, {ui.Key7, "7"}, {ui.KeyU, "U"},
	{ui.KeyI, "I"},
}

// keyNotes are the notes of computerKeys, in semitones above the lower
// row's C; the comma plays the C after M, as Q does.
var keyNotes = func() map[ui.Key]int {
	m := map[ui.Key]int{ui.KeyComma: 12}
	for i, k := range computerKeys {
		m[k.key] = i
	}
	return m
}()

// keysDown are the computer's keys held, with the note each pressed.
var keysDown = map[ui.Key]int{}

func (p *piano) input(ev ui.InputEvent) bool {
	switch ev.Kind {
	case ui.InputKeyDown, ui.InputKeyUp:
		if ev.Mods&(ui.Super|ui.Ctrl|ui.Alt) != 0 {
			return false
		}
		down := ev.Kind == ui.InputKeyDown
		switch k := ev.PhysicalKey; k {
		case ui.KeySpace:
			p.pedal(down)
		case ui.KeyLeft, ui.KeyRight:
			if down && !ev.Repeat {
				if k == ui.KeyLeft {
					p.octave = max(p.octave-1, 3)
				} else {
					p.octave = min(p.octave+1, 4)
				}
			}
		default:
			off, ok := keyNotes[k]
			if !ok {
				return false
			}
			if ev.Repeat {
				return true
			}
			if down {
				note := (p.octave+1)*12 + off
				keysDown[k] = note
				p.down(note)
			} else if note, ok := keysDown[k]; ok {
				delete(keysDown, k)
				p.up(note)
			}
		}
		return true
	case ui.InputPointerDown, ui.InputPointerMove:
		if ev.Kind == ui.InputPointerMove && ev.Button != 0 {
			return false
		}
		// Dragging across the keys plays each it enters.
		note := keyAt(ev.X/p.scale, ev.Y/p.scale)
		if note != p.pointer {
			if p.pointer >= 0 {
				p.up(p.pointer)
			}
			p.pointer = note
			if note >= 0 {
				p.down(note)
			}
		}
		return true
	case ui.InputPointerUp:
		if p.pointer >= 0 {
			p.up(p.pointer)
			p.pointer = -1
		}
		return true
	}
	return false
}

func (p *piano) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Center().Gap(18).Padding(20).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			for _, s := range songs {
				if ui.Button(c.Key(s.name), s.name).Clicked() {
					p.play(s)
				}
			}
			if ui.Button(c.Key("stop"), "Stop").Disabled(p.song == nil).Clicked() {
				p.stop()
			}
		})
		// As wide as it can be up to its size, keeping its shape.
		ui.Box(c).FillWidth().MaxWidth(keyboardWidth).AspectRatio(keyboardWidth / keyboardHeight).
			Focusable().Focus().
			Label("Piano").
			HandleInput(p.input).
			Draw(func(pt *ui.Painter, r ui.Rect) { p.draw(pt, r, t) })
		hint := fmt.Sprintf("Keys Z–M and Q–I play from C%d · Space sustains · ← → change octaves", p.octave)
		ui.Text(c, hint).FontSize(12).TextColor(t.TextMuted)
		if err := audio.Err(); err != nil {
			ui.Text(c, err.Error()).FontSize(12).TextColor(t.TextMuted)
		}
	})
}

// The keyboard's geometry, in DIPs.
const (
	whiteWidth     = 36
	blackWidth     = 22
	keyboardHeight = 180
	blackHeight    = 110
)

var keyboardWidth = float32(whites()) * whiteWidth

// black reports whether note is a black key.
func black(note int) bool { return "010100101010"[note%12] == '1' }

func whites() int {
	n := 0
	for note := lowest; note <= highest; note++ {
		if !black(note) {
			n++
		}
	}
	return n
}

// keyRect returns where note's key is on the keyboard.
func keyRect(note int) ui.Rect {
	w := 0
	for n := lowest; n < note; n++ {
		if !black(n) {
			w++
		}
	}
	if black(note) {
		return ui.Rect{X: float32(w)*whiteWidth - blackWidth/2, W: blackWidth, H: blackHeight}
	}
	return ui.Rect{X: float32(w) * whiteWidth, W: whiteWidth, H: keyboardHeight}
}

func inside(r ui.Rect, x, y float32) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// keyAt returns the note of the key at x, y, black keys over white, or -1.
func keyAt(x, y float32) int {
	for note := lowest; note <= highest; note++ {
		if black(note) && inside(keyRect(note), x, y) {
			return note
		}
	}
	for note := lowest; note <= highest; note++ {
		if !black(note) && inside(keyRect(note), x, y) {
			return note
		}
	}
	return -1
}

// labels are the computer keys that play each note.
func (p *piano) labels() map[int]string {
	m := map[int]string{}
	for i, k := range computerKeys {
		m[(p.octave+1)*12+i] = k.label
	}
	return m
}

func (p *piano) draw(pt *ui.Painter, r ui.Rect, t *ui.Theme) {
	labels := p.labels()
	sounding := func(note int) bool { return p.sources[note] > 0 }
	// The keyboard shrinks with the window; input scales back.
	s := r.W / keyboardWidth
	p.scale = s
	at := func(k ui.Rect) ui.Rect {
		return ui.Rect{X: r.X + k.X*s, Y: r.Y + k.Y*s, W: k.W * s, H: k.H * s}
	}
	for note := lowest; note <= highest; note++ {
		if black(note) {
			continue
		}
		k := at(keyRect(note))
		face := ui.Hex("#fbfbf8")
		if sounding(note) {
			face = t.Accent.Mix(face, 0.55)
		}
		pt.Fill(k, face, 4*s)
		pt.Stroke(k, ui.Hex("#a1a1aa"), 4*s, 1)
		if l := labels[note]; l != "" {
			pt.Text(k.X+k.W/2-4*s, k.Y+k.H-24*s, l, 11*s, ui.Hex("#71717a"))
		}
		if note%12 == 0 {
			pt.Text(k.X+4*s, k.Y+k.H-42*s, fmt.Sprintf("C%d", note/12-1), 10*s, ui.Hex("#a1a1aa"))
		}
	}
	for note := lowest; note <= highest; note++ {
		if !black(note) {
			continue
		}
		k := at(keyRect(note))
		face := ui.Hex("#18181b")
		if sounding(note) {
			face = t.Accent
		}
		pt.Fill(k, face, 3*s)
		if l := labels[note]; l != "" {
			pt.Text(k.X+k.W/2-4*s, k.Y+k.H-20*s, l, 10*s, ui.Hex("#d4d4d8"))
		}
	}
}

func main() {
	p := newPiano()
	mygo.App.WhenReady(func() {
		p.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Piano",
			Width:   int(keyboardWidth) + 48,
			Height:  330,
			Content: ui.View(p.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
