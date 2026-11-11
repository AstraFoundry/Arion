# Date input

<!-- preview date-input 320 -->

`ui.DateInput` creates a field showing a `*time.Time` as the user's
regional formats write a date in numbers, as 1/5/2026 in the United States,
05/01/2026 in the United Kingdom or 2026/1/5 in China, which a
[calendar](calendar.md) below it changes. A click, Enter or Space opens the
calendar, where a click chooses a day, as do the arrows and Enter; Page Up
and Page Down, or its buttons, move by months, and Escape closes it.
`Changed` reports a new date, which keeps the time of day and the location
of the value.

```go
ui.Field(c, "Birthday", func() {
	ui.DateInput(c, &app.birthday)
})
```

With a [time input](time-input.md) beside it, the two edit one `time.Time`:

```go
ui.Row(c).Gap(8).Children(func() {
	ui.DateInput(c, &app.meeting).Label("Date")
	ui.TimeInput(c, &app.meeting).Label("Time")
})
```

## Accessibility

Assistive technology sees a pop-up button whose value is the date, named by
its `Label` or its [field](form.md), and the calendar while it is open.
