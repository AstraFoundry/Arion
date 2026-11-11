# Tooltip

<!-- preview tooltip 160 -->

`Tooltip` shows a tip once the pointer rests on an element, or the keyboard
focus comes to it, and describes the element to assistive technology:

```go
ui.Button(c, "").Label("Share").Tooltip("Share with others").Children(func() {
	ui.Icon(c, share)
})
```

The tip shows once the pointer has rested on the element for 0.6 seconds,
and at once as the keyboard focus comes to it from the keyboard. It goes
above the element, centered, or below it where there is no room above,
wherever the pointer is on the element, as Base UI's does. Within 0.4
seconds of a tip going, the next one shows at once, as the pointer moves
along a toolbar. It goes as the pointer leaves, and a press or a click on
the element, or Escape, hides it until the pointer leaves and comes back,
or the focus does: it does not show again after a click, nor beside the
menu a menu button opens. Over elements inside one another, the innermost
one's tip shows, and one tip shows at a time. Give one to buttons showing
only an icon, and to what a label alone does not explain.

The focus counts for the element an element gives it to, its
`ActiveDescendant`, as a menu drawn in the window gives it to the item the
keys highlight: that one's tip shows.

The tip takes the theme's `Inverse` and `InverseText` colors, its text and
background turned over unless the theme sets them (see
[themes](styling.md#themes)).

## The delay

The theme's `TooltipDelay` sets how long the pointer rests on an element
before its tip shows, for `Tooltip` and `TooltipBase` alike: 0 is 0.6
seconds, and a negative delay shows the tip at once. The time within which
the next tip shows at once follows it, two thirds of it (0.4 seconds by
default). A dense settings page shows its tips sooner, and a toolbar of
many small buttons, or a page that also shows help inline, later:

```go
t := *c.Theme()
t.TooltipDelay = 300 * time.Millisecond
c.SetTheme(&t)
```

For a part of the window only, as a toolbar, build it in `ui.Themed`
with that theme (see [a theme for part of the
window](styling.md#a-theme-for-part-of-the-window)); each element's tip
waits as the theme it is built with says.

## Without a look

`TooltipBase` shows a tip of your own by an element, as Base UI's Tooltip
does: it decides when the tip shows and goes, as above, and `fn` styles the
tip and builds its content. The tip goes above the element, centered, or
below it where there is no room above; place it elsewhere with `AttachTo`,
its margins keeping it apart from the element. `AttachedAnchors` tells the
side it went to, and `AttachedOrigin` scales it from the edge nearest the
element (see [overlays](overlays.md#your-own-overlays)):

```go
b := ui.Button(c, "").Label("Share").Description("Share with others")
b.Children(func() { ui.Icon(c, share) })
ui.TooltipBase(c, b, func(tip ui.Element) {
	tip.AttachTo(b, ui.AnchorRight, ui.AnchorLeft).Margin(0, 0, 0, 6)
	tip.Padding(4, 8).Radius(6).Background(ink).TextColor(paper)
	tip.Children(func() {
		ui.Text(c, "Share with others")
		ui.Text(c, "⌘S").TextColor(muted)
	})
})
```

It returns the tip, or nil while it does not show. The pointer goes
through the tip.

## Accessibility

Assistive technology reads the tip of `Tooltip` as the element's
description, after its name, as help text, unless `Description` gives
another. It sees the tip of `TooltipBase` as a tooltip, which it does not
read: give the element a `Description` telling what the tip does.
