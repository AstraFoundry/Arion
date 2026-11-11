package ui

import (
	"math"
	"math/big"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// decimalTester shows a DecimalInput of *value in a field named Amount,
// and a text input named Note, which takes the focus from it.
func decimalTester(t *testing.T, value *string, lo, hi, step string) *Tester {
	t.Helper()
	note := ""
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(10).Gap(12).AlignItems(Start).Children(func() {
			coreField(c, "Amount", func() { coreDecimalInput(c, value, lo, hi, step) })
			coreField(c, "Note", func() { coreTextInput(c, &note) })
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	return tt
}

// amount returns the node of the input named Amount.
func amount(t *testing.T, tt *Tester) platform.AccessNode {
	t.Helper()
	return accessNode(t, tt.h.access, platform.RoleTextField, "Amount")
}

// retype focuses the input named Amount and replaces its text with s.
func retype(tt *Tester, s string) {
	tt.Click("Amount")
	tt.Key(Cmd, KeyA)
	if s == "" {
		tt.Key(0, KeyBackspace)
		return
	}
	tt.Type(s)
}

func TestDecimalInputExact(t *testing.T) {
	// A float64 holds 90071992547409.94 here.
	v := "0.00"
	tt := decimalTester(t, &v, "0", "1000000000000000", "0.01")
	retype(tt, "90071992547409.93")
	if v != "90071992547409.93" {
		t.Fatalf("typed 90071992547409.93: %q", v)
	}
	tt.Key(0, KeyUp)
	if v != "90071992547409.94" {
		t.Errorf("Up: %q", v)
	}
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	if v != "90071992547409.92" {
		t.Errorf("Down twice: %q", v)
	}
	// Assistive technology reads the value as its text, exactly.
	if n := amount(t, tt); n.Value != "90071992547409.92" || n.Step != 0.01 || n.Min != 0 || n.Max != 1e15 {
		t.Errorf("the input: %+v", n)
	}
}

func TestDecimalInputDecimals(t *testing.T) {
	v := "1.00"
	tt := decimalTester(t, &v, "", "", "0.01")
	// Zeros past the step's decimals round nothing.
	retype(tt, "1.500")
	if v != "1.50" {
		t.Fatalf("typed 1.500: %q", v)
	}
	if n := amount(t, tt); n.Value != "1.500" || n.States&platform.AccessInvalid != 0 {
		t.Errorf("as typed while focused: %+v", n)
	}
	tt.Click("Note")
	if n := amount(t, tt); n.Value != "1.50" {
		t.Errorf("once the focus went: %q", n.Value)
	}
	// Digits grouped by commas, and a sign.
	for _, c := range []struct{ typed, want string }{
		{"1,234.5", "1234.50"},
		{"-12,345,678", "-12345678.00"},
		{"+.5", "0.50"},
		{" 7 ", "7.00"},
	} {
		retype(tt, c.typed)
		if v != c.want {
			t.Errorf("typed %q: %q", c.typed, v)
		}
	}
	// A third decimal would round: the value stays, and the input says
	// why, below it too, until it is corrected.
	retype(tt, "1.005")
	const msg = "Use at most 2 decimal places"
	if v != "7.00" {
		t.Errorf("1.005 changed the value to %q", v)
	}
	if n := amount(t, tt); n.States&platform.AccessInvalid == 0 || n.Description != msg || !tt.HasText(msg) {
		t.Errorf("1.005 is not marked: %+v, %q", n, tt.Texts())
	}
	tt.Click("Note")
	if n := amount(t, tt); n.Value != "1.005" || n.States&platform.AccessInvalid == 0 || !tt.HasText(msg) || v != "7.00" {
		t.Errorf("once the focus went: %+v, %q", n, v)
	}
	retype(tt, "1.01")
	if n := amount(t, tt); v != "1.01" || n.States&platform.AccessInvalid != 0 || tt.HasText(msg) {
		t.Errorf("corrected: %q, %+v", v, n)
	}
	// What is not a number.
	for _, s := range []string{"abc", "1,2", "1.2.3", "1e3", "-", "12,34.5", "1 000"} {
		retype(tt, s)
		if n := amount(t, tt); v != "1.01" || n.States&platform.AccessInvalid == 0 || n.Description != "Enter a number" {
			t.Errorf("typed %q: %q, %+v", s, v, n)
		}
	}
	// Nothing typed leaves the value, shown again once the focus goes.
	retype(tt, "")
	if n := amount(t, tt); v != "1.01" || n.States&platform.AccessInvalid != 0 {
		t.Errorf("emptied: %q, %+v", v, n)
	}
	tt.Click("Note")
	if n := amount(t, tt); n.Value != "1.01" {
		t.Errorf("emptied, then the focus went: %q", n.Value)
	}
}

func TestDecimalInputWholeNumbers(t *testing.T) {
	v := "3"
	tt := decimalTester(t, &v, "1", "99", "1")
	retype(tt, "4.0")
	if v != "4" {
		t.Errorf("typed 4.0: %q", v)
	}
	retype(tt, "4.5")
	if n := amount(t, tt); v != "4" || n.Description != "Enter a whole number" {
		t.Errorf("typed 4.5: %q, %+v", v, n)
	}
}

func TestDecimalInputBounds(t *testing.T) {
	v := "50.00"
	tt := decimalTester(t, &v, "0", "100", "0.01")
	for _, c := range []struct{ typed, msg string }{
		{"100.01", "Enter 100.00 or less"},
		{"-0.01", "Enter 0.00 or more"},
	} {
		retype(tt, c.typed)
		if n := amount(t, tt); v != "50.00" || n.States&platform.AccessInvalid == 0 || n.Description != c.msg || !tt.HasText(c.msg) {
			t.Errorf("typed %q: %q, %+v", c.typed, v, n)
		}
	}
	retype(tt, "100")
	if v != "100.00" {
		t.Errorf("typed 100, the top: %q", v)
	}
	retype(tt, "0")
	if v != "0.00" {
		t.Errorf("typed 0, the bottom: %q", v)
	}

	// "" is no limit, which assistive technology reads as infinite.
	w := "0"
	tt = decimalTester(t, &w, "", "", "1")
	retype(tt, "-123456789012345678901234567890")
	if w != "-123456789012345678901234567890" {
		t.Errorf("no limits: %q", w)
	}
	if n := amount(t, tt); !math.IsInf(n.Min, -1) || !math.IsInf(n.Max, 1) || n.Step != 1 {
		t.Errorf("no limits: %+v", n)
	}
	// One limit.
	x := "5.0"
	tt = decimalTester(t, &x, "", "10.0", "0.5")
	retype(tt, "10.5")
	if n := amount(t, tt); x != "5.0" || n.Description != "Enter 10.0 or less" {
		t.Errorf("above the top: %q, %+v", x, n)
	}
	retype(tt, "-1000000000.5")
	if x != "-1000000000.5" {
		t.Errorf("no bottom: %q", x)
	}
}

func TestDecimalInputSteps(t *testing.T) {
	v := "0.75"
	tt := decimalTester(t, &v, "0", "1", "0.25")
	tt.Click("Amount")
	tt.Key(0, KeyUp)
	if v != "1.00" {
		t.Errorf("Up from 0.75: %q", v)
	}
	tt.Key(0, KeyUp)
	if v != "1.00" {
		t.Errorf("Up past the top: %q", v)
	}
	// At the top, the button stepping up is disabled.
	if n := accessNode(t, tt.h.access, platform.RoleButton, "Increase"); n.States&platform.AccessDisabled == 0 {
		t.Errorf("Increase at the top: %+v", n)
	}
	for range 5 {
		tt.Key(0, KeyDown)
	}
	if v != "0.00" {
		t.Errorf("Down past the bottom: %q", v)
	}
	if n := amount(t, tt); n.Value != "0.00" || n.Now != 0 {
		t.Errorf("at the bottom: %+v", n)
	}
	if err := tt.Click("Increase"); err != nil {
		t.Fatal(err)
	}
	tt.Click("Increase")
	tt.Click("Decrease")
	if v != "0.25" {
		t.Errorf("two steps up and one down from 0: %q", v)
	}
	// A step replaces what is wrong with the value stepped.
	retype(tt, "0.333")
	tt.Key(0, KeyUp)
	if n := amount(t, tt); v != "0.50" || n.Value != "0.50" || n.States&platform.AccessInvalid != 0 {
		t.Errorf("Up from 0.333 mistyped: %q, %+v", v, n)
	}
	// Steps from nothing start at 0.
	e := ""
	tt = decimalTester(t, &e, "", "", "0.1")
	tt.Click("Increase")
	if e != "0.1" {
		t.Errorf("Increase from nothing: %q", e)
	}
}

func TestDecimalInputAppSets(t *testing.T) {
	v := "1.00"
	tt := decimalTester(t, &v, "0", "", "0.01")
	retype(tt, "1.005")
	// The app sets the value while the mistyped text has the focus.
	v = "2.5"
	tt.Frame()
	if n := amount(t, tt); n.Value != "2.50" || n.States&platform.AccessInvalid != 0 || tt.HasText("decimal places") {
		t.Errorf("the app set 2.5: %+v", n)
	}
	if v != "2.5" {
		t.Errorf("showing the app's value changed it: %q", v)
	}
	// What the app sets is never rounded either.
	v = "3.141"
	tt.Frame()
	if n := amount(t, tt); n.Value != "3.141" || n.States&platform.AccessInvalid == 0 {
		t.Errorf("the app set 3.141: %+v", n)
	}
	v = "-1.00"
	tt.Frame()
	if n := amount(t, tt); n.Value != "-1.00" || n.Description != "Enter 0.00 or more" {
		t.Errorf("the app set -1.00: %+v", n)
	}
	v = ""
	tt.Frame()
	if n := amount(t, tt); n.Value != "" || n.States&platform.AccessInvalid != 0 {
		t.Errorf("the app set nothing: %+v", n)
	}
}

func TestDecimalInputChanged(t *testing.T) {
	committed, changes := "1.00", 0
	tt := NewTester(func(c *Context) {
		value := committed
		Field(c, "Amount", func() {
			if DecimalInput(c, &value, "0", "10", "0.01").Changed() {
				committed = value
				changes++
			}
		})
	}, 300, 150)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	retype(tt, "2.5")
	if committed != "2.50" || changes != 1 {
		t.Fatalf("typed 2.5: %q, %d changes", committed, changes)
	}
	tt.Frame()
	if changes != 1 {
		t.Fatal("a change reported again on a rebuild")
	}
	// The same value written another way, and a value rejected, change
	// nothing.
	retype(tt, "2.500")
	retype(tt, "20")
	if committed != "2.50" || changes != 1 {
		t.Errorf("2.500 and 20: %q, %d changes", committed, changes)
	}
	tt.Key(0, KeyUp)
	if committed != "2.51" || changes != 2 {
		t.Errorf("Up: %q, %d changes", committed, changes)
	}
	tt.Click("Increase")
	if committed != "2.52" || changes != 3 {
		t.Errorf("Increase: %q, %d changes", committed, changes)
	}
}

func TestDecimalInputLimits(t *testing.T) {
	for _, c := range []struct{ lo, hi, step string }{
		{"0", "", "0"},
		{"0", "", "-0.01"},
		{"0", "", "a"},
		{"0.001", "", "0.01"},
		{"", "x", "1"},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%q, %q, %q did not panic", c.lo, c.hi, c.step)
				}
			}()
			v := ""
			coreNewTester(func(ctx *context) { coreDecimalInput(ctx, &v, c.lo, c.hi, c.step) }, 200, 100)
		}()
	}
}

