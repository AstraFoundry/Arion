// Package gpu turns scenes into what the GPU renderers draw: one instanced
// quad per operation, for a shader that computes rounded rectangles,
// borders, gradients, stripes, shadows and atlas coverage itself, in
// batches that share a scissor rectangle and an image. Every renderer
// (d3d11, metal, gl) draws the same instances, the way internal/raster
// draws scenes on the CPU.
//
// An effect (scene.Effect) draws with a pipeline of its own, made from
// its shader, in a batch of its own. One reading its backdrop has its
// batch start after a Backdrop step, which renderers take between
// batches: they read the backdrop's area of what they drew so far,
// average it over squares (the pass "down") into a texture, blur that
// along rows into another and along columns back (the pass "blur"),
// keeping 8 bits a channel, and bind it for the batch (the shaders'
// backdrop texture), as internal/raster computes it.
//
// The shaders write a second color for blending, the source's alpha for
// each channel: the alpha of the color for everything but subpixel
// glyphs, whose subpixels cover each channel by its own. Renderers blend
// with dual-source blending: the destination times one minus the second
// color, plus the first.
//
// A hole (scene.OpHole) is an opaque fill in a batch of its own, which
// renderers blend with none of the source: the destination times one
// minus the second color, its coverage, which leaves it transparent.
package gpu

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"unsafe"

	"github.com/egoist/mygo/internal/scene"
)

// GoBytes returns the Go declaration of a variable name holding b, as
// code compiled ahead of time is generated.
func GoBytes(name string, b []byte) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "var %s = []byte{", name)
	for i, c := range b {
		if i%16 == 0 {
			sb.WriteString("\n\t")
		}
		fmt.Fprintf(&sb, "0x%02x, ", c)
	}
	sb.WriteString("\n}\n")
	return sb.String()
}

// Instance is the data of one quad, in device pixels, as the shaders read
// it: thirteen float4s.
type Instance struct {
	// Rect is x, y, width, height; Radii the corners' radii (top-left,
	// top-right, bottom-right, bottom-left), negative for continuous
	// corners (scene.Corners), which only Metal's shader draws, or a
	// glyph's gamma ratios
	// (scene.TextParams); Inner those of a border's inner edge, or of the
	// box casting a shadow, or a glyph's contrast and thin boost in Inner[0]
	// and Inner[1].
	Rect, Radii, Inner [4]float32
	// Color and Color2 (a gradient's end) and Border are straight RGBA.
	Color, Color2, Border [4]float32
	// Grad holds a gradient's start and end points, or the stripes' unit
	// vector across them, width and period; UV the texture rectangle,
	// normalized, a fill's border widths (top, right, bottom, left), or
	// the box casting a shadow, which shows only outside it (none when
	// empty).
	//
	// An effect has its parameters (scene.EffectOp.Params) in Inner,
	// Color, Color2, Border and Grad, and in UV where its backdrop's area
	// starts in the frame and its size in texels.
	Grad, UV [4]float32
	// Clip and ClipRadii are the innermost clip, which the shader cuts.
	Clip, ClipRadii [4]float32
	// Params is the kind (0 fill, 1 shadow, 2 mask glyph, 3 color glyph, 4
	// image, 5 subpixel glyph, 6 effect); 1 for a dashed border, a
	// grayscale image or an inner shadow, whose box is in Rect and Radii
	// and the hole it leaves in UV and Inner (none when empty); the
	// shadow's sigma (0 for none), the paint (scene.Paint) or the size of
	// the squares an effect's backdrop averages; and the opacity.
	Params [4]float32
	// Transform rows map local pixels to the frame. Transform0.w enables
	// transformed antialiasing, Transform1.w selects a cached clip mask.
	// Transform2 holds the projective row's g and h, the frame point of
	// local (x, y) being Transform0's and Transform1's dot products with
	// (x, y, 1) over g*x + h*y + 1: shaders interpolate in perspective.
	Transform0, Transform1, Transform2 [4]float32
}

// InstanceSize is the size of an Instance in bytes.
const InstanceSize = int(unsafe.Sizeof(Instance{}))

// Scissor is a scissor rectangle in device pixels, the layout of a
// Direct3D RECT.
type Scissor struct{ Left, Top, Right, Bottom int32 }

// Empty reports whether the rectangle has no pixels.
func (s Scissor) Empty() bool { return s.Right <= s.Left || s.Bottom <= s.Top }

