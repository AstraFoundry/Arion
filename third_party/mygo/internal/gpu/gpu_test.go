package gpu

import (
	"bytes"
	"testing"

	"github.com/egoist/mygo/internal/scene"
)

func TestAffineInstancesPreserveGeometryAndWideColors(t *testing.T) {
	m := scene.Translation(30, 10).Mul(scene.Rotation(20))
	s := &scene.Scene{Width: 80, Height: 60, Clear: scene.Color{B: 120, A: 80}, Ops: []scene.Op{{Kind: scene.OpFill, Rect: scene.Rect{W: 25, H: 15}, Color: scene.Color{R: 255, A: 150}, Transform: m, Wide: 1}}, Wide: []scene.WideColors{{Set: scene.WideColor, Color: [4]float32{1.2, -0.1, 0.2, 0.8}}}}
	var b Builder
	b.Wide = true
	defer b.Release()
	uploads := 0
	shown := b.Build(s, func(*scene.Image) uintptr { uploads++; return 1 })
	if shown != s || uploads != 0 || len(b.Instances) != 1 {
		t.Fatal("transformed fill was rasterized or uploaded as an image")
	}
	in := b.Instances[0]
	if in.Params[0] != 0 || in.Rect != [4]float32{0, 0, 25, 15} || in.Color != s.Wide[0].Color {
		t.Fatal("local geometry or wide color was lost")
	}
	if in.Transform0 != [4]float32{m.A, m.C, m.X, 1} || in.Transform1 != [4]float32{m.B, m.D, m.Y, 0} {
		t.Fatal("vertex transform does not match scene geometry")
	}
	s.Ops[0].Transform = scene.Scaling(0, 1)
	b.Build(s, func(*scene.Image) uintptr { return 1 })
	if len(b.Instances) != 0 {
		t.Fatal("singular transform emitted GPU geometry")
	}
}

func TestAffineClipMaskCacheFollowsClipInsteadOfContent(t *testing.T) {
	s := &scene.Scene{Width: 100, Height: 80, Ops: []scene.Op{
		{Kind: scene.OpPushClip, Rect: scene.Rect{X: 10, Y: 10, W: 70, H: 50}, Radii: [4]float32{8, 8, 8, 8}},
		{Kind: scene.OpFill, Rect: scene.Rect{W: 25, H: 15}, Color: scene.Color{R: 255, A: 255}, Transform: scene.Translation(20, 15)},
		{Kind: scene.OpPopClip},
	}}
	var b Builder
	defer b.Release()
	var mask *scene.Image
	build := func() { b.Build(s, func(m *scene.Image) uintptr { mask = m; return 1 }) }
	build()
	if mask == nil || b.Batches[0].ClipMask != 1 || b.Instances[0].Transform1[3] != 1 {
		t.Fatal("nested clip was not bound as a mask")
	}
	id, version := mask.ID(), mask.Version()
	pixels := append([]byte(nil), mask.Pix...)
	s.Ops[1].Transform = scene.Translation(35, 25)
	build()
	if mask.ID() != id || mask.Version() != version || !bytes.Equal(mask.Pix, pixels) {
		t.Fatal("moving content rerasterized its stationary clip")
	}
	s.Ops[0].Radii = [4]float32{12, 12, 12, 12}
	build()
	if mask.ID() != id || mask.Version() == version {
		t.Fatal("changing clip did not update the retained mask")
	}
	s.Ops[0].Rect.W = 60
	build()
	if mask.ID() != id || mask.W != 60 {
		t.Fatal("resizing clip accumulated a new cached image")
	}
	clipped := s.Ops
	s.Ops = s.Ops[1:2]
	build()
	if len(b.maskSlots) != 1 {
		t.Fatal("a removed clip did not keep its mask")
	}
	version = mask.Version()
	s.Ops = clipped
	build()
	if mask.ID() != id || mask.Version() != version {
		t.Fatal("a clip coming back rasterized its mask again")
	}
}

func TestTransformComingAndGoingKeepsMasks(t *testing.T) {
	// A badge toggling between scaled and not, as in #255, beside a
	// rounded card holding a transformed clip of its own while it is.
	card := scene.Op{Kind: scene.OpPushClip, Rect: scene.Rect{X: 10, Y: 10, W: 200, H: 150}, Radii: [4]float32{12, 12, 12, 12}}
	inner := scene.Op{Kind: scene.OpPushClip, Rect: scene.Rect{W: 80, H: 40}, Transform: scene.Translation(30, 30).Mul(scene.Rotation(5))}
	fill := scene.Op{Kind: scene.OpFill, Rect: scene.Rect{W: 80, H: 40}, Color: scene.Color{A: 255}, Transform: inner.Transform}
	other := scene.Op{Kind: scene.OpPushClip, Rect: scene.Rect{X: 220, Y: 10, W: 60, H: 60}, Radii: [4]float32{6, 6, 6, 6}}
	pop := scene.Op{Kind: scene.OpPopClip}
	shown := &scene.Scene{Width: 300, Height: 200, Ops: []scene.Op{card, inner, fill, pop, pop}}
	elsewhere := &scene.Scene{Width: 300, Height: 200, Ops: []scene.Op{other, fill, pop}}
	hidden := &scene.Scene{Width: 300, Height: 200, Ops: []scene.Op{card, pop}}
	var b Builder
	defer b.Release()
	versions := map[uint64]uint64{}
	rasterized := 0
	build := func(s *scene.Scene) {
		b.Build(s, func(m *scene.Image) uintptr {
			if v, ok := versions[m.ID()]; !ok || v != m.Version() {
				rasterized++
				versions[m.ID()] = m.Version()
			}
			return uintptr(m.ID())
		})
	}
	build(shown)
	first := rasterized
	for range 5 {
		build(hidden)
		build(elsewhere)
		build(shown)
	}
	if first == 0 || rasterized != first+1 {
		t.Fatalf("rasterized %d masks after the first frame's %d, want only the other clip's", rasterized-first, first)
	}
}