func TestParseDecimal(t *testing.T) {
	for _, c := range []struct {
		s     string
		scale int
		want  string // the units, "" for a fault
		fault decimalFault
	}{
		{"0", 2, "0", decimalFine},
		{"1.", 2, "100", decimalFine},
		{".5", 2, "50", decimalFine},
		{"-0.00", 2, "0", decimalFine},
		{"123,456.7", 1, "1234567", decimalFine},
		{"1.230000", 2, "123", decimalFine},
		{"1.231", 2, "", decimalTooExact},
		{"1.5", 0, "", decimalTooExact},
		{"", 2, "", decimalInvalid},
		{".", 2, "", decimalInvalid},
		{"-.", 2, "", decimalInvalid},
		{",123", 0, "", decimalInvalid},
		{"1234,567", 0, "", decimalInvalid},
		{"1,234,", 0, "", decimalInvalid},
		{"1.2,3", 2, "", decimalInvalid},
		{"--1", 0, "", decimalInvalid},
		{"١٢", 0, "", decimalInvalid},
	} {
		v, f := parseDecimal(c.s, c.scale)
		if f != c.fault || f == decimalFine && v.String() != c.want {
			t.Errorf("parseDecimal(%q, %d) = %v, %d; want %s, %d", c.s, c.scale, v, f, c.want, c.fault)
		}
	}
	for _, c := range []struct {
		v     int64
		scale int
		want  string
	}{{0, 2, "0.00"}, {5, 2, "0.05"}, {-5, 2, "-0.05"}, {-150, 2, "-1.50"}, {42, 0, "42"}, {123456, 3, "123.456"}} {
		if s := formatDecimal(big.NewInt(c.v), c.scale); s != c.want {
			t.Errorf("formatDecimal(%d, %d) = %q, want %q", c.v, c.scale, s, c.want)
		}
	}
}

func TestDecimalInputOutsideField(t *testing.T) {
	v := "1.00"
	tt := coreNewTester(func(c *context) {
		coreDecimalInput(c, &v, "0", "", "0.01").Label("Price")
	}, 300, 100)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	r, _ := tt.Find("Price")
	tt.ClickAt(r.X+20, r.Y+r.H/2)
	tt.Key(Cmd, KeyA)
	tt.Type("1.001")
	// Marked, with nowhere to show the message but to assistive
	// technology.
	if n := accessNode(t, tt.h.access, platform.RoleTextField, ""); v != "1.00" || n.States&platform.AccessInvalid == 0 || n.Description != "Use at most 2 decimal places" || tt.HasText("decimal places") {
		t.Errorf("typed 1.001: %q, %+v", v, n)
	}
}
