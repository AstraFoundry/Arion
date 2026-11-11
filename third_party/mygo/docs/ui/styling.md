# Styling and themes

`Background`, `Border`, `Radius`, `Shadow` and `Opacity` style an
element's box, and `Cursor` sets the pointer over it:

```go
ui.Column(c).Padding(18).Gap(12).Radius(10).
	Background(t.Background).Border(1, t.Border).
	Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.06))
```

- **Borders.** `Border(1, c)` draws one inside every edge; `BorderWidth`
  sets the sides apart, CSS style, with `BorderColor`, as a line under a
  header with `BorderWidth(0, 0, 1, 0)`, and `BorderStyle(ui.BorderDashed)`
  dashes it. As with CSS's `box-sizing: border-box`, the border takes room
  within the element's size, and the padding and the children are inside
  it.
- **Shadows.** `Shadow(x, y, blur, spread, c)` casts a box shadow, as CSS's
  `box-shadow` does: several stack, and each shows only outside the box,
  so a translucent background never shows its own shadow through.
  `InsetShadow(x, y, blur, spread, c)` draws an inner one, as
  `box-shadow: inset`: inside the box's padding edge, within its border
  and rounded corners, above the background and below the children; the
  offset moves it, and a positive spread grows it inward. Inner shadows
  stack too, and go with outer ones, as a hairline that edges a field in
  the dark:

  ```go
  field.Radius(12).Background(t.Surface).
  	Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.3)).
  	InsetShadow(0, 0, 1, 0, ui.RGBA(255, 255, 255, 0.2))
  ```

  `NoShadow()` takes off the shadows added so far, as `box-shadow: none`,
  to flatten a card a helper returned with one; shadows added after it
  start afresh. A [transition](transitions.md#what-moves) fades shadows
  as they change.
- **Gradients and stripes.** `Gradient(from, to, angle)` fills the box with
  a linear gradient; `LinearGradient` also places its colors along the line
  (`Start`, `End`) and mixes them in Oklab, which keeps their lightness,
  instead of sRGB, or takes as many colors as its `Stops`, each a
  `GradientStop{Color, At}` at its place from 0 to 1. `ConicGradient` turns
  its stops around a point instead, as CSS's `conic-gradient()` (SwiftUI's
  `AngularGradient`): from `Angle`, in degrees clockwise from upwards,
  around `Center`, the middle unless set. `RadialGradient` spreads them
  out from `Center` in circles, as CSS's `radial-gradient()` (SwiftUI's
  and Flutter's `RadialGradient`): from 0 there to 1 at `Radius` DIPs
  away, or at the farthest corner of the box without one, as CSS's
  `farthest-corner`, its first color before the center and its last past
  the radius, for glows, spotlights and vignettes:

  ```go
  ui.Box(c).Size(240, 140).Radius(16).RadialGradient(ui.RadialGradient{
  	Center: ui.Point{X: 0.3, Y: 0.2}, Oklab: true,
  	Stops:  []ui.GradientStop{{ui.Hex("#a5b4fc"), 0}, {ui.Hex("#312e81"), 1}},
  })
  ```

  `BorderLinearGradient` and `BorderConicGradient` paint the border with
  one in place of its color, as SwiftUI's `strokeBorder` with a gradient:

  ```go
  rainbow := []ui.GradientStop{{red, 0}, {yellow, 0.33}, {blue, 0.66}, {red, 1}}
  ui.Box(c).Size(120, 80).Radius(16).Border(4, ui.Color{}).
  	BorderConicGradient(ui.ConicGradient{Stops: rainbow, Oklab: true})
  ```

  `Stripes(c, width, gap, angle)` draws stripes over the background, as on
  what is unavailable. A painter fills rectangles and paths with the same
  gradients (`FillGradient`, `FillConicGradient`, `FillRadialGradient`,
  `FillPathGradient`, `FillPathConicGradient`, `StrokePathConicGradient`).
