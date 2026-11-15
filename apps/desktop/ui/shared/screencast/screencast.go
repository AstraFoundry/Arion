// Package screencast is what an illustration needs to play like a screen
// recording: values keyed along a loop, a camera that zooms toward where
// the pointer works without uncovering the stage's edges, and the pointer
// with the ring its clicks leave. Scenes compute a frame from the clock
// and draw it; nothing here keeps state between frames.
package screencast

import (
	"time"
	"unicode/utf8"

	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Point is a position on the stage, in DIPs.
type Point struct{ X, Y float32 }

// Lens is the camera: how far it zooms in, toward which point.
type Lens struct {
	Zoom  float32
	Focus Point
}

// Key holds a value from At milliseconds into the loop, easing from the
// key before it. Two keys with one value hold it still between them.
type Key[T any] struct {
	At    float32
	Value T
}

// Pt is the point (x, y).
func Pt(x, y float32) Point { return Point{X: x, Y: y} }

// View is a lens zoomed zoom times toward focus.
func View(zoom float32, focus Point) Lens { return Lens{Zoom: zoom, Focus: focus} }

// At is the key holding value from ms milliseconds into the loop.
func At[T any](ms float32, value T) Key[T] { return Key[T]{At: ms, Value: value} }

// How long a click presses the pointer and how long its ring spreads.
const (
	press  = 160 * time.Millisecond
	ripple = 450 * time.Millisecond
)

// Sample eases between the keys around at.
func Sample[T any](keys []Key[T], at time.Duration, mix func(a, b T, p float32) T) T {
	t := float32(at) / float32(time.Millisecond)
	if t <= keys[0].At {
		return keys[0].Value
	}
	for i := 1; i < len(keys); i++ {
		if t < keys[i].At {
			from, to := keys[i-1], keys[i]
			return mix(from.Value, to.Value, ui.EaseInOut((t-from.At)/(to.At-from.At)))
		}
	}
	return keys[len(keys)-1].Value
}

func lerp(a, b, p float32) float32 { return a + (b-a)*p }

// MixPoint is the point p of the way from a to b.
func MixPoint(a, b Point, p float32) Point { return Point{lerp(a.X, b.X, p), lerp(a.Y, b.Y, p)} }

// MixLens is the lens p of the way from a to b.
func MixLens(a, b Lens, p float32) Lens {
	return Lens{lerp(a.Zoom, b.Zoom, p), MixPoint(a.Focus, b.Focus, p)}
}

// Shift is the translation that brings the lens's focus to the middle of
// a width×height stage scaled around its center, kept so the stage's
// edges never come into view.
func Shift(l Lens, width, height float32) (float32, float32) {
	axis := func(focus, size float32) float32 {
		half := size / 2
		low, high := half/l.Zoom, size-half/l.Zoom
		focus = min(max(focus, low), high)
		return (half - focus) * l.Zoom
	}
	return axis(l.Focus.X, width), axis(l.Focus.Y, height)
}

// Cursor is the pointer at one moment: where it is, how far it is pressed
// and how far the ring of its last click has spread from RippleAt
// (negative without one).
type Cursor struct {
	At            Point
	Press, Ripple float32
	RippleAt      Point
}

// CursorAt is the pointer t into a loop where it follows path and clicks
// at each of clicks.
func CursorAt(path []Key[Point], clicks []time.Duration, t time.Duration) Cursor {
	cursor := Cursor{At: Sample(path, t, MixPoint), Ripple: -1}
	for _, click := range clicks {
		since := t - click
		if since < 0 {
			continue
		}
		if since < press {
			cursor.Press = 1 - float32(since)/float32(press)
		}
		if since < ripple {
			cursor.Ripple = float32(since) / float32(ripple)
			cursor.RippleAt = Sample(path, click, MixPoint)
		}
	}
	return cursor
}

// Stage lays scene out on a width×height stage seen through lens. The
// caller's box clips it.
func Stage(c *ui.Context, width, height float32, lens Lens, scene func()) ui.Element {
	x, y := Shift(lens, width, height)
	return ui.Column(c).Absolute().Top(0).Left(0).Size(width, height).TransformOrigin(0.5, 0.5).
		Scale(lens.Zoom, lens.Zoom).Translate(x, y).Children(scene)
}

// Pointer draws the arrow and the ring a click leaves.
func Pointer(c *ui.Context, tokens theme.Tokens, cursor Cursor) {
	if cursor.Ripple >= 0 {
		size := 14 + 22*cursor.Ripple
		ui.Box(c).Absolute().Left(cursor.RippleAt.X-size/2).Top(cursor.RippleAt.Y-size/2).Size(size, size).Radius(size/2).
			Border(2, tokens.AccentInk).Opacity(1 - cursor.Ripple).PassThrough()
	}
	scale := 1 - 0.15*cursor.Press
	// The arrow's tip sits at (4, 2) of its 20 DIP box.
	ui.Image(c, icons.Pointer).Absolute().Left(cursor.At.X-4).Top(cursor.At.Y-2).Size(20, 20).
		TransformOrigin(0.2, 0.1).Scale(scale, scale).PassThrough()
}

// Head returns the first fraction of text's runes, as text streaming in.
func Head(text string, fraction float32) string {
	n := int(float32(utf8.RuneCountInString(text)) * fraction)
	return string([]rune(text)[:n])
}
