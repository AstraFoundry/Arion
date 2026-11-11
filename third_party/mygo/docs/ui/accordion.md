# Accordion

<!-- preview accordion 300 -->

`ui.Accordion` creates sections, one above the other in a bordered box,
that `ui.AccordionItem` builds: a header showing a title, which a click
opens and closes, as do Enter and Space, and below it what its function
builds while its `*bool` is true. Each opens and closes on its own; Up and
Down move the focus between their headers, as do Home and End. An item's
`Changed` reports a click.

```go
ui.Accordion(c, func() {
	ui.AccordionItem(c, "General", &app.general, func() { app.generalSettings(c) })
	ui.AccordionItem(c, "Privacy", &app.privacy, func() { app.privacySettings(c) })
	ui.AccordionItem(c, "Advanced", &app.advanced, func() { app.advancedSettings(c) })
})
```

For one section open at a time, close the others as one opens:

```go
for i, s := range sections {
	open := app.section == i
	ui.AccordionItem(c, s.Title, &open, s.Build).OnChange(func() {
		app.section = -1
		if open {
			app.section = i
		}
	})
}
```

Tab goes from a header into the content of its section while it is open.

## Without a look

`ui.AccordionBase` is an accordion without a look, for headers and panels
of your own layout: a `Root` column holding the sections, each made by
`Item`, a [`CollapsibleBase`](collapsible.md#without-a-look) open while
its `*bool` is true, whose `Trigger` is the header and whose `Panel`
builds the content. The headers have the keys of `Accordion`'s: Up and
Down move the focus between them, as do Home and End, and Enter and Space
open and close them.

```go
acc := ui.AccordionBase(c)
acc.Root.Gap(4).Children(func() {
	for i := range app.sections {
		s := &app.sections[i]
		item := acc.Item(&s.open)
		item.Trigger.Gap(8).Padding(8, 12).Radius(8).Children(func() {
			ui.Icon(c, s.icon)
			ui.Text(c, s.title).Grow(1)
			ui.Text(c, s.summary).TextColor(t.TextMuted)
			ui.Icon(c, chevron).Rotate(180 * item.Progress())
		})
		item.Panel(func() { s.build(c) })
	}
})
```

`Accordion` is `AccordionBase` with the theme's look: its headers ring
the focus inside them (`FocusRingOffset`), as the box clips what goes
past it. See [custom widgets](custom-widgets.md).

## Accessibility

Assistive technology sees each header as a disclosure, expanded or
collapsed, before the content of its section.
