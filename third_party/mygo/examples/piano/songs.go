package main

import (
	"strconv"
	"strings"
	"time"
)

// A song is a tune in the public domain: notes as "E5:0.5", a name with
// its octave and a length in beats, "C4+E4+G4:1" for a chord, and "r:1"
// for a rest.
type song struct {
	name  string
	bpm   float64
	notes string
}

var songs = []*song{
	{"Twinkle, Twinkle", 100, `
		C4+C3:1 C4:1 G4+E3:1 G4:1 A4+F3:1 A4:1 G4+E3:2
		F4+D3:1 F4:1 E4+C3:1 E4:1 D4+G3:1 D4:1 C4+C3:2
		G4+E3:1 G4:1 F4+D3:1 F4:1 E4+C3:1 E4:1 D4+G3:2
		G4+E3:1 G4:1 F4+D3:1 F4:1 E4+C3:1 E4:1 D4+G3:2
		C4+C3:1 C4:1 G4+E3:1 G4:1 A4+F3:1 A4:1 G4+E3:2
		F4+D3:1 F4:1 E4+C3:1 E4:1 D4+G3:1 D4:1 C4+C3:2`},
	{"Ode to Joy", 120, `
		E4+C3:1 E4:1 F4:1 G4:1 G4+G3:1 F4:1 E4:1 D4:1
		C4+A3:1 C4:1 D4:1 E4:1 E4+G3:1.5 D4:0.5 D4:2
		E4+C3:1 E4:1 F4:1 G4:1 G4+G3:1 F4:1 E4:1 D4:1
		C4+A3:1 C4:1 D4:1 E4:1 D4+G3:1.5 C4:0.5 C4+C3:2`},
	{"Für Elise", 66, `
		E5:0.25 D#5:0.25 E5:0.25 D#5:0.25 E5:0.25 B4:0.25 D5:0.25 C5:0.25
		A4+A3:0.5 r:0.25 C4:0.25 E4:0.25 A4:0.25
		B4+E3:0.5 r:0.25 E4:0.25 G#4:0.25 B4:0.25
		C5+A3:0.5 r:0.25 E4:0.25 E5:0.25 D#5:0.25
		E5:0.25 D#5:0.25 E5:0.25 B4:0.25 D5:0.25 C5:0.25
		A4+A3:0.5 r:0.25 C4:0.25 E4:0.25 A4:0.25
		B4+E3:0.5 r:0.25 E4:0.25 C5:0.25 B4:0.25
		A4+A3:1.5`},
	{"Happy Birthday", 100, `
		G4:0.75 G4:0.25 A4+F3:1 G4:1 C5+E3:1 B4+G3:2
		G4:0.75 G4:0.25 A4+G3:1 G4:1 D5+F3:1 C5+E3:2
		G4:0.75 G4:0.25 G5+E3:1 E5:1 C5+A3:1 B4:1 A4+F3:1
		F5:0.75 F5:0.25 E5+C3:1 C5:1 D5+G3:1 C5+C3:2`},
}

// An event of a song: notes held for a length in beats.
type event struct {
	notes []int
	beats float64
}

func (s *song) events() []event {
	var evs []event
	for _, tok := range strings.Fields(s.notes) {
		name, length, _ := strings.Cut(tok, ":")
		beats, err := strconv.ParseFloat(length, 64)
		if err != nil {
			panic("piano: bad length in " + tok)
		}
		var notes []int
		if name != "r" {
			for _, n := range strings.Split(name, "+") {
				notes = append(notes, noteNumber(n))
			}
		}
		evs = append(evs, event{notes, beats})
	}
	return evs
}

// noteNumber returns the MIDI number of a name such as C4 or G#3.
func noteNumber(name string) int {
	semi := map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}[name[0]]
	rest := name[1:]
	if strings.HasPrefix(rest, "#") {
		semi++
		rest = rest[1:]
	}
	octave, err := strconv.Atoi(rest)
	if err != nil {
		panic("piano: bad note " + name)
	}
	return (octave+1)*12 + semi
}

// play plays s, pressing the keys on the main thread, on time: each event
// is scheduled from the start, not after the last, so the tune keeps its
// tempo.
func (p *piano) play(s *song) {
	p.stop()
	if p.win == nil {
		return
	}
	p.song = s
	go func() {
		beat := time.Duration(float64(time.Minute) / s.bpm)
		start := time.Now()
		at := time.Duration(0)
		for _, ev := range s.events() {
			length := time.Duration(ev.beats * float64(beat))
			if !p.at(s, start.Add(at), func() { p.songNotes(ev.notes, true) }) {
				return
			}
			// Held for most of its length, as a pianist would.
			if !p.at(s, start.Add(at+length*9/10), func() { p.songNotes(ev.notes, false) }) {
				return
			}
			at += length
		}
		p.win.Update(func() {
			if p.song == s {
				p.song = nil
			}
		})
	}()
}

// at runs fn on the main thread at t, unless s stopped meanwhile.
func (p *piano) at(s *song, t time.Time, fn func()) bool {
	time.Sleep(time.Until(t))
	done := make(chan bool, 1)
	p.win.Update(func() {
		if p.song != s {
			done <- false
			return
		}
		fn()
		done <- true
	})
	return <-done
}

// songNotes presses or lets go of the notes a song holds.
func (p *piano) songNotes(notes []int, down bool) {
	for _, n := range notes {
		if down {
			p.down(n)
			p.songHeld = append(p.songHeld, n)
		} else {
			p.up(n)
			for i, h := range p.songHeld {
				if h == n {
					p.songHeld = append(p.songHeld[:i], p.songHeld[i+1:]...)
					break
				}
			}
		}
	}
}

// stop stops the song playing, and lets go of what it holds.
func (p *piano) stop() {
	p.song = nil
	for _, n := range p.songHeld {
		p.up(n)
	}
	p.songHeld = nil
}
