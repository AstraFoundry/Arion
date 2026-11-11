//go:build darwin

package metal

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo/internal/gpu"
	"github.com/egoist/mygo/internal/gpu/gputest"
	"github.com/egoist/mygo/internal/scene"
)

// unavailable ends a test for which Metal cannot be used: it skips it, or, where
// MYGO_GPU_REQUIRED is set (a machine with a GPU, CI for it), fails it, so that
// a renderer that does not start does not turn the identity tests into passing
// skips.
func unavailable(t testing.TB, err error) {
	t.Helper()
	if os.Getenv("MYGO_GPU_REQUIRED") != "" {
		t.Fatal("no Metal (MYGO_GPU_REQUIRED is set):", err)
	}
	t.Skip("no Metal:", err)
}

func init() {
	// Tests compare the specialized pipelines with the general ones, so frames
	// make the pipelines they need first (TestFramesNeverWaitForACompile
	// tests the other way).
	syncPipelines = true
}

func TestDrawsAsTheCPURenderer(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	s := gputest.Scene()
	// Twice: the second frame updates what the first uploaded.
	for range 2 {
		pix, err := r.renderOffscreen(s)
		if err != nil {
			t.Fatal(err)
		}
		gputest.Compare(t, "metal", pix, s.Width*4, s)
	}

	for frame := range 3 {
		transformed := gputest.TransformedScene()
		if frame == 2 {
			transformed.Ops[0].Transform = scene.Translation(40, 40)
		}
		pix, err := r.renderOffscreen(transformed)
		if err != nil {
			t.Fatal(err)
		}
		gputest.Compare(t, "metal-affine", pix, transformed.Width*4, transformed)
	}
	for range 2 {
		s = gputest.PerspectiveScene()
		pix, err := r.renderOffscreen(s)
		if err != nil {
			t.Fatal(err)
		}
		gputest.Compare(t, "metal-perspective", pix, s.Width*4, s)
	}
	for range 2 {
		s = gputest.BehindOriginScene()
		pix, err := r.renderOffscreen(s)
		if err != nil {
			t.Fatal(err)
		}
		gputest.Compare(t, "metal-behind-origin", pix, s.Width*4, s)
	}
	for i, s := range gputest.TranslatedScenes() {
		pix, err := r.renderOffscreen(s)
		if err != nil {
			t.Fatal(err)
		}
		gputest.Compare(t, "metal-translated"+[]string{"", "-between-pixels"}[i], pix, s.Width*4, s)
	}
	for range 2 {
		s = gputest.MaskedClipScene()
		pix, err := r.renderOffscreen(s)
		if err != nil {
			t.Fatal(err)
		}
		gputest.Compare(t, "metal-masked-clips", pix, s.Width*4, s)
	}
	for range 2 {
		s = gputest.CornerClipScene()
		pix, err := r.renderOffscreen(s)
		if err != nil {
			t.Fatal(err)
		}
		gputest.Compare(t, "metal-corner-clips", pix, s.Width*4, s)
	}
	s = gputest.ContinuousScene()
	pix, err := r.renderOffscreen(s)
	if err != nil {
		t.Fatal(err)
	}
	gputest.Compare(t, "metal-continuous", pix, s.Width*4, s)
}

// The CPU presenter starts without drawing resources; the first GPU
// frame creates them, and a frame after they are freed draws correctly.
func TestDrawingResourcesOnDemand(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	if r.queue != 0 || r.sampler != 0 || r.empty != 0 || r.cur.pipeline != 0 {
		t.Fatal("created GPU drawing resources before a GPU frame")
	}
	s := gputest.Scene()
	for range 2 {
		pix, err := r.renderOffscreen(s)
		if err != nil {
			t.Fatal(err)
		}
		gputest.Compare(t, "metal-on-demand", pix, s.Width*4, s)
		pool(r.releaseTextures)
	}
}

// TestShaderLibrary checks that the library compiled ahead of time comes
// from shader.metal as it is, and that Metal loads it.
func TestShaderLibrary(t *testing.T) {
	if gpu.SourceSum(shaderSource) != shaderLibrarySum {
		t.Fatal("shader.metal changed since shaderlib.go was generated: run go generate ./internal/gpu/metal on macOS")
	}
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	var lib id
	pool(func() { lib = r.compiledLibrary() })
	if lib == 0 {
		t.Fatal("Metal does not load the compiled library")
	}
	pool(func() { release(&lib) })
}

// TestTrimDrawables checks that a renderer keeps three drawables, and that
// trimming them shrinks them until the next frame.
func TestTrimDrawables(t *testing.T) {
	if err := load(); err != nil {
		t.Skip("no Metal:", err)
	}
	var layer id
	pool(func() { layer = send(class("CALayer"), "new") })
	if layer == 0 {
		t.Skip("no Core Animation")
	}
	defer pool(func() { release(&layer) })
	r, err := New(uintptr(layer))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Release()
	pool(func() {
		r.fit(800, 600, 1)
		if n := send(r.layer, "maximumDrawableCount"); n != 3 {
			t.Errorf("drawable count = %d, want 3", n)
		}
		r.trim()
		if !r.trimmed {
			t.Error("idle trim did not shrink the drawables")
		}
		r.fit(800, 600, 1)
		if r.trimmed {
			t.Error("next frame did not restore the drawables")
		}
	})
}

