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

Colors mix in sRGB, unless the `Animation` has `Oklab`: then in Oklab, as a
gradient's `Oklab` mixes, whose way between two saturated colors keeps
their lightness and saturation rather than going through a dull, darker
middle (yellow to blue through blues, not gray). It mixes the colors of
a [transition](transitions.md#what-moves) with it too, and those of a
`ContentTransition`'s morph, and `Color.MixOklab` mixes two colors so:

```go
badge.Background(badge.AnimateColor("bg", level, ui.Animation{Spring: ui.SpringOptions{Duration: 300 * time.Millisecond}, Oklab: true}))
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

A `Pose` moves by `X` and `Y`, and by `XPercent` and `YPercent` percent of
the element's own width and height (as [`TranslatePercent`](transforms.md)
moves it, following its size as it changes), scales by `Scale` (0 keeps
the size), turns by `Rotate` degrees, and fades by `Fade`, from 0, opaque,
to 1, gone.
The pointer, carets and overlays follow the element as it moves, and
`Transitioning()` reports whether its pose was moving.

A pose starts where the element is first posed: an element that the
frame before did not pose, as one built for the first time, shows at its
pose at once, and moves only when a later frame poses it elsewhere. To
come in from a pose, as a flyout's panel rising and fading in as it
shows, pose the element there in one build, ask for another frame with
`c.Invalidate()`, and pose it at its place in the next:

```go
panel := ui.Column(c)
entered := ui.Local(panel, "entered", func() bool { return false })
p := ui.Pose{}
if !*entered {
	p, *entered = ui.Pose{Y: 8, Fade: 1}, true
	c.Invalidate()
}
panel.AnimatePose(p, ui.Snappy)
```

Without the second build, the element stays where its first pose put
it. An element appearing in a parent that the frame before showed comes
in with the `Enter` of its [transition](transitions.md#coming-and-going)
too, as a row added to a list.

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

The last moves of a drag come while `Dragged` still reports the press, and
`DragEnded` in the build after: the dial turns as far as the pointer went
before it springs to a notch.

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
building what changes. `Keyframes` and timelines play on their own rather
than towards a target, and go on as they were; seek a timeline to jump it.
`c.Instant()` reports whether the frame does, for a view that moves a
value of its own, so that it settles in the same frames.

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

## Timelines

A `Timeline` plays several tweens along one playhead, so motion that belongs
together is arranged, controlled and tested together: a card arriving while
its title fades in, rows following one another, a small bounce after the
card lands. `AnimateTo` moves one value of one element towards a target, and
`Keyframes` plays one value along timed stops; a timeline coordinates many
such values, each tween of its own, around one clock.

An element keeps a timeline under a key, as it keeps `Keyframes`: build it
in the view, as everything else, and give each tween the value it shows
this frame:

```go
card := ui.Column(c)
tl := card.Timeline("intro", ui.TimelineOptions{Run: app.opened})
x := tl.Tween(ui.Tween{From: -200, To: 0, Duration: 400 * time.Millisecond})
fade := tl.Tween(ui.Tween{From: 0, To: 1, Duration: 300 * time.Millisecond},
	ui.AfterPrevious(-150*time.Millisecond))
card.Translate(x, 0).Opacity(fade)
```

A tween eases out (`EaseOut`) unless its `Ease` says otherwise, as an
`Animation` does. A tween with no placement follows the one before it at
once, so here the second would start at 400ms; placements put it
elsewhere, as GSAP's position parameter does:

| Placement | Starts the tween |
| --- | --- |
| `AfterPrevious(50*time.Millisecond)` | 50ms after the one before it ends; a negative offset overlaps it, as `-150ms` above starts the fade at 250ms |
| `WithPrevious(0)` | where the one before it starts, to move along with it, or an offset after that |
| `AtLabel("rows", 100*time.Millisecond)` | 100ms after a label |
| `AtTime(500*time.Millisecond)` | at a time of the timeline |

Nothing starts before the timeline does: a placement before its start is
at its start.

### Labels and stagger

`Label` marks a time of the timeline for tweens to start at, and an offset
from it staggers them by index:

```go
tl := list.Timeline("rows", ui.TimelineOptions{Run: app.shown})
tl.Label("rows", ui.AtTime(300*time.Millisecond))
for i := range app.rows {
	app.rows[i] = tl.Tween(ui.Tween{From: 0, To: 1, Duration: 200 * time.Millisecond},
		ui.AtLabel("rows", time.Duration(i)*40*time.Millisecond))
}
for i, shown := range app.rows {
	rows[i].Opacity(shown).Translate(0, (1-shown)*12)
}
```

### Playing it

`Duration` is the end of the last tween, `Time` where the playhead is, and
`Progress` how far along it is, from 0 to 1, for a scrubber. `Paused` holds
the playhead where it is: time a paused timeline spends is not time it
played, so letting a pause go plays on from where it held, however long the
pause was. `Reverse` plays it backward from where the playhead is to its
start, as a panel that closes the way it opened (`Reverse: !app.open`), and
a timeline played over while reversed plays from its end. `Seek` and
`Progress` hold it at a time or a part of the timeline for as long as they
are given, going on from there when they are not; give one of them at
most. What the options say takes
effect in the frame it is said in, so a button that pauses or plays the
timeline again shows in the frame it was clicked in:

```go
o := ui.TimelineOptions{Run: app.run, Paused: app.paused, Reverse: app.reverse}
if app.scrubbing {
	o.Progress = &app.scrub
}
tl := card.Timeline("intro", o)
```

`Repeat` is how many more times the timeline plays after the first, as
`KeyframeOptions.Repeat` is, `RepeatForever` for as long as the view builds
it, with `Alternate` playing every other time backward; `Delay` holds the
timeline at its first value for a while before it plays, and `RepeatDelay`
the value it got to between plays. `Settled` reports that it rests, at
its end or, in reverse, at its start, and `Arrived` is true in the one frame
in which it came to rest, as `Settled(key)` and `Arrived(key)` are for a
value.

Change `Run` to play the timeline again, or a changed one. Building it
without changing `Run` keeps the playhead where it is, so a view may
rearrange its tweens from frame to frame: the timeline plays the ones it
arranged this frame. Arranging a longer one goes on from where the playhead
is, and it is not `Settled` until the timeline it arranged has played. A
timeline goes with its element's state when the view stops building the
element. An element asks for each of its timelines once per build, and the
`*Timeline` it returns is good while the view builds that frame.

While the desktop asks for less motion, a timeline that would play is at
once where it would come to rest, every tween at its `To`, or at its `From`
when it plays back to its start, as values, poses and `Keyframes` are, and
it asks for no frame, unless its `NonMotion` says it only fades (see
[Less motion](#less-motion-and-frames)). What the view does with the
playhead itself still works, as a drag still moves a `Draggable` element:
`Seek` and `Progress` put it where a scrubber asks, and `Paused` holds it
there.
`DisableAnimations` does not change a timeline, which plays on its own as
`Keyframes` do; `Seek` jumps it. A timeline draws frames only while it
plays, timed by `c.Now()`.

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

`InViewMargin(margin...)` grows the window and the scroll containers around
the element by `margin` DIPs before it looks, as IntersectionObserver's
`rootMargin` and `scrollMargin` do (Motion's `margin`): CSS style as
`Padding`, all sides, vertical and horizontal, or top, right, bottom and
left. A thumbnail starts loading 300 DIPs before it scrolls in, so that it
never shows blank:

```go
thumb := ui.Image(c, placeholder).Key(it.ID).Size(160, 120)
if thumb.InViewMargin(300) {
	app.load(it.ID) // once; the next frames show the picture
}
```

A negative margin shrinks them instead, to count an element in view only
once it is that far inside. Other clips around it (`Clip`, `ClipInset`)
clip it as they do, without the margin. `InViewMargin()`, with no margin,
is `InView()`. A `List` or `GridView` builds only the rows near its edges,
two beyond each, so in one the margin reaches no further than the rows it
builds.

`InViewAmount(amount, margin...)` waits for that share of the element's
area, from 0 to 1, to show, as IntersectionObserver's `threshold` and
Motion's `amount` do: an impression counts once half of a card shows,
not as it peeks in at the bottom of a list, and an animation starts at a
fifth of it. `1` waits for all of it, `0` is `InViewMargin`; the margin
grows the window and the scroll containers as `InViewMargin`'s does. Its
area is that of its box as transforms move it:

```go
card := ui.Column(c).Key(ad.ID)
if card.InViewAmount(0.5) {
	app.counted(ad.ID)
}
```

`ScrollState.ScrollTo(x, y, animation)` scrolls a container smoothly, and
`ScrollIntoView` with an `Animation` in its options the containers around
an element (see [Scrolling](scroll.md)).

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

`Morph` moves the characters the old and the new text share from where
they were to where they are, those of the old text alone fading out and
those of the new fading in, as Keynote's Magic Move by characters: words
reflowing, a title rewritten, letters rearranged. Rich text morphs too,
its characters going to the colors of their spans. A text flying with a
[`Match`](transitions.md#text-that-morphs) morphs from the text of the
element of its `Match`.

```go
ui.Text(c, app.title).ContentTransition(ui.ContentTransition{Morph: true})
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

What a value moves is the view's to say. One that moves nothing, as an
opacity, a blur or a crossfade the view drives itself, says so with
`NonMotion` on its `Animation`, `KeyframeOptions` or `TimelineOptions`,
and then animates with less motion too, as an element's own fades do:

```go
op := e.AnimateTo("fade", target, ui.Animation{Duration: 200 * time.Millisecond, NonMotion: true})
ui.Image(c, first).Opacity(1 - op)
ui.Image(c, second).Opacity(op)
```

Leave it off for a position, a size, a scale or a scroll, and give what
moves a timeline of its own beside its fades', which goes to rest at once.

Animations draw frames only while something moves, at the display's rate,
and none while the window is out of sight (below). Frames moving values
the view reads run the view; poses, content transitions, and drawings that
animate from their `Draw` only paint.

## Out of sight

While nothing of a window shows (on macOS: hidden, minimized, on another
Space, or entirely covered by other windows; in a browser: a tab in the
background), MyGo draws no frames for it, as browsers draw nothing for
pages out of sight and as Apple asks of Mac apps. The view does not run,
`Draw` is not called, and the frames `After` and `Painter.After` ask for
wait. As soon as some of the window shows, it draws once, with everything
that changed meanwhile; until then the screen shows its last frame, for
about a refresh. A frame still builds after input (keys typed into a key
window another app's window covers), when assistive technology turns on,
and for `Window.CapturePage`, so that a capture shows the window as it is
now. Windows and Linux do not tell MyGo when a window is out of sight, so
there it draws as it always did.

What this means for motion:

- **What MyGo animates goes on from where the time puts it.** A spring,
  an `Animate` on its way, a `Loop`, keyframes and timelines running when
  the window went out of sight are further along, or done, when it shows.
- **A target that changed out of sight animates when the window shows.**
  The view first sees the new target then, so the value moves to it from
  where it was, as a CSS transition does for a change made in a hidden
  tab.
- **Compute your own motion from the time, not from frames.** A start
  time taken in the first frame that draws something, or a count of
  frames, starts over when the window shows, and plays what happened
  meanwhile as if it just began. Take the time from when the state
  changed, in the code that changes it, and the view shows a change made
  out of sight as done:

```go
// counter eases from one value to the next over 400 ms, by the clock.
type counter struct {
	from, to float32
	changed  time.Time
}

const ease = 400 * time.Millisecond

// at returns the value shown at t.
func (n counter) at(t time.Time) float32 {
	p := min(float32(t.Sub(n.changed))/float32(ease), 1)
	return n.from + (n.to-n.from)*ui.EaseOut(p)
}

// Where the count changes, outside the view:
now := time.Now()
app.count = counter{from: app.count.at(now), to: float32(n), changed: now}
win.Update(func() {})

// In the view:
value := app.count.at(c.Now())
if c.Now().Sub(app.count.changed) < ease {
	c.AnimationFrame()
}
```

- **Keep work out of the view.** Polling, saving, or starting jobs from
  the view, from `After` or from a `Draw` waits while the window is out of
  sight. Run such work from the app (a goroutine or a `time.Ticker`) and
  show what it does with `Window.Update`.
