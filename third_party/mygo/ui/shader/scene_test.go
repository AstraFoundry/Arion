package shader

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/egoist/mygo/internal/raster"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/shader"
	"github.com/egoist/mygo/internal/shaderir"
)

// testSource reads two images, one exactly and one sampled, and uniforms
// of each kind, as the renderers must alike.
const testSource = `package main

var Seed uint
var Tint vec4
var Count int
var Spots [3]vec3
var Turn mat2

// Bools last: Metal's take a byte each, at the end of its struct.
var On, Off, Third bool

func pcg(v uint) uint {
	s := v*747796405 + 2891336453
	w := ((s >> ((s >> 28) + 4)) ^ s) * 277803737
	return (w >> 22) ^ w
}

func Fragment(dst vec4, pos vec2) vec4 {
	q := Turn * (pos - rectSize()/2)
	glow := 0.0
	for i := 0; i < Count; i++ {
		d := distance(q, Spots[i].xy)
		glow += exp(-d*d/(Spots[i].z*Spots[i].z))
	}
	h := float(pcg(uint(int(pos.x/4))*0x9E3779B9^pcg(uint(int(pos.y/4))^Seed))>>8) / 16777216.0
	a := image0At(pos * pixelScale() / 2)
	b := image1Sample(pos / rectSize() * image1Size())
	c := mix(a, b, 0.5)*0.6 + Tint*clamp(glow, 0, 1)*0.3 + vec4(h*0.1)
	if On && !Off && Third {
		c.g += 0.1
	}
	return c
}
`

// testScene returns a scene of s over a gray frame at scale 2, reading a
// gradient bitmap and a canvas of three rectangles, which differs top to
// bottom and left to right, as a renderer keeping its rows going up must
// show it.
func testScene(s *Shader) *scene.Scene {
	const scale = 2
	grad := image.NewRGBA(image.Rect(0, 0, 64, 40))
	for y := range 40 {
		for x := range 64 {
			grad.Set(x, y, color.RGBA{uint8(x * 4), uint8(y * 6), 128, 255})
		}
	}
	canvas := scene.NewCanvas(20, 12)
	canvas.Scene.Ops = []scene.Op{
		{Kind: scene.OpFill, Rect: scene.Rect{W: 10, H: 6}, Color: scene.Color{R: 220, G: 40, B: 40, A: 255}},
		{Kind: scene.OpFill, Rect: scene.Rect{X: 10, W: 10, H: 6}, Color: scene.Color{R: 37, G: 99, B: 235, A: 255}},
		{Kind: scene.OpFill, Rect: scene.Rect{Y: 6, W: 20, H: 6}, Color: scene.Color{R: 30, G: 160, B: 60, A: 255}},
	}
	images := [4]*scene.Image{scene.NewImage(grad), canvas}

	r := scene.Rect{X: 8, Y: 6, W: 224, H: 108}
	words := make([]uint32, s.words)
	for i, img := range images {
		if img != nil {
			words[2*i], words[2*i+1] = math.Float32bits(float32(img.W)), math.Float32bits(float32(img.H))
		}
	}
	words[scaleWord-2], words[scaleWord-1] = math.Float32bits(r.W/scale), math.Float32bits(r.H/scale)
	words[scaleWord] = math.Float32bits(scale)
	values := map[string]any{
		"Seed":  uint32(0x3fa1c2d9),
		"Tint":  []float32{1, 0.8, 0.2, 1},
		"Count": 2,
		"Spots": []float32{-20, 0, 18, 25, 10, 12, 0, 0, 1},
		"Turn":  []float32{0.8, 0.6, -0.6, 0.8},
		"On":    true,
		"Off":   false,
		"Third": true,
	}
	for name, v := range values {
		u := s.uniforms[name]
		kind := u.t.Scalar()
		if u.t.Main == shaderir.Array {
			kind = u.t.Sub[0].Scalar()
		}
		set(words[u.word:u.word+u.n], kind, name, v)
	}
	sc := &scene.Scene{Width: 240, Height: 120, Clear: scene.Color{R: 240, G: 240, B: 240, A: 255}}
	sc.Ops = append(sc.Ops, scene.Op{Kind: scene.OpEffect, Rect: r, Radii: [4]float32{16, 16, 16, 16}})
	sc.Effects = append(sc.Effects, scene.EffectOp{Effect: s.fx, Uniforms: words, Images: images})
	return sc
}