// TestChangedSince checks what frames drawn in memory copy into the
// drawables, which take turns, and after the GPU drew into one.
// TestWideColors draws wide colors into a float16 target, which keeps what
// leaves 0 to 1, and an Oklab gradient between them, which does not clamp;
// and the same scene into a BGRA8 one, which draws the nearest sRGB colors.
func TestWideColors(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	if !r.SetWide(true) {
		t.Fatal(r.wideErr)
	}
	s := &scene.Scene{}
	s.Reset(32, 16, scene.Color{})
	green, red := [4]float32{-0.25, 1.1, -0.1, 1}, [4]float32{1.15, -0.2, -0.1, 1}
	s.Wide = []scene.WideColors{{Color: green, Set: scene.WideColor}, {Color: green, Color2: red, Set: scene.WideColor | scene.WideColor2}}
	srgbGreen, srgbRed := scene.Color{G: 255, A: 255}, scene.Color{R: 255, A: 255}
	s.Ops = append(s.Ops,
		scene.Op{Kind: scene.OpFill, Rect: scene.Rect{W: 32, H: 8}, Color: srgbGreen, Wide: 1},
		scene.Op{Kind: scene.OpFill, Rect: scene.Rect{Y: 8, W: 32, H: 8}, Paint: scene.PaintOklab, Gradient: [4]float32{0, 0, 32, 0},
			Color: srgbGreen, Color2: srgbRed, Wide: 2})
	// General transforms must preserve the float16 wide-gamut path.
	for i := range s.Ops {
		s.Ops[i].Transform = scene.Scaling(1.1, 1)
	}
	var px [2][4]float32
	pool(func() {
		r.waitLast()
		r.cur = &r.formats[1]
		tex := r.newTexture(s.Width, s.Height, pixelFormatRGBA16Float, usageRenderTarget|usageShaderRead, nil, 0)
		if tex == 0 {
			err = errors.New("no target")
			return
		}
		defer release(&tex)
		var cb id
		if cb, err = r.encode(s, tex); err != nil {
			return
		}
		blit := send(cb, "blitCommandEncoder")
		send(blit, "synchronizeResource:", tex)
		send(blit, "endEncoding")
		send(cb, "commit")
		send(cb, "waitUntilCompleted")
		pix := make([]uint16, s.Width*s.Height*4)
		msgGetBytes(tex, sel("getBytes:bytesPerRow:fromRegion:mipmapLevel:"), unsafe.Pointer(&pix[0]), uint(s.Width*8),
			mtlRegion{W: uint(s.Width), H: uint(s.Height), D: 1}, 0)
		at := func(x, y int) (c [4]float32) {
			for i := range c {
				c[i] = half(pix[(y*s.Width+x)*4+i])
			}
			return c
		}
		px[0], px[1] = at(4, 2), at(16, 12)
	})
	if err != nil {
		t.Fatal(err)
	}
	// The pixel holds the straight color premultiplied by alpha 1, in the
	// order of the target's components (RGBA).
	for i, want := range green {
		if d := px[0][i] - want; d > 0.01 || d < -0.01 {
			t.Errorf("solid wide color: component %d is %v, want %v", i, px[0][i], want)
		}
	}
	// Midway between two colors, at least one component is out of 0 to 1:
	// the mix of the greens and reds of the ends, in Oklab, keeps the
	// chroma the ends have beyond sRGB.
	if m := px[1]; !(m[0] < 0 || m[0] > 1 || m[1] < 0 || m[1] > 1 || m[2] < 0 || m[2] > 1) {
		t.Errorf("gradient midpoint %v is inside sRGB", m)
	}

	// A BGRA8 target draws the sRGB colors of the ops.
	pix, err := r.renderOffscreen(s)
	if err != nil {
		t.Fatal(err)
	}
	if b, g, red, a := pix[(2*s.Width+4)*4], pix[(2*s.Width+4)*4+1], pix[(2*s.Width+4)*4+2], pix[(2*s.Width+4)*4+3]; b != 0 || g != 255 || red != 0 || a != 255 {
		t.Errorf("BGRA8 target: %v %v %v %v, want the sRGB green", b, g, red, a)
	}
}

// half converts an IEEE half-precision float.
func half(h uint16) float32 {
	sign := float32(1)
	if h&0x8000 != 0 {
		sign = -1
	}
	e, m := int(h>>10)&0x1f, float32(h&0x3ff)
	switch e {
	case 0:
		return sign * m / 1024 / 16384
	case 31:
		return sign * float32(math.Inf(1))
	}
	return sign * (1 + m/1024) * float32(math.Pow(2, float64(e-15)))
}

// TestShadowSplitDrawsTheSamePixels draws panels with stacks of shadows of
// several radii, blurs, spreads and offsets, with the shadows drawn around the
// box casting them and whole, and requires the same bytes.
func TestShadowSplitDrawsTheSamePixels(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	for i, c := range gputest.ShadowCases() {
		s := gputest.ShadowPanelScene(c)
		r.b.NoShadowSplit = false
		split, err := r.renderOffscreen(s)
		if err != nil {
			t.Fatal(err)
		}
		instances := len(r.b.Instances)
		r.b.NoShadowSplit = true
		whole, err := r.renderOffscreen(s)
		if err != nil {
			t.Fatal(err)
		}
		if instances <= len(r.b.Instances) {
			t.Errorf("case %d: the split drew %d instances, whole %d", i, instances, len(r.b.Instances))
		}
		if !bytes.Equal(split, whole) {
			n := 0
			for k := range split {
				if split[k] != whole[k] {
					n++
				}
			}
			t.Errorf("case %d: %d bytes differ between split and whole shadows", i, n)
		}
		gputest.Compare(t, "metal-shadow-split", split, s.Width*4, s)
	}
}

