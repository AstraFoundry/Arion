# Shaders

A shader computes the color of every pixel of a rectangle, on the GPU,
from its position, values the app passes it each frame and images it
reads: noise and gradients that move, a texture lit pixel by pixel, a
ripple across the window. Package `ui/shader` compiles shaders written in
Go's syntax, and draws them on whichever GPU renderer draws the window
(Metal, Direct3D 11, OpenGL or WebGL 2), and on the CPU where none can,
the same pixels within rounding.

```go
import "github.com/egoist/mygo/ui/shader"

var waves = shader.MustCompile(`package main

var Time float

func Fragment(dst vec4, pos vec2) vec4 {
	v := 0.5 + 0.5*sin(pos.x/20+Time)
	return vec4(v*0.2, v*0.4, v, 1)
}
`)

ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
	waves.Draw(p, r, shader.Options{
		Uniforms: map[string]any{"Time": p.Now().Sub(start).Seconds()},
	})
	p.AnimationFrame()
})
```

`Draw` fills a rectangle of the painter, clipped, transformed and faded
with it as anything it paints; `Options.Radius` rounds its corners.

## Text, paths and icons

A shader also fills shapes in place of a color, as a gradient does, and
as SwiftUI fills a shape or text with a shader: `FillPath` and
`StrokePath` paint a `ui.Path`, `Text` a line of text, and `Fill`
whatever text, paths and icons a function paints, in a box of your own:

```go
var circle ui.Path
circle.Circle(60, 60, 50)

ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
	opts := shader.Options{Uniforms: map[string]any{"Time": t}}
	waves.FillPath(p, &circle, opts)
	waves.Text(p, 140, 40, "Hello", 48, opts)
	waves.Fill(p, ui.Rect{X: 140, Y: 100, W: 200, H: 40}, opts, func() {
		p.RichText(140, 100, 200, ui.Span{Text: "and more", Size: 24, Weight: 700})
		p.Icon(star, ui.Rect{X: 300, Y: 100, W: 24, H: 24}, ui.RGB(0, 0, 0))
	})
})
```

Each pixel shows the shader's color times how much of it the shape
covers, the edges of text corrected as the platform's text is for that
color. `pos` runs across the shape's box (the path's, grown by half the
stroke's width, the text's, or the one `Fill` takes), which `rectSize()`
returns; emoji keep their own colors, and rectangles and images `Fill`'s
function paints paint as they would.

## Effects on elements

A shader can also draw an element from what it renders, as SwiftUI's
`colorEffect`, `distortionEffect` and `layerEffect` do. Its entry point
says which, in place of `Fragment`:

```go
// The content's colors, changed pixel by pixel.
func ColorEffect(pos vec2, color vec4) vec4

// Where to read the content for the pixel at pos.
func DistortionEffect(pos vec2) vec2

// The pixel's color, from the content read anywhere.
func LayerEffect(pos vec2) vec4
```

`Element.Effect` takes the shader's `Effect`:

```go
var wave = shader.MustCompile(`package main

var Time float

func DistortionEffect(pos vec2) vec2 {
	return pos + vec2(0, 6*sin(pos.x/12+Time))
}
`)

ui.Text(c, "Wavy").FontSize(24).Effect(wave.Effect(shader.Options{
	Uniforms:        map[string]any{"Time": t},
	MaxSampleOffset: ui.Point{Y: 6},
}))
```

The element and its children, with their shadows and what their `Draw`
callbacks paint, render into a bitmap on the GPU every frame the element
paints, and the effect draws over it in place of the element. `pos` runs
from the corner of the element's box, whose size `rectSize()` returns. A
`LayerEffect` reads the content with `layerAt(p)`, its pixel at `p`, or
`layerSample(p)`, blended between pixels, `p` in DIPs as `pos` is; a
`DistortionEffect` reads it with `layerSample`. Content reads as
transparent beyond what renders: `Options.MaxSampleOffset` renders that
much more around the box, for an effect that reads or moves pixels that
far, and the effect draws over it all. The element stays where it is for
clicks, focus and accessibility: only what shows changes. An effect's
shader reads images 0 to 2 of its own; image 3 is the content.