// dfdySource is green where y grows downward on the screen, as the
// position's does, and red elsewhere: GPUs differ in which way their
// window's y goes.
const dfdySource = `package main

func Fragment(dst vec4, pos vec2) vec4 {
	if dfdy(pos.y) > 0 {
		return vec4(0, 1, 0, 1)
	}
	return vec4(1, 0, 0, 1)
}
`

// dfdyScene returns a scene of dfdySource in a canvas, and the canvas
// shown, as renderers flip canvases: dfdy must be positive in both.
func dfdyScene() *scene.Scene {
	s := MustCompile(dfdySource)
	words := make([]uint32, s.words)
	words[scaleWord] = math.Float32bits(1)
	canvas := scene.NewCanvas(16, 16)
	canvas.Scene.Ops = []scene.Op{{Kind: scene.OpEffect, Rect: scene.Rect{W: 16, H: 16}}}
	canvas.Scene.Effects = []scene.EffectOp{{Effect: s.fx, Uniforms: words}}
	sc := &scene.Scene{Width: 40, Height: 20}
	sc.Ops = []scene.Op{
		{Kind: scene.OpEffect, Rect: scene.Rect{W: 20, H: 20}},
		{Kind: scene.OpImage, Rect: scene.Rect{X: 22, Y: 2, W: 16, H: 16}, Image: canvas, Src: scene.Rect{W: 16, H: 16}},
	}
	sc.Effects = []scene.EffectOp{{Effect: s.fx, Uniforms: words}}
	sc.Canvases = []*scene.Image{canvas}
	return sc
}

// checkDfdy checks that the window's pixel (10, 10) and the canvas's
// (30, 10) are green, in BGRA or RGBA rows of stride bytes.
func checkDfdy(t *testing.T, pix []byte, stride int) {
	t.Helper()
	for _, x := range []int{10, 30} {
		p := pix[10*stride+4*x:][:4]
		if p[1] < 200 || p[0] > 50 && p[2] > 50 {
			t.Errorf("pixel (%d, 10) is %v, want green: dfdy goes the wrong way", x, p)
		}
	}
}

// boolSource reads only bools besides the bridge's uniforms, which leaves
// Metal's struct aligned to 8, the bools in its last bytes.
const boolSource = `package main

var A, B, C bool

func Fragment(dst vec4, pos vec2) vec4 {
	v := vec4(0, 0, 0, 1)
	if A {
		v.r = 1
	}
	if B {
		v.g = 1
	}
	if C {
		v.b = 1
	}
	return v
}
`

// boolScene returns a scene of boolSource with A, B and C true: white.
func boolScene() *scene.Scene {
	s := MustCompile(boolSource)
	words := make([]uint32, s.words)
	words[scaleWord] = math.Float32bits(1)
	for _, name := range []string{"A", "B", "C"} {
		set(words[s.uniforms[name].word:][:1], shaderir.Bool, name, true)
	}
	sc := &scene.Scene{Width: 20, Height: 20}
	sc.Ops = []scene.Op{{Kind: scene.OpEffect, Rect: scene.Rect{W: 20, H: 20}}}
	sc.Effects = []scene.EffectOp{{Effect: s.fx, Uniforms: words}}
	return sc
}