- **Opacity.** `Opacity(o)` fades the element and its children as one
  group, as CSS's `opacity`: what they paint is put together first, then
  faded, so a badge over a card's edge, or two bars of a chart that
  overlap, show as one, not each through the other.

  ```go
  ui.Row(c).Gap(8).Children(func() { /* … */ }).Opacity(0.5) // a toolbar, faded
  ```

  Children that overlap nothing paint faded each on their own, which
  looks the same at no cost; where they overlap, they render into a
  bitmap of the GPU first, as a blur's, its text smoothed in grayscale.
  Web views, and materials that show what is behind them, fade on their
  own, as no bitmap shows them; so does what `Disabled` dims.
- **Blur.** `Blur(sigma)` blurs the element and its children, as CSS's
  `filter: blur()`: what they draw spreads about 3 sigma DIPs around,
  transparent beyond, on the GPU. They render into a bitmap first, as an
  [effect](shaders.md#effects-on-elements) does, every frame they paint;
  clicks, focus and assistive technology take the element where it is.
  Glows are borders blurred, a sharp one and blurred ones stacked:

  ```go
  for _, l := range []struct{ width, blur float32 }{{6, 0}, {9, 4}, {15, 15}} {
  	ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().
  		Radius(16).Border(l.width, ui.Color{}).BorderConicGradient(glow).Blur(l.blur)
  }
  ```
- **Clipping to an inset.** `ClipInset(top, right, bottom, left, radius)`
  clips the element and everything inside it (background, shadows,
  border, children) to its box inset by those DIPs, as CSS's
  `clip-path: inset()`. `ClipInsetPercent` takes percent of the box
  instead. A negative inset reaches beyond the box on that side, so
  `ClipInset(0, -40, 0, -40, 0)` clips top and bottom while a shadow
  still shows left and right. The insets are drawn as a clip of the
  scene, not a layer, so they can change every frame. Animating one
  wipes the element in:

  ```go
  hidden := float32(100)
  if app.open {
  	hidden = 0
  }
  card.ClipInsetPercent(0, card.AnimateTo("wipe", hidden, ui.Smooth), 0, 0, 12)
  ```

  What the inset hides takes no clicks.
- **Gradient masks.** `MaskLinearGradient`, `MaskRadialGradient` and
  `MaskConicGradient` mask the element and what is inside it by a
  gradient across its box, as CSS's `mask-image`. Each point shows as
  much as the gradient's alpha there; the colors don't matter. Nothing
  shows outside the box (or outside the inset, with `ClipInset`). A
  scrolling list whose rows fade out at its edges:

  ```go
  black := ui.RGB(0, 0, 0)
  fade := []ui.GradientStop{{ui.Transparent, 0}, {black, 0.08}, {black, 0.92}, {ui.Transparent, 1}}
  list.MaskLinearGradient(ui.LinearGradient{Angle: 180, Stops: fade})
  ```

  The end of clamped text fades the same way, along its width:
  `MaskLinearGradient(ui.LinearGradient{Angle: 90, Stops: …})`.
  Unlike `Opacity`, the mask applies to each thing drawn on its own; the
  element isn't drawn into a layer first. So where a row's background
  is faded, what is under the row shows through it too. Clicks are not
  masked.
- **Layers of masks.** `Masks` masks the element by several layers, as
  CSS's `mask-image` with `mask-composite`: each a gradient (a
  `ui.Mask`'s `Image`, a `LinearGradient`, `RadialGradient` or
  `ConicGradient`) across the border box, or across the padding box or the
  content box (its `Box`, as `mask-origin` and `mask-clip`), the first
  on top. Each layer goes with those under it, after it in the list, as
  its `Composite` says: `MaskAdd`, by default, shows what either shows,
  `MaskSubtract` the layer where those under it don't show, `MaskIntersect`
  where both do, and `MaskExclude` where one does but not both. A
  gradient ring 1.5 DIPs wide, the content box cut out of the border box,
  as the gradient-border trick of CSS:

  ```go
  black := ui.RGB(0, 0, 0)
  solid := ui.LinearGradient{From: black, To: black}
  ui.Box(c).Size(120, 40).Radius(20).Padding(1.5).Gradient(pink, violet, 90).Masks(
  	ui.Mask{Image: solid, Box: ui.MaskContentBox, Composite: ui.MaskExclude},
  	ui.Mask{Image: solid},
  )
  ```

  and a crescent, a disc less another beside it:

  ```go
  disc := func(x, y float32) ui.RadialGradient {
  	return ui.RadialGradient{Center: ui.Point{X: x, Y: y}, Radius: 40,
  		Stops: []ui.GradientStop{{black, 0.98}, {ui.Transparent, 1}}}
  }
  moon := ui.Box(c).Size(80, 80).Background(t.Text)
  moon.Masks(ui.Mask{Image: disc(0.5, 0.5), Composite: ui.MaskSubtract}, ui.Mask{Image: disc(0.7, 0.35)})
  ```

  `Masks` replaces the gradient masks above, and they replace it. One
  layer across the border box masks as they do; more make a mask of their
  alpha, on the CPU, once for each size the element shows at, which each
  renderer draws as a path's clip. Like a gradient mask, it applies to
  each thing drawn on its own.
