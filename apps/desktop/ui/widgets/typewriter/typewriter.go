// Package typewriter illustrates dictation set up for the first time: a
// studio microphone wired to a typewriter, drawn live with the Anatomy
// kit's rules (ui/shared/iso). It arrives with its back to the viewer and
// turns round 180°, the camera closes in, and it comes to rest square in
// front of the paper, where the typewriter types Herodotus's story of
// Arion (Histories 1.23–24) in the original Greek, in a serif, for as long
// as the figure shows.
package typewriter

import (
	"image"
	"math"
	"time"

	"arion.local/desktop/ui/shared/assets"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/iso"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// The arrival, from when the figure first shows: it turns round, closes
// in, then settles square in front of the paper, and starts to type.
const (
	turnFrom, turnTo   = 250 * time.Millisecond, 1450 * time.Millisecond
	zoomFrom, zoomTo   = 1000 * time.Millisecond, 2700 * time.Millisecond
	frontFrom, frontTo = 1700 * time.Millisecond, 3000 * time.Millisecond
	typingFrom         = 3200 * time.Millisecond
)

// The camera at the overview and in front of the paper.
const (
	overviewAzimuth, frontAzimuth     = 124, 90
	overviewElevation, frontElevation = 30, tilt    // square to the paper
	frontSpan                         = paperW + 12 // world units across the box, in front
)

// still is how far the typing has got in a still figure (Reduce Motion):
// a few lines typed, the carriage back at the margin.
const stillLines = 6

// pose is the figure at one frame.
type pose struct {
	turn, zoom, front float32
	typed             state
}

// text is the story set for the paper.
var text = typeset(assets.Arion, (paperW-2*margin)/em)

// Render fills its box with the figure, described by label. Since is when
// the step it illustrates was entered: the arrival plays from then, and
// the typing starts once it is over and goes on as long as it shows.
func Render(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, label copy.Key, since time.Time) {
	box := ui.Box(c).Fill().Clip().Background(tokens.Card).Role(ui.RoleImage).Label(messages.Text(label))
	at := poseAt(c.Now().Sub(since), c.Preferences().ReduceMotion)
	if !c.Preferences().ReduceMotion {
		elapsed := c.Now().Sub(since)
		if elapsed < typingFrom {
			c.AnimationFrame()
		} else if wait := text.next(elapsed - typingFrom); wait > 0 {
			c.After(wait)
		} else {
			c.AnimationFrame()
		}
	}
	ink := colorsOf(tokens)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Image(view.show(r, p.Scale(), frame{ink, at}), r, ui.FillBox)
	})
}

func poseAt(elapsed time.Duration, still bool) pose {
	if still {
		return pose{turn: 1, zoom: 1, front: 1, typed: text.at(text.lines[stillLines].start)}
	}
	phase := func(from, to time.Duration) float32 {
		return ui.EaseInOut(min(1, max(0, float32(elapsed-from)/float32(to-from))))
	}
	at := pose{turn: phase(turnFrom, turnTo), zoom: phase(zoomFrom, zoomTo), front: phase(frontFrom, frontTo)}
	at.typed = text.at(max(0, elapsed-typingFrom))
	if elapsed < typingFrom {
		// The paper waits at the margin of its first line.
		at.typed = text.at(0)
	}
	return at
}

// frame is everything a picture depends on: the same frame is not drawn
// twice, so the figure between two strokes costs nothing.
type frame struct {
	ink colors
	at  pose
}

// picture keeps the figure's bitmap. Two images take turns, as the GPU
// may still read the one last shown.
type picture struct {
	canvas iso.Canvas
	images [2]*image.RGBA
	turn   int
	bitmap *ui.Bitmap
	last   frame
	size   image.Point
}

// view is the one figure onboarding shows.
var view picture

func (v *picture) show(r ui.Rect, scale float32, f frame) *ui.Bitmap {
	size := image.Pt(int(math.Ceil(float64(r.W*scale))), int(math.Ceil(float64(r.H*scale))))
	if v.bitmap != nil && size == v.size && f == v.last {
		return v.bitmap
	}
	if size != v.size {
		v.images = [2]*image.RGBA{image.NewRGBA(image.Rectangle{Max: size}), image.NewRGBA(image.Rectangle{Max: size})}
		v.size = size
	}
	v.turn = 1 - v.turn
	img := v.images[v.turn]
	v.canvas.Begin(img, ui.Rect{W: r.W, H: r.H}, scale)
	draw(&v.canvas, ui.Rect{W: r.W, H: r.H}, f.ink, f.at)
	v.canvas.End()
	if v.bitmap == nil {
		v.bitmap = ui.NewBitmap(img)
	} else {
		v.bitmap.Update(img)
	}
	v.last = f
	return v.bitmap
}

// colors are the figure's inks, all from the Sunbeam tokens.
type colors struct {
	body, metal, rubber, lit iso.Palette
	paper, back, well, halo  ui.Color
	ink                      [3]ui.Color
	legend, ribbon           ui.Color
}

func colorsOf(t theme.Tokens) colors {
	shade, well, top, edge := float32(0.16), float32(0.12), t.Card, t.Faint
	paper := t.Card
	if t.Dark {
		shade, well, top, edge = 0.5, 0.5, t.Card.Mix(t.Text, 0.05), t.Faint.Mix(t.Card, 0.35)
		paper = t.Card.Mix(t.Text, 0.1)
	}
	body := iso.Palette{Top: top.Mix(t.Shade, 0.04), Shadow: t.Card.Mix(t.Shade, shade), Lit: t.Card.Mix(t.Shade, shade/4), Edge: edge, Crease: t.CardBorder}
	metal := iso.Palette{Top: top.Mix(t.Text, 0.06), Shadow: t.Card.Mix(t.Shade, shade*1.3).Mix(t.Text, 0.08), Lit: t.Card.Mix(t.Text, 0.04), Edge: edge, Crease: t.CardBorder}
	rubber := iso.Palette{Top: t.Card.Mix(t.Shade, 0.5), Shadow: t.Card.Mix(t.Shade, 0.72), Lit: t.Card.Mix(t.Shade, 0.56), Edge: edge.Mix(t.Shade, 0.4), Crease: edge}
	lit := iso.Palette{Top: top.Mix(t.Accent, 0.3), Shadow: t.Card.Mix(t.Accent, 0.34).Mix(t.Shade, shade), Lit: t.Card.Mix(t.Accent, 0.36), Edge: t.AccentInk, Crease: t.AccentInk.Alpha(0.5)}
	return colors{
		body: body, metal: metal, rubber: rubber, lit: lit,
		paper: paper, back: paper.Mix(t.Shade, shade*0.6),
		well:   t.Card.Mix(t.Shade, well+0.12),
		halo:   t.Shade.Alpha(map[bool]float32{false: 0.08, true: 0.22}[t.Dark]),
		ink:    [3]ui.Color{t.Text, t.Text.Alpha(0.86), t.Text.Alpha(0.74)},
		legend: edge, ribbon: t.Card.Mix(t.Shade, 0.62),
	}
}

func lerp(a, b, t float32) float32 { return a + (b-a)*t }
