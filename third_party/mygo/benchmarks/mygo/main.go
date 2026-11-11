// Command mygo is MyGo's app for the comparison of benchmarks/README.md, with
// the scenes the other frameworks' apps have.
package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// recorder times frames and says on standard output how the run goes.
type recorder struct {
	warmup, frames int
	seen           int
	last, started  time.Time
	intervals      []time.Duration
	// at is the time of the frame being built, which its build passes share,
	// and more whether the run wants another frame.
	at   time.Time
	more bool
}

func newRecorder() *recorder {
	return &recorder{warmup: env("BENCH_WARMUP", 120), frames: env("BENCH_FRAMES", 600)}
}

func env(name string, fallback int) int {
	if n, err := strconv.Atoi(os.Getenv(name)); err == nil {
		return n
	}
	return fallback
}

// frame is called by every build pass, and reports whether this pass starts a
// frame the scene should advance for: a frame can have several passes.
func (r *recorder) frame(c *ui.Context) bool {
	now := c.Now()
	started := !now.Equal(r.at)
	if started {
		r.at = now
		r.more = r.tick()
	}
	if r.more {
		c.AnimationFrame()
	}
	return started && r.more
}

// tick counts a frame, and reports whether the run wants another.
func (r *recorder) tick() bool {
	now := time.Now()
	r.seen++
	switch {
	case r.seen == r.warmup:
		fmt.Println("start")
		r.started = now
	case r.seen > r.warmup && r.seen <= r.warmup+r.frames:
		r.intervals = append(r.intervals, now.Sub(r.last))
		if len(r.intervals) == r.frames {
			slices.Sort(r.intervals)
			fmt.Printf(`done {"frames":%d,"wallUs":%d,"p50Us":%d,"p99Us":%d,"maxUs":%d}`+"\n",
				r.frames, now.Sub(r.started).Microseconds(),
				r.intervals[r.frames/2].Microseconds(), r.intervals[r.frames*99/100].Microseconds(), r.intervals[r.frames-1].Microseconds())
		}
	}
	r.last = now
	return r.seen <= r.warmup+r.frames
}

var wordList = [...]string{
	"window", "button", "layout", "render", "scroll", "update", "shadow", "cursor",
	"memory", "thread", "signal", "bitmap", "stroke", "column", "editor", "kernel",
	"packet", "vector", "syntax", "buffer", "format", "cipher", "socket", "module",
	"branch", "commit", "server", "client", "screen", "pointer", "display", "palette",
}

// words is the text of a seed, the same in every framework's app.
func words(seed, n int) string {
	x := uint32(seed)
	var b []byte
	for i := range n {
		x = x*1103515245 + 12345
		if i > 0 {
			b = append(b, ' ')
		}
		b = append(b, wordList[(x>>16)%uint32(len(wordList))]...)
	}
	return string(b)
}

func cellValue(cell, frame int) float64 {
	return float64((cell*7919+frame*31)%100000) / 100
}

// counter: the smallest app, for starting and idling.
type counter struct{ n int }

func (s *counter) view(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(16).Children(func() {
		ui.Textf(c, "%d", s.n).FontSize(56).Bold()
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c.Key("decrement"), "−").Width(44).Clicked() {
				s.n--
			}
			if ui.Button(c.Key("reset"), "Reset").Clicked() {
				s.n = 0
			}
			if ui.Button(c.Key("increment"), "+").Width(44).Clicked() {
				s.n++
			}
		})
	})
}

// list: 100,000 rows of text, scrolling 8 DIPs a frame.
type list struct {
	rec    *recorder
	state  ui.ListState
	scroll ui.ScrollState
}

func (s *list) view(c *ui.Context) {
	if s.rec.frame(c) {
		s.scroll.Y += 8
	}
	muted := c.Theme().TextMuted
	ui.List(c, &s.state, 100_000, func(i int) {
		ui.Row(c).Height(40).Padding(0, 12).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).Children(func() {
				ui.Textf(c, "Item %d", i).FontSize(13).NoWrap()
				ui.Text(c, words(i+1, 6)).FontSize(11).TextColor(muted).NoWrap()
			})
			ui.Textf(c, "%.2f", cellValue(i, 0)).FontSize(13).NoWrap()
		})
	}).Fill().TrackScroll(&s.scroll)
}

// grid: 600 cells of a live dashboard, each changing every frame.
type grid struct {
	rec   *recorder
	frame int
}

func (s *grid) view(c *ui.Context) {
	if s.rec.frame(c) {
		s.frame++
	}
	ui.Column(c).Fill().Children(func() {
		for row := range 30 {
			ui.Row(c).Children(func() {
				for col := range 20 {
					v := cellValue(row*20+col, s.frame)
					ui.Box(c).Size(40, 20).Center().Background(cellColor(v)).Children(func() {
						ui.Textf(c, "%.2f", v).FontSize(11).TextColor(ui.RGB(0x10, 0x10, 0x10)).NoWrap()
					})
				}
			})
		}
	})
}

func cellColor(v float64) ui.Color {
	switch {
	case v < 333:
		return ui.RGB(0xf6, 0xc9, 0xc4)
	case v < 666:
		return ui.RGB(0xf4, 0xe8, 0xb8)
	}
	return ui.RGB(0xc6, 0xe9, 0xc8)
}

// document: 40 paragraphs wrapping to a width that changes every frame, as
// when a window of text is resized.
type document struct {
	rec        *recorder
	paragraphs []string
	frame      int
}

func (s *document) view(c *ui.Context) {
	if s.rec.frame(c) {
		s.frame++
	}
	width := float32(math.Round(600 + 150*math.Sin(float64(s.frame)/30)))
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Width(width).Gap(8).Padding(12).Children(func() {
			for _, p := range s.paragraphs {
				ui.Text(c, p).FontSize(13).Wrap()
			}
		})
	})
}

func main() {
	scene := "counter"
	if len(os.Args) > 1 {
		scene = os.Args[1]
	}
	var content mygo.Content
	width, height := 800, 600
	switch scene {
	case "counter":
		width, height = 360, 260
		content = ui.View((&counter{}).view)
	case "list":
		content = ui.View((&list{rec: newRecorder()}).view)
	case "grid":
		content = ui.View((&grid{rec: newRecorder()}).view)
	case "document":
		d := &document{rec: newRecorder()}
		for i := range 40 {
			d.paragraphs = append(d.paragraphs, words(i+1, 70))
		}
		content = ui.View(d.view)
	default:
		log.Fatalf("no scene %q", scene)
	}
	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{Title: "Benchmark", Width: width, Height: height, UseContentSize: true, Content: content})
		// Called once the first frame is drawn. Windows show from here too.
		w.OnReadyToShow(func() { fmt.Println("ready") })
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
