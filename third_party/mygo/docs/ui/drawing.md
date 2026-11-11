# Drawing and animation

`Draw` paints on an element after its background, and `DrawOver` after its
children, with a `*ui.Painter` in DIPs of the window: rectangles with
`Fill`, `FillGradient`, `FillConicGradient` and `FillRadialGradient`,
`Stroke` and `StrokeDashed`, `Shadow`, `Line` (horizontal or vertical),
`Image`, `Clip` and `ClipPath`, paths of lines and curves with `FillPath` and
`StrokePath`, or in a gradient with `FillPathGradient` and
`StrokePathGradient`, and text with `Text`, or `RichText` in spans of any
style, which `MeasureText` measures first, to center or align it:

```go
ui.Box(c).Height(120).Draw(func(p *ui.Painter, r ui.Rect) {
	var wave ui.Path
	for i := 0; i <= 100; i++ {
		x := r.X + r.W*float32(i)/100
		y := r.Y + r.H/2 + 40*float32(math.Sin(float64(i)/8))
		if i == 0 {
			wave.MoveTo(x, y)
		} else {
			wave.LineTo(x, y)
		}
	}
	p.StrokePath(&wave, 2, c.Theme().Accent)
})
```

```go
ui.Box(c).Size(200, 40).Draw(func(p *ui.Painter, r ui.Rect) {
	label := ui.Span{Text: "42%", Weight: 600, Color: c.Theme().TextMuted}
	w, h := p.MeasureText(0, label)
	p.RichText(r.X+(r.W-w)/2, r.Y+(r.H-h)/2, 0, label)
})
```

