# Decimal input

<!-- preview decimal-input 160 -->

`ui.DecimalInput` is a [number input](number-input.md) for amounts of
money: it edits a `*string` holding an exact decimal between two bounds,
which Up and Down, and the buttons beside it, change by a step. The value
is written with exactly the decimals of the step, as `"1234.50"`; the
bounds are written the same way, `""` for no limit. Inside, it counts in
`math/big` integers of units of the step's last decimal, so no amount is
ever held in binary floating point.

```go
ui.Field(c, "Amount", func() {
	if ui.DecimalInput(c, &app.amount, "0", "", "0.01").Changed() {
		app.save()
	}
})
```

What is typed applies as soon as it is a number in range, with `.` before
its decimals, `,` between groups of three digits, and an optional `-`:
`1,234.5` sets `"1234.50"`, and shows so once the input loses the focus.
Text and amounts in `*string` convert to whatever decimal type the app
uses (`shopspring/decimal`, `apd`, `big.Rat`), and fit a database's TEXT
columns.

## Nothing is rounded

At a step of `0.01`, `1.500` is `1.50`, but `1.005` is not a number the
input takes: rather than round it, it keeps the value as it was and marks
the input with `Error`, whose message a [field](form.md) around it shows
below, until the text is corrected. Text that is not a number, or out of
range, is marked the same way, and stays when the input loses the focus, so
that a typo is never silently replaced. Emptying the input leaves the value
as it was, and shows it again once the input loses the focus.

A value the app sets replaces what was typed, a mistake included. A value
the app sets with more decimals than the step's, or out of range, shows as
it is, marked. `Changed` reports a new value: one written another way, as
`1.5` for `1.50`, is not.

As `NumberInput`, it reads `.` and `,` the same way in every language the
window shows: `1,50` is not one and a half, but text that is not a number,
which the input marks rather than read as 150.

## Accessibility

Assistive technology sees a spin button of the value, named by its `Label`
or its field, which it increments and decrements by the step, and reads
the value as the input's text, exactly. An error's message describes the
input, which it reads as invalid.
