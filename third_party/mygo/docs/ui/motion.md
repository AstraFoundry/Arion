# Motion

<!-- preview motion 220 -->

An element animates values for you: you give the target, it returns the
value to show in this frame, and draws frames until the value gets there.
The view says where things are; the animation finds the way:

```go
card := ui.Box(c).Size(120, 80).Radius(12).Background(t.Accent)
target := float32(0)
if app.open {
	target = 200
}
card.Translate(card.AnimateTo("x", target, ui.Snappy), 0)
```

The key (`"x"`) tells apart the values of one element, which needs the same
ID from frame to frame: a `Key`, or the same place among its siblings. A
value starts at its first target. Use the value in any way: a transform
(see [Transforms](transforms.md)), a size, an opacity, a color mix, or what
a `Draw` paints.

## Animations

An `Animation` says how a value goes to a new target:

- **Springs**, the zero `Animation` and the presets `ui.Smooth` (no
  bounce), `ui.Snappy` (a little) and `ui.Bouncy` (a visible one). A
  spring keeps its velocity when its target changes on the way: clicked
  again mid-flight, the card above turns around smoothly instead of
  stopping dead.
- **Curves**: a `Duration` and an `Ease` (`ui.EaseOut` when nil), as
  `ui.Animation{Duration: 150 * time.Millisecond, Ease: ui.EaseInOut}`.
  `Animate(key, target, d)` and `AnimateWith(key, target, d, ease)` are
  short for them.

A spring's `SpringOptions` describe it as one perceives it, with a
`Duration` (500 ms when zero), about how long it takes to arrive, and a
`Bounce`, from 0, arriving without overshooting, toward 1; negative values
arrive more slowly still. These are SwiftUI's and Motion's parameters.
`Stiffness`, `Damping` and `Mass` describe it physically instead:

```go
gentle := ui.Animation{Spring: ui.SpringOptions{Duration: 700 * time.Millisecond, Bounce: 0.1}}
stiff := ui.Animation{Spring: ui.SpringOptions{Stiffness: 400, Damping: 30}}
x := card.AnimateTo("x", target, gentle)
```

`Spring(key, target, options)` is short for `AnimateTo` with a spring.
Springs are solved exactly rather than stepped, so a slow frame does not
change their path. A value can change its kind of animation on the way: a
spring taking over a curve keeps the curve's velocity.

`Easing` functions are `ui.Linear`, `ui.EaseIn`, `ui.EaseOut`,
`ui.EaseInOut`, `ui.Bounce(e)`, CSS's `ui.CubicBezier(x1, y1, x2, y2)`, or
any `func(t float32) float32`.

`AnimateColor(key, target, a)` animates a color the same way, mixed with
premultiplied alpha, so that a color fading in from transparent keeps its
hue:

```go
row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
```

## Poses that only paint

The values above are read by the view, so every frame that moves runs it.
`AnimatePose` moves an element instead: it translates, scales, turns and
fades it, after its own transform, and the frames that move it only paint,
without running the view, as Core Animation and the Web Animations API run
transforms and opacity. It is the cheapest motion there is, which keeps a
large view smooth:

```go
lift := ui.Pose{}
if card.Hovered() {
	lift = ui.Pose{Y: -4, Scale: 1.02}
}
card.AnimatePose(lift, ui.Snappy)
```

A `Pose` moves by `X` and `Y`, scales by `Scale` (0 keeps the size),
turns by `Rotate` degrees, and fades by `Fade`, from 0, opaque, to 1, gone.
The pointer, carets and overlays follow the element as it moves, and
`Transitioning()` reports whether its pose was moving.

Poses nest: each acts after those of the elements around it, so an
element inside another adds its own motion, along its own curve. In the
[confetti example](../../examples/confetti), a burst falls and fades, each
piece in it bursts out along a curve that slows down, and spins inside:
hundreds of pieces fly without building the view again. Each element
posed still costs every frame that paints it, so share a pose among the
elements that move alike, as the burst's fall, rather than giving each
its own.

## Delays and staggering

`Delay` keeps the value going as it was for a while after its target
changes. Delays growing with the index stagger a list:

```go
for i, it := range app.items {
	row := ui.Row(c.Key(it.ID))
	a := ui.Snappy
	a.Delay = time.Duration(i) * 30 * time.Millisecond
	row.Opacity(row.AnimateTo("shown", 1, a))
}
```

## Gestures

`Draggable` lets the pointer move an element and throw it, as motion.dev's
`drag` does: the element follows the pointer, and once let go, the
pointer's velocity carries it to where it comes to rest, with a spring:

