package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/scene"
)

// The stress scenes: windows of 900×640 DIPs at twice the density whose
// every frame changes what they show, as an animation, a live dashboard
// or a list scrolling do. BenchmarkStressFrame times their frames built,
// laid out and painted, and BenchmarkStressFrameGPU (on macOS and Windows) the GPU
// drawing them.
var stressScenes = []struct {
	name string
	// view returns the scene's view, and step, which changes its state
	// before each frame.
	view func() (view func(c *Context), step func())
}{
	{"boxes", stressBoxes},
	{"labels", stressLabels},
	{"translated_column", stressTranslatedColumn},
	{"list", stressList},
	{"dashboard", stressDashboard},
}

const stressW, stressH, stressScale = 900, 640, 2

// stressBoxes moves 600 rounded boxes of 24 DIPs casting shadows, and
// changes their colors, every frame.
func stressBoxes() (func(c *Context), func()) {
	frame := 0
	view := func(c *Context) {
		Box(c).Fill().Background(RGB(246, 247, 249)).Children(func() {
			for i := range 600 {
				t := float32(frame + i)
				x := 8 + float32(i%30)*29 + float32(frame%40)*0.25
				y := 8 + float32(i/30)*31 + float32((frame+i)%20)*0.5
				Box(c).Absolute().Left(x).Top(y).Size(24, 24).Radius(6).
					Shadow(0, 2, 8, 0, RGBA(0, 0, 0, 0.27)).
					Background(RGB(uint8(40+int(t)*7%180), uint8(80+i*13%150), uint8(120+frame*3%120)))
			}
		})
	}
	return view, func() { frame++ }
}

// stressLabels shows a grid of 15×20 labels whose numbers change every
// frame.
func stressLabels() (func(c *Context), func()) {
	frame := 0
	view := func(c *Context) {
		Column(c).Fill().Padding(8).Gap(4).Children(func() {
			for r := range 20 {
				Row(c).Gap(4).Children(func() {
					for col := range 15 {
						Textf(c, "%d", (frame*31+r*15+col)*7919%100000).Width(54).FontSize(13)
					}
				})
			}
		})
	}
	return view, func() { frame++ }
}

// stressTranslatedColumn moves a column of 1000 rows, none virtualized,
// inside a box that clips it, by a translation changing every frame, as
// a scroll view of its own would.
func stressTranslatedColumn() (func(c *Context), func()) {
	frame := 0
	view := func(c *Context) {
		Box(c).Fill().Clip().Children(func() {
			Column(c).FillWidth().Translate(0, -float32(frame%2000)*3).Children(func() {
				for i := range 1000 {
					Row(c).Height(28).PaddingX(12).Gap(8).Children(func() {
						Textf(c, "Row %d", i).Grow(1)
						Text(c, "detail").TextColor(RGB(110, 110, 120))
					})
				}
			})
		})
	}
	return view, func() { frame++ }
}

// stressList scrolls a virtual List of 100,000 rows by 3 rows a frame.
func stressList() (func(c *Context), func()) {
	var s ListState
	frame := 0
	view := func(c *Context) {
		List(c, &s, 100_000).Fill().Rows(func(row ListRow) {
			Row(row.Context).Height(28).PaddingX(12).Gap(8).Children(func() {
				Textf(row.Context, "Row %d", row.Index).Grow(1)
				Text(row.Context, "detail").TextColor(RGB(110, 110, 120))
			})
		})
	}
	return view, func() {
		frame++
		s.ScrollTo(frame*3%99_000, Start)
	}
}