// maskedScene returns a scene of s filling mask glyphs, discs of a mask
// atlas: a row of them, straight, and another rotated, its positions from
// the corner of the row's box, as text and paths a shader fills.
func maskedScene(s *Shader) *scene.Scene {
	atlas := scene.NewAtlas(1, 64, 64)
	disc := make([]byte, 16*16)
	for y := range 16 {
		for x := range 16 {
			dx, dy := float64(x)-7.5, float64(y)-7.5
			disc[y*16+x] = byte(255 * max(0, min(1, 7.5-math.Sqrt(dx*dx+dy*dy))))
		}
	}
	u, v, _ := atlas.Alloc(16, 16)
	atlas.Put(u, v, 16, 16, disc, 16)
	sc := &scene.Scene{Width: 240, Height: 120, Clear: scene.Color{R: 240, G: 240, B: 240, A: 255}, MaskAtlas: atlas,
		Text: scene.TextParams{GammaRatios: scene.GammaRatios(1.8), Contrast: 1}}
	words := make([]uint32, s.words)
	words[scaleWord-2], words[scaleWord-1] = math.Float32bits(200), math.Float32bits(16)
	words[scaleWord] = math.Float32bits(1)
	rotate := scene.Affine{Set: true, A: 0.94, B: 0.34, C: -0.34, D: 0.94, X: 40, Y: 50}
	for row, m := range []scene.Affine{{}, rotate} {
		start := int32(len(sc.Glyphs))
		for i := range 12 {
			sc.Glyphs = append(sc.Glyphs, scene.Glyph{X: 10 + float32(i*17), Y: 10 + float32(row*20), W: 16, H: 16,
				U: uint16(u), V: uint16(v), UW: 16, VH: 16, Color: scene.Color{R: 255, G: 255, B: 255, A: 255}})
		}
		sc.Ops = append(sc.Ops, scene.Op{Kind: scene.OpEffect, Rect: scene.Rect{X: 10, Y: 10 + float32(row*20), W: 220, H: 17},
			Transform: m, Start: int32(len(sc.Effects))})
		sc.Effects = append(sc.Effects, scene.EffectOp{Effect: s.fx, Uniforms: words,
			Glyphs: [2]int32{start, int32(len(sc.Glyphs))}, Origin: [2]float32{10, 10 + float32(row*20)}})
	}
	return sc
}

// maskedSource colors by position, across the box it fills.
const maskedSource = `package main

func Fragment(dst vec4, pos vec2) vec4 {
	t := clamp(pos.x/rectSize().x, 0, 1)
	return vec4(1-t, 0.3*(pos.y/16), t, 1)
}
`

// effectSource moves its content in a wave, and tints it by position.
const effectSource = `package main

func DistortionEffect(pos vec2) vec2 {
	return pos + vec2(3*sin(pos.y/5), 4*sin(pos.x/9))
}
`

// effectScene returns a scene of s, an effect's shader, drawing content
// of three rectangles in a canvas, the effect's box inset in it by 6
// pixels, as Element.Effect draws an element.
func effectScene(s *Shader) *scene.Scene { return effectSceneIn(s, 112, 52) }

// effectScenes are effectScene's, and the same content in the corner of
// a larger canvas, as Element.Effect keeps one as the element's size
// changes: what the larger canvas holds beyond the content's 112×52
// pixels, an opaque square, reads as transparent, and draws nothing.
func effectScenes(s *Shader) map[string]*scene.Scene {
	return map[string]*scene.Scene{"exact": effectSceneIn(s, 112, 52), "larger canvas": effectSceneIn(s, 176, 64)}
}