```go
card := ui.Box(c).Size(90, 60).Radius(10).Background(t.Accent)
card.Draggable(&app.card, ui.DraggableOptions{
	Axis:   ui.Horizontal,
	Bounds: &ui.Rect{W: 200},
	Snap:   ui.Nearest(ui.Point{}, ui.Point{X: 200}),
})
```

`app.card` is a `ui.Point`, the element's offset from its place in the
layout. As the pointer drags the element, it follows the pointer; once the
pointer lets go, it is where the element comes to rest; and setting it
moves the element there, as a button turning a page does. A nil offset is
kept in the element. The options say how it moves:

- **`Axis`**, `ui.Horizontal` or `ui.Vertical`, is the direction it moves
  in; both when zero.
- **`Bounds`** hold the offset within a rectangle at rest, from `X` to
  `X+W` and from `Y` to `Y+H`. The pointer pulls the element past them less
  and less far, as iOS holds content pulled past its edges, and it springs
  back within them once let go. `Rigid` stops it at them.
- **`Snap`** chooses where it comes to rest, from where the throw would
  carry it (`ui.Project`, held within the bounds), so that a flick goes
  on to the next page or height: `ui.Nearest(points...)` chooses the
  nearest of points. A `Snap` returning its point lets the
  element glide where the throw carries it. Without `Snap`, it rests where
  the pointer lets it go, as an item placed on a canvas.
- **`Animation`** is how it goes to rest, and to an offset the app sets:
  `ui.Smooth` when zero. A spring starts at the pointer's velocity.

The pointer drags the element once it moved a few DIPs, and that press is
no click; a press on the element as it moves catches it. The element moves
after its own transform and its pose, along its parent's axes, so it
stays under the pointer as it scales or turns, and it goes to rest in
frames that only paint, without building the view, as `AnimatePose`
moves it. `Dragging()` reports that the pointer drags it, to lift it as
it does:

```go
lift := ui.Pose{}
if card.Dragging() {
	lift = ui.Pose{Scale: 1.05}
}
card.AnimatePose(lift, ui.Snappy)
```

`Transitioning()` is false once it is at rest, for the view to go on, as
removing a row swiped away:

```go
for i := range app.rows {
	r := &app.rows[i]
	row := ui.Row(c).Key(r.ID)
	row.Draggable(&r.swipe, ui.DraggableOptions{
		Axis: ui.Horizontal,
		Snap: func(p ui.Point) ui.Point {
			if p.X < -160 {
				return ui.Point{X: -400} // away to the left
			}
			return ui.Point{}
		},
	})
	if r.swipe.X == -400 && !row.Transitioning() {
		app.remove(r.ID)
	}
}
```

`DragOffset()` returns where the element shows in this frame, following
the pointer or on its way to rest, for what follows it, as a backdrop
fading as a sheet goes down. Reading it builds the view in every frame
the element moves in, as `AnimateTo` does; call it after `Draggable`:

```go
backdrop := ui.Box(c).Absolute().Fill().Background(ui.RGBA(0, 0, 0, 0.4))
sheet := ui.Column(c).Draggable(&app.sheet, ui.DraggableOptions{
	Axis:   ui.Vertical,
	Bounds: &ui.Rect{H: 400},
	Snap:   ui.Nearest(ui.Point{}, ui.Point{Y: 400}),
})
backdrop.Opacity(1 - sheet.DragOffset().Y/400)
```

While the desktop asks for less motion, the element follows the pointer,
and goes to rest at once.

### By hand

For a value other than an offset that follows the pointer, as the angle
of a dial, build the gesture from smaller pieces. `Jump` sets a value
at once, following the pointer, and keeps track of how fast it moves; the
spring that takes over when the pointer lets go starts at that speed.
`DragEnded` reports the pointer's velocity as it let go, which
`SpringOptions.Velocity` hands to the spring:

```go
dial := ui.Box(c).Size(80, 80)
var turn float32
if dx, _, ok := dial.Dragged(); ok {
	app.turn += dx
	turn = dial.Jump("turn", app.turn)
} else {
	a := ui.Snappy
	if vx, _, ok := dial.DragEnded(); ok {
		// Where the throw would carry it decides the notch it lands on.
		app.turn = float32(math.Round(float64(ui.Project(app.turn, vx)/45))) * 45
		a.Spring.Velocity = vx
	}
	turn = dial.AnimateTo("turn", app.turn, a)
}
dial.Transform(ui.Rotation(turn))
```

