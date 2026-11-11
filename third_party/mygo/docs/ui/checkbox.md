# Checkbox

<!-- preview checkbox 220 -->

`ui.Checkbox` creates a check box toggling a `*bool`, with a label: a click
on the box or its label checks it or unchecks it, as does Space while it
has the keyboard focus, and `Changed` reports a click.

```go
ui.Checkbox(c, &app.settings.Sync, "Sync across devices")
if ui.Checkbox(c, &app.showHidden, "Show hidden files").Changed() {
	app.reload()
}
```

The check box changes the value itself, the moment the user clicks: read
it, or ask `Changed` for work that follows.

## Groups

`ui.CheckboxGroup` creates a check box over the check boxes built inside
it, indented below it, as a list of permissions: it is checked when they
all are, mixed when some are, and a click, or Space while it has the focus,
checks them all, or none when they all are. `Changed` reports a click.

```go
ui.CheckboxGroup(c, "Notifications", func() {
	ui.Checkbox(c, &app.mail, "Mail")
	ui.Checkbox(c, &app.calendar, "Calendar")
	ui.Checkbox(c, &app.messages, "Messages")
})
```

## Mixed

`ui.CheckboxState` is a check box that can be mixed, as a "select all"
box in a table's header: checked when every row is, mixed when some are.
It sets a `*ui.CheckState`, `ui.CheckOff`, `ui.CheckOn` or
`ui.CheckMixed`, and shows a dash while mixed; a click, or Space while it
has the focus, checks it, or unchecks it once it is checked, and `Changed`
reports a click:

```go
all := ui.CheckOff
switch n := app.selected(); {
case n == len(app.rows):
	all = ui.CheckOn
case n > 0:
	all = ui.CheckMixed
}
if ui.CheckboxState(c, &all, "").Label("Select all").Changed() {
	app.selectAll(all == ui.CheckOn)
}
```

`CheckboxGroup` is such a check box over the check boxes it holds.

## Without a look

`ui.CheckboxBase` is a check box without a look: a row that toggles its
`*bool` when clicked, which you draw from the value, and
`ui.CheckboxStateBase` one that sets a `*ui.CheckState`, as
`CheckboxState` does: see [custom widgets](custom-widgets.md).

## Accessibility

Assistive technology sees a check box named by its label, checked or not,
which it toggles as a click does, and one that can be mixed checked,
unchecked or mixed. A check box group is a check box, mixed while some are
checked, and a group of the others named by its label.
