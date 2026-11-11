# Time input

<!-- preview time-input 120 -->

`ui.TimeInput` creates a field of the time of day of a `*time.Time`, in the
hour cycle of the user's regional formats (`Context.Region`): 15:04, or
3:04 PM, whose hours, minutes and half of the day each take the keyboard
focus, as the segments of AppKit's date picker: Up and Down step the one
with the focus round the clock, digits typed set it, going on to the
minutes once the hours are typed, the first letter of AM or PM (or of the
language's names of them) chooses the half of the day, and Left and Right
move between them. The names of the halves of the day and their place are
the window's language's: 下午3:04 in Chinese, with the half first. `Changed` reports a
new time, which keeps the date and the location of the value.

```go
ui.TimeInput(c, &app.alarm).Label("Alarm")
```

A click on the hours or the minutes gives them the focus. With a
[date input](date-input.md), the two edit one `time.Time`.

## Accessibility

Assistive technology sees spin buttons, named after the field's `Label`
and "hours", "minutes" and "AM/PM", in the window's language.