// Batch is a run of instances that draw with the same scissor rectangle
// and image.
type Batch struct {
	Start, Count int
	Scissor      Scissor
	// Image is the renderer's texture of the batch's image, or 0.
	Image uintptr
	// ClipMask is a cached nested clip texture, sampled in frame coordinates.
	ClipMask uintptr
	// Effect is the effect of the batch's one instance, drawn with a
	// pipeline of its own, and Backdrop, unless 0, 1 + the index in
	// Builder.Backdrops of its backdrop, which the renderer reads before
	// drawing it.
	Effect   *scene.Effect
	Backdrop int
	// Hole tells that the batch's instances make their shapes transparent
	// (scene.OpHole): renderers blend them with a source factor of zero.
	Hole bool
}

// Builder builds the instances and batches of scenes, reusing its memory
// from frame to frame.
type Builder struct {
	Instances []Instance
	Batches   []Batch
	Backdrops []scene.Backdrop
	// Wide draws the colors of scenes outside the sRGB gamut
	// (Scene.Wide), for a target that keeps them, as Metal's float16
	// drawables do; without it, their nearest sRGB colors.
	Wide  bool
	stack []clip
	// encloses tells, for each op of the scene, that it pushes a clip with
	// a transformed op or glyph inside it.
	encloses     []bool
	pushes       []int
	clipShapes   []scene.ClipShape
	maskSlots    []clipMaskSlot
	maskUsed     int
	frame        uint64
	imageTexture func(*scene.Image) uintptr
	preparedMask uintptr
}

// Pass is what a pass computing a backdrop reads, the layout of the
// shaders' Pass: down reads the area from Origin to Limit, frame pixels,
// at their place less Shift in its source, and averages squares of Down
// pixels a side; blur reads texels up to Limit, Radius of them each way
// along Dir, weighted by a Gaussian of standard deviation Sigma.
type Pass struct {
	Origin, Limit, Shift, Dir [2]int32
	Down, Radius              int32
	Sigma, _                  float32
}

// DownPass returns the pass averaging the squares of bk's area, whose
// frame pixel (x, y) is at (x, y) less shift in the texture it reads.
func DownPass(bk scene.Backdrop, shift [2]int32) Pass {
	return Pass{
		Origin: [2]int32{int32(bk.Area.Min.X), int32(bk.Area.Min.Y)},
		Limit:  [2]int32{int32(bk.Area.Max.X - 1), int32(bk.Area.Max.Y - 1)},
		Shift:  shift,
		Down:   int32(bk.Down),
	}
}

// BlurPass returns the pass blurring bk's texels along rows (dir 1, 0)
// or columns (0, 1).
func BlurPass(bk scene.Backdrop, dir [2]int32) Pass {
	w, h := bk.Size()
	return Pass{Limit: [2]int32{int32(w - 1), int32(h - 1)}, Dir: dir, Radius: int32(bk.Radius), Sigma: bk.Sigma}
}

// BackdropSize returns how large the textures of the backdrops of the
// last scene built must be, in texels: as the largest.
func (b *Builder) BackdropSize() (w, h int) {
	for _, bk := range b.Backdrops {
		bw, bh := bk.Size()
		w, h = max(w, bw), max(h, bh)
	}
	return w, h
}

type clip struct {
	rect    scene.Rect
	radii   [4]float32
	bounds  scene.Rect
	scissor Scissor
	shape   scene.ClipShape
	// moved tells that this clip or one it is nested in is transformed,
	// which the scissor rectangles don't follow.
	moved       bool
	masked      bool
	maskSlot    int
	maskTexture uintptr
}