## The backdrop

A shader can read what the window drew behind it, as Flutter's
`BackdropFilter` with a shader does: frosted panes, lenses, a tint that
follows what scrolls under it. `backdropAt(p)` returns its pixel at `p`,
and `backdropSample(p)` blends the pixels around `p`, `p` in DIPs as
`pos` is; `Options.BackdropBlur` blurs it first, its standard deviation
in DIPs:

```go
var frost = shader.MustCompile(`package main

var Tint vec4

func Fragment(dst vec4, pos vec2) vec4 {
	b := backdropSample(pos)
	return mix(b, Tint, Tint.a)
}
`)

ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
	frost.Draw(p, r, shader.Options{
		BackdropBlur: 12,
		Uniforms:     map[string]any{"Tint": ui.RGBA(255, 255, 255, 0.3)},
		Radius:       16,
	})
})
```

The renderers capture the backdrop as they do for the glass plugin's
panes, every frame, so it follows whatever moves behind. An element's
effect reads it too. A shader reading its backdrop draws rectangles
(`Draw`, `Effect`), not shapes (`Fill`), and reads images 0, 1 and 3 of
its own; image 2 is the backdrop.

## The language

A shader is a Go file of package main. Its function `Fragment` returns the
pixel's color, premultiplied (each color component multiplied by the
alpha):

```go
func Fragment(dst vec4, pos vec2) vec4
```

`dst` is the pixel's center in pixels of the window, or of the canvas
drawn into, and `pos` the same point in DIPs from the rectangle's top
left corner. `Fragment` may take `dst` alone, or nothing.

The language is [Ebitengine's Kage](https://ebitengine.org/en/documents/shader.html),
whose compiler MyGo's is forked from: Go's syntax, with these types:

| Types | |
| --- | --- |
| `bool`, `int`, `uint`, `float` | scalars of 32 bits |
| `vec2`, `vec3`, `vec4` | vectors of floats, with swizzles (`v.xy`, `c.rgb`) |
| `ivec2` to `ivec4`, `uvec2` to `uvec4` | vectors of ints and uints |
| `mat2`, `mat3`, `mat4` | matrices of floats, by columns |
| `[N]T` | arrays of any of them |

Functions of your own, `if`, `for` loops (to bounds that need not be
constants, and `range` over an int), constants, and GLSL's built-in
functions: `sin`, `cos`, `atan2`, `pow`, `exp`, `sqrt`, `abs`, `floor`,
`fract`, `mod`, `min`, `max`, `clamp`, `mix`, `step`, `smoothstep`,
`length`, `distance`, `dot`, `cross`, `normalize`, and the others. ints
and uints are two kinds of integers that mix only in shifts, as in Go;
uints wrap around, which hashes need:

```go
func pcg(v uint) uint {
	s := v*747796405 + 2891336453
	w := ((s >> ((s >> 28) + 4)) ^ s) * 277803737
	return (w >> 22) ^ w
}
```

`discard()` leaves the pixel as it was. `dfdx`, `dfdy` and `fwidth` give
how a value changes to the next pixel on the GPU, and zero on the CPU.

## Uniforms

Exported variables at the package level are uniforms: values the app sets
for each draw in `Options.Uniforms`, by name. A number sets a scalar; a
slice or an array of numbers sets a vector, a matrix (by columns) or an
array (elements in order); a `ui.Color` sets a `vec4`, its components from
0 to 1. A uniform the options leave out is zero.

```go
var Taps [6]vec4 // x, y, when, how hard
var Count int
```

```go
taps := make([]float32, 0, 24)
for _, t := range ripples {
	taps = append(taps, t.x, t.y, t.at, t.force)
}
fx.Draw(p, r, shader.Options{Uniforms: map[string]any{"Taps": taps, "Count": len(ripples)}})
```

For values set every frame, `Values` sets them by variables looked up
once, with no map, no conversions and no allocation each draw, as
Flutter's `FragmentShader` and Skia's `SkRuntimeShaderBuilder` hold them:

```go
var (
	timeVar = waves.Uniform("Time") // panics for a name the shader lacks
	tintVar = waves.Uniform("Tint")
	values  = waves.NewValues()
)

values.Float(timeVar, float32(t)).Color(tintVar, theme.Accent)
waves.Draw(p, r, shader.Options{Values: values})
```

Each setter checks the variable's type (`Float`, `Vec2` to `Vec4`, `Int`,
`Uint`, `Bool`, and `Floats`, `Ints` and `Uints` for vectors, matrices and
arrays) and panics for another, as for too many numbers; drawing copies
the values, so one `Values` serves every draw. `Options.Uniforms` then
sets the names it holds over them.