`DragVelocity` is the pointer's velocity while it presses, in DIPs per
second along the element's axes, zero once it rests a moment. `Velocity(key)`
is the velocity of an animated value. A click, which does not move, ends no
drag.

`ui.Project(value, velocity)` is where a fling would carry a value as
iOS's scroll views slow down, to choose where it lands, as a page of a
pager. `ui.Rubberband(value, lo, hi, dimension)` holds a value dragged past
lo or hi as iOS holds content pulled past its edges, going less and less
far:

```go
if _, dy, ok := sheet.Dragged(); ok {
	app.y += dy
	y = sheet.Jump("y", ui.Rubberband(app.y, 0, 400, 600))
}
```

## When it is done

`Settled(key)` reports whether a value rests at its target, and
`Arrived(key)` is true in the one frame in which it came to rest, for the
view to go on, as removing what faded out:

```go
toast.Opacity(toast.AnimateTo("fade", 0, ui.Animation{Duration: 200 * time.Millisecond}))
if toast.Arrived("fade") {
	app.dismiss(toast)
}
```

## Without animation

`c.DisableAnimations()` makes the values, transitions, poses and router
pages of the frame go to their targets at once, as SwiftUI's
disablesAnimations: for a change that should not show as motion, as
restoring a window's state or jumping to a search result. Call it before
building what changes.

## Keyframes

`Keyframes` plays timed values, each segment along the `Ease` of the frame
it arrives at, and holds the last:

```go
tile := ui.Box(c).Key("tile").Size(80, 50)
x := tile.Keyframes("x", []ui.Keyframe{
	{Value: 0},
	{At: 200 * time.Millisecond, Value: 100, Ease: ui.EaseOut},
	{At: 700 * time.Millisecond, Value: 40},
	{At: time.Second, Value: 0, Ease: ui.EaseInOut},
}, ui.KeyframeOptions{Run: app.shakes})
tile.Translate(x, 0)
```

The first time must be zero and later times must increase. Change `Run` to
play the sequence again, or a changed one: interrupted, it starts from the
value shown rather than jumping to its first, the first time through.
`Repeat` is how many more times it plays after the first, `RepeatForever`
for as long as it is built, with `Alternate` playing every other time
backward; `Delay` holds the first value for a while before it plays, and
`RepeatDelay` the value it got to between plays.

## Scrolling

`InView()` reports whether some of an element showed in the frame before,
within the scroll containers around it, to bring it in as it scrolls into
view. A `ScrollState` (see [Scrolling](scroll.md)) gives how far its
container scrolled, for effects that follow the scrolling:

```go
card := ui.Box(c).Key(it.ID).Height(120)
shown := float32(0)
if card.InView() {
	shown = 1
}
card.Opacity(card.AnimateTo("shown", shown, ui.Smooth))

header.Opacity(1 - min(app.scroll.Y/80, 1)) // fades as the page scrolls
```

`ScrollState.ScrollTo(x, y, animation)` scrolls a container smoothly (see
[Scrolling](scroll.md)).

## Content that changes

`ContentTransition` animates an element's content as it changes: the text
of a `Text`, the icon of an `Icon`, the picture of an `Image`. By default
the old content fades out as the new fades in, and an icon shrinks away as
the new one grows in. `Numeric` rolls the characters of a line of text that
change, as a counter's digits, those that stay staying: up, or with `Down`
down, as SwiftUI's numericText:

```go
ui.Text(c, strconv.Itoa(app.count)).FontSize(32).Bold().
	ContentTransition(ui.ContentTransition{Numeric: true, Down: app.decreased})
ui.Icon(c, playOrPause).ContentTransition(ui.ContentTransition{})
```

Content transitions run in frames that only paint. While the desktop asks
for less motion, content fades without moving.

## Loops and your own motion

`Loop` returns the progress of an animation that starts over every period,
for spinners and pulses (see [Drawing and animation](drawing.md#animation)).
To animate otherwise, compute from `c.Now()` and call `c.AnimationFrame()`
in every frame that moves.

## Less motion and frames

When the desktop asks for less motion (`c.Preferences().ReduceMotion`),
values and poses go to their targets at once and sequences end at their
last value. What changes without moving still changes smoothly: colors
(`AnimateColor`), a pose's `Fade`, content fading across, and
[transitions](transitions.md)' colors and fades. `Loop` goes on, as the
system's spinners do.

Animations draw frames only while something moves, at the display's rate,
and none while other windows cover the window. Frames moving values the
view reads run the view; poses, content transitions, and drawings that
animate from their `Draw` only paint.