// Build turns a scene directly into GPU instances, retaining local geometry
// and composing transforms in the vertex shader. Nested affine clips use
// cached masks; stationary clipping never rasterizes again as content moves.
// A clip is masked only when it is transformed, nested in a transformed
// clip, or rounded (or masked) with transformed content inside: the others
// cut in the shader as in a scene without transforms.
func (b *Builder) Build(s *scene.Scene, image func(*scene.Image) uintptr) *scene.Scene {
	b.markEnclosing(s)
	b.imageTexture = image
	b.maskUsed = 0
	b.frame++
	b.clipShapes = b.clipShapes[:0]
	b.Instances = b.Instances[:0]
	b.Batches = b.Batches[:0]
	b.Backdrops = b.Backdrops[:0]
	everything := scene.Rect{X: -1e6, Y: -1e6, W: 2e6, H: 2e6}
	b.stack = append(b.stack[:0], clip{rect: everything, bounds: scene.Rect{W: float32(s.Width), H: float32(s.Height)},
		scissor: Scissor{0, 0, int32(s.Width), int32(s.Height)}})
	for i := range s.Ops {
		op := &s.Ops[i]
		switch op.Kind {
		case scene.OpPushClip:
			top := b.stack[len(b.stack)-1]
			bounds := top.bounds.Intersect(op.Transform.Bounds(op.Rect))
			// The innermost clip shapes its edges in the shader, the
			// others cut with the scissor rectangle.
			c := clip{rect: op.Rect, radii: scene.Corners(op.Rect, op.Radii, op.Continuous), bounds: bounds,
				shape: scene.ClipShape{Rect: op.Rect, Radii: op.Radii, Transform: op.Transform, Continuous: op.Continuous}, maskSlot: -1}
			c.moved = top.moved || op.Transform.Set
			c.masked = c.moved || b.encloses[i] && (top.masked || c.radii != [4]float32{})
			if c.masked {
				c.maskSlot = b.maskUsed
				b.maskUsed++
			}
			b.clipShapes = append(b.clipShapes, c.shape)
			if bounds.W > 0 && bounds.H > 0 {
				c.scissor = Scissor{int32(math.Floor(float64(bounds.X))), int32(math.Floor(float64(bounds.Y))),
					int32(math.Ceil(float64(bounds.X + bounds.W))), int32(math.Ceil(float64(bounds.Y + bounds.H)))}
			}
			b.stack = append(b.stack, c)
		case scene.OpPopClip:
			if len(b.stack) > 1 {
				b.stack = b.stack[:len(b.stack)-1]
				b.clipShapes = b.clipShapes[:len(b.clipShapes)-1]
			}
		case scene.OpFill:
			if op.Rect.Empty() {
				continue
			}
			bw := op.Border
			if op.BorderColor.A == 0 {
				bw = [4]float32{}
			}
			radii := scene.Corners(op.Rect, op.Radii, op.Continuous)
			var inner [4]float32
			if scene.HasBorder(bw) {
				_, inner = scene.InnerRadii(op.Rect, radii, bw)
			}
			dashed := float32(0)
			if op.Dashed {
				dashed = 1
			}
			w := b.wideOf(s, op.Wide)
			b.add(Instance{
				Rect: rect(op.Rect), Radii: radii, Inner: inner,
				Color: wideColor(w, scene.WideColor, op.Color), Color2: wideColor(w, scene.WideColor2, op.Color2), Border: wideColor(w, scene.WideBorder, op.BorderColor), Grad: op.Gradient,
				UV:     bw,
				Params: [4]float32{0, dashed, float32(op.Paint), opacity(op.Opacity)},
			}, 0, op.Transform)
		case scene.OpShadow:
			if op.Inset {
				// The box in Rect, the hole in UV and Inner, which an
				// inner shadow's 1 in Params[1] tells.
				if op.Cast.Empty() {
					continue
				}
				sigma := op.Blur / 2
				if sigma < 0.5 {
					sigma = 0
				}
				in := Instance{
					Rect: rect(op.Cast), Radii: scene.Corners(op.Cast, op.CastRadii, op.Continuous),
					Color: wideColor(b.wideOf(s, op.Wide), scene.WideColor, op.Color), Params: [4]float32{1, 1, sigma, opacity(op.Opacity)},
				}
				if !op.Rect.Empty() {
					in.UV, in.Inner = rect(op.Rect), scene.Corners(op.Rect, op.Radii, op.Continuous)
				}
				b.add(in, 0, op.Transform)
				continue
			}
			if op.Rect.Empty() {
				continue
			}
			sigma := op.Blur / 2
			if sigma < 0.5 {
				sigma = 0
			}
			in := Instance{
				Rect: rect(op.Rect), Radii: scene.Corners(op.Rect, op.Radii, op.Continuous),
				Color: wideColor(b.wideOf(s, op.Wide), scene.WideColor, op.Color), Params: [4]float32{1, 0, sigma, opacity(op.Opacity)},
			}
			if !op.Cast.Empty() {
				in.UV, in.Inner = rect(op.Cast), scene.Corners(op.Cast, op.CastRadii, op.Continuous)
			}
			b.add(in, 0, op.Transform)
		case scene.OpGlyphs:
			grad := op.Paint == scene.PaintLinear || op.Paint == scene.PaintOklab
			var c1, c2 [4]float32
			if grad {
				w := b.wideOf(s, op.Wide)
				c1, c2 = wideColor(w, scene.WideColor, op.Color), wideColor(w, scene.WideColor2, op.Color2)
			}
			for _, g := range s.Glyphs[op.Start:op.End] {
				a, kind, contrast := s.MaskAtlas, float32(2), s.Text.Contrast
				switch {
				case g.Colored:
					a, kind = s.ColorAtlas, 3
				case g.Subpixel:
					a, kind, contrast = s.ColorAtlas, 5, s.Text.SubpixelContrast
				}
				if a == nil {
					continue
				}
				boost := float32(0)
				if g.Thin {
					boost = scene.ThinBoost
				}
				aw, ah := float32(a.W), float32(a.H)
				in := Instance{
					Rect:   [4]float32{float32(math.Round(float64(g.X))), float32(math.Round(float64(g.Y))), float32(g.UW), float32(g.VH)},
					Radii:  s.Text.GammaRatios,
					Inner:  [4]float32{contrast, boost},
					UV:     [4]float32{float32(g.U) / aw, float32(g.V) / ah, float32(g.U+g.UW) / aw, float32(g.V+g.VH) / ah},
					Color:  straight(g.Color),
					Params: [4]float32{kind, 0, 0, 1},
				}
				if w := b.wideOf(s, g.Wide); w != nil {
					in.Color = w.Color
				}
				if grad && !g.Colored {
					in.Color, in.Color2, in.Grad = c1, c2, op.Gradient
					in.Params[2], in.Params[3] = float32(op.Paint), opacity(op.Opacity)
				}
				b.add(in, 0, op.Transform.Mul(g.Transform))
			}
		case scene.OpEffect:
			if !op.Rect.Empty() && int(op.Start) < len(s.Effects) && s.Effects[op.Start].Effect != nil {
				b.addEffect(s, op)
			}
		case scene.OpHole:
			if !op.Rect.Empty() {
				b.addHole(op)
			}
		case scene.OpImage:
			img := op.Image
			if img == nil || op.Rect.Empty() || img.W == 0 || img.H == 0 {
				continue
			}
			tex := image(img)
			if tex == 0 {
				continue
			}
			iw, ih := float32(img.W), float32(img.H)
			gray := float32(0)
			if op.Grayscale {
				gray = 1
			}
			b.add(Instance{
				Rect:   rect(op.Rect),
				Radii:  scene.Corners(op.Rect, op.Radii, op.Continuous),
				UV:     [4]float32{op.Src.X / iw, op.Src.Y / ih, (op.Src.X + op.Src.W) / iw, (op.Src.Y + op.Src.H) / ih},
				Params: [4]float32{4, gray, 0, opacity(op.Opacity)},
			}, tex, op.Transform)
		}
	}

	b.dropIdleMasks(idleMaskBytes)
	b.imageTexture = nil
	return s
}