- **Caching.** `Cache()` keeps what the element and its children paint in
  a bitmap of the GPU, as Flutter's `RepaintBoundary` and SwiftUI's
  `drawingGroup()` do, drawn again only when what they paint changes:
  other content, another size, the window's scale or theme. A still card
  of gradients, blurred images and large shadows around a drawing that
  moves then costs one image wherever a frame redraws over it, and
  turning or scaling the element with a transform moves the bitmap
  without drawing it again. The bitmap holds the box, its shadows and
  children beyond it that it does not clip; text over no opaque
  background of the subtree's own is smoothed in grayscale, as in a blur.
  With `Blur` or an effect, their bitmap is the one kept.

  ```go
  ui.Box(c).Size(900, 212).Radius(16).Gradient(from, to, 30).
  	Shadow(0, 10, 40, 0, shade).Cache().Children(func() { /* … */ })
  ```
- **Materials.** `Material(m)` fills the box with a material a package
  paints, in place of a background, as the [glass
  plugin](../plugins/glass.md)'s Liquid Glass, which what is under it shows
  through, frosted and bent along its edges, its scroll edges and its
  backdrop blur, which may fade along a gradient.
- **Pointer.** `Cursor` takes the shapes of the platforms: `CursorPointer`,
  `CursorText`, `CursorMove`, `CursorGrab` and `CursorGrabbing`, the
  resize cursors (both ways, toward one side as `CursorResizeE`, and of
  columns and rows), `CursorCopy` and `CursorAlias` for drops,
  `CursorContextMenu`, `CursorVerticalText`, `CursorNotAllowed`,
  `CursorCrosshair`, `CursorNone`, which hides it, CSS's `progress` and
  `wait` (`CursorProgress` over a control that is busy but still takes
  input, as one with `aria-busy`, and `CursorWait` while nothing can be
  done), `CursorHelp`, `CursorCell`, `CursorZoomIn` and `CursorZoomOut`,
  `CursorAllScroll` and `CursorNoDrop`. Windows has no cursors for
  zooming and cells, which MyGo draws, as browsers bring their own;
  macOS has `CursorWait`, `CursorHelp`, the zoom cursors and
  `CursorAllScroll` among AppKit's private cursors, as WebKit uses them,
  and shows the nearest public one where a version lacks them (the
  crosshair for `CursorCell`). An element's
  cursor shows over its children too, except over those that take clicks
  (a button, or anything with `Clicked` or `OnClick`): those show their
  own, the default arrow unless they set one, so the buttons of a field
  with `CursorText` show no I-beam.