// renderRaw draws s into a target of the renderer's format, BGRA8 or, when
// wide, float16, and returns its bytes as the GPU keeps them.
func renderRaw(t *testing.T, r *Renderer, s *scene.Scene, wide bool) (raw []byte) {
	t.Helper()
	var err error
	pool(func() {
		r.waitLast()
		r.cur = &r.formats[0]
		format, bpp := uint(pixelFormatBGRA8Unorm), 4
		if wide {
			r.cur, format, bpp = &r.formats[1], pixelFormatRGBA16Float, 8
		}
		tex := r.newTexture(s.Width, s.Height, format, usageRenderTarget|usageShaderRead, nil, 0)
		if tex == 0 {
			err = errors.New("no target")
			return
		}
		defer release(&tex)
		var cb id
		if cb, err = r.encode(s, tex); err != nil {
			return
		}
		blit := send(cb, "blitCommandEncoder")
		send(blit, "synchronizeResource:", tex)
		send(blit, "endEncoding")
		send(cb, "commit")
		send(cb, "waitUntilCompleted")
		if int(send(cb, "status")) == statusError {
			err = fmt.Errorf("the frame failed: %s", describe(send(cb, "error")))
			return
		}
		raw = make([]byte, s.Width*s.Height*bpp)
		msgGetBytes(tex, sel("getBytes:bytesPerRow:fromRegion:mipmapLevel:"), unsafe.Pointer(&raw[0]), uint(s.Width*bpp),
			mtlRegion{W: uint(s.Width), H: uint(s.Height), D: 1}, 0)
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func differing(a, b []byte) (n int) {
	for i := range a {
		if a[i] != b[i] {
			n++
		}
	}
	return n
}

// TestCullDrawsTheSamePixels draws scenes of overlapping fills with the ops
// that later opaque fills cover left out and with all of them, in BGRA8 and
// float16 targets, and requires the same bytes, and the CPU renderer's pixels.
func TestCullDrawsTheSamePixels(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	cases := append(gputest.CullCases(), gputest.CullCase{Name: "reference scene", Scene: gputest.Scene(), Culled: -1})
	for _, wide := range []bool{false, true} {
		if wide && !r.SetWide(true) {
			t.Log("no wide target:", r.wideErr)
			continue
		}
		for _, c := range cases {
			r.b.NoCull = false
			culled := renderRaw(t, r, c.Scene, wide)
			n, instances := r.b.Culled, len(r.b.Instances)
			r.b.NoCull = true
			whole := renderRaw(t, r, c.Scene, wide)
			if c.Culled >= 0 && n != c.Culled {
				t.Errorf("%s wide=%v: %d ops left out, want %d", c.Name, wide, n, c.Culled)
			}
			if n > 0 && instances >= len(r.b.Instances) {
				t.Errorf("%s wide=%v: %d instances with the cull, %d without", c.Name, wide, instances, len(r.b.Instances))
			}
			if d := differing(culled, whole); d != 0 {
				t.Errorf("%s wide=%v: %d bytes differ between culled and whole", c.Name, wide, d)
			}
			if !wide {
				gputest.Compare(t, "metal-cull-"+c.Name, culled, c.Scene.Width*4, c.Scene)
			}
		}
	}
	r.b.NoCull = false
}

// TestRingSplitDrawsTheSamePixels draws border-only fills around the core
// inside their border and whole, in BGRA8 and float16 targets, and requires
// the same bytes, and the CPU renderer's pixels.
func TestRingSplitDrawsTheSamePixels(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	for _, wide := range []bool{false, true} {
		if wide && !r.SetWide(true) {
			t.Log("no wide target:", r.wideErr)
			continue
		}
		for _, c := range gputest.RingCases() {
			r.b.NoRingSplit = false
			split := renderRaw(t, r, c.Scene, wide)
			instances := len(r.b.Instances)
			r.b.NoRingSplit = true
			whole := renderRaw(t, r, c.Scene, wide)
			// Split, a ring is more pieces, or none where it shows nothing.
			if c.Split && instances == len(r.b.Instances) {
				t.Errorf("%s wide=%v: the split drew %d instances, as whole", c.Name, wide, instances)
			}
			if d := differing(split, whole); d != 0 {
				t.Errorf("%s wide=%v: %d bytes differ between split and whole", c.Name, wide, d)
			}
			if !wide {
				gputest.Compare(t, "metal-ring-"+c.Name, split, c.Scene.Width*4, c.Scene)
			}
		}
	}
	r.b.NoRingSplit = false
}

// TestOpaqueCoreDrawsTheSamePixels draws fills as opaque cores, drawn without
// blending, with blended edges around them, and whole and blended, in BGRA8
// and float16 targets, with and without culling, and requires the same bytes
// (and the CPU renderer's pixels).
func TestOpaqueCoreDrawsTheSamePixels(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	cases := identityCases()
	cores := 0
	for _, wide := range []bool{false, true} {
		if wide && !r.SetWide(true) {
			t.Log("no wide target:", r.wideErr)
			continue
		}
		for _, noCull := range []bool{false, true} {
			r.b.NoCull = noCull
			for _, c := range cases {
				r.b.Opaque = true
				core := renderRaw(t, r, c.Scene, wide)
				n, opaque := r.b.OpaqueCores, 0
				for _, bt := range r.b.Batches {
					if bt.Opaque {
						opaque++
					}
				}
				r.b.Opaque = false
				blended := renderRaw(t, r, c.Scene, wide)
				if n > 0 && opaque == 0 {
					t.Errorf("%s: %d cores, no opaque batch", c.Name, n)
				}
				cores += n
				if d := differing(core, blended); d != 0 {
					t.Errorf("%s wide=%v noCull=%v: %d bytes differ between opaque cores and blended fills", c.Name, wide, noCull, d)
				}
				if !wide && !noCull && c.Scene.Width*c.Scene.Height < 1e6 {
					gputest.Compare(t, "metal-opaque-"+c.Name, core, c.Scene.Width*4, c.Scene)
				}
			}
		}
	}
	if cores == 0 {
		t.Error("no fill was drawn as an opaque core")
	}
	r.b.NoCull, r.b.Opaque = false, true
}

// identityCases are the scenes of the identity tests of changes to how
// scenes are drawn: the cases of the cull, ring and shadow tests, the
// scenes of fills and opaque fills at window sizes, the reference scene and
// the scenes of transformed content.
func identityCases() []gputest.CullCase {
	cases := []gputest.CullCase{{Name: "opaque scene", Scene: gputest.OpaqueScene(900, 700)},
		{Name: "window opaque scene", Scene: gputest.OpaqueScene(3456, 2000)},
		{Name: "odd opaque scene", Scene: gputest.OpaqueScene(2049, 1031)},
		{Name: "window fills", Scene: gputest.FillScene(3456, 2000)}, {Name: "odd window fills", Scene: gputest.FillScene(2049, 1031)},
		{Name: "reference scene", Scene: gputest.Scene()}, {Name: "transformed scene", Scene: gputest.TransformedScene()},
		{Name: "perspective scene", Scene: gputest.PerspectiveScene()}, {Name: "axis clip scene", Scene: gputest.AxisClipScene()},
		{Name: "continuous scene", Scene: gputest.ContinuousScene()},
		{Name: "canvas scene", Scene: gputest.CanvasScene()}}
	cases = append(cases, gputest.CullCases()...)
	for _, c := range gputest.RingCases() {
		cases = append(cases, gputest.CullCase{Name: "ring " + c.Name, Scene: c.Scene})
	}
	for i, c := range gputest.ShadowCases() {
		cases = append(cases, gputest.CullCase{Name: fmt.Sprint("shadow case ", i), Scene: gputest.ShadowPanelScene(c)})
	}
	cases = append(cases, gputest.CullCase{Name: "wide colors", Scene: gputest.WideScene()})
	for _, c := range gputest.MaskedCases() {
		cases = append(cases, gputest.CullCase{Name: c.Name, Scene: c.Scene})
	}
	// Every case again with a transformed op in front, which gives rounded
	// clips masks (the splits must not touch the clip's mask area).
	n := len(cases)
	for _, c := range cases[:n] {
		cases = append(cases, gputest.CullCase{Name: c.Name + " (transformed scene)", Scene: gputest.Transformed(c.Scene)})
	}
	return cases
}

// TestClassesDrawTheSamePixels draws every scene with a fragment function
// specialized for each batch (gpu.Class), small batches joined (with one of
// their classes, gpu.Batch.Joined) and apart (gpu.Builder.NoJoin), and with
// the general one, in BGRA8 and float16 targets, and requires the same bytes.
func TestClassesDrawTheSamePixels(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	var seen [gpu.NumClasses]int
	joined := map[uint32]int{}
	for _, wide := range []bool{false, true} {
		if wide && !r.SetWide(true) {
			t.Log("no wide target:", r.wideErr)
			continue
		}
		// And scenes whose small batches of classes interleave.
		cases := append(shaderIdentityCases(), gputest.CullCase{Name: "boxes", Scene: gputest.BoxesScene(640, 400, 120, false)},
			gputest.CullCase{Name: "translated boxes", Scene: gputest.BoxesScene(640, 400, 120, true)},
			gputest.CullCase{Name: "rows", Scene: gputest.RowsScene(640, 400, 30)})
		for _, c := range cases {
			r.b.NoClasses = true
			general := renderRaw(t, r, c.Scene, wide)
			for _, apart := range []bool{false, true} {
				r.b.NoClasses, r.b.NoJoin = false, apart
				c.Scene.EachCanvas(func(c *scene.Image) { c.Changed() })
				special := renderRaw(t, r, c.Scene, wide)
				var classes [gpu.NumClasses]int
				for _, bt := range r.b.Batches {
					classes[bt.Class]++
					if bt.Joined != 0 {
						joined[bt.Joined]++
					}
				}
				for k, n := range classes {
					seen[k] += n
				}
				if d := differing(special, general); d != 0 {
					t.Errorf("%s wide=%v apart=%v: %d bytes differ between specialized and general shaders (classes %v)", c.Name, wide, apart, d, classes)
				}
			}
		}
	}
	r.b.NoClasses, r.b.NoJoin = false, false
	for k, n := range seen {
		if n == 0 {
			t.Errorf("no batch of class %d was drawn", k)
		}
		// The test proves nothing if a specialized pipeline did not build and
		// the frames drew with the general one in its place.
		cl := gpu.Class(k)
		if cl != gpu.ClassGeneral && specializable(pixelFormatBGRA8Unorm, 1<<cl, blendOver) &&
			!r.usedSpecialized(pipeKey{pixelFormatBGRA8Unorm, 1 << cl, false, blendOver}) && !r.usedSpecialized(pipeKey{pixelFormatBGRA8Unorm, 1 << cl, true, blendOver}) {
			t.Errorf("class %d drew with no specialized pipeline", k)
		}
	}
	// Shadows joined with fills (boxes), and fills with glyphs (rows).
	for _, set := range []uint32{1<<gpu.ClassShadow | 1<<gpu.ClassPlain, 1<<gpu.ClassPlain | 1<<gpu.ClassGlyph} {
		if joined[set] == 0 {
			t.Errorf("no batch joined the classes %b", set)
		}
		if !r.usedSpecialized(pipeKey{pixelFormatBGRA8Unorm, set, false, blendOver}) && !r.usedSpecialized(pipeKey{pixelFormatBGRA8Unorm, set, true, blendOver}) {
			t.Errorf("the classes %b drew with no specialized pipeline", set)
		}
	}
	t.Logf("joined classes: %v", joined)
	if built, failed := r.pipelineCounts(); failed != 0 || built == 0 {
		t.Errorf("%d specialized pipelines built, %d failed", built, failed)
	}
}

// TestSplitsDrawTheSamePixels draws every identity case with the shadow and
// ring splits and without them, in BGRA8 and float16, and requires the same
// bytes. The cases include rounded, continuous and rotated clips with masks,
// where the pieces must not change the clip's mask area.
func TestSplitsDrawTheSamePixels(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	pieces := 0
	for _, wide := range []bool{false, true} {
		if wide && !r.SetWide(true) {
			t.Log("no wide target:", r.wideErr)
			continue
		}
		for _, c := range identityCases() {
			r.b.NoRingSplit, r.b.NoShadowSplit = false, false
			split := renderRaw(t, r, c.Scene, wide)
			n := len(r.b.Instances)
			r.b.NoRingSplit, r.b.NoShadowSplit = true, true
			whole := renderRaw(t, r, c.Scene, wide)
			pieces += n - len(r.b.Instances)
			if d := differing(split, whole); d != 0 {
				t.Errorf("%s wide=%v: %d bytes differ between split and whole", c.Name, wide, d)
			}
		}
	}
	if pieces == 0 {
		t.Error("nothing was split")
	}
	r.b.NoRingSplit, r.b.NoShadowSplit = false, false
}

// TestCornerSplitDrawsTheSamePixels draws every identity case, and each
// with continuous corners everywhere, with shapes of continuous corners
// drawn as nine-patches (their corners and a twin with square corners) and
// whole (gpu.Builder.NoCornerSplit), in BGRA8 and float16, and requires the
// same bytes, and of the cards, the CPU renderer's pixels.
func TestCornerSplitDrawsTheSamePixels(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	cases := []gputest.CullCase{{Name: "cards", Scene: gputest.CardsScene(1400, 900)},
		{Name: "odd cards", Scene: gputest.CardsScene(1001, 777)}, {Name: "retina cards", Scene: gputest.CardsScene(2560, 1600)}}
	for _, c := range identityCases() {
		cases = append(cases, c, gputest.CullCase{Name: c.Name + " (continuous)", Scene: gputest.Continuously(c.Scene)})
	}
	pieces := 0
	for _, wide := range []bool{false, true} {
		if wide && !r.SetWide(true) {
			t.Log("no wide target:", r.wideErr)
			continue
		}
		for _, c := range cases {
			r.b.NoCornerSplit = false
			split := renderRaw(t, r, c.Scene, wide)
			n := len(r.b.Instances)
			r.b.NoCornerSplit = true
			whole := renderRaw(t, r, c.Scene, wide)
			pieces += n - len(r.b.Instances)
			if d := differing(split, whole); d != 0 {
				t.Errorf("%s wide=%v: %d bytes differ between nine-patches and whole shapes", c.Name, wide, d)
			}
			if !wide && strings.HasSuffix(c.Name, "cards") {
				r.b.NoCornerSplit = false
				pix, err := r.renderOffscreen(c.Scene)
				if err != nil {
					t.Fatal(err)
				}
				gputest.Compare(t, "metal-"+strings.ReplaceAll(c.Name, " ", "-"), pix, c.Scene.Width*4, c.Scene)
			}
		}
	}
	if pieces == 0 {
		t.Error("nothing was drawn in pieces")
	}
	r.b.NoCornerSplit = false
}

// holdPipelineBuilds blocks specialized compilation, including an accidental
// synchronous build in pipeFor. The watchdog only breaks a deadlock: a passing
// test must finish its frame while the gate remains closed. A separate startup
// deadline also catches a worker that never enters a build.
func holdPipelineBuilds(t *testing.T, r *Renderer) (waitStarted, resume func()) {
	t.Helper()
	started, gate, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var entered, released sync.Once
	unblock := func() { released.Do(func() { close(gate) }) }
	r.pipes.beforeBuild = func() {
		entered.Do(func() { close(started) })
		<-gate
	}
	go func() {
		defer close(done)
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		select {
		case <-gate:
			return
		case <-started:
			timer.Reset(30 * time.Second)
		case <-timer.C:
			t.Error("the pipeline worker did not start a build")
			unblock()
			return
		}
		select {
		case <-gate:
		case <-timer.C:
			t.Error("the frame or Release waited for the blocked pipeline compile")
			unblock()
		}
	}()
	return func() {
		select {
		case <-started:
		case <-done:
			t.Fatal("no blocked pipeline build to test")
		}
	}, func() { unblock(); <-done }
}

// TestFramesNeverWaitForACompile checks that a frame whose batches need a
// specialized pipeline that is not made yet draws with the general one at
// once, whatever the compile takes, that the worker then makes it and later
// frames use it, drawing the same pixels, and that a second renderer of the
// device finds the pipelines made.
func TestFramesNeverWaitForACompile(t *testing.T) {
	syncPipelines = false
	defer func() { syncPipelines = true }()
	// Other tests made pipelines the device's cache holds: start empty.
	pipeCache.Lock()
	for k, pipe := range pipeCache.m {
		release(&pipe)
		delete(pipeCache.m, k)
	}
	pipeCache.Unlock()
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	waitStarted, resume := holdPipelineBuilds(t, r)
	defer resume() // unblock the worker before Release, including on a failed test
	s := gputest.OpaqueScene(900, 700)
	first := renderRaw(t, r, s, false)
	waitStarted()
	if built, failed := r.pipelineCounts(); built != 0 || failed != 0 {
		t.Fatalf("the first frame completed after compiling pipelines: built=%d failed=%d", built, failed)
	}
	if r.pipes.fallbacks == 0 {
		t.Fatal("the first frame did not draw with a general fallback")
	}
	f := r.cur
	if p := r.pipeFor(f, 1<<gpu.ClassGlyph, true, blendOver); p != f.pipeline {
		t.Error("a pipeline that is not made yet is not the general one")
	}
	// The frame finished while compilation was held. Let the worker make
	// the pipelines, one after the other, and wait for the plain fill's.
	resume()
	plain := f.pipeline
	for i := 0; i < 200 && plain == f.pipeline; i++ {
		time.Sleep(50 * time.Millisecond)
		plain = r.pipeFor(f, 1<<gpu.ClassPlain, false, blendOver)
		if plain == 0 {
			t.Fatal("no pipeline")
		}
		if plain != f.pipeline {
			break
		}
	}
	if plain == f.pipeline {
		t.Fatal("the worker did not make the specialized pipeline")
	}
	second := renderRaw(t, r, s, false)
	if d := differing(first, second); d != 0 {
		t.Errorf("%d bytes differ between the frame drawn with the general pipelines and with the specialized ones", d)
	}
	// Another renderer of the device: the pipelines are in the cache.
	r2, err := newRenderer()
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Release()
	renderRaw(t, r2, s, false)
	if p := r2.pipeFor(r2.cur, 1<<gpu.ClassPlain, false, blendOver); p == r2.cur.pipeline {
		t.Error("a second renderer did not find the pipeline made by the first")
	}
}

//go:embed testdata/shader_upstream.metal
var upstreamShader string

// TestMatchesTheUpstreamShader draws every identity case with the renderer as it
// is, every optimization on, and with the shader of upstream f2eca9f7 (a copy in
// testdata, compiled when the test runs) and a builder that does none of
// them (no cull, no splits, no opaque cores, no classes, no clip skip), in BGRA8
// and float16 targets, and requires the same bytes. It guards what the
// knob-against-knob tests cannot: the restructured plain path, the early return
// of frameOf and the guarded paths of the shader, against the old shader.
func TestMatchesTheUpstreamShader(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	ref, err := newRenderer()
	if err != nil {
		t.Fatal(err)
	}
	defer ref.Release()
	ref.source = upstreamShader
	ref.b.Opaque, ref.b.NoCull, ref.b.NoShadowSplit, ref.b.NoRingSplit, ref.b.NoClasses, ref.b.NoClipSkip = false, true, true, true, true, true
	for _, wide := range []bool{false, true} {
		if wide && !(r.SetWide(true) && ref.SetWide(true)) {
			t.Log("no wide target:", r.wideErr)
			continue
		}
		for _, c := range shaderIdentityCases() {
			// What the upstream shader draws: no gradients of stops yet.
			sc := gputest.WithoutNewPaints(c.Scene)
			got := renderRaw(t, r, sc, wide)
			want := renderRaw(t, ref, sc, wide)
			if d := differing(got, want); d != 0 {
				t.Errorf("%s wide=%v: %d bytes differ from the upstream shader", c.Name, wide, d)
			}
		}
	}
}

// TestCullTransformedShadows draws a shadow under a transform, scaled, rotated,
// scaled unevenly and in perspective, with an opaque fill over exactly what the
// planner took to be the shadow's reach before the blur was grown in the shadow's
// own units (the blur's reach, in device pixels, grows with the scale), and
// requires that culling changes no pixel; and that a cover of the whole frame
// still culls it.
func TestCullTransformedShadows(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	cases := []struct {
		name string
		m    scene.Affine
		rect scene.Rect
		blur float32
		cast bool
	}{
		{"scale4", scene.Scaling(4, 4), scene.Rect{X: 50, Y: 50, W: 40, H: 40}, 20, false},
		{"scale4cast", scene.Scaling(4, 4), scene.Rect{X: 50, Y: 50, W: 40, H: 40}, 20, true},
		{"nonuniform", scene.Scaling(4, 1.5), scene.Rect{X: 50, Y: 150, W: 40, H: 40}, 20, false},
		{"rotate30x3", scene.Translation(320, 320).Mul(scene.Rotation(30)).Mul(scene.Scaling(3, 3)).Mul(scene.Translation(-20, -20)), scene.Rect{W: 40, H: 40}, 16, false},
		{"scale1.5", scene.Scaling(1.5, 1.5), scene.Rect{X: 150, Y: 150, W: 60, H: 60}, 60, false},
		{"perspective", scene.Projective(2, 0, 0, 2, 100, 100, 0.0008, 0), scene.Rect{X: 50, Y: 50, W: 60, H: 60}, 20, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := c.m.Bounds(c.rect)
			g := 2*c.blur + 3 + 1 + 2 // the growth the planner had, plus the occluder's inset
			x0, y0 := float32(math.Floor(float64(b.X-g))), float32(math.Floor(float64(b.Y-g)))
			x1, y1 := float32(math.Ceil(float64(b.X+b.W+g))), float32(math.Ceil(float64(b.Y+b.H+g)))
			sh := scene.Op{Kind: scene.OpShadow, Rect: c.rect, Color: scene.Color{A: 255}, Blur: c.blur, Transform: c.m}
			if c.cast {
				sh.Cast = scene.Rect{X: c.rect.X + 4, Y: c.rect.Y + 4, W: c.rect.W - 8, H: c.rect.H - 8}
			}
			s := &scene.Scene{Width: 640, Height: 640, Clear: scene.Color{R: 255, G: 255, B: 255, A: 255}}
			s.Ops = []scene.Op{sh, {Kind: scene.OpFill, Rect: scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, Color: scene.Color{R: 180, G: 180, B: 180, A: 255}}}
			r.b.NoCull = false
			got := renderRaw(t, r, s, false)
			culled := r.b.Culled
			r.b.NoCull = true
			want := renderRaw(t, r, s, false)
			r.b.NoCull = false
			if d := differing(got, want); d != 0 {
				t.Errorf("culled %d op(s), %d bytes differ", culled, d)
			}
		})
	}
	s := &scene.Scene{Width: 640, Height: 640}
	s.Ops = []scene.Op{
		{Kind: scene.OpShadow, Rect: scene.Rect{X: 50, Y: 50, W: 40, H: 40}, Color: scene.Color{A: 255}, Blur: 20, Transform: scene.Scaling(4, 4)},
		{Kind: scene.OpFill, Rect: scene.Rect{X: 0, Y: 0, W: 640, H: 640}, Color: scene.Color{R: 180, G: 180, B: 180, A: 255}},
	}
	renderRaw(t, r, s, false)
	if r.b.Culled != 1 {
		t.Errorf("a full-frame cover culled %d ops, want 1", r.b.Culled)
	}
}

// TestFastFillDrawsTheSamePixels draws scenes of plain fills along the
// shader's fast path (and with the clip's coverage skipped where the builder
// found it to be 1) and along its general path, in BGRA8 and float16
// targets, at the size of a window of a retina display too, and requires the
// same bytes.
func TestFastFillDrawsTheSamePixels(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	cases := []gputest.CullCase{{Name: "reference scene", Scene: gputest.Scene()}, {Name: "small fills", Scene: gputest.FillScene(900, 700)},
		{Name: "window fills", Scene: gputest.FillScene(3456, 2000)}, {Name: "odd window fills", Scene: gputest.FillScene(2049, 1031)}}
	for _, c := range gputest.CullCases() {
		cases = append(cases, c)
	}
	for _, wide := range []bool{false, true} {
		if wide && !r.SetWide(true) {
			t.Log("no wide target:", r.wideErr)
			continue
		}
		for _, c := range cases {
			r.b.GeneralFill, r.b.NoClipSkip = false, false
			fast := renderRaw(t, r, c.Scene, wide)
			plain, unclipped := 0, 0
			for _, in := range r.b.Instances {
				if in.Params[0] == 0 && in.Params[2] == 0 && in.UV == [4]float32{} {
					plain++
				}
				if in.Transform1[3] == 3 {
					unclipped++
				}
			}
			r.b.GeneralFill, r.b.NoClipSkip = true, true
			general := renderRaw(t, r, c.Scene, wide)
			if c.Name == "window fills" && (plain == 0 || unclipped == 0) {
				t.Errorf("%s: %d plain fills, %d without clip coverage", c.Name, plain, unclipped)
			}
			for _, in := range r.b.Instances {
				if in.Params[0] == 0 && in.Transform1[3] == 3 {
					t.Fatalf("%s: a fill without clip coverage along the general path", c.Name)
				}
			}
			if d := differing(fast, general); d != 0 {
				t.Errorf("%s wide=%v: %d bytes differ between the fast and the general path", c.Name, wide, d)
			}
		}
	}
	r.b.GeneralFill, r.b.NoClipSkip = false, false
}

// Fractionally scaled pixel art uses different filtering in the CPU renderer.
// Compare these scenes byte for byte with the current upstream GPU shader.
func shaderIdentityCases() []gputest.CullCase {
	return append(identityCases(), gputest.CullCase{Name: "pixelated sprites", Scene: gputest.SpriteScene(320, 240, 1.5, 2, 30)})
}

// TestFramesInFlight encodes and commits frames one after the other without
// waiting for the GPU (as Render does, with a ring of instance buffers), among
// them frames that change an atlas and an image in place, which waits for the
// frames in flight, and requires each frame to have the pixels it has when
// drawn alone.
func TestFramesInFlight(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	const frames = 12
	const w, h = 480, 360
	img := scene.NewImageRGBA(8, 8, make([]byte, 8*8*4))
	s := gputest.OpaqueScene(w, h)
	x, y, _ := s.MaskAtlas.Alloc(16, 16)
	s.Glyphs = append(s.Glyphs, scene.Glyph{X: 300, Y: 300, W: 16, H: 16, U: uint16(x), V: uint16(y), UW: 16, VH: 16, Color: scene.Color{A: 255}})
	s.Ops = append(s.Ops, scene.Op{Kind: scene.OpImage, Rect: scene.Rect{X: 20, Y: 250, W: 80, H: 80}, Image: img, Src: scene.Rect{W: 8, H: 8}},
		scene.Op{Kind: scene.OpGlyphs, Start: int32(len(s.Glyphs) - 1), End: int32(len(s.Glyphs))},
		scene.Op{Kind: scene.OpFill, Rect: scene.Rect{X: 30, Y: 40, W: 200, H: 150}, Radii: [4]float32{12, 12, 12, 12}, Color: scene.Color{G: 120, B: 200, A: 255}})
	last := len(s.Ops) - 1
	canvas := scene.NewCanvas(160, 100)
	canvas.Scene.Ops = []scene.Op{{Kind: scene.OpFill, Rect: scene.Rect{W: 160, H: 100}, Color: scene.Color{R: 40, A: 255}},
		{Kind: scene.OpFill, Rect: scene.Rect{X: 10, Y: 10, W: 40, H: 40}, Color: scene.Color{G: 220, A: 255}}}
	s.Ops = append(s.Ops, scene.Op{Kind: scene.OpImage, Rect: scene.Rect{X: 300, Y: 20, W: 160, H: 100}, Image: canvas, Src: scene.Rect{W: 160, H: 100}})
	// step makes frame i: the fill moves, and the atlas and the image change
	// in place, the atlas at the frames 4 and 8, the image at 5 and 9.
	step := func(i int) {
		canvas.Scene.Ops[1].Rect.X = 10 + float32(i)*7
		canvas.Scene.Ops[1].Color.R = byte(i * 20)
		canvas.Changed()
		s.Ops[last].Rect.X = 30 + float32(i)*20
		s.Ops[last].Color.R = byte(20 * i)
		if i%4 == 0 {
			pix := make([]byte, 16*16)
			for k := range pix {
				pix[k] = byte(k*i + 1)
			}
			s.MaskAtlas.Put(x, y, 16, 16, pix, 16)
		}
		if i%4 == 1 {
			for k := range img.Pix {
				img.Pix[k] = byte(k * (i + 3))
			}
			img.Changed()
		}
	}
	want := make([][]byte, frames)
	for i := range want { // each frame drawn alone and waited for
		step(i)
		if want[i], err = r.renderOffscreen(s); err != nil {
			t.Fatal(err)
		}
	}
	// A new renderer, so that its textures are made as the frames in flight
	// find them.
	r2, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r2.Release()
	clear(img.Pix) // as the first frame had it
	img.Changed()
	got := make([][]byte, frames)
	var targets [frames]id
	pool(func() {
		r2.cur = &r2.formats[0]
		for i := range targets {
			step(i)
			targets[i] = r2.newTexture(w, h, pixelFormatBGRA8Unorm, usageRenderTarget|usageShaderRead, nil, 0)
			cb, err := r2.encode(s, targets[i])
			if err != nil {
				t.Error(err)
				return
			}
			send(cb, "commit")
			r2.track(cb)
		}
		r2.waitLast()
		if r2.err != nil {
			t.Error(r2.err)
			return
		}
		for i := range targets {
			cb := send(r2.queue, "commandBuffer")
			blit := send(cb, "blitCommandEncoder")
			send(blit, "synchronizeResource:", targets[i])
			send(blit, "endEncoding")
			send(cb, "commit")
			send(cb, "waitUntilCompleted")
			got[i] = make([]byte, w*h*4)
			msgGetBytes(targets[i], sel("getBytes:bytesPerRow:fromRegion:mipmapLevel:"), unsafe.Pointer(&got[i][0]), uint(w*4),
				mtlRegion{W: w, H: h, D: 1}, 0)
			release(&targets[i])
		}
	})
	for i := range got {
		if got[i] == nil {
			t.Errorf("frame %d was not rendered", i)
			continue
		}
		if d := differing(got[i], want[i]); d != 0 {
			t.Errorf("frame %d: %d bytes differ from drawing it alone", i, d)
		}
	}
	if differing(want[0], want[1]) == 0 || differing(want[3], want[4]) == 0 || differing(want[4], want[5]) == 0 {
		t.Error("the frames do not differ: the test checks nothing")
	}
}

func TestCullKeepsHardShadowsWithNegativeBlur(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	for _, inset := range []bool{false, true} {
		shadow := scene.Op{Kind: scene.OpShadow, Rect: scene.Rect{X: 10, Y: 10, W: 100, H: 100}, Color: scene.Color{A: 255}, Blur: -10}
		if inset {
			shadow.Inset = true
			shadow.Cast = shadow.Rect
			shadow.Rect = scene.Rect{X: 20, Y: 20, W: 80, H: 80}
		}
		s := &scene.Scene{Width: 120, Height: 120, Clear: scene.Color{R: 255, G: 255, B: 255, A: 255}, Ops: []scene.Op{shadow, {Kind: scene.OpFill, Rect: scene.Rect{X: 25, Y: 25, W: 70, H: 70}, Color: scene.Color{R: 180, G: 180, B: 180, A: 255}}}}
		r.b.NoCull = false
		got := renderRaw(t, r, s, false)
		n := r.b.Culled
		r.b.NoCull = true
		want := renderRaw(t, r, s, false)
		if !bytes.Equal(got, want) {
			t.Errorf("inset=%v culled=%d changed=%d bytes", inset, n, differing(got, want))
		}
	}
}

// Closing a window waits for an active compile to release its library safely,
// but does not compile the rest of that window's speculative warmup queue.
func TestReleaseStopsQueuedPipelineCompiles(t *testing.T) {
	syncPipelines = false
	defer func() { syncPipelines = true }()
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	waitStarted, resume := holdPipelineBuilds(t, r)
	defer resume()
	renderRaw(t, r, gputest.Scene(), false)
	waitStarted()
	r.pipes.mu.Lock()
	pending, before := len(r.pipes.work), r.pipes.nbuilt
	r.pipes.mu.Unlock()
	if pending < 2 {
		t.Fatalf("need queued warmup work, got %d", pending)
	}
	// Release must close the queue before waiting for the active build. Let
	// that build finish only after observing the closed state.
	unblocked := make(chan struct{})
	go func() {
		defer close(unblocked)
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		tick := time.NewTicker(time.Millisecond)
		defer tick.Stop()
		for {
			r.pipes.mu.Lock()
			closed := r.pipes.closed
			r.pipes.mu.Unlock()
			if closed {
				resume()
				return
			}
			select {
			case <-tick.C:
			case <-timer.C:
				t.Error("Release did not close the pipeline queue")
				resume()
				return
			}
		}
	}()
	r.Release()
	<-unblocked
	after, failed := r.pipelineCounts()
	if after-before > 1 {
		t.Fatalf("Release built %d queued pipelines; only an active compile may finish", after-before)
	}
	if failed != 0 {
		t.Fatalf("%d pipeline builds failed", failed)
	}
}

// A shader precompiled (PrecompileShader) is the library and the pipeline
// a renderer on the default GPU draws it with.
func TestPrecompileShader(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		t.Skip(err)
	}
	defer r.Release()
	src := ShaderSource("struct Uniforms { float4 a; };\nfloat4 shaderMain(constant Uniforms& uniforms, texture2d<float> T0, texture2d<float> T1, texture2d<float> T2, texture2d<float> T3, float2 p, float4 rect) { return float4(0.25, 0.5, 0.75, 1); }\n")
	PrecompileShader(src)
	lib, pipe, ok := sharedShader(r.device, src, precompiledFormat)
	if !ok || lib == 0 || pipe == 0 {
		t.Fatalf("the renderer's device %#x shares nothing (default device %#x): library %#x, pipeline %#x", r.device, shaders.device, lib, pipe)
	}
	release(&lib)
	release(&pipe)
	if _, _, ok := sharedShader(r.device, src+"// another\n", precompiledFormat); ok {
		t.Error("a source never precompiled is shared")
	}
}

// Boxes that translations move draw the pixels of the same boxes placed
// without transforms, and those of the CPU renderer.
func TestTranslatedBoxes(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	placed, err := r.renderOffscreen(gputest.BoxesScene(400, 300, 60, false))
	if err != nil {
		t.Fatal(err)
	}
	s := gputest.BoxesScene(400, 300, 60, true)
	moved, err := r.renderOffscreen(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(moved, placed) {
		t.Error("translated boxes draw other pixels than boxes in their places")
	}
	gputest.Compare(t, "metal-translated-boxes", moved, s.Width*4, s)
}

// TestBorderedCardDrawsAsTheCPURenderer draws the card of #531, a large
// opaque card with a border of 2 pixels and continuous corners at scale 2,
// over an opaque backdrop, and a small button like it, with each way the
// builder can split them (an opaque core and pieces around it, nine-patches
// of continuous corners, rings, specialized shaders) and without, and
// requires the CPU renderer's pixels along their straight edges, exactly:
// Metal drew a second line of border color at about 65% inside the card's
// top edge, in the rows under its border, which Compare's tolerance (as
// in the other parity tests) lets through.
func TestBorderedCardDrawsAsTheCPURenderer(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		unavailable(t, err)
	}
	defer r.Release()
	reset := func() {
		r.b.Opaque, r.b.NoCornerSplit, r.b.NoRingSplit, r.b.NoCull = true, false, false, false
		r.b.NoClasses, r.b.GeneralFill, r.b.NoClipSkip, r.b.NoJoin = false, false, false, false
	}
	defer reset()
	wide := r.SetWide(true)
	if !wide {
		t.Log("no wide target:", r.wideErr)
	}
	ways := []struct {
		name string
		set  func()
	}{
		{"", func() {}},
		{"-blended", func() { r.b.Opaque = false }},
		{"-whole-corners", func() { r.b.NoCornerSplit = true }},
		{"-blended-whole-corners", func() { r.b.Opaque, r.b.NoCornerSplit = false, true }},
		{"-no-cull", func() { r.b.NoCull, r.b.NoRingSplit = true, true }},
		{"-general", func() { r.b.NoClasses, r.b.GeneralFill, r.b.NoClipSkip = true, true, true }},
	}
	for _, continuous := range []bool{true, false} {
		s := gputest.BorderedCardScene(continuous)
		name := "metal-bordered-card"
		if !continuous {
			name += "-round"
		}
		for _, w := range ways {
			reset()
			w.set()
			// Twice: the second frame draws with what the first made.
			for range 2 {
				pix, err := r.renderOffscreen(s)
				if err != nil {
					t.Fatal(err)
				}
				gputest.CompareStraightEdges(t, name+w.name, pix, s.Width*4, s)
			}
			if !wide {
				continue
			}
			// In float16, as windows showing wide colors draw.
			raw := renderRaw(t, r, s, true)
			pix := make([]byte, s.Width*s.Height*4)
			for i := range s.Width * s.Height {
				for c, from := range [4]int{2, 1, 0, 3} { // RGBA to BGRA
					v := half(uint16(raw[8*i+2*from]) | uint16(raw[8*i+2*from+1])<<8)
					pix[4*i+c] = byte(math.Round(float64(min(max(v, 0), 1) * 255)))
				}
			}
			gputest.CompareStraightEdges(t, name+w.name+"-float16", pix, s.Width*4, s)
		}
	}
}
