# Accessibility

Assistive technology, such as VoiceOver on macOS, Orca on Linux and
Narrator or NVDA on Windows, reads a window's native UI as it reads other
apps: the widgets with their roles, names, values and states, the texts,
and the elements that take the focus, as the content changes. It acts on
them as the keyboard and the pointer would: it presses buttons, checks
boxes, moves sliders, edits text inputs and moves the focus.

## Names

Widgets describe themselves, and the text inside an element names it, as a
button's does. Elements without text need a `Label`, which also finds them
in [tests](testing.md):

```go
ui.Slider(c, &app.volume, 0, 100).Label("Volume")
ui.TextInput(c, &app.query).Placeholder("Search").Label("Search")
ui.Box(c).Size(24, 24).Draw(drawIcon).Label("Unread messages")
```

A [field](form.md) names its control with its label, and `LabelledBy`
names a control by a text the view draws as its caption, a click on which
goes to the control ([labels of your own](form.md#labels-of-your-own)).

## Roles

Other elements get a role from what they do: one that is clickable and
takes the focus is a button, a scroll container a scroll area, and one with
a `Label`, or that takes the focus, a group. `Role` sets it for an element
drawn as a widget it is not built from, and `ui.RoleNone` leaves an element
out but not its children ([`AccessHidden`](#out-of-assistive-technologys-way)
leaves them out too):

```go
toggle := ui.Box(c).Size(36, 20).Focusable().Role(ui.RoleSwitch).Label("Wi-Fi")
```

Besides the roles of MyGo's widgets, `ui.RoleHeading` is the title of a
section, ranked with `Level`, `ui.RoleNavigation` a landmark of links
that move around the app or a document, as a breadcrumb trail or a
navigation bar, which screen readers list and jump to (as
[`Breadcrumbs`](breadcrumbs.md) is), and `ui.RoleMenu`, `ui.RoleMenuBar`,
`ui.RoleMenuItem`, `ui.RoleMenuItemCheckBox` and `ui.RoleMenuItemRadio`
make menus drawn in the window, as a drop-down menu or a menu bar of your
own (every menu MyGo draws in the window,
[`PopoverMenu`](context-menu.md#drawn-in-the-window)'s,
[`PopoverMenuButton`](menu-button.md#drawn-in-the-window)'s and [a
window's](context-menu.md#every-menu-of-a-window), has them, as do
[`MenuBase`](menu-button.md#without-a-look) and `ContextMenuBase`,
menus without a look that keep their keys and focus):

```go
ui.Text(c, "Appearance").Bold().Role(ui.RoleHeading).Level(2)
ui.Row(c).Role(ui.RoleNavigation).Label("Sections").Children(sectionLinks)
```

A table of your own builds its rows with `ui.RoleRow`, its cells with
`ui.RoleCell` and the headers of its columns with `ui.RoleColumnHeader`.
A header is named by the text inside it, and keeps the elements inside it
in the tree, as ARIA's `columnheader` does: a sort button and a column's
menu button in it are reached and pressed as any other button.

```go
ui.Row(c).Role(ui.RoleColumnHeader).Children(func() { // named "Sort Options"
	if ui.Button(c, "Sort").Clicked() {
		app.sortBy("name")
	}
	ui.Button(c, "Options")
})
```

`ui.RoleGrid` and `ui.RoleTreeGrid` are tables of your own whose cells
take the focus and the arrow keys, as a spreadsheet's, where
`ui.RoleTable` is one to read: see [Grids](#grids).

`ui.RoleListBox` is a list of options to choose from, and `ui.RoleOption`
one of its options, selected with `Checked`, as ARIA's `listbox`, `option`
and `aria-selected`: a list box on Windows (UI Automation's List, with
`Selection` and `SelectionItem`), on Linux (ATK's list box, with
`AtkSelection`) and on macOS (a table whose rows are selected, as
AppKit's lists are), as the
popups of [`SelectBase`](select.md#without-a-look) and
[`ComboboxBase`](combobox.md) are. Suggestions shown at a text area's
caret, as for mentions, are one, the text area keeping the focus and
pointing at the option highlighted with `ActiveDescendant`:

```go
area := ui.TextArea(c, &app.draft)
if len(app.matches) > 0 {
	ui.Column(c).Role(ui.RoleListBox).Label("People").Children(func() {
		for i, p := range app.matches {
			opt := ui.Row(c).Padding(4, 8).Role(ui.RoleOption).Checked(i == app.hi)
			if opt.Clicked() {
				app.mention(p)
			}
			opt.Children(func() { ui.Text(c, p.Name) })
			if i == app.hi {
				area.ActiveDescendant(opt)
			}
		}
	})
}
```

Pressing an option, as assistive technology choosing it does, clicks it;
it takes the focus only if it is `Focusable`.

## States

The bases tell assistive technology what their widgets show. A widget of
your own tells it with these, as ARIA's states do on the web:

| Method | What it tells |
|---|---|
| `Checked(on)`, `Mixed()` | whether a check box, switch, radio button, toggle or menu item is on, or partly on, or whether an option is selected |
| `AccessSelected(on)` | whether a tab, option, row, cell or tree item is selected, without the accent `Selected` paints ([below](#selected-without-the-look)) |
| `Expanded(open)` | whether what the element opens shows, as a popup or a section |
| `Value(s)` | its value, as the choice a button opening a popup shows |
| `Range(lo, hi, value)`, `Step(step)` | the range and value of a slider, progress bar, meter or stepper, and how far the keys move it |
| `Level(n)` | the rank of a heading, or how deep an item of a tree is |
| `PositionInSet(pos, size)` | which item of how many an item of a list, tree, menu or tab list is, from 1, where what is built does not say ([below](#place-in-a-set)) |
| `ActiveDescendant(e)` | the option that has the focus while the element keeps it, as the one the arrows are on in a menu or a list |
| `Busy(on)` | that the element's content is about to change, as a region loading: screen readers wait for it before reading it |
| `Current(kind)` | that the element is the current one of the items around it: `ui.CurrentPage` (the page the user is on), `CurrentStep`, `CurrentLocation`, `CurrentDate`, `CurrentTime` or `CurrentItem` |
| `AutoComplete(kind)` | what a text input's suggestions complete as the user types: `ui.AutoCompleteList` (a popup of them), `AutoCompleteInline` (the text completed in place) or `AutoCompleteBoth` ([below](#suggestions)) |
| `RoleDescription(s)` | what to call its role, in place of the role's own name, as "carousel" for a group of slides and "slide" for each |
| `Controls(e)` | that it controls the content or presence of `e`, as a disclosure button the region it opens; once for each |
| `HasPopup(kind)` | the kind of popup it opens: `ui.PopupMenu`, `PopupListBox`, `PopupTree`, `PopupGrid` or `PopupDialog` |
| `Sort(dir)` | the order a table's rows are sorted in by the column whose header it is: `ui.SortAscending`, `SortDescending`, `SortOther` or `SortNone` |
| `Orientation(o)` | which way a slider, separator, splitter, toolbar, tab list or menu of your own runs: `ui.Vertical` or `ui.Horizontal` ([below](#orientation)) |
| `DescribedBy(e)`, `Invalid(on)`, `ErrorMessage(e)` | the element `e` whose text describes it, that its value is not valid, and the element holding its error ([below](#descriptions-and-errors)) |
| `Multiselectable(on)` | that the user chooses several of the items of a list, grid, tree or tree grid of your own |
| `RowCount(n)`, `ColumnCount(n)` | how many rows and columns a table or grid has, built or not |
| `RowIndex(i)`, `ColumnIndex(i)` | which of all the rows a row or a cell is in, and which of all the columns a cell is in, from 1 |

```go
all := ui.CheckboxBase(c, &app.all).Label("Select all")
if app.some {
	all.Mixed()
}

date := ui.ButtonBase(c).Role(ui.RolePopUpButton).Label("Due").Value(app.due.Format("Jan 2")).Expanded(app.picking)
```

A region loading says so while a placeholder stands in for it, and the
link to the page shown in a navigation bar says it is the current page:

```go
ui.Column(c).Label("Messages").Busy(app.loading).Children(func() {
	if app.loading {
		ui.Spinner(c).Label("Loading")
		return
	}
	messageList(c, app.messages)
})

ui.Row(c).Role(ui.RoleNavigation).Label("Main").Children(func() {
	for _, p := range pages {
		l := ui.Link(c, p.Title, p.Path)
		if p.Path == app.path {
			l.Current(ui.CurrentPage)
		}
	}
})
```

An element with no role of its own and no `Label` is still seen once
`Busy` or `Current` marks it, as a group, as ARIA's states make a
generic element seen on the web. `Busy` is AppKit's `AXElementBusy`,
ATK's busy state and UI Automation's `AriaProperties` (`busy=true`);
`Current` is AppKit's `AXARIACurrent`, ATK's `current` attribute with the
active state, and `AriaProperties` (`current=page`).

A cell of a grid says whether it is chosen with `AccessSelected`, or
`Checked`, as ARIA's `aria-selected` on a `gridcell`, without painting
it, as a calendar's days or the cells of a spreadsheet of your own;
`Selected` paints it in the accent color too. Cells (`RoleCell`, `RoleColumnHeader`,
`RoleRowHeader`) that call neither are not chosen among others, and a
table or a [grid](#grids) (`RoleTable`, `RoleGrid`, `RoleTreeGrid`)
holding cells that do chooses them. A carousel, a
disclosure and a button opening a list box, built on the others:

```go
ui.Column(c).Role(ui.RoleGroup).Label("Highlights").RoleDescription("carousel").Children(func() {
	for i, s := range slides {
		ui.Column(c).Role(ui.RoleGroup).Label(fmt.Sprintf("%d of %d", i+1, len(slides))).
			RoleDescription("slide").AccessHidden(i != app.slide).Children(s)
	}
})

toggle := ui.Button(c, "Details").Expanded(app.open)
if app.open {
	toggle.Controls(ui.Column(c).Role(ui.RoleRegion).Label("Details").Children(details))
}

ui.ButtonBase(c).Role(ui.RolePopUpButton).Label("Fruit").Value(app.fruit).
	HasPopup(ui.PopupListBox).Expanded(app.picking)

day := ui.Box(c).Role(ui.RoleCell).Label(d.Format("January 2")).AccessSelected(d.Equal(app.picked))
```

They reach each platform as WAI-ARIA's Core Accessibility API Mappings
say. `RoleDescription` is AppKit's `AXRoleDescription`, ATK's
`roledescription` attribute and UI Automation's `LocalizedControlType`.
`Controls` is AppKit's `AXLinkedUIElements`, ATK's `controller-for`
relation (and `controlled-by` on what it controls) and UI Automation's
`ControllerFor`, which screen readers offer to move to (VoiceOver's
VO-J). `HasPopup` is AppKit's `AXPopupValue` with the `AXShowMenu`
action, which presses the element, ATK's `has-popup` state and
`haspopup` attribute, and UI Automation's ExpandCollapse pattern (with
`Expanded`) and `AriaProperties` (`haspopup=listbox`). `Sort` is
AppKit's `AXSortDirection` (unknown for another order) on a header that
is a sort button (`AXSortButton`), as NSTableHeaderView's are, ATK's
`sort` attribute, and UI Automation's `ItemStatus` and `AriaProperties`
(`sort=other`), as [`Table`](table.md)'s headers say from
`ListState.Sort`. A chosen cell is AppKit's `AXSelected` with its
table's `AXSelectedCells`, and its row in the table's `AXSelectedRows`,
as NSTableView tells its choice; as VoiceOver reads neither the state of
a cell nor its change, the cell says "selected" after its name, in the
app's language, and a cell chosen away from the focus is read out. It is
ATK's selectable and selected states, and UI Automation's SelectionItem,
in the table's Selection.

A [test](testing.md#what-assistive-technology-sees) reads them as
assistive technology gets them, with `Tester.AccessByRole`:

```go
if a, _ := tt.AccessByRole(ui.RoleGroup, "Messages"); !a.Busy {
	t.Error("Messages is not busy while loading")
}
if day, _ := tt.AccessByRole(ui.RoleCell, "November 2"); !day.Selected {
	t.Error("the day picked is not selected")
}
```

A check box that is mixed as often as not is better built on
[`CheckboxStateBase`](checkbox.md#without-a-look), which tells it itself
from its `ui.CheckState`, and checks it as it is pressed.

## Suggestions

`AutoComplete(kind)` tells screen readers what a text input's suggestions
complete as the user types, as ARIA's `aria-autocomplete`, so that they
say a field has them: `ui.AutoCompleteList` for a popup of the values
matching what was typed, `AutoCompleteInline` for the rest of the value
inserted after the caret, selected, and `AutoCompleteBoth` for both. A
[combobox](combobox.md), an [autocomplete](autocomplete.md) and a
[token field](token-field.md) with suggestions tell `AutoCompleteList`
themselves; a combobox of your own, with `Expanded` for its popup and
`ActiveDescendant` for the option the arrows are on, tells it too:

```go
in := ui.TextInputBase(c, &app.query).Role(ui.RoleComboBox).Label("Search")
in.Expanded(len(app.matches) > 0).AutoComplete(ui.AutoCompleteList)
```

It only tells: the view shows the suggestions itself. `AutoCompleteNone`,
the default, tells nothing, as for a combobox whose popup shows the same
options whatever is typed. It is AppKit's `AXAutocompleteValue` on macOS,
as WebKit gives it, ATK's `autocomplete` object attribute with the
supports-autocompletion state on Linux, as Core-AAM maps it, and
`autocomplete=list` in UI Automation's `AriaProperties` on Windows, the
place of ARIA's properties UI Automation has none of. A
[test](testing.md#what-assistive-technology-sees) reads it in the
`AutoComplete` of `Tester.AccessByRole`'s answer, and the
[automation plugin](../plugins/automation.md) in a node's
`autoComplete`.

## Descriptions and errors

`Description` tells more than the name, as help text read after it, as a
[tooltip](tooltip.md) does, and `Error` marks a value as invalid, for a
message read with it; a [field](form.md) gives its control the texts below
it.

A hint or a help text the view draws as an element of its own describes a
control with `DescribedBy`, as ARIA's `aria-describedby`: screen readers
read its text after the control's name and value, after the control's
`Description`, and a sentence with a link in it reads as one text. It
counts even when hidden from assistive technology, so the hint can be left
out of the reading order and still be read with the field:

```go
hint := ui.Text(c, "16 digits, no spaces").TextColor(t.TextMuted)
ui.TextInput(c, &app.card).Label("Card number").DescribedBy(hint)
```

`Invalid` marks a value as invalid without a message, as `aria-invalid`
alone: a required field marked as the user leaves it empty, whose message
waits for the form to be submitted. A text input then draws its border in
the theme's `Danger` color, and its description stays as it is.
`ErrorMessage` points at the element holding the message, as
`aria-errormessage`: while the control is invalid, its text is read first,
before the description, and screen readers that follow the relation go to
it; while the control is valid, it says nothing:

```go
in := ui.TextInput(c, &app.email).Label("Email").Invalid(app.emailTouched && app.email == "")
if app.submitted && app.email == "" {
	in.ErrorMessage(ui.Text(c, "Enter your email").TextColor(t.Danger))
}
```

The description is AppKit's `AXHelp`, ATK's description with the
`described-by` relation (and `description-for` on the hint), and UI
Automation's `FullDescription` with `DescribedBy`. The message is WebKit's
`AXErrorMessageElements`, ATK's `error-message` relation (and `error-for`
on the message), and UI Automation's `ControllerFor`, as Core-AAM maps
them; invalid is `AXInvalid`, ATK's `invalid-entry` state and UI
Automation's `IsDataValidForForm`.

## Orientation

`Orientation` tells which way an element of your own runs, as
`aria-orientation`: a slider, separator, splitter, toolbar, tab list or
menu drawn up and down, whose keys go Up and Down, is `ui.Vertical`, which
screen readers read with it. It only tells it: the view lays the element
out and takes its keys ([`FocusGroup`](input.md#focus-groups) moves the focus
with the arrows). A slider or a separator told nothing runs across, a
separator taller than wide up and down, and a
[`SliderBase`](slider.md) going up (`Vertical`) says so itself:

```go
ui.Box(c).Size(6, 160).Draw(drawTrack).Focusable().
	Role(ui.RoleSlider).Label("Volume").Range(0, 100, app.volume).
	Orientation(ui.Vertical)
ui.Column(c).Role(ui.RoleToolbar).Label("Tools").Orientation(ui.Vertical).Children(tools)
```

It is AppKit's `AXOrientation`, ATK's vertical or horizontal state, and UI
Automation's `Orientation`.

## Keyboard shortcuts

`KeyShortcuts(mods, key)` tells screen readers a key that presses or
focuses the element, as ARIA's `aria-keyshortcuts`: a toolbar button
whose command the window's shortcut runs, or the button opening a command
palette, is read with its key after its name. It only tells the key,
which the view handles itself, as with
[`c.Shortcut`](input.md#shortcuts); call it once for each key that does:

```go
if c.Shortcut(ui.Cmd, ui.KeyK) {
	app.palette = true
}
ui.Toolbar(c, func() {
	if ui.Button(c, "Search").KeyShortcuts(ui.Cmd, ui.KeyK).Clicked() {
		app.palette = true
	}
	ui.Button(c, "Redo").KeyShortcuts(ui.Cmd|ui.Shift, ui.KeyZ).KeyShortcuts(ui.Cmd, ui.KeyY)
})
```

The keys are written as the platform's menus write them, separated by
spaces: `⌘K` and `⇧⌘Z ⌘Y` on macOS, in AppKit's `AXKeyShortcutsValue` as
WebKit gives it; `Ctrl+K` and `Ctrl+Shift+Z Ctrl+Y` on Linux, in ATK's
`keyshortcuts` object attribute, and on Windows, in UI Automation's
`AcceleratorKey`, where Super is `Win`. A
[test](testing.md#what-assistive-technology-sees) reads them in the
`KeyShortcuts` of `Tester.AccessByRole`'s answer, and the
[automation plugin](../plugins/automation.md) in a node's `keyShortcuts`.

## Formats of text

The [rich text editor](rich-text-editor.md) tells screen readers the formats
of its runs as they read them: bold, italic, underlined, struck through,
links, fonts, colors and headings. VoiceOver reads them through
`AXAttributedStringForRange`, Orca through ATK's run attributes, and
Narrator through UI Automation's Text pattern, which every text field has,
by characters, formats, words and lines. Screen readers move the selection
of text fields too.

## Lists and tables

A [list](list.md)'s rows are list items, and a [table](table.md)'s rows,
named by the text inside them, and each says which of all the rows it is,
"5 of 10,000", although the list builds only those in view. A list choosing
its rows gives assistive technology the focus on the row chosen: Up and
Down are read as they move the choice, and focusing a row chooses it. A
screen reader moving out of view asks the list to scroll there, which
builds the rows it reaches. `Label` names a list or a table. A table of
your own, or one whose cells take the arrow keys, is a
[grid](#grids).

## Place in a set

Screen readers read an item with its place, "3 of 50", which they count
from the items around it. A list, list box, tree, menu or tab list of
your own whose items are not all built, as one showing only those in
view, or whose items are not siblings, as a tree laid out flat in one
column, tells each item's place with `PositionInSet(pos, size)`, as
ARIA's `aria-posinset` and `aria-setsize`: `pos` from 1, `size` how many
the set has, built or not. Either at 0 or less is not given.

```go
ui.Column(c).Role(ui.RoleListBox).Label("Results").Children(func() {
	for i := app.first; i < app.last; i++ { // only those in view
		ui.Row(c).Role(ui.RoleOption).PositionInSet(i+1, len(app.results)).
			Children(func() { ui.Text(c, app.results[i]) })
	}
})

ui.Column(c).Role(ui.RoleTree).Label("Files").Children(func() {
	for _, f := range app.flat { // each item at its depth, one below the other
		ui.Row(c).Role(ui.RoleTreeItem).Label(f.Name).Level(f.Depth+1).PositionInSet(f.Index+1, f.Siblings)
	}
})
```

The place the app gives wins over the one a `ComboboxBase`,
`SelectBase`, `MenuBase`, `Tree` or `GridView` gives its items, which
count those they build, as for a combobox showing the first 50 of 1,000
results; a [List](list.md)'s and a [Table](table.md)'s rows keep theirs,
which already say which of all their rows they are. The rows and cells
of a [grid](#grids) of your own say their place with `RowIndex` and
`ColumnIndex`. It is ATK's `posinset` and `setsize` attributes on Linux,
UI Automation's `PositionInSet` and `SizeOfSet` on Windows, and AppKit's
`AXIndex` on macOS, where a list's `AXRowCount` is the size its items
give when it has none of its own. A
[test](testing.md#what-assistive-technology-sees) reads them as
`PosInSet` and `SetSize`.

## Selected, without the look

`Selected(true)` shows an element as chosen in the accent color, with
the accent's text color, and tells assistive technology it is selected.
A tab, an option, a row, a cell or an item of a tree that shows its
choice its own way, underlined, bold or with a check mark, tells only
the state with `AccessSelected(on)`, ARIA's `aria-selected`:

```go
ui.Row(c).Role(ui.RoleTabList).Label("Views").Children(func() {
	for i, name := range views {
		tab := ui.ButtonBase(c).Role(ui.RoleTab).Label(name).AccessSelected(i == app.view)
		if tab.Clicked() {
			app.view = i
		}
		tab.Children(func() {
			label := ui.Text(c, name)
			if i == app.view {
				label.Bold()
			}
		})
	}
})
```

It sets the state `Selected` sets, which `Checked` also sets on these
roles, so it reaches every platform as `Selected` does: a row or a cell
saying whether it is selected, `false` or `true`, is one its table or
grid chooses among others, the table or grid holding the choice. It
only tells: the view shows the choice and changes it as the element is
clicked. A [test](testing.md#what-assistive-technology-sees) reads it
as `Selected`.

## Grids

A grid is a table whose cells take the focus and move it with the arrow
keys, as a spreadsheet's or a data grid's, where `ui.RoleTable` is one to
read; `ui.RoleTreeGrid` is a grid whose rows open and close as a tree's
items do, with `Level` and `Expanded` on its rows. Their rows, cells and
headers are `ui.RoleRow`, `ui.RoleCell`, `ui.RoleColumnHeader` and
`ui.RoleRowHeader`, as a table's: build the rows directly inside the
grid (an element without a role between them is left out), and the
cells directly inside the rows.

One of thousands of rows builds only those in view, so it tells screen
readers how many it has, `RowCount` and `ColumnCount` (ARIA's
`aria-rowcount` and `aria-colcount`, header rows counted), and each row
which of all it is, `RowIndex` from 1 (`aria-rowindex`); a cell of a
column not built, or after one left out, says its column with
`ColumnIndex` (`aria-colindex`), and the cells after it follow it.
Screen readers then read "row 4,121 of 10,000". A cell covering several
columns or rows, as a total under two columns, says so with
`ColumnSpan` and `RowSpan` (`aria-colspan`, `aria-rowspan`), which also
lay it out across them where its row is a [Grid](grid.md): the cell after
it in its row is in the column after those it covers, and the cells of
the rows below one spanning rows say their columns with `ColumnIndex`
where they follow it. `Multiselectable(true)`
says that the user chooses several rows (`aria-multiselectable`), and
each row says whether it is chosen with `Selected`, or `Checked`, which
does not paint it:

```go
ui.Column(c).Role(ui.RoleGrid).Label("Sheet").RowCount(len(app.rows)+1).ColumnCount(len(cols)).Multiselectable(true).Children(func() {
	ui.Row(c).Role(ui.RoleRow).RowIndex(1).Children(func() {
		for _, col := range cols {
			ui.Box(c).Role(ui.RoleColumnHeader).Children(func() { ui.Text(c, col.Title) })
		}
	})
	for _, r := range app.visibleRows() { // 4120 to 4160
		ui.Row(c).Role(ui.RoleRow).RowIndex(r + 2).Selected(app.chosen[r]).Children(func() {
			for j, col := range cols {
				ui.Box(c).Role(ui.RoleCell).Focusable().Children(func() { ui.Text(c, app.cell(r, j)) })
			}
		})
	}
})
```

A [List](list.md) shown as a grid builds only the rows in view itself:
`Role(ui.RoleGrid)` makes its rows rows rather than list items, which
already say which of all they are, as its `Selection` chooses several:

```go
ui.List(c, &app.rows, len(app.data), func(i int) {
	ui.Row(c).Children(func() {
		for j := range cols {
			ui.Box(c).Role(ui.RoleCell).Children(func() { ui.Text(c, app.cell(i, j)) })
		}
	})
}).Role(ui.RoleGrid).Label("Sheet").ColumnCount(len(cols))
```

A cell across two columns of a hand-built data grid, whose rows lay out
their cells in the grid's columns:

```go
ui.Grid(c).Role(ui.RoleRow).ColumnTracks(ui.Fixed(120), ui.Fr(1), ui.Fr(1), ui.Fr(1)).Children(func() {
	ui.Box(c).Role(ui.RoleRowHeader).Children(func() { ui.Text(c, "Design") })
	ui.Box(c).Role(ui.RoleCell).ColumnSpan(2).Children(func() { ui.Text(c, "First half: 9") })
	ui.Box(c).Role(ui.RoleCell).Children(func() { ui.Text(c, "Total: 14") }) // the fourth column
})
```

A [Table](table.md)'s rows say which of all they are themselves, and so
does a list's: the counts and indexes are for grids and tables of your
own. A grid is AppKit's `AXTable`, its rows and cells telling the counts
and indexes in `AXARIARowCount`, `AXARIAColumnCount`, `AXARIARowIndex`
and `AXARIAColumnIndex` as WebKit's do, and its rows `AXIndex` from them;
ATK's table (a tree grid's tree table) with the `xml-roles` attribute
`grid` (`treegrid`) and the `rowcount`, `colcount`, `rowindex` and
`colindex` attributes, as Chromium gives them; and UI Automation's
DataGrid, whose `Grid` and `Table` patterns, and its cells' `GridItem`
and `TableItem`, say where each cell is among all, from 0, and which
column and row headers name it, as tables' do. A cell's spans are the
lengths of its `AXColumnIndexRange` and `AXRowIndexRange` on macOS, its
`colspan` and `rowspan` attributes on Linux, and `GridItem`'s
`ColumnSpan` and `RowSpan` on Windows, where the grid finds the cell at
each place it covers. Cells say whether they
are chosen with [`Checked`](#states) too, the grid holding their
choice, as a table's do. A grid choosing several rows is
`multiselectable` on Linux and `CanSelectMultiple` on Windows, its rows
`selectable` and `selected` there, and in its `AXSelectedRows` on macOS.

## Unavailable, still reachable

`Disabled(true)` does three things: the element reads as disabled, it
leaves the focus and the Tab order and takes no clicks, and it shows at
half its opacity (unless `DimDisabled(false)`). A screen reader moving
by Tab then never reaches it. `AccessDisabled(true)` does the first alone,
as HTML's `aria-disabled="true"`, for what the user must still find while
it cannot be used, as the APG asks of a menu item whose action is
blocked, a tab disabled but still in the arrows' order, or a submit
button while the form is not valid: the element, and those inside it,
read as disabled (dimmed, unavailable), and keep the focus, their place
in the Tab order, the pointer and their look.

```go
del := ui.Button(c, "Delete project").AccessDisabled(app.locked)
if app.locked {
	del.Description("Unlock the project to delete it")
}
if del.Clicked() && !app.locked {
	app.deleteProject()
}
```

It only tells: clicks, Enter and Space still press the element, which
the view ignores while it is unavailable, as on the web, and it shows as
the view styles it. Assistive technology focuses it and reads it, its
`Description` with it, but neither presses nor edits it. It is AppKit's
`AXEnabled` false on macOS, ATK's enabled and sensitive states left out
on Linux, and UI Automation's `IsEnabled` false on Windows, with
`IsKeyboardFocusable` true and its `SetFocus` taken, as Chromium's
aria-disabled. A [test](testing.md#what-assistive-technology-sees)
reads it as `Disabled` with `Focusable` in `Tester.AccessByRole`'s
answer.

## Out of the pointer's way

`IgnoreInput(true)` leaves an element, and those inside it, to assistive
technology alone: the pointer goes through it to what is under it, Tab
and clicks give it no focus, but assistive technology still finds it, with
its role and name, and presses it, which `Clicked` reports. A button that
shows while its row is hovered can so stay built at rest, at `Opacity(0)`,
where a screen reader finds it and a click on the row is the row's:

```go
row := ui.Row(c).Padding(8)
hovered := row.Hovered()
if row.Clicked() {
	app.open(item)
}
row.Children(func() {
	ui.Text(c, item.Title).Grow(1)
	del := ui.Button(c, "Delete").IgnoreInput(!hovered)
	if !hovered {
		del.Opacity(0)
	}
	if del.Clicked() {
		app.remove(item)
	}
})
```

Assistive technology can neither focus it nor move a slider inside it, as
both go through the keyboard focus; it can set a text input's value.
Unlike HTML's `inert`, which hides an element from assistive technology
too, it stays in the tree: `Invisible` and `Hidden` take it out of it,
`AccessHidden` too while it still shows, and `Disabled` keeps it there
without its press, as `AccessDisabled` does with its focus too.

## Out of assistive technology's way

`AccessHidden(true)` leaves an element, and those inside it, out of what
assistive technology reads, while they still paint, as HTML's
`aria-hidden="true"` does: none of them is in its tree, their text does
not name the element around them, and it can neither press, focus nor edit
them. It is for what shows but is not there to read, or is read already:

- a page or a panel sliding out beside the one replacing it, while both
  show;
- the cards peeking from behind the front one of a stack, the slides
  beside the current one of a carousel;
- a copy of a text drawn twice, as a label fading into its next one.

With `IgnoreInput(true)`, it is HTML's `inert`: the pointer goes through
them, Tab passes them by, they lose the focus and their text cannot be
selected, as for the copy of an
[element leaving](transitions.md#coming-and-going) or a
[page going away](navigation.md#transitions):

```go
ui.Box(c).Fill().Children(func() {
	if app.leaving >= 0 { // the tab that was selected, sliding out
		page(c, app.leaving).Absolute().Translate(-x, 0).IgnoreInput(true).AccessHidden(true)
	}
	page(c, app.tab).Translate(width-x, 0)
})
```

Alone, it leaves them to the pointer and the keyboard, and their text to
selection, as `aria-hidden` does: keep the keyboard focus out of them, as
assistive technology cannot follow it there. `RoleNone` leaves out an
element alone, which suits a leaf, as a decorative icon.

## Announcements

News that does not move the focus, as a search done or a file saved,
reaches screen readers with `c.Announce(text)`, which they read once, after
what they are reading: VoiceOver's and Orca's announcements, and a live
region that Narrator and NVDA read on Windows. [Toasts](toast.md) announce
their text, and a [router](navigation.md) the title of a page shown while
the focus stays outside it.

```go
if ui.Button(c, "Search").Clicked() {
	app.results = search(app.query)
	c.Announce(fmt.Sprintf("%d results", len(app.results)))
}
```

MyGo describes frames only once assistive technology asked, so apps pay
nothing for it otherwise.

## Landmarks, live regions and rules

These roles are ARIA's of the same names, which reach each platform as
WAI-ARIA's Core Accessibility API Mappings say:

| Role | What it is | macOS | Linux (ATK) | Windows (UI Automation) |
|---|---|---|---|---|
| `ui.RoleRegion` | a landmark with a `Label`, as an activity feed or a panel; without one, a group | `AXGroup`, `AXLandmarkRegion` | landmark, `xml-roles:region` | Group, the custom landmark type "region" |
| `ui.RoleSearch` | a landmark holding what searches the app or a document, as the search field and its button at the top of a window | `AXGroup`, `AXLandmarkSearch` | landmark, `xml-roles:search` | Group, the search landmark type |
| `ui.RoleArticle` | a self-contained piece of content, as a post of a feed or a comment of a thread, named by its `Label`; no landmark | `AXGroup`, `AXDocumentArticle` | article | Group "article" |
| `ui.RoleLog` | a live region of messages added in order, as a chat's transcript | `AXGroup`, `AXApplicationLog` | log | Group "log", `LiveSetting` polite |
| `ui.RoleTimer` | a count of time, as a countdown, read only when the user goes to it | `AXGroup`, `AXApplicationTimer` | timer | Group "timer" |
| `ui.RoleAlert` | a live region of important news, as a field's error, read at once | `AXGroup`, `AXApplicationAlert` | notification | Group "alert", `LiveSetting` assertive |
| `ui.RoleSeparator` | a rule between groups of content or controls, running up and down when taller than wide | `AXSplitter` | separator, vertical or horizontal | Separator, with `Orientation` |
| `ui.RoleRowHeader` | the cell naming its row of a table, named by its content, whose buttons and other elements stay reachable, as a cell's | `AXCell` | row header | HeaderItem "row header" |
| `ui.RoleDeletion` | content removed from a document, as a diff's removed lines or an old price struck out, as HTML's `<del>`; its text is inside it | `AXGroup`, `AXDeleteStyleGroup` | content deletion, `xml-roles:deletion` | Text "deletion" |
| `ui.RoleInsertion` | content added to a document, as a diff's added lines, as HTML's `<ins>` | `AXGroup`, `AXInsertStyleGroup` | content insertion, `xml-roles:insertion` | Text "insertion" |

Live regions say how they are read (`AXARIALive`, ATK's `live` and
`container-live` attributes, `LiveSetting`), and MyGo reads them out as
the web does: once a log has shown, what is added to it is announced,
after what screen readers are reading, as `c.Announce` does; an alert's
text is read out at once, interrupting, as it appears and whenever it
changes (VoiceOver's announcements of high priority, ATK's `notification`
signal, assertive, where ATK has it (2.50), else an announcement, and an
assertive live region on Windows). What they show as assistive
technology comes, or as a log first shows, is not read out, and a timer
reads nothing by itself:

```go
ui.Column(c).Role(ui.RoleLog).Label("Messages").Children(func() {
	for _, m := range app.messages {
		ui.Row(c).Key(m.ID).Gap(4).Children(func() {
			ui.Text(c, m.From+":").Bold()
			ui.Text(c, m.Text)
		})
	}
})
if app.nameErr != "" {
	ui.Text(c, app.nameErr).TextColor(t.Danger).Role(ui.RoleAlert)
}
ui.Row(c).Role(ui.RoleToolbar).Label("Format").Children(func() {
	boldAndItalic(c)
	ui.Divider(c).Role(ui.RoleSeparator)
	alignment(c)
})
```

A log reads out the elements it gained, those of each row together; an
alert, its text. Keyed rows (`Key`) keep what a log showed apart from
what it gained when rows come before others.

Screen readers list landmarks and jump to them, so a search field and its
button at the top of a window go in a search landmark, as HTML's
`<search>`:

```go
ui.Row(c).Role(ui.RoleSearch).Label("Search mail").Gap(6).Children(func() {
	ui.SearchField(c, &app.query).Label("Search mail").Grow(1)
	if ui.Button(c, "Search").Clicked() {
		app.search()
	}
})
```

Each post of a feed, or comment of a thread, is an article, as HTML's
`<article>`, which screen readers step through one by one:

```go
for _, p := range app.posts {
	ui.Column(c).Key(p.ID).Role(ui.RoleArticle).Label(p.Title).Gap(4).Children(func() {
		ui.Text(c, p.Title).Bold().Role(ui.RoleHeading).Level(3)
		ui.Text(c, p.Body)
	})
}
```

Content removed and added, as the lines of a diff or an old price beside
the new one, says so, rather than being read as plain text:

```go
ui.Row(c).Gap(6).Children(func() {
	ui.Text(c, "$40").Strikethrough().TextColor(t.TextMuted).Role(ui.RoleDeletion)
	ui.Text(c, "$30").Role(ui.RoleInsertion)
})
```

Each component's page says what assistive technology sees of it.