// markEnclosing sets b.encloses for the ops of s.
func (b *Builder) markEnclosing(s *scene.Scene) {
	b.encloses = slices.Grow(b.encloses[:0], len(s.Ops))[:len(s.Ops)]
	clear(b.encloses)
	b.pushes = b.pushes[:0]
	for i := range s.Ops {
		op := &s.Ops[i]
		if op.Kind == scene.OpPopClip {
			if n := len(b.pushes); n > 0 {
				b.pushes = b.pushes[:n-1]
			}
			continue
		}
		transformed := op.Transform.Set
		if op.Kind == scene.OpGlyphs {
			for _, g := range s.Glyphs[op.Start:op.End] {
				transformed = transformed || g.Transform.Set
			}
		}
		// Every clip open around it encloses it, from the innermost out to
		// the first already marked, whose own are.
		for j := len(b.pushes) - 1; transformed && j >= 0 && !b.encloses[b.pushes[j]]; j-- {
			b.encloses[b.pushes[j]] = true
		}
		if op.Kind == scene.OpPushClip {
			b.pushes = append(b.pushes, i)
		}
	}
}

// Release frees the retained clipping resources and instance buffers.
func (b *Builder) Release() { *b = Builder{} }

// add appends an instance within the current clip, starting a batch when
// the scissor rectangle or the image changes, or after an effect's.
func (b *Builder) add(in Instance, image uintptr, m scene.Affine) {
	cur := &b.stack[len(b.stack)-1]
	if !b.prepare(&in, m, cur) {
		return
	}
	n := len(b.Batches)
	if n == 0 || b.Batches[n-1].Scissor != cur.scissor || b.Batches[n-1].Effect != nil || b.Batches[n-1].Hole ||
		b.Batches[n-1].ClipMask != b.preparedMask ||
		(image != 0 && b.Batches[n-1].Image != 0 && b.Batches[n-1].Image != image) {
		b.Batches = append(b.Batches, Batch{Start: len(b.Instances), Scissor: cur.scissor, Image: image, ClipMask: b.preparedMask})
		n++
	} else if image != 0 {
		b.Batches[n-1].Image = image
	}
	b.Instances = append(b.Instances, in)
	b.Batches[n-1].Count++
}

