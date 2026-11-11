# Loading

The loading plugin has the 29 loading indicators of
[loading.dev](https://loading.dev), Jakub Krehel's `loading-dev` for
React, for native UI: arcs, rings, dots, bars and blocks that MyGo draws
itself, each an element that moves while it shows. They look and move as
loading.dev's do: the same shapes, keyframes and timing functions, the
same pace. It is all Go: no JavaScript package, and nothing to
`mygo.Use`. Pages, which have CSS, can use `loading-dev` itself.

[examples/loading](../../examples/loading) shows them all, with their
options:

<!-- app loading 480 -->

```go
import "github.com/egoist/mygo/plugins/loading"

ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
	loading.Arc(c).Label("Saving")
	ui.Text(c, "Saving…")
})
```

## The indicators

| Function | What it draws |
| --- | --- |
| `Arc` | A single open stroke turning in a circle. |
| `Atom` | Three rings tumbling inside a circle. |
| `Blocks` | Nine blocks shrinking and growing in a sweep across a grid. |
| `BouncingDots` | Three staggered dots bouncing up and down. |
| `Cascade` | Three nested arcs fanning into a spiral, then snapping back in line. |
| `CircularDots` | Eight dots in a ring, the brightest going around. |
| `Classic` | Twelve fading bars in a radial pattern. |
| `ClassicV2` | Eight ticks in a ring, lit in turn. |
| `Clock` | A clock hand sweeping around a faint face. |
| `Comet` | A full ring fading into its tail. |
| `Compass` | Four ticks turning a quarter turn at a time. |
| `Dual` | Two arcs turning in opposite directions. |
| `Eclipse` | Two dots trading places, one passing behind the other. |
| `Flip` | A square flipping over one axis, then the other. |
| `Gather` | Four blocks pulling together, turning, and pushing apart. |
| `Leap` | Three dots in a row, the last one leaping to the front. |
| `LinearDots` | Three dots lighting up in turn from left to right. |
| `Loading` | The loading.dev mark, its brightest segment circling the ring. |
| `Morph` | A square rounding into a circle and back as it turns. |
| `Orbit` | A fading half ring turning around a dot. |
| `Pulse` | A ring rippling outward from a dot. |
| `Radar` | A fading beam sweeping around a faint dish. |
| `Ring` | An arc turning in a faint circle. |
| `Ripple` | Three rings spreading out from the center. |
| `Slide` | Three dots sliding into the empty corner of a square. |
| `Snake` | An arc stretching and shrinking as it circles. |
| `Swirl` | A bright cell chasing its trail around a square. |
| `Trace` | A dash tracing the outline of a rounded square. |
| `Wave` | Five bars rising and falling in a wave. |

`loading.All()` lists them with their names, for a gallery or a setting
that lets the user choose one:

```go
for _, s := range loading.All() {
	ui.Column(c).Gap(8).Center().Children(func() {
		s.New(c).Label(s.Name)
		ui.Text(c, s.Name)
	})
}
```

## Options

Each function takes `loading.Options`, whose zero values are
loading.dev's defaults:

```go
loading.Ring(c, loading.Options{
	Size:     32,                 // DIPs high: 20 by default
	Color:    t.Accent,           // the theme's text color by default
	Duration: 600 * time.Millisecond, // one cycle: the indicator's own by default
	Easing:   loading.EaseInOut,
	Cap:      loading.FlatCap,
})
```

- **`Size`** is the indicator's height, and its width, except for
  `BouncingDots` and `LinearDots`, rows of dots 1.15 and 0.9375 times as
  wide as high. The element's own size, as `Size`, `Fill` or a parent's
  layout give it, scales the drawing to fit, centered.
- **`Color`** is what it draws in, with the faint parts (a ring's track,
  a fading dot) in it at their opacity. Zero is the theme's `Text`.
- **`Duration`** is one cycle of its motion, which differs from one to
  another: 800 ms for `Arc`, 2.4 s for `Slide`.
- **`Paused`** stops it where it is; set back to false, it goes on from
  there.
- **`Easing`** is how the indicators that turn (`Arc`, `Atom`, `Clock`,
  `Comet`, `Dual`, `Orbit`, `Radar`, `Ring`, `Snake`, `Trace`) go
  around: `Linear` at one speed, `EaseInOut` slowing at the end of each
  turn, `Stacked` both added, two turns a cycle, fast in the middle.
- **`Cap`** ends the strokes of `Arc`, `Cascade`, `Dual`, `Ring`,
  `Snake` and `Trace`: `RoundCap` or `FlatCap`.
- **`Sweep`** is the order of `Blocks`'s blocks: `Diagonal`, `Rows` or
  `Columns`.
- **`Direction`** sends `Ripple`'s rings out from the center
  (`RippleOut`) or in to it (`RippleIn`).
- **`Origin`** grows `Wave`'s bars from their middle (`WaveCenter`) or
  their bottom (`WaveBottom`).

## Motion and accessibility

An indicator asks for a frame each time it is painted, as a spinner
does (`Painter.AnimationFrame`), and only while it shows: one out of view
is not painted and asks for none, and it does not build the view again.
Each starts its cycle as it first shows, as a CSS animation does.

Where the desktop asks for less motion (`ui.Preferences.ReduceMotion`:
macOS's Reduce Motion, Windows's animation effects or GNOME's animations
turned off), an indicator stands still in loading.dev's pose for reduced
motion, as dimmed dots or a ring of rings, and asks for no frames.

Assistive technology sees a progress indicator of unknown length. Name it
with `Label`, as what is loading: `loading.Ring(c).Label("Loading
messages")`.