Paragraphs a view lays out itself, as the pages of a book reader, are
shaped once with `ui.ShapeRichTextWith`, justified and indented if need
be, and drawn with `Painter.TextLayout`; their `Lines` tell where each
line is, to break pages between them (see [retained text
geometry](text-input-client.md#retained-text-geometry)).

Draw functions only paint: MyGo may call them more than once a frame.
`Painter.Icon` and `Painter.Image` draw SVGs and bitmaps,
`Painter.ImageRegion` parts of bitmaps (see [Parts of
bitmaps](#parts-of-bitmaps)), and
`Painter.FocusRing` the ring of the keyboard focus, as the theme draws it
(see [the focus ring](styling.md#the-focus-ring)), for widgets that draw
their own.

`Path.Trim(from, to)` keeps the part of a path's length from `from` to
`to`, as fractions of it, which draws a path in as `to` animates from 0 to
1, as SwiftUI's `trim` and Motion's `pathLength`:

```go
drawn := float32(0)
if app.done {
	drawn = 1
}
drawn = box.AnimateTo("check", drawn, ui.Smooth)
box.Draw(func(p *ui.Painter, r ui.Rect) {
	check := (&ui.Path{}).MoveTo(r.X+4, r.Y+12).LineTo(r.X+10, r.Y+18).LineTo(r.X+20, r.Y+6)
	p.StrokePath(check.Trim(0, drawn), 2, c.Theme().Accent)
})
```

A stroke has round ends and corners unless its path sets them, as SVG's
`stroke-linecap`, `stroke-linejoin` and `stroke-miterlimit` do:
`Path.Cap` ends it flat (`ui.ButtCap`) or square (`ui.SquareCap`), and
`Path.Join` turns its corners in points (`ui.MiterJoin`, bevelled past
`Path.MiterLimit`) or cuts them (`ui.BevelJoin`). `Path.Dash` strokes it
in dashes, as `stroke-dasharray` and `stroke-dashoffset` do, and they stay
where they are as `Trim` draws the path in:

```go
box.Draw(func(p *ui.Painter, r ui.Rect) {
	edge := (&ui.Path{}).MoveTo(r.X, r.Y+r.H/2).LineTo(r.X+r.W, r.Y+r.H/2)
	p.StrokePath(edge.Cap(ui.ButtCap).Dash(0, 6, 4), 1.5, c.Theme().TextMuted)
})
```

## Fill rules and hits

A path fills, and clips, the points its subpaths wind around (SVG's
`nonzero`), so a subpath inside another one leaves a hole only when it
goes the other way round. `Path.FillRule(ui.EvenOdd)` fills the points
its subpaths go around an odd number of times instead, as SVG's
`fill-rule` and `clip-rule: evenodd` do, which makes a ring or a donut of
two circles, and a shape with a cut-out, whichever way their path data
draws them:

```go
box.Draw(func(p *ui.Painter, r ui.Rect) {
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	donut := (&ui.Path{}).Circle(cx, cy, r.W/2).Circle(cx, cy, r.W/4).FillRule(ui.EvenOdd)
	p.FillPath(donut, c.Theme().Accent)
})
```

`Path.Contains(x, y)` reports whether a point is inside a path as it
fills, by its rule, as the web's `isPointInPath`: a view's handler can
tell which segment of a pie a press is in. `HitTest` makes an element
take the pointer only where a function says, given the pointer in the
coordinates `Draw` paints in, so a pie's segments, a map's regions or a
ring take presses, hover and the cursor only where they paint, as SVG's
`pointer-events: visiblePainted`, and the pointer reaches what is under
them elsewhere:

```go
seg := func(r ui.Rect) *ui.Path { return slice(r, from, to) } // a segment's outline
ui.Box(c).Absolute().Fill().
	Draw(func(p *ui.Painter, r ui.Rect) { p.FillPath(seg(r), color) }).
	HitTest(func(x, y float32, r ui.Rect) bool { return seg(r).Contains(x, y) }).
	OnClick(func() { app.pick(i) })
```

## Paths from SVG data

`ui.ParsePath(d)` reads the path data of an SVG `<path d="…">` (all of
its commands, absolute and relative, arcs too) into a `*ui.Path`, so an
icon's outline copied from an SVG can be filled, stroked, trimmed or
clipped to like any path. It returns an error that gives the offset of
the first thing it can't read; `ui.MustParsePath` panics instead, for
data written into the program:

```go
var heart = ui.MustParsePath("M12 21 L10.6 19.7 C5.4 15 2 11.9 2 8.1 C2 5 4.4 2.6 7.5 2.6 " +
	"C9.2 2.6 10.9 3.4 12 4.7 C13.1 3.4 14.8 2.6 16.5 2.6 C19.6 2.6 22 5 22 8.1 " +
	"C22 11.9 18.6 15 13.4 19.7 Z")

box.Draw(func(p *ui.Painter, r ui.Rect) {
	p.Transform(ui.Scaling(r.W/24, r.H/24).Then(ui.Translation(r.X, r.Y)), 0, 0, func() {
		p.StrokePath(heart.Trim(0, drawn), 1.5, c.Theme().Accent)
	})
})
```

The path is in the data's own units; a transform scales it to the box.

## Clipping to a path

`Painter.Clip(r, radius, fn)` clips what `fn` draws to a rounded
rectangle. `Painter.ClipPath(path, fn)` clips it to the inside of any
path (by its `FillRule`), with smooth edges, as SVG's and CSS's
`clip-path`: a chart drawn into a polygon, or a picture cut to a star.

```go
star := ui.MustParsePath("M50 0 L61 35 L98 35 L68 57 L79 91 L50 70 L21 91 L32 57 L2 35 L39 35 Z")
ui.Box(c).Size(100, 100).Draw(func(p *ui.Painter, r ui.Rect) {
	p.Transform(ui.Translation(r.X, r.Y), 0, 0, func() {
		p.ClipPath(star, func() {
			p.FillGradient(ui.Rect{W: 100, H: 100}, ui.LinearGradient{From: gold, To: orange, Angle: 135}, 0)
		})
	})
})
```

Path clips nest with each other and with `Clip`, and turn with a
transform. The clip is a mask of the path's coverage that every
renderer samples, made once for each shape and size it is drawn at and
kept while frames draw it, so a still path costs nothing to clip to
again. A moving or changing one makes a new mask every frame, sized to
the path in device pixels (at most 8192 each way).

To clip an element and its children rather than a drawing, see
`ClipInset` and the gradient masks in [styling](styling.md).

## Parts of bitmaps

A bitmap can hold many pictures: the icons of a sheet, the frames of an
animation, the cells of a contact sheet. `Painter.ImageRegion` draws one
part of it, `src` in the bitmap's pixels, into a rectangle; nothing of the
bitmap around the part shows, smooth or pixelated. Its `ImageOptions`
mirror the part (`FlipX`, `FlipY`) and tint it (`Tint` multiplies its
colors and alpha):

```go
icons, _ := ui.DecodeBitmap(iconsPNG) // 16×16 icons in a row
icon := func(i int) ui.Rect { return ui.Rect{X: float32(i * 16), W: 16, H: 16} }

box.Draw(func(p *ui.Painter, r ui.Rect) {
	p.ImageRegion(icons, icon(3), ui.Rect{X: r.X, Y: r.Y, W: 24, H: 24},
		ui.ImageOptions{Tint: c.Theme().TextMuted})
})
```

A part outside what the painter shows draws nothing, so a view of many
can draw them all without culling them itself, as games draw their
sprites and tiles.

## Canvases

A `ui.Canvas` is a bitmap a painter draws, on the GPU where the window
draws there: render to texture, as Skia's `SkSurface` and LÖVE's
`Canvas`. `Draw` draws it anew during a frame, with a painter in its
pixels, and it keeps what it holds, showing anywhere a bitmap does through
`Bitmap`:

```go
grid := ui.NewCanvas(800, 400)
drawn := false

box.Draw(func(p *ui.Painter, r ui.Rect) {
	if !drawn { // once, then every frame for the price of an image
		drawn = true
		grid.Draw(p, func(cp *ui.Painter) { drawGrid(cp, 800, 400) })
	}
	p.Image(grid.Bitmap(), r, ui.FillBox)
	drawSeries(p, r)
})
```

A canvas drawn with a tint or an opacity fades or colors a whole layer at
once. Canvases show other canvases, and text. A canvas's pixels are its
units: draw it at the window's scale (`Painter.Scale`) for one as sharp as
the window, or smaller and make its bitmap `Pixelated` to scale pixel art
up crisp (the [game plugin](../plugins/game.md#pixel-perfect-scaling)
shows one at whole multiples). A [shader](shaders.md) reads canvases as
images, and draws into them.

## Transforms

`Painter.Transform(t, ox, oy, fn)` transforms what `fn` draws by `t`, a
`ui.Transform` such as `Translation`, `Scaling` or `Rotation`, around the
point `(ox, oy)`, which stays where it is. Transforms nest, after the
painter's own, so a view that pans and zooms, as a map or a diagram,
draws its content through one, and turns a part of it inside:

```go
ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
	view := ui.Translation(-doc.panX, -doc.panY).Then(ui.Scaling(doc.zoom, doc.zoom))
	p.Transform(view, r.X, r.Y, func() {
		for _, n := range doc.nodes {
			p.Fill(n.box, n.color, 6) // culled where the view does not look
		}
		cx, cy := doc.dial.X, doc.dial.Y
		p.Transform(ui.Rotation(doc.dial.angle), cx, cy, func() {
			p.Line(cx, cy, cx, cy-40, 2, c.Theme().Accent)
		})
	})
})
```

Clips made inside turn with what they clip. Input is not transformed: the
pointer comes in the element's coordinates, which the app maps to its
content with the inverse of its view. Under a transform, images and their
parts are placed where they are rather than on whole pixels, so that one
moving through a zoomed view glides.

## Shaping text

Text that a widget lays out itself, as in the cells of a grid, is shaped
once with `ui.Shape`, which returns glyphs placed along a line by the
system's text engine, with ligatures, kerning and fallback fonts, and the
runes each comes from. The glyphs come in visual order on every platform,
from the left: through Hebrew or Arabic their `Cluster` descends, and
`RTL` marks them, so find a range of runes' glyphs by `Cluster` and
`Runes`. Move their `X` and draw them with `Painter.Glyphs`;
`Font.Metrics` returns the font's ascent, descent and line gap, and
`Painter.Scale` the device pixels of a DIP, to line things up with the
display's pixels. `Shape` caches nothing, unlike the text of elements: keep
the glyphs of text drawn in many frames. `Font.Features` turns OpenType
features on and off, and `Font.Thicken` draws text with a thicker stroke on
macOS, as Ghostty's `font-thicken`, and `Font.Antialiased` a thinner one,
without font smoothing, as browsers draw text styled
`-webkit-font-smoothing: antialiased`.

```go
font := ui.Font{Family: "monospace", Size: 13}
glyphs := ui.Shape("grid", font)
for i := range glyphs {
	glyphs[i].X = float32(glyphs[i].Cluster) * cellWidth // one per cell
}
ui.Box(c).Height(20).Draw(func(p *ui.Painter, r ui.Rect) {
	p.Glyphs(glyphs, r.X, r.Y+font.Metrics().Ascent, c.Theme().Text)
})
```

## Animation

For motion, `Element.Animate` returns a value that eases to a target and
draws frames until it gets there:

```go
panel := ui.Column(c).Clip()
width := float32(0)
if app.sidebar {
	width = 280
}
panel.Width(panel.Animate("width", width, 200*time.Millisecond))
```

`AnimateWith` takes the easing: `ui.Linear`, `ui.EaseIn`, `ui.EaseOut` (as
`Animate`), `ui.EaseInOut`, `ui.CubicBezier(x1, y1, x2, y2)`, any
`func(t float32) float32`, or `ui.Bounce(e)`, which goes along `e` and
comes back. `AnimateTo` takes springs as well (see [Motion](motion.md)). `Loop` returns the
progress of an animation that starts over every period, for spinners and
pulses; `Rotate` turns an element, as an icon:

```go
spin := ui.Icon(c, loader)
spin.Rotate(spin.Loop("spin", time.Second, ui.Linear) * 360)

skeleton := ui.Box(c).Height(14).Radius(7).Background(t.Border)
skeleton.Opacity(0.4 + 0.6*skeleton.Loop("pulse", 1600*time.Millisecond, ui.Bounce(ui.EaseInOut)))
```

To animate in other ways, compute from `c.Now()` and call
`c.AnimationFrame()` in every frame that moves: MyGo draws the next frame
when the display can show it, and draws nothing while nothing changes.

A drawing that moves while the layout stays, as a spinner or a chart
scrolling by, animates from its `Draw` instead: compute it from `p.Now()`
and call `p.AnimationFrame()` while it moves, or `p.After(d)` when it
changes in steps. Unless something else changed, those frames paint the
elements of the last frame again without running the view, so they cost
only the painting; and an element out of view is not painted, so it asks
for none. While the window is out of sight (hidden, minimized, or covered
by other windows), MyGo draws no frames at all, from `Draw` or the view,
until some of it shows again: compute what moves from the time, as here,
and it shows where the time puts it (see [Motion](motion.md#out-of-sight)):

```go
ui.Box(c).Height(4).Draw(func(p *ui.Painter, r ui.Rect) {
	p.AnimationFrame()
	x := r.X + (r.W-20)*float32(p.Now().UnixMilli()%1000)/1000
	p.Fill(ui.Rect{X: x, Y: r.Y, W: 20, H: r.H}, c.Theme().Accent, 2)
})
```

A drawing that looks as smooth at fewer frames, as a record turning, a
slow wave or a level meter, can ask for them with
`p.AnimationFrameRate(fps)` in place of `p.AnimationFrame()`: the window
paints it again that often, rather than at each of the display's
refreshes (120 a second on ProMotion displays), and on macOS has the
display refresh as often, which saves power. The rate rounds up to one the
display shows evenly, a whole number of refreshes a frame: 60, 40 or 30
frames a second on a 120 Hz display, 60 or 30 on a 60 Hz one. The window
draws at the fastest rate anything in it asks for, so a drawing asking for
`AnimationFrame` or a view that animates still draws at the display's.
Pacing with `p.After(time.Second / 60)` instead draws unevenly: a frame
asked for later than a refresh waits for the next.

Such a `Draw` sees the state as the view last ran: what it reads that
changes otherwise is the view's to build anew, after `Window.Update`.

When the desktop asks for less motion (`c.Preferences().ReduceMotion`),
`Animate` and `AnimateWith` go to their target at once. `Loop` goes on, as
the system's spinners do: it shows that something is going on. Motion you
compute yourself should read the preference too.

[Motion](motion.md) has springs that keep their velocity, delays, values
following the pointer, keyframes, and how to tell when an animation is
done; [Transforms](transforms.md) move, scale and rotate any element.