// addEffect appends the instance of an effect within the current clip, in
// a batch of its own, which reads its backdrop first when it reads one.
func (b *Builder) addEffect(s *scene.Scene, op *scene.Op) {
	cur := &b.stack[len(b.stack)-1]
	if cur.scissor.Empty() {
		return
	}
	fx := &s.Effects[op.Start]
	p := &fx.Params
	in := Instance{
		Rect: rect(op.Rect), Radii: scene.Corners(op.Rect, op.Radii, op.Continuous),
		Inner: p[0], Color: p[1], Color2: p[2], Border: p[3], Grad: p[4],
		Clip: rect(cur.rect), ClipRadii: cur.radii,
		Params: [4]float32{6, 0, 1, opacity(op.Opacity)},
	}
	if !b.prepare(&in, op.Transform, cur) {
		return
	}
	batch := Batch{Start: len(b.Instances), Count: 1, Scissor: cur.scissor, Effect: fx.Effect, ClipMask: b.preparedMask}
	if fx.Effect.Backdrop {
		bk := scene.BackdropOf(op.Transform.Bounds(op.Rect), fx.Blur, s.Width, s.Height)
		if bk.Area.Empty() {
			return
		}
		b.Backdrops = append(b.Backdrops, bk)
		batch.Backdrop = len(b.Backdrops)
		tw, th := bk.Size()
		in.UV = [4]float32{float32(bk.Area.Min.X), float32(bk.Area.Min.Y), float32(tw), float32(th)}
		in.Params[2] = float32(bk.Down)
	}
	b.Batches = append(b.Batches, batch)
	b.Instances = append(b.Instances, in)
}

// addHole appends a hole within the current clip, in a batch of holes:
// an opaque fill, whose coverage the renderers take away.
func (b *Builder) addHole(op *scene.Op) {
	cur := &b.stack[len(b.stack)-1]
	in := Instance{
		Rect: rect(op.Rect), Radii: scene.Corners(op.Rect, op.Radii, op.Continuous),
		Color:  [4]float32{0, 0, 0, 1},
		Params: [4]float32{0, 0, float32(scene.PaintSolid), opacity(op.Opacity)},
	}
	if !b.prepare(&in, op.Transform, cur) {
		return
	}
	if n := len(b.Batches); n == 0 || !b.Batches[n-1].Hole || b.Batches[n-1].Scissor != cur.scissor ||
		b.Batches[n-1].ClipMask != b.preparedMask {
		b.Batches = append(b.Batches, Batch{Start: len(b.Instances), Scissor: cur.scissor, Hole: true, ClipMask: b.preparedMask})
	}
	b.Instances = append(b.Instances, in)
	b.Batches[len(b.Batches)-1].Count++
}

func rect(r scene.Rect) [4]float32 { return [4]float32{r.X, r.Y, r.W, r.H} }

func straight(c scene.Color) [4]float32 {
	return [4]float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255, float32(c.A) / 255}
}

// wideOf returns the colors outside the sRGB gamut of an op or a glyph of
// s whose Wide is i, or nil when it has none or b does not draw them.
func (b *Builder) wideOf(s *scene.Scene, i uint16) *scene.WideColors {
	if !b.Wide || i == 0 || int(i) > len(s.Wide) {
		return nil
	}
	return &s.Wide[i-1]
}

// wideColor returns the color of w that bit names, or c when w has none.
func wideColor(w *scene.WideColors, bit scene.WideSet, c scene.Color) [4]float32 {
	if w == nil || w.Set&bit == 0 {
		return straight(c)
	}
	switch bit {
	case scene.WideColor:
		return w.Color
	case scene.WideColor2:
		return w.Color2
	}
	return w.Border
}

func opacity(o float32) float32 {
	if o == 0 {
		return 1
	}
	return o
}
