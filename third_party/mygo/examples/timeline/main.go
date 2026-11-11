// Timeline shows what a ui.Timeline is for: several tweens on one playhead,
// arranged and controlled together.
//
//	go run ./examples/timeline
//
// The card slides in from the left as it grows to its size, its title
// follows it, the rows come one after another, and the card bounces a
// little as it lands. One playhead drives all of it, so Play, Pause,
// Reverse, Restart and the scrubber move the whole intro together. The
// timeline reads the frame's clock and asks the window for a frame only
// while it is moving, as the rest of MyGo's motion does.
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const windowW, windowH = 520, 480

// app is the window's state: what the intro's controls set, and what its
// last frame showed.
type app struct {
	run     uint64 // plays the intro again when it changes
	paused  bool
	reverse bool
	// scrub is the scrubber, held whether the pointer held it in the last
	// frame, and seek the playhead it asks for while it does.
	scrub float64
	held  bool
	seek  float32
}

func main() {
	s := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Timeline",
			Width:   windowW,
			Height:  windowH,
			Content: ui.View(s.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// view builds the window from the state, as every frame of the intro runs
// the view again.
func (s *app) view(c *ui.Context) {
	t := c.Theme()
	o := ui.TimelineOptions{Run: s.run, Paused: s.paused, Reverse: s.reverse}
	if s.held {
		// The pointer held the scrubber, so the playhead stays where it
		// put it, and goes on from there when it is let go.
		s.seek = float32(s.scrub)
		o.Progress = &s.seek
	}
	ui.Column(c).Fill().Center().Gap(18).Padding(24).Children(func() {
		// One playhead for the whole intro, kept by the card: the card
		// slides in as it grows, the title follows it, the rows come one
		// after another, and the card bounces a little as it lands.
		card := ui.Column(c).Key("card")
		tl := card.Timeline("intro", o)
		slide := tl.Tween(ui.Tween{From: -420, To: 0, Duration: 450 * time.Millisecond})
		grow := tl.Tween(ui.Tween{From: 0.9, To: 1, Duration: 450 * time.Millisecond}, ui.WithPrevious(0))
		fade := tl.Tween(ui.Tween{From: 0, To: 1, Duration: 300 * time.Millisecond, Ease: ui.Linear},
			ui.AtTime(150*time.Millisecond))
		tl.Label("rows") // where the card lands, at 450ms
		names := [3]string{"Plan", "Build", "Ship"}
		rows := [3]float32{}
		for i := range rows {
			rows[i] = tl.Tween(ui.Tween{From: 0, To: 1, Duration: 240 * time.Millisecond, Ease: ui.Linear},
				ui.AtLabel("rows", time.Duration(i)*90*time.Millisecond))
		}
		bounce := tl.Tween(ui.Tween{From: 10, To: 0, Duration: 360 * time.Millisecond, Ease: ui.Bounce(ui.EaseOut)},
			ui.AtTime(450*time.Millisecond))

		card.Width(300).Padding(22).Gap(10).
			Radius(16).Background(t.Surface).
			Translate(slide, bounce).Scale(grow, grow).
			Children(func() {
				ui.Text(c, "A timeline").FontSize(20).Bold().TextColor(t.Text).
					Opacity(fade).Translate(0, (1-fade)*12)
				for i, name := range names {
					ui.Row(c).Gap(8).Opacity(rows[i]).Translate(0, (1-rows[i])*14).Children(func() {
						ui.Text(c, name).TextColor(t.TextMuted)
					})
				}
			})
		ui.Row(c).Gap(8).Children(func() {
			if ui.PrimaryButton(c.Key("restart"), "Restart").Clicked() {
				s.run++
			}
			if ui.Button(c.Key("pause"), "Pause").Disabled(s.paused).Clicked() {
				s.paused = true
			}
			if ui.Button(c.Key("play"), "Play").Disabled(!s.paused).Clicked() {
				s.paused = false
			}
			if ui.Button(c.Key("reverse"), "Reverse").Clicked() {
				s.reverse = !s.reverse
			}
		})
		ui.Row(c).Gap(10).Children(func() {
			sl := ui.Slider(c, &s.scrub, 0, 1).Grow(1)
			s.held = sl.Dragging()
			if !s.held {
				s.scrub = float64(tl.Progress())
			}
			ui.Textf(c, "%d / %d ms", tl.Time().Milliseconds(), tl.Duration().Milliseconds()).
				FontSize(12).TextColor(t.TextMuted).Width(96)
		})
	})
}
