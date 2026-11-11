package terminal

import (
	"fmt"
	"strings"
	"testing"
)

// A server's terminal has no View, but applications must still be able
// to query the colors configured for it.
func TestHeadlessThemeQueries(t *testing.T) {
	loadLib(t)
	for _, tc := range []struct {
		name  string
		theme *Theme
		mode  int
	}{
		{"light", LightTheme(), 2},
		{"dark", DarkTheme(), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newPipe()
			term, err := New(Options{Conn: p, Theme: tc.theme})
			if err != nil {
				t.Fatal(err)
			}
			defer term.Close()
			term.Feed([]byte("\x1b]10;?\x1b\\\x1b]11;?\x1b\\\x1b[?996n"))
			got := p.take(term)
			for _, color := range []struct {
				index   int
				r, g, b uint8
			}{
				{10, tc.theme.Foreground.R, tc.theme.Foreground.G, tc.theme.Foreground.B},
				{11, tc.theme.Background.R, tc.theme.Background.G, tc.theme.Background.B},
			} {
				want := fmt.Sprintf("\x1b]%d;rgb:%04x/%04x/%04x\x1b\\", color.index,
					uint16(color.r)*257, uint16(color.g)*257, uint16(color.b)*257)
				if !strings.Contains(got, want) {
					t.Errorf("query OSC %d: got %q, want %q", color.index, got, want)
				}
			}
			if want := fmt.Sprintf("\x1b[?997;%dn", tc.mode); !strings.Contains(got, want) {
				t.Errorf("scheme query: got %q, want %q", got, want)
			}
		})
	}
}

func TestHeadlessAppearanceChanges(t *testing.T) {
	loadLib(t)
	p := newPipe()
	term, err := New(Options{Conn: p, Theme: LightTheme()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()
	// Applications that did not subscribe must receive no unsolicited
	// input. Once subscribed, each preference change produces one report.
	term.UseTheme(DarkTheme())
	if got := p.take(term); got != "" {
		t.Fatalf("unsubscribed application received %q", got)
	}
	term.Feed([]byte("\x1b[?2031h"))
	p.take(term)
	term.UseTheme(LightTheme())
	if got, want := p.take(term), "\x1b[?997;2n"; got != want {
		t.Fatalf("light change = %q, want %q", got, want)
	}
	term.UseTheme(LightTheme())
	if got := p.take(term); got != "" {
		t.Fatalf("unchanged preference sent %q", got)
	}
	term.UseTheme(DarkTheme())
	if got, want := p.take(term), "\x1b[?997;1n"; got != want {
		t.Fatalf("dark change = %q, want %q", got, want)
	}
	term.Feed([]byte("\x1b[?996n"))
	if got, want := p.take(term), "\x1b[?997;1n"; got != want {
		t.Fatalf("query after change = %q, want %q", got, want)
	}
	term.Feed([]byte("\x1b[?2031l"))
	term.UseTheme(LightTheme())
	if got := p.take(term); got != "" {
		t.Fatalf("unsubscribed application received %q", got)
	}
}