A color (`Values.Color`, or a `ui.Color` in `Options.Uniforms`) sets a
`vec3` or `vec4` to its components from 0 to 1, not premultiplied, in
sRGB; where the window draws a wide gamut, a color outside sRGB, as
`ui.Oklch` makes, reaches the shader in extended sRGB (components below 0
or above 1), as the window's own colors do.

## Images

A shader reads up to four images, `Options.Images`: bitmaps, or canvases'
(`Canvas.Bitmap`), in their pixels from their top left corners:

| Function | |
| --- | --- |
| `image0At(p vec2) vec4` | the pixel at `p`, transparent outside the image |
| `image0Sample(p vec2) vec4` | blended between the four pixels around `p`, clamped to the edges |
| `image0Size() vec2` | the image's size in pixels |

and `image1At` to `image3At`, and so on. `rectSize()` returns the size of
the rectangle the shader fills, in DIPs, and `pixelScale()` the pixels a
DIP.

A shader drawing into a [canvas](drawing.md#canvases) makes the input of
another, as the passes of an effect do: one computes a field at a few
samples into a small canvas, the next reads it, sampled, at every pixel of
the window.

```go
field := ui.NewCanvas(cols, rows)

box.Draw(func(p *ui.Painter, r ui.Rect) {
	field.Draw(p, func(cp *ui.Painter) {
		windShader.Draw(cp, ui.Rect{W: float32(cols), H: float32(rows)}, shader.Options{Uniforms: wind})
	})
	glyphShader.Draw(p, r, shader.Options{Images: [4]*ui.Bitmap{field.Bitmap(), atlas}})
})
```

## Errors and cost

`Compile` reports what is wrong with a shader with its lines; compile
shaders once, as package variables with `MustCompile`. `Compile` takes
about a millisecond: it checks the shader and translates it into the
GPU's language (Metal, HLSL or GLSL), which the GPU's driver then
compiles, as SwiftUI's `Shader.compile(as:)` has it done ahead of use.
On macOS, `Compile` starts that on a goroutine of its own: Metal's
library and pipeline take about 0.2 s together the first time the app
runs on the Mac, and the renderers of every window draw with them,
waiting only for what is left; a shader compiled as a package variable
is ready long before its first frame, and Metal keeps both on disk, so
later launches have them at once. Linux's Mesa
drivers keep theirs on disk too (in `~/.cache/mesa_shader_cache`): from
a tenth to a third of a second on the CPU's llvmpipe, the first time,
and a few milliseconds after. On Windows, Direct3D's compiler takes from
a tenth of a second for a short shader to a second for one of a few
hundred lines, so `Compile` starts it on a goroutine of its own: the
renderers of every window share the code, waiting on their first draw
only for what is left, and it stays in the user's cache directory
(`MyGo\ShaderCache`) for later launches. A shader costs
little to draw on a GPU, whatever it computes per pixel; on the CPU (the
renderer of last resort, and `ui.Tester`) it runs interpreted, a few
microseconds a pixel: seconds for a window. A shader covering much of the
window can ask `p.Accelerated()`, which is false there and where the GPU's
renderer draws on the CPU (WARP, llvmpipe, SwiftShader), and draw less,
or nothing:

```go
ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
	if !p.Accelerated() {
		return // a backdrop, which the window does without
	}
	backdrop.Draw(p, r, shader.Options{})
})
```

`Draw` panics on a uniform set to a value that is
not a number.
