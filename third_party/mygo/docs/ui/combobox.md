# Combobox

<!-- preview combobox 280 -->

`ui.Combobox` creates a combobox choosing a `*string` among options: a
field showing the choice, whose text filters the options below it as the
user types, those starting with it first, while a click or Down shows them
all; a click showing them selects the choice, for typing to replace. Up
and Down move among them and Enter chooses, as does a click; Escape closes
them, and the text goes back to the choice when the field loses the focus.
`Changed` reports a new choice.

```go
ui.Combobox(c, &app.font, fonts).Label("Font")
```

The field keeps the keyboard focus while the user types and picks.
`Placeholder` shows a text in it while its text is empty.

For a text that may be anything, with suggestions, use
[autocomplete](autocomplete.md); for a short list without typing, a
[select](select.md).

## Without a look

`ui.ComboboxBase` is a combobox without a look: an `Input` whose text
filters the `Item`s of a `Popup` below it, which the arrows highlight and
Enter or a click chooses (`Chosen`): see [custom widgets](custom-widgets.md).
When the options are those matching what was typed, as here, tell
assistive technology with
[`cb.Input.AutoComplete(ui.AutoCompleteList)`](accessibility.md#suggestions).

A popup without options hides, unless it holds something else to read, as
a row saying that nothing matches, which then shows in their place:

```go
cb.Popup(func(panel ui.Element) {
	panel.Padding(4).Background(t.Background).Border(1, t.Border)
	matches := app.matching(app.query)
	for _, f := range matches {
		cb.Item(f).Padding(6, 10).Children(func() { ui.Text(c, f) })
	}
	if len(matches) == 0 {
		ui.Text(c, "No matches found").Padding(6, 10).TextColor(t.TextMuted)
	}
})
```

## Accessibility

Assistive technology sees a combo box named by its `Label`, editable as a
text field, which expands, and whose options complete what was typed
(`aria-autocomplete="list"`); the option the arrows are on has the focus
for it, and reads as which of how many it is. With no options, the focus stays
in the field, and a popup showing a row in their place is read out as it
appears.
