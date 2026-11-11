# Calendar

<!-- preview calendar 340 -->

`ui.Calendar` creates a month's calendar choosing the day of a
`*time.Time`, as SwiftUI's graphical date picker: a click chooses a day, as
the arrows do while the calendar has the keyboard focus; Page Up and Page
Down, or its buttons, move by months, and Home and End go to the first and
the last day of the month. `Changed` reports a new date, which keeps the
time of day and the location of the value.

```go
ui.Calendar(c, &app.day).Label("Due date")
```

It shows six weeks from the first day of the week of the user's regional
formats (`Context.Region`: Sunday in the United States, Monday in most of
Europe), the days of the months around in gray, and marks today. Its title
("January 2026", "2026年1月"), the names of the weekdays and the days' names
for assistive technology are CLDR's in the window's language, its days in
the language's digits; the buttons' names are
[MyGo's own strings](../i18n.md#mygos-own-strings).

For a date in a compact field, use a [date input](date-input.md), which
opens this calendar below it.

## Browsing without choosing

`Browse` lets the user look about the calendar without choosing a day:
its buttons and Page Up and Page Down only show other months, and the
arrows, Home and End only move among the days, until a click, Enter or
Space chooses one, as in a date input's calendar. `Changed` then reports
only a day the user picked, as a calendar in a [popover](popover.md) that
closes when a day is picked needs:

```go
chip := ui.Button(c, task.Due.Format("Jan 2"))
if chip.Clicked() {
	app.dueOpen = !app.dueOpen
}
ui.Popover(c, chip, &app.dueOpen, func() {
	if ui.Calendar(c, &task.Due).Browse().Changed() {
		app.dueOpen = false
	}
})
```

The calendar goes back to the value's month as the app sets another date.

## Accessibility

Assistive technology sees a table named by its month, whose days are
buttons named by their dates, the day chosen checked, and reads the day
chosen as the arrows move; with `Browse`, the day the arrows are on.