// effectSceneIn is effectScene with a canvas of w×h pixels, of which the
// content is the top left 112×52.
func effectSceneIn(s *Shader, w, h int) *scene.Scene {
	content := scene.NewCanvas(w, h)
	if w > 112 {
		content.Scene.Ops = append(content.Scene.Ops, scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: 112, Y: 0, W: float32(w - 112), H: float32(h)}, Color: scene.Color{R: 255, A: 255}})
	}
	content.Scene.Ops = append(content.Scene.Ops, []scene.Op{
		{Kind: scene.OpFill, Rect: scene.Rect{X: 6, Y: 6, W: 50, H: 20}, Color: scene.Color{R: 220, G: 40, B: 40, A: 255}},
		{Kind: scene.OpFill, Rect: scene.Rect{X: 56, Y: 6, W: 50, H: 20}, Color: scene.Color{R: 37, G: 99, B: 235, A: 255}, Radii: [4]float32{8, 8, 8, 8}},
		{Kind: scene.OpFill, Rect: scene.Rect{X: 6, Y: 26, W: 100, H: 20}, Color: scene.Color{R: 30, G: 160, B: 60, A: 200}},
	}...)
	words := make([]uint32, s.words)
	words[2*shader.LayerImage], words[2*shader.LayerImage+1] = math.Float32bits(112), math.Float32bits(52)
	words[scaleWord-2], words[scaleWord-1] = math.Float32bits(100), math.Float32bits(40)
	words[scaleWord] = math.Float32bits(1)
	words[insetWord], words[insetWord+1] = math.Float32bits(6), math.Float32bits(6)
	sc := &scene.Scene{Width: 240, Height: 120, Clear: scene.Color{R: 240, G: 240, B: 240, A: 255}}
	sc.Ops = append(sc.Ops, scene.Op{Kind: scene.OpEffect, Rect: scene.Rect{X: 20, Y: 30, W: 112, H: 52}})
	var images [4]*scene.Image
	images[shader.LayerImage] = content
	sc.Effects = append(sc.Effects, scene.EffectOp{Effect: s.fx, Uniforms: words, Images: images})
	return sc
}

// backdropSource inverts what is behind it, and dims it toward the right.
const backdropSource = `package main

func Fragment(dst vec4, pos vec2) vec4 {
	b := backdropSample(pos + vec2(0, 3*sin(pos.x/8)))
	k := 1 - 0.5*pos.x/rectSize().x
	return vec4((b.a-b.rgb)*k, b.a)
}
`

// backdropScene returns a scene of s over stripes, reading its backdrop
// blurred by blur pixels, its op under m, with the bridge's words as
// Painter.EffectBackdrop sets them at scale 1.
func backdropScene(s *Shader, blur float32, m scene.Affine) *scene.Scene {
	return backdropSceneIn(s, blur, m, scene.Rect{X: 40, Y: 30, W: 150, H: 60}, 240, 120)
}

// backdropSceneIn is backdropScene with the shader filling r, in a frame
// of w×h pixels, of which the stripes take the top left 240×120: a
// larger frame holds an opaque square beyond them, and says they are what
// is drawn (Scene.Drawn), as an effect's layer larger than its content.
func backdropSceneIn(s *Shader, blur float32, m scene.Affine, r scene.Rect, w, h int) *scene.Scene {
	sc := &scene.Scene{Width: w, Height: h, Clear: scene.Color{R: 250, G: 250, B: 250, A: 255}}
	if w > 240 || h > 120 {
		sc.Drawn = image.Pt(240, 120)
		sc.Ops = append(sc.Ops, scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: 240, W: float32(w - 240), H: float32(h)}, Color: scene.Color{R: 255, A: 255}})
	}
	for i := range 12 {
		c := scene.Color{R: uint8(40 + 18*i), G: uint8(200 - 15*i), B: 90, A: 255}
		sc.Ops = append(sc.Ops, scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: float32(i * 20), Y: 0, W: 10, H: 120}, Color: c})
	}
	bk := sc.BackdropOf(m.Bounds(r), blur, sc.Width, sc.Height)
	tw, th := bk.Size()
	words := make([]uint32, s.words)
	f := func(i int, v float32) { words[i] = math.Float32bits(v) }
	f(2*shader.BackdropImage, float32(tw))
	f(2*shader.BackdropImage+1, float32(th))
	f(scaleWord-2, r.W)
	f(scaleWord-1, r.H)
	f(scaleWord, 1)
	a, b, c, d, x, y := float32(1), float32(0), float32(0), float32(1), float32(0), float32(0)
	if m.Set {
		a, b, c, d, x, y = m.A, m.B, m.C, m.D, m.X, m.Y
	}
	for k, v := range [3]float32{a, c, a*r.X + c*r.Y + x} {
		f(row0Word+k, v)
	}
	for k, v := range [3]float32{b, d, b*r.X + d*r.Y + y} {
		f(row1Word+k, v)
	}
	f(areaWord, float32(bk.Area.Min.X))
	f(areaWord+1, float32(bk.Area.Min.Y))
	f(areaWord+2, float32(bk.Down))
	sc.Ops = append(sc.Ops, scene.Op{Kind: scene.OpEffect, Rect: r, Radii: [4]float32{12, 12, 12, 12}, Transform: m, Start: int32(len(sc.Effects))})
	sc.Effects = append(sc.Effects, scene.EffectOp{Effect: s.fx, Uniforms: words, Blur: blur})
	return sc
}

