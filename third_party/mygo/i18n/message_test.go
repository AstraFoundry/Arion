package i18n

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"testing"
	"time"
)

// messageFixtures are the cases of testdata/messages.json, which
// mygo-runtime's formatter checks too.
type messageFixtures struct {
	Format []messageCase
	Errors []string
	Pseudo []messageCase
}

type messageCase struct {
	Message string
	Locale  string
	Args    map[string]any
	Want    string
}

func readMessageFixtures(t *testing.T) messageFixtures {
	t.Helper()
	data, err := os.ReadFile("testdata/messages.json")
	if err != nil {
		t.Fatal(err)
	}
	var f messageFixtures
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestMessageFixtures(t *testing.T) {
	f := readMessageFixtures(t)
	if len(f.Format) < 50 || len(f.Errors) < 20 || len(f.Pseudo) < 4 {
		t.Fatalf("the fixtures lost cases: %d, %d, %d", len(f.Format), len(f.Errors), len(f.Pseudo))
	}
	for _, c := range append(f.Format, f.Pseudo...) {
		m, err := ParseMessage(c.Message)
		if err != nil {
			t.Errorf("ParseMessage(%q): %v", c.Message, err)
			continue
		}
		locale := c.Locale
		if locale == "" {
			locale = "en"
		}
		if got := m.Format(locale, c.Args); got != c.Want {
			t.Errorf("%q in %s with %v = %q, want %q", c.Message, locale, c.Args, got, c.Want)
		}
		if m.String() != c.Message {
			t.Errorf("String() = %q, want %q", m.String(), c.Message)
		}
	}
	for _, s := range f.Errors {
		_, err := ParseMessage(s)
		var syntax *SyntaxError
		if !errors.As(err, &syntax) {
			t.Errorf("ParseMessage(%q) = %v, want a SyntaxError", s, err)
		}
	}
}

func TestMessageArgs(t *testing.T) {
	m := MustParseMessage("{host} and {n, plural, offset:1 one {# {g, select, a {{host}} other {{when, date, short}}}} other {{n, number}}} {x, selectordinal, other {#}} {t, time} {p, number, percent}")
	want := []Arg{{"host", ""}, {"n", "plural"}, {"g", "select"}, {"when", "date"}, {"x", "selectordinal"}, {"t", "time"}, {"p", "number"}}
	if got := m.Args(); !slices.Equal(got, want) {
		t.Errorf("Args() = %v, want %v", got, want)
	}
	if got := m.Cases("n"); !slices.Equal(got, []string{"one", "other"}) {
		t.Errorf("Cases(n) = %v", got)
	}
	if got := m.Cases("g"); !slices.Equal(got, []string{"a", "other"}) {
		t.Errorf("Cases(g) = %v", got)
	}
	m = MustParseMessage("{n, plural, =0 {none} one {#} other {{n, plural, few {#} other {#}}}}")
	if got := m.Cases("n"); !slices.Equal(got, []string{"=0", "one", "other", "few"}) {
		t.Errorf("Cases(n) = %v", got)
	}
}

func TestMessageValues(t *testing.T) {
	day := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)
	for _, c := range []struct {
		msg  string
		args map[string]any
		want string
	}{
		{"{d, date}", map[string]any{"d": day}, "Jan 2, 2026"},
		{"{d, date, short}", map[string]any{"d": day}, "1/2/26"},
		{"{d, date, full}", map[string]any{"d": day}, "Friday, January 2, 2026"},
		{"{d, time, short}", map[string]any{"d": day}, "3:04 PM"},
		{"{d, time}", map[string]any{"d": day}, "3:04:05 PM"},
		{"{d}", map[string]any{"d": day}, "1/2/26, 3:04 PM"},
		{"{d, date, ::yMMMM}", map[string]any{"d": day}, "January 2026"},
		{"{d, date, ::MMMd}", map[string]any{"d": day}, "Jan 2"},
		{"{d, date}", map[string]any{"d": "tomorrow"}, "tomorrow"},
		{"{e}", map[string]any{"e": errors.New("boom")}, "boom"},
		{"{v}", map[string]any{"v": time.Second}, "1s"},
		{"{v}", map[string]any{"v": nil}, ""},
		{"{v}", map[string]any{"v": []int{1}}, "[1]"},
		{"{n}", map[string]any{"n": uint8(7)}, "7"},
		{"{n, plural, one {#} other {# x}}", map[string]any{"n": int64(-2)}, "-2 x"},
		{"{n, plural, one {#} other {# x}}", map[string]any{"n": float32(1.5)}, "1.5 x"},
		{"{n, plural, one {#} other {# x}}", map[string]any{"n": Operands{I: 1, V: 2}}, "1.00 x"},
		{"{n, plural, one {#} other {# x}}", map[string]any{"n": Operands{I: 1}}, "1"},
		{"{n, plural, one {#} other {# x}}", map[string]any{"n": uint64(1)}, "1"},
		{"{s, select, a {A} other {O}}", map[string]any{"s": stringer("a")}, "A"},
		{"{s, select, 2 {A} other {O}}", map[string]any{"s": 2.0}, "A"},
	} {
		if got := MustParseMessage(c.msg).Format("en", c.args); got != c.want {
			t.Errorf("%q with %v = %q, want %q", c.msg, c.args, got, c.want)
		}
	}
}

type stringer string

func (s stringer) String() string { return string(s) }

func TestMustParseMessagePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("MustParseMessage of a bad message did not panic")
		}
	}()
	MustParseMessage("{")
}

func TestMessageNesting(t *testing.T) {
	deep := ""
	for range 40 {
		deep = "{a, select, other {" + deep + "}}"
	}
	if _, err := ParseMessage(deep); err == nil {
		t.Error("a message nested 40 deep parsed")
	}
}

func FuzzParseMessage(f *testing.F) {
	for _, s := range []string{"Hello {name}", "{n, plural, one {#} other {# {g, select, a {x} other {y}}}}", "'{'", "{n, number, ::currency/EUR}"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		m, err := ParseMessage(s)
		if err != nil {
			return
		}
		_ = m.Format("ar", map[string]any{"n": 3, "g": "a", "name": "x"})
		_ = m.Format("en-XA", nil)
		_ = m.Args()
	})
}