`ui.Oklch(l, c, h)` is CSS's `oklch()`; `.Alpha(a)` is its `/ a`. It
returns a `ui.Color` like any other, so fills, borders, gradients,
stripes, shadows, dividers, text, decorations, text backgrounds, the
glass's tint and the theme (`t.Accent = ui.Oklch(0.6, 0.2, 240)`) all take
it. It can name colors outside sRGB, more vivid than most screens show.
Such a color keeps both: its real value, drawn when the window is on a
screen that shows a wider range (Display P3 on recent Macs), and the
nearest sRGB color, chosen as CSS does (less saturated, same lightness and
hue). That fallback is its `R`, `G` and `B`, and is what sRGB screens,
Linux, Windows and SVG pictures draw; `SRGB()` returns it without the real
value. `Mix` and transitions mix the real values: a mix back inside sRGB
is an ordinary color. Don't assign `R`, `G` or `B` of such a color, as the
real value would no longer match: make a new one.

Colors come from `ui.RGB`, `ui.RGBA` and `ui.Hex("#2563eb")`; `Mix` blends
two, `MixOklab` blends them in Oklab, as a gradient's `Oklab` does (yellow
and blue meet in a blue rather than in sRGB's gray), and `Alpha` makes one
translucent.

## State, not selectors

There are no style sheets and no state selectors: the view is code, so an
element's look follows the state in the view itself, as
`if row.Hovered() { row.Background(t.SurfaceHover) }`, and one element's
state can style another, as a group's hover does in CSS.

A widget returns its element, so a call after it styles it differently
from the rest: `ui.Button(c, "Save").Padding(10, 20).Radius(999)`. For a
look of your own, build on the widgets' bases, which have none: see
[custom widgets](custom-widgets.md).

## Themes

Widgets take their colors and metrics from the theme, `c.Theme()`: the
light or the dark theme, following the system's appearance as it changes,
and the settings of the desktop that the system's own controls follow. Use
its colors in your own elements so they follow too. To change it, set a
copy:

```go
t := *c.Theme() // the default, which follows the system
t.Accent, t.Radius = ui.Hex("#7c3aed"), 8
t.Spacing = 3 // compact
c.SetTheme(&t)
```

`Spacing` is the unit of the room widgets leave: their paddings and gaps,
and the sizes of check boxes, switches, sliders and the rows of tables and
trees, are multiples of it. It is 4 by default; 3 makes every widget
compact, 5 roomy. `FontSize` sizes their text, `Radius` rounds their
corners, and `ScrollbarWidth` sets the width of scroll bars. On macOS,
rounded corners, the widgets' and your elements', curve continuously as
AppKit's and SwiftUI's do, and elsewhere as quarter circles, as Windows
and GTK draw them. Size your own
elements with the theme too, and they follow it: `t.Space(3)` is three
units of its spacing, and `t.Rem(2)` twice its font size, as CSS's rem.

Tooltips and toasts are the theme turned over, so that they stand out from
what they are over: filled with `Inverse`, with `InverseText` on it, which
left zero are the theme's `Text` and `Background`. A dark app that would
rather keep them dark sets both:

```go
t.Inverse, t.InverseText = ui.Hex("#3f3f46"), ui.Hex("#fafafa")
```