// stressDashboard shows 12 cards casting shadows, each with a title, a
// number and a progress bar that change every frame.
func stressDashboard() (func(c *Context), func()) {
	frame := 0
	view := func(c *Context) {
		Column(c).Fill().Padding(16).Gap(16).Background(RGB(242, 243, 246)).Children(func() {
			for r := range 3 {
				Row(c).Gap(16).Grow(1).Children(func() {
					for col := range 4 {
						i := r*4 + col
						v := (frame*(i+3) + i*17) % 1000
						Column(c).Grow(1).Basis(0).Padding(14).Gap(8).Radius(10).Background(RGB(255, 255, 255)).
							Shadow(0, 4, 16, 0, RGBA(0, 0, 0, 0.12)).Children(func() {
							Textf(c, "Metric %d", i+1).FontSize(12).TextColor(RGB(110, 110, 120))
							Textf(c, "%d.%d%%", v/10, v%10).FontSize(28).Bold()
							Progress(c, float64(v)/1000).FillWidth()
							Textf(c, "%d requests, %d errors", v*13, v%7).FontSize(12)
						})
					}
				})
			}
		})
	}
	return view, func() { frame++ }
}

// stressHost is a window's host without a screen: present hands each
// frame's scene to render, if set, rather than drawing it with the
// software renderer as a Tester's host does.
type stressHost struct {
	*headless
	render func(*scene.Scene)
}

func (h *stressHost) present(s *scene.Scene) {
	if h.render != nil {
		h.render(s)
	}
}

// stressFrames runs the frames of a stress scene, a 60th of a second
// apart, presenting them to render.
type stressFrames struct {
	rt   *engine
	step func()
	now  time.Time
}

func newStressFrames(build func() (func(c *Context), func()), render func(*scene.Scene)) *stressFrames {
	view, step := build()
	h := &stressHost{headless: &headless{w: stressW, h: stressH, scale: stressScale, hz: 60}, render: render}
	f := &stressFrames{step: step, now: time.Unix(1000, 0)}
	f.rt = newRuntime(func(c *context) { view(makeContext(c)) }, h)
	f.rt.clock = func() time.Time { return f.now }
	// The first frames lay out, shape and cache what the next ones reuse.
	for range 3 {
		f.next()
	}
	return f
}

// next changes the scene's state and runs a frame of it.
func (f *stressFrames) next() {
	f.step()
	f.now = f.now.Add(time.Second / 60)
	f.rt.runFrame()
}

// BenchmarkStressFrame builds, lays out and paints the frames of the
// stress scenes: what a frame costs the CPU before the GPU draws it.
func BenchmarkStressFrame(b *testing.B) {
	for _, sc := range stressScenes {
		b.Run(sc.name, func(b *testing.B) {
			f := newStressFrames(sc.view, nil)
			b.ReportAllocs()
			for b.Loop() {
				f.next()
			}
		})
	}
}

// TestStressScenes checks the stress scenes show what they are meant to,
// and change every frame.
func TestStressScenes(t *testing.T) {
	for _, sc := range stressScenes {
		t.Run(sc.name, func(t *testing.T) {
			var last *scene.Scene
			var sums []string
			f := newStressFrames(sc.view, func(s *scene.Scene) { last = s })
			for range 2 {
				f.next()
				if last.Width != stressW*stressScale || last.Height != stressH*stressScale {
					t.Fatalf("a scene of %d×%d", last.Width, last.Height)
				}
				sums = append(sums, fmt.Sprint(last.Ops, last.Glyphs))
			}
			if sums[0] == sums[1] {
				t.Error("two frames drew the same")
			}
			var fills, shadows, glyphs int
			for _, op := range last.Ops {
				switch op.Kind {
				case scene.OpFill:
					fills++
				case scene.OpShadow:
					shadows++
				case scene.OpGlyphs:
					glyphs += int(op.End - op.Start)
				}
			}
			t.Logf("%d ops: %d fills, %d shadows, %d glyphs", len(last.Ops), fills, shadows, glyphs)
			switch sc.name {
			case "boxes":
				if fills < 600 || shadows < 600 {
					t.Errorf("%d fills, %d shadows, want 600 boxes", fills, shadows)
				}
			case "dashboard":
				if shadows < 12 {
					t.Errorf("%d shadows, want 12 cards", shadows)
				}
			case "labels":
				if glyphs < 300*3 {
					t.Errorf("%d glyphs, want 300 numbers", glyphs)
				}
			default:
				// The rows that show, about 22.
				if glyphs < 200 {
					t.Errorf("%d glyphs, want the rows that show", glyphs)
				}
			}
		})
	}
}