func TestIdleMasksStayWithinBudget(t *testing.T) {
	// Two masks of 24 MiB each in a frame, then none: one idle mask fits
	// the budget, two don't.
	clip := func(x float32) []scene.Op {
		return []scene.Op{
			{Kind: scene.OpPushClip, Rect: scene.Rect{X: x, W: 3000, H: 2100}, Transform: scene.Rotation(0.01)},
			{Kind: scene.OpFill, Rect: scene.Rect{X: x, W: 10, H: 10}, Color: scene.Color{A: 255}},
			{Kind: scene.OpPopClip},
		}
	}
	var b Builder
	defer b.Release()
	build := func(s *scene.Scene) { b.Build(s, func(*scene.Image) uintptr { return 1 }) }
	build(&scene.Scene{Width: 6200, Height: 2100, Ops: append(clip(0), clip(3100)...)})
	if len(b.maskSlots) != 2 {
		t.Fatalf("kept %d masks, want 2", len(b.maskSlots))
	}
	build(&scene.Scene{Width: 10, Height: 10})
	if len(b.maskSlots) != 1 {
		t.Fatalf("kept %d idle masks, want 1 within the budget", len(b.maskSlots))
	}
}

func TestTransformOutsideAClipLeavesItUnmasked(t *testing.T) {
	// A scaled badge beside a rounded card of clipped rows: nothing in the
	// card is transformed, so its clips stay in the shader.
	s := &scene.Scene{Width: 400, Height: 300, Ops: []scene.Op{
		{Kind: scene.OpFill, Rect: scene.Rect{W: 40, H: 40}, Color: scene.Color{B: 255, A: 255}, Transform: scene.Scaling(1.2, 1.2)},
		{Kind: scene.OpPushClip, Rect: scene.Rect{Y: 50, W: 400, H: 250}, Radii: [4]float32{11, 11, 11, 11}},
		{Kind: scene.OpPushClip, Rect: scene.Rect{Y: 50, W: 200, H: 32}},
		{Kind: scene.OpFill, Rect: scene.Rect{Y: 50, W: 300, H: 32}, Color: scene.Color{A: 255}},
		{Kind: scene.OpPopClip},
		{Kind: scene.OpPopClip},
	}}
	var b Builder
	defer b.Release()
	masks := 0
	b.Build(s, func(*scene.Image) uintptr { masks++; return 1 })
	if masks != 0 || len(b.maskSlots) != 0 {
		t.Fatalf("rasterized %d clip masks for clips with no transform inside", masks)
	}
	if in := b.Instances[1]; in.Transform1[3] != 0 || in.Clip != [4]float32{0, 50, 200, 32} {
		t.Fatalf("row clip %v, mask mode %v", in.Clip, in.Transform1[3])
	}
}

func TestRoundedClipMasksOnlyTheBranchWithATransform(t *testing.T) {
	// A zoomed image beside a list in the same rounded card: the image's
	// clip keeps the card's corners, the list's rows stay in the shader.
	s := &scene.Scene{Width: 400, Height: 300, Ops: []scene.Op{
		{Kind: scene.OpPushClip, Rect: scene.Rect{W: 400, H: 300}, Radii: [4]float32{11, 11, 11, 11}},
		{Kind: scene.OpPushClip, Rect: scene.Rect{W: 200, H: 300}},
		{Kind: scene.OpFill, Rect: scene.Rect{W: 200, H: 300}, Color: scene.Color{A: 255}, Transform: scene.Scaling(1.5, 1.5)},
		{Kind: scene.OpPopClip},
		{Kind: scene.OpPushClip, Rect: scene.Rect{X: 200, W: 200, H: 32}},
		{Kind: scene.OpFill, Rect: scene.Rect{X: 200, W: 200, H: 32}, Color: scene.Color{A: 255}},
		{Kind: scene.OpPopClip},
		{Kind: scene.OpPopClip},
	}}
	var b Builder
	defer b.Release()
	b.Build(s, func(*scene.Image) uintptr { return 1 })
	if b.Instances[0].Transform1[3] != 1 {
		t.Fatal("zoomed content lost its rounded ancestor's mask")
	}
	if b.Instances[1].Transform1[3] != 0 || b.Batches[len(b.Batches)-1].ClipMask != 0 {
		t.Fatal("untransformed row was masked")
	}
}