`TooltipDelay` is how long the pointer rests on an element before its
tooltip shows, 0.6 seconds when left zero (see
[Tooltip](tooltip.md#the-delay)).

In a window that is not the one the user's input goes to, the widgets
fade their accent to gray, as AppKit fades an inactive window's (see
[Active and inactive windows](windows.md#active-and-inactive-windows)):
selected rows, checked controls, sliders and progress bars are filled
with `InactiveAccent`, which left zero is a gray `AccentText` stands out
on, and primary buttons look as the others. A theme that keeps its accent
in every window sets it to the accent:

```go
t.InactiveAccent = t.Accent
```

## The focus ring

The element with the keyboard focus from the keyboard shows a ring, which
the theme draws: `FocusRingWidth` DIPs wide (2 by default), its inner edge
`FocusRingOffset` DIPs outside the element's edge (1 in the light and dark
themes; 0 puts it on the edge, and a negative offset inside it), in
`Focus`. Its corners follow the element's, as CSS's outline does. A theme
of a thicker ring, further out:

```go
t.FocusRingWidth, t.FocusRingOffset = 3, 2
```

An element sets its own offset with `FocusRingOffset`, and its color with
`FocusRingColor`, as a control whose ring must sit inside it because an
element around it clips what goes past its edges, as a header in an
accordion's box does:

```go
header.FocusRingOffset(-4) // inside, its outer edge 2 DIPs in
```

Widgets that ring a part of themselves, as a check box its box, take the
same settings. `FocusRing(false)` turns the ring off for a widget that
draws its own with `Painter.FocusRing`, the theme's ring around any
rectangle, while `FocusVisible`.

## A theme for part of the window

`SetTheme` themes the whole window: the root, its background and its text,
and the widgets built after the call. `ui.Themed(c, t)` themes a part of
it instead: a column whose children build with `t`, as a theme picker
previewing light beside dark, or a light preview pane inside dark chrome:

```go
ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
	for _, t := range []*ui.Theme{ui.LightTheme(), ui.DarkTheme()} {
		ui.Themed(c, t).Grow(1).Padding(16).Gap(8).Children(func() {
			ui.Text(c, "Preview")
			ui.Button(c, "OK")
		})
	}
})
```

Inside, everything is as in a window of `t`: `c.Theme()` returns it while
the children build, and `Painter.Theme()` while they paint, so widgets of
your own follow too; the widgets take its colors and its metrics
(`Spacing`, `Radius`, `FontSize`, `ScrollbarWidth`, so `t.Space` and
`t.Rem`); and the popovers, menus, tooltips and dialogs opened from inside
take it, though they show above the window. The column paints `t`'s
`Background`, and its text takes `t`'s `Text` color, `FontSize` and
`Font`, as the root does the window's theme's; `.Background(ui.Transparent)`
leaves what is behind it showing instead, as for the root. The root, and
what is built after the column, keep the window's theme; toasts are the
window's, and keep it too. Themes nest, and a nil theme makes it a plain
column. `SetTheme` called inside still themes the window, from the root to
what follows the column, while the column's children keep `t`.

## The desktop's preferences

`c.Preferences()` returns the settings of the desktop, which the default
theme follows and a frame follows the changes of:

- **`Accent`.** The accent color the user chose (macOS, Windows, and
  desktops whose portal gives one, as GNOME 47 and KDE do), which colors
  primary buttons, the choice and the focus ring, with text on it that
  stands out.
- **`HighContrast`.** macOS's Increase Contrast, Windows's contrast themes,
  the portal's higher contrast: borders and secondary text are darker
  (lighter in the dark), and the focus ring opaque.
- **`TextScale`.** Windows's and GNOME's text size: `FontSize` is that many
  times larger.
- **`ReduceMotion`.** macOS's Reduce Motion, Windows's animation effects and
  GNOME's animations turned off: `Animate` goes to its target at once.
- **`Pointer`, `AnyPointer`, `Hover()` and `AnyHover()`.** What the
  pointing devices can do, as Media Queries 4's `pointer`, `any-pointer`,
  `hover` and `any-hover`: `Pointer` is how accurately the primary one
  points, `ui.PointerFine` for a mouse, a touchpad or a pen,
  `ui.PointerCoarse` for a touch screen, `ui.PointerNone` for none, and
  `AnyPointer` the most accurate of all; `NoHover` and `NoAnyHover`
  (`Hover()` and `AnyHover()` the other way round) tell that the primary
  one, or every one, cannot hover. A touch-first layout gives larger
  targets to a coarse pointer, and shows on hover alone only what a
  pointer that hovers can reach:

  ```go
  p := c.Preferences()
  size := float32(28)
  if p.Pointer == ui.PointerCoarse {
  	size = 44
  }
  del := ui.Button(c, "×").Size(size, size).Label("Delete")
  if p.Hover() {
  	del.Tooltip("Delete")
  }
  ```

  Linux reads the seat's devices (GDK's), and follows them as they come
  and go; Windows reads whether a mouse, a pen or a touch screen is
  there, the touch screen being the primary one in tablet mode; browsers
  answer the media queries. macOS's, and those not known, are a mouse's:
  fine, and hovering.