// backdropCases are the backdrops the GPU tests draw: sharp, blurred, and
// under a rotation.
func backdropCases() map[string]*scene.Scene {
	s := MustCompile(backdropSource)
	rotate := scene.Translation(115, 60).Mul(scene.Rotation(12)).Mul(scene.Translation(-115, -60))
	return map[string]*scene.Scene{"sharp": backdropScene(s, 0, scene.Affine{}), "blurred": backdropScene(s, 6, scene.Affine{}), "rotated": backdropScene(s, 3, rotate)}
}

// TestEffectLargerCanvas checks that an effect drawn from the corner of a
// larger canvas draws as from a canvas of the content's size: what the
// canvas holds beyond the content does not show.
func TestEffectLargerCanvas(t *testing.T) {
	s := MustCompile(effectSource)
	render := func(sc *scene.Scene) *raster.Image {
		img := raster.NewImage(sc.Width, sc.Height)
		raster.Render(img, sc)
		return img
	}
	exact, larger := render(effectSceneIn(s, 112, 52)), render(effectSceneIn(s, 176, 64))
	if !bytes.Equal(exact.Pix, larger.Pix) {
		t.Error("an effect from a larger canvas draws otherwise")
	}
}

// edgeBackdrops are a backdrop at the right edge of the stripes, sharp and
// blurred, in a frame of their size and in a larger one that holds an
// opaque square beyond them, which the backdrop reads as beyond the frame.
func edgeBackdrops() map[string][2]*scene.Scene {
	s := MustCompile(backdropSource)
	r := scene.Rect{X: 150, Y: 30, W: 90, H: 60}
	out := map[string][2]*scene.Scene{}
	for name, blur := range map[string]float32{"sharp": 0, "blurred": 6} {
		out[name] = [2]*scene.Scene{backdropSceneIn(s, blur, scene.Affine{}, r, 240, 120), backdropSceneIn(s, blur, scene.Affine{}, r, 304, 128)}
	}
	return out
}

// TestBackdropEndsAtDrawn checks that a backdrop in a frame larger than
// what it draws (Scene.Drawn) ends at the edge of that, as in a frame of
// its size: what lies beyond does not show through.
func TestBackdropEndsAtDrawn(t *testing.T) {
	for name, scs := range edgeBackdrops() {
		exact, larger := raster.NewImage(240, 120), raster.NewImage(304, 128)
		raster.Render(exact, scs[0])
		raster.Render(larger, scs[1])
		for y := range 120 {
			if !bytes.Equal(exact.Pix[y*exact.Stride:][:4*240], larger.Pix[y*larger.Stride:][:4*240]) {
				t.Errorf("%s: row %d differs in the larger frame", name, y)
				break
			}
		}
	}
}
