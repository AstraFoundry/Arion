# Transforms

<!-- preview transforms 200 -->

`Translate`, `Scale`, and `Rotate` change an element and everything inside
it visually, while layout reserves its original space. They work on boxes,
text, images, controls, and custom drawings. The pointer follows what is
drawn, including nested transforms and clips:

```go
ui.Column(c).Size(180, 110).Padding(12).Gap(8).
	Scale(1.1, 0.9).Rotate(12).Translate(40, 0).
	Children(func() {
		ui.Text(c, "Transformed controls")
		ui.TextInput(c, &app.name)
	})
```

Transforms in a method chain act in call order: this example scales, then
rotates, then translates. A parent's transform acts after its children's.
Each scale and rotation uses the center of the element's border box by
default. `TransformOrigin(0, 0)` uses its top-left; `(1, 1)` its bottom-right.
Other fractions, including values outside the box, are allowed. Negative
scales reflect content; a zero scale makes the element invisible and gives
it no pointer hit region. Transform values must be finite.

For reusable compositions, build a `Transform` and set it with `Transform`:

```go
pose := ui.Scaling(1.2, 1.2).Then(ui.Rotation(15)).Then(ui.Translation(30, 0))
ui.Box(c).Transform(pose).TransformOrigin(0, 0)
```

The zero `Transform` is identity. Setting `Transform` replaces the element's
previous transform, `TranslatePercent`'s too; the convenience methods
append to it.

`TranslatePercent(x, y)` moves an element by percents of its own width and
height, as CSS's `translate(-50%, -50%)`: of its border box as the frame
lays it out, so that it is in place from its first frame, without waiting
for its `Bounds`, and follows its size in the frame it changes. It acts
after the element's other transforms, wherever it is called among them,
as CSS's `translate` property does, and before its pose; calls add up. A
callout centered above the point it points at, whatever its size:

```go
ui.Box(c).Absolute().Left(at.X).Top(at.Y - 8).TranslatePercent(-50, -100).Children(callout)
```

A transition's `Motion` and a `Pose` move by percents of the element's
size too (`XPercent`, `YPercent`). Set transforms
in the view, before the frame is laid out. `Rotate` on an icon keeps its
existing behavior of rotating the SVG artwork around its center. To rotate
an icon's whole border box, use `Transform(ui.Rotation(degrees))`.

## Skew and 3D

`Skew(x, y)` slants an element by degrees along each axis, as CSS's
`skew()`. `RotateX` and `RotateY` turn it in 3D around the horizontal and
vertical axes through its origin, and `Perspective(distance)` after them
sees it from that many DIPs in front: the shorter, the deeper; 800 or so
looks natural. Without `Perspective`, a turned element only looks narrower
or shorter:

```go
// A card turning over: its back, turned half a turn more, shows past
// 90 degrees, and each face hides while it is turned away.
card := ui.Box(c).Size(160, 100)
turn := float32(0)
if app.flipped {
	turn = 180
}
turn = card.AnimateTo("turn", turn, ui.Snappy)
card.Children(func() {
	front := ui.Box(c).Background(t.Surface)
	back := ui.Box(c).Background(t.Accent)
	front.Absolute().Fill().RotateY(turn).Perspective(800).BackfaceHidden()
	back.Absolute().Fill().RotateY(turn + 180).Perspective(800).BackfaceHidden()
})
```

`TranslateZ(z)` moves an element along Z, toward the viewer for positive
values and away for negative ones, as CSS's `translateZ()`. Seen in
`Perspective(distance)` after it, it looks `distance/(distance-z)` its
size, about its origin: 120 DIPs back seen from 800, `800/920` its size.
Without `Perspective`, a move along Z shows nothing, as in CSS; one as far
as the viewer, or past it, puts the element behind the viewer, where it
shows nothing and takes no pointer input. Before a turn, it pushes an
element out from the axis, as the items of a carousel; after it, it
lifts the turned element toward the viewer:

```go
// A carousel: each item out on a ring of 300 DIPs, turned by spin, the
// ring's center 300 behind the plane so that the item in front shows at
// its size. Those farther round look smaller, and each hides while the
// viewer sees its back.
for i, item := range app.items {
	angle := spin + float32(i)*360/float32(len(app.items))
	ring := ui.TranslationZ(300).Then(ui.RotationY(angle)).Then(ui.TranslationZ(-300)).Then(ui.Perspective(800))
	ui.Box(c).Absolute().Left(200).Top(40).Size(160, 100).Background(item.color).
		Transform(ring).BackfaceHidden()
}

// A stack in depth: each card 60 DIPs behind the one in front of it,
// the farthest drawn first.
for i := 2; i >= 0; i-- {
	ui.Box(c).Absolute().Fill().Radius(12).Background(t.Surface).Border(1, t.Border).
		TranslateZ(-60 * float32(i)).Perspective(800)
}
```

Elements moved along Z still draw in their order, not by depth, as each
flattens into its parent's plane: draw the nearest last.

`Transform` values compose the same way, with `Skewing`, `TranslationZ`,
`Translation3D(x, y, z)` (CSS's `translate3d()`), `RotationX`,
`RotationY` and `Perspective`. Each element flattens what its 3D transform
makes of it into its parent's plane, as CSS's default `transform-style:
flat`: a parent turning in 3D turns its children's drawing with it.
`BackfaceHidden` hides an element, and takes it from the pointer, while
its own 3D transform turns its back to the viewer: from where the viewer
sees it in perspective, as browsers have it, so that an item pushed out
along Z turns its back as it passes the viewer's line of sight.

The pointer, clips, text input and accessibility follow 3D transforms as
they follow 2D ones, and the renderers draw them in perspective, smooth
edges included.

## Coordinates, clips, and overlays

`Bounds()` is the previous frame's transformed bounding rectangle in
window DIPs. `LayoutBounds()` is its box before transforms. `LocalToWindow`
and `WindowToLocal` convert points relative to that box; `WindowToLocal`
reports false when the transform has no inverse. `PointerPosition`,
`Dragged`, and `HandleInput` report local positions and vectors. Captured
pointer input stays local even outside the transformed box.

`Draw` and `DrawOver` receive the original layout coordinates, as before:
the painter applies the transform to everything they draw, including paths,
text, images, shadows, and effects. Clips retain the coordinate system in
which they were pushed and compose through nested elements. IME candidate
windows and accessibility receive the transformed bounding rectangles of
carets and elements. Popovers stay upright in the overlay and attach to
the transformed bounds of their anchors, including during layout transitions.

## Animating transforms

Transforms take animated values as well as fixed ones, Z's included: a spring that keeps
its velocity, a curve, keyframes, or a value following the pointer (see
[Motion](motion.md)):

```go
tile := ui.Box(c).Key("tile").Size(80, 50)
target := float32(0)
if app.expanded {
	target = 120
}
tile.Translate(tile.AnimateTo("x", target, ui.Snappy), 0)
```

[Transitions](transitions.md) scale and rotate elements coming and going,
and move an element appearing from where another showed (`Match`), with
transforms too.

## Rendering

Metal, Direct3D 11, and OpenGL apply general transforms to their instanced
geometry on the GPU. Fills, borders, shadows, images, text, and effects keep
local coordinates; backdrop reads map back into window coordinates. Metal
preserves wide-gamut colors under transforms.

Nested rounded or transformed clips use retained coverage masks once
content inside them is scaled, rotated, skewed or in perspective. Content
that a transform only moves, such as a label translated by a fraction of a
pixel, needs no mask: its clips are cut as for content that isn't
transformed. A clip that reaches into the rounded corners of a clip around
it, such as a clipped column filling the bottom of a rounded card, uses the
rounded clip's mask too, whatever its content. Moving content under a
stationary clip reuses that mask without CPU pixel work.
Hard clips sharing their content's coordinates, including transformed text
inputs, are evaluated directly in the fragment shader and reuse their
ancestor's mask. Changing a complex clip recomputes its coverage; resizing
it replaces its texture in place rather than accumulating cached images.

The CPU compositor remains the renderer for machines without a GPU and for
small resting updates. Active transformed motion uses the GPU where one is
available. Text bitmaps are sampled bilinearly when transformed, and
transformed text uses grayscale coverage. GPU antialiasing follows the CPU
renderer in device-pixel coordinates. To zoom a whole window, sharp, use
the content's zoom (`Context.SetZoom`, see
[Windows](windows.md#zoom)) rather than a transform.

The gallery's **Motion** page shows nested transformed editable controls,
an anchored popover, springs retargeted and thrown, keyframe sequences, and
shared elements.
