# Alert dialog

<!-- preview alert-dialog 240 -->

`ui.AlertDialog` asks about something important over the window while a
`*bool` is true, as AppKit's alerts: a title, a message, and buttons, the
last of which is the default, in the accent color, with the focus, which
Enter clicks. Escape closes the alert choosing no button, as canceling does,
whatever the buttons say, in any language; a click outside the alert does
nothing. It returns the index of the button clicked, in the frame it is,
which closes the alert, `ui.AlertDismissed` in the frame Escape closes it,
and -1 otherwise.

```go
if ui.Button(c, "Delete").Clicked() {
	app.asking = true
}
switch ui.AlertDialog(c, &app.asking, "Delete “Notes”?", "You can't undo this.", "Cancel", "Delete") {
case 1:
	app.delete()
}
```

An alert that must be answered stays open by setting the `*bool` back to
true as Escape dismisses it:

```go
if ui.AlertDialog(c, &app.updating, "Update required", "Restart to finish updating.", "Restart") == ui.AlertDismissed {
	app.updating = true
}
```

For a dialog of your own content, use a [dialog](dialog.md); for a choice
the user can undo, act at once and offer it in a [toast](toast.md).

## Accessibility

Assistive technology sees an alert dialog named by its title and described
by its message, with the focus on its default button.
