package ui

import (
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/internal/l10n"
	"github.com/egoist/mygo/internal/platform"
)

func TestCalendar(t *testing.T) {
	day := time.Date(2026, 10, 15, 9, 41, 0, 0, time.UTC)
	changes := 0
	tt := coreNewTester(func(c *context) {
		if coreCalendar(c, &day).Changed() {
			changes++
		}
	}, 400, 400)
	tt.Click("October 20, 2026")
	if !sameDay(day, time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)) || day.Hour() != 9 || changes != 1 {
		t.Fatalf("a click: %v (%d changes)", day, changes)
	}
	// The arrows choose as they move.
	for _, step := range []struct {
		key  Key
		want string
	}{{KeyRight, "2026-10-21"}, {KeyDown, "2026-10-28"}, {KeyEnd, "2026-10-31"}, {KeyHome, "2026-10-01"}, {KeyPageDown, "2026-11-01"}, {KeyLeft, "2026-10-31"}} {
		tt.Key(0, step.key)
		if got := day.Format("2006-01-02"); got != step.want {
			t.Fatalf("%v: %s, not %s", step.key, got, step.want)
		}
	}
	if !tt.HasText("October 2026") {
		t.Error("the calendar does not follow the day into its month")
	}
	// Assistive technology reads the day chosen.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleButton, "October 31, 2026"); tt.h.access.Focus != n.ID || n.States&platform.AccessChecked == 0 {
		t.Errorf("the focus on %d, the day %+v", tt.h.access.Focus, n)
	}
}

func TestCalendarBrowse(t *testing.T) {
	date := time.Date(2026, 10, 29, 9, 41, 0, 0, time.UTC)
	changed := 0
	tt := NewTester(func(c *Context) {
		if Calendar(c, &date).Browse().Changed() {
			changed++
		}
	}, 400, 400)
	tt.Frame()
	if err := tt.Click("Next month"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if changed != 0 || date.Day() != 29 || date.Month() != time.October {
		t.Fatalf("paging to the next month chose %s (%d changes)", date.Format("2006-01-02"), changed)
	}
	if !tt.HasText("November 2026") {
		t.Fatalf("Next month shows %q", tt.Texts())
	}
	// A click chooses.
	if err := tt.Click("November 12, 2026"); err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 11, 12, 9, 41, 0, 0, time.UTC); !date.Equal(want) || changed != 1 {
		t.Fatalf("a click on the 12th: %v (%d changes)", date, changed)
	}
	// The keys move without choosing, Enter chooses.
	for _, key := range []Key{KeyPageDown, KeyRight, KeyEnd, KeyHome, KeyDown} {
		tt.Key(0, key)
	}
	if changed != 1 || !sameDay(date, time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("Page Down and the arrows chose %v (%d changes)", date, changed)
	}
	if !tt.HasText("December 2026") {
		t.Fatalf("Page Down shows %q", tt.Texts())
	}
	// Assistive technology reads the day under the keys, the day chosen
	// marked so.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleButton, "December 8, 2026"); tt.h.access.Focus != n.ID || n.States&platform.AccessChecked != 0 {
		t.Errorf("the focus on %d, the day %+v", tt.h.access.Focus, n)
	}
	tt.Key(0, KeyPageUp)
	if n := accessNode(t, tt.h.access, platform.RoleButton, "November 12, 2026"); n.States&platform.AccessChecked == 0 {
		t.Errorf("the day chosen %+v", n)
	}
	tt.Key(0, KeyEnter)
	if want := time.Date(2026, 11, 8, 9, 41, 0, 0, time.UTC); !date.Equal(want) || changed != 2 {
		t.Fatalf("Enter on the 8th: %v (%d changes)", date, changed)
	}
	// The calendar goes to a date the app sets.
	tt.Key(0, KeyPageDown)
	date = time.Date(2027, 3, 3, 9, 41, 0, 0, time.UTC)
	tt.Frame()
	if !tt.HasText("March 2027") || changed != 2 {
		t.Fatalf("a date the app set shows %q (%d changes)", tt.Texts(), changed)
	}
	tt.Key(0, KeyRight)
	tt.Key(0, KeySpace)
	if want := time.Date(2027, 3, 4, 9, 41, 0, 0, time.UTC); !date.Equal(want) || changed != 3 {
		t.Fatalf("Right and Space from the 3rd: %v (%d changes)", date, changed)
	}
}

// A calendar in a popover that closes as the user picks a day.
func TestCalendarBrowsePopover(t *testing.T) {
	due := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	open := false
	tt := NewTester(func(c *Context) {
		chip := Button(c, due.Format("Jan 2"))
		if chip.Clicked() {
			open = !open
		}
		Popover(c, chip, &open, func() {
			if Calendar(c, &due).Browse().Changed() {
				open = false
			}
		})
	}, 400, 420)
	tt.Click("Oct 29")
	tt.Click("Next month")
	tt.Click("Next month")
	if !open || !tt.HasText("December 2026") || due.Month() != time.October {
		t.Fatalf("paging: open %v, due %v, texts %q", open, due, tt.Texts())
	}
	tt.Click("December 24, 2026")
	if open || !sameDay(due, time.Date(2026, 12, 24, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("a click on the 24th: open %v, due %v", open, due)
	}
}

func TestTimeInput(t *testing.T) {
	alarm := time.Date(2026, 10, 4, 7, 30, 0, 0, time.UTC)
	changes := 0
	tt := coreNewTester(func(c *context) {
		if coreTimeInput(c, &alarm).Label("Alarm").Changed() {
			changes++
		}
	}, 300, 100)
	tt.SetRegion(i18n.Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24})
	if !tt.HasText("07") || !tt.HasText("30") {
		t.Fatalf("shows %q", tt.Texts())
	}
	tt.Click("07")
	tt.Key(0, KeyUp)
	if alarm.Hour() != 8 || changes != 1 {
		t.Fatalf("Up: %v", alarm)
	}
	// Round the clock.
	for range 9 {
		tt.Key(0, KeyDown)
	}
	if alarm.Hour() != 23 {
		t.Fatalf("Down nine times from 8: %v", alarm)
	}
	// Digits typed set the hours, then go on to the minutes.
	tt.Key(0, Key1)
	tt.Key(0, Key5)
	if alarm.Hour() != 15 || !tt.Focused("30") {
		t.Fatalf("1 and 5: %v, minutes focused %v", alarm, tt.Focused("30"))
	}
	tt.Key(0, Key4)
	tt.Key(0, Key5)
	if alarm.Minute() != 45 {
		t.Fatalf("4 and 5 in the minutes: %v", alarm)
	}
	tt.Key(0, KeyLeft)
	tt.Key(0, Key9)
	if alarm.Hour() != 9 || !tt.Focused("45") {
		t.Errorf("9, which can't begin two digits of hours: %v, minutes focused %v", alarm, tt.Focused("45"))
	}
	if y, m, d := alarm.Date(); y != 2026 || m != 10 || d != 4 {
		t.Errorf("the date changed: %v", alarm)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	hours := accessNode(t, tt.h.access, platform.RoleStepper, "Alarm hours")
	if hours.Now != 9 || hours.Max != 23 {
		t.Errorf("hours: %+v", hours)
	}
	accessNode(t, tt.h.access, platform.RoleStepper, "Alarm minutes")
	// Assistive technology steps the hours, not the focus to the minutes.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: hours.ID, Action: platform.AccessIncrement})
	if alarm.Hour() != 10 {
		t.Errorf("incremented the hours to %v", alarm)
	}
}

// TestTimeInput12 checks a time field in 12 hours: the half of the day a
// segment of its own, after the hours in English and before them in
// Chinese, as the locale's pattern writes it.
func TestTimeInput12(t *testing.T) {
	alarm := time.Date(2026, 10, 4, 7, 30, 0, 0, time.UTC)
	tt := coreNewTester(func(c *context) { coreTimeInput(c, &alarm).Label("Alarm") }, 300, 100)
	tt.SetRegion(i18n.Region{Locale: "en-US", FirstWeekday: time.Sunday, HourCycle: 12})
	if got := tt.Texts(); !slices.Equal(got, []string{"Alarm", "hours", "7", ":", "minutes", "30", "AM/PM", "AM"}) {
		t.Fatalf("shows %q", got)
	}
	tt.Click("AM")
	tt.Key(0, KeyUp)
	if alarm.Hour() != 19 {
		t.Fatalf("Up on AM: %v", alarm)
	}
	tt.Key(0, KeyA)
	if alarm.Hour() != 7 {
		t.Fatalf("A: %v", alarm)
	}
	tt.Click("7")
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	if alarm.Hour() != 0 || !tt.HasText("12") {
		t.Fatalf("seven hours down from 7 AM: %v, %q", alarm, tt.Texts())
	}
	tt.Key(0, Key1)
	tt.Key(0, Key1)
	if alarm.Hour() != 11 || !tt.Focused("30") {
		t.Fatalf("1 and 1: %v", alarm)
	}
	tt.SetLocale("zh-Hans")
	if got := tt.Texts(); !slices.Equal(got, []string{"Alarm", "上午/下午", "上午", "小时", "11", ":", "分钟", "30"}) {
		t.Errorf("in Chinese, shows %q", got)
	}
}

// TestCalendarRegion checks that a calendar starts its weeks on the first
// day of the user's formats and says its names in the window's language.
func TestCalendarRegion(t *testing.T) {
	day := time.Date(2026, 10, 15, 9, 41, 0, 0, time.UTC)
	tt := coreNewTester(func(c *context) { coreCalendar(c, &day) }, 400, 400)
	tt.SetRegion(i18n.Region{Locale: "en-US", FirstWeekday: time.Sunday, HourCycle: 12})
	week := func(first string) []string {
		texts := tt.Texts()
		i := slices.Index(texts, first)
		if i < 0 || i+8 > len(texts) {
			return texts
		}
		return texts[i : i+8]
	}
	// October 2026 starts on a Thursday: the 27th of September before it.
	if got := week("Su"); !slices.Equal(got, []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa", "September 27, 2026"}) {
		t.Errorf("the days of a week from Sunday: %q", got)
	}
	tt.SetRegion(i18n.Region{Locale: "en-GB", FirstWeekday: time.Monday, HourCycle: 24})
	if got := week("Mo"); !slices.Equal(got, []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su", "28 September 2026"}) {
		t.Errorf("the days of a week from Monday: %q", got)
	}
	tt.SetLocale("zh-Hans")
	if !tt.HasText("2026年10月") || !tt.HasText("一") {
		t.Errorf("in Chinese: %q", tt.Texts())
	}
	tt.SetLocale("fa")
	if !tt.HasText("۱۵") {
		t.Errorf("in Persian, the days are not in its digits: %q", tt.Texts())
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.SetLocale("de")
	accessNode(t, tt.h.access, platform.RoleButton, "15. Oktober 2026")
	// MyGo's own strings have no German: the source language's.
	accessNode(t, tt.h.access, platform.RoleButton, "Next month")
	tt.SetLocale("zh-Hant")
	accessNode(t, tt.h.access, platform.RoleButton, l10n.In("zh-Hant").NextMonth())
}

func TestColorPicker(t *testing.T) {
	color := Hex("#3b82f6")
	changes := 0
	tt := coreNewTester(func(c *context) {
		if coreColorPicker(c, &color).Changed() {
			changes++
		}
	}, 400, 500)
	// A swatch.
	tt.Click("Red")
	if color != Hex("#ef4444") || changes != 1 {
		t.Fatalf("Red: %v", hexOf(color))
	}
	// The hex typed.
	tt.Click("Hex")
	tt.Key(Cmd, KeyA)
	tt.Type("#00ff0080")
	if color != Hex("#00ff0080") {
		t.Fatalf("typed: %v", hexOf(color))
	}
	// The square: its top-right corner is the hue itself; its bottom black.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	sq := accessNode(t, tt.h.access, platform.RoleSlider, "Saturation and brightness").Bounds
	tt.ClickAt(float32(sq.X+sq.W-1), float32(sq.Y+1))
	if h := toHSVA(color); h.s < 0.98 || h.v < 0.98 || h.h < 115 || h.h > 125 {
		t.Fatalf("the top-right corner: %v (%+v)", hexOf(color), h)
	}
	// Let go at the bottom-left corner before a frame: black, though the
	// pointer went back.
	tt.Press(float32(sq.X+sq.W-1), float32(sq.Y+1))
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerUp, X: sq.X + 1, Y: sq.Y + sq.H - 1})
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: sq.X + sq.W - 1, Y: sq.Y + 1})
	tt.settle()
	if h := toHSVA(color); h.v > 0.02 {
		t.Errorf("let go at the bottom-left corner before a frame: %v (%+v)", hexOf(color), h)
	}
	// A gray keeps the hue it came from.
	tt.ClickAt(float32(sq.X+1), float32(sq.Y+sq.H/2))
	tt.Key(0, KeyRight)
	if h := toHSVA(color); h.h < 115 || h.h > 125 {
		t.Errorf("out of the grays, the hue is %v", h.h)
	}
	// The app sets another color.
	color = Hex("#ffffff")
	tt.Frame()
	if n := accessNode(t, tt.h.access, platform.RoleTextField, "Hex"); n.Value != "#ffffff" {
		t.Errorf("the hex shows %q", n.Value)
	}
	n := accessNode(t, tt.h.access, platform.RoleSlider, "Hue")
	if n.Max != 360 {
		t.Errorf("the hue: %+v", n)
	}
}

func TestColorWell(t *testing.T) {
	color := Hex("#3b82f6")
	tt := coreNewTester(func(c *context) {
		coreColorWell(c, &color).Label("Tint")
	}, 400, 500)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleColorWell, "Tint"); n.Value != "#3b82f6" || n.States&platform.AccessExpanded != 0 {
		t.Fatalf("the well: %+v", n)
	}
	tt.Click("Tint")
	if !tt.HasText("Red") {
		t.Fatal("the picker did not open")
	}
	tt.Click("Green")
	if color != Hex("#22c55e") {
		t.Errorf("chose %v", hexOf(color))
	}
	if n := accessNode(t, tt.h.access, platform.RoleColorWell, "Tint"); n.Value != "#22c55e" || n.States&platform.AccessExpanded == 0 {
		t.Errorf("the well: %+v", n)
	}
	tt.Key(0, KeyEscape)
	if tt.HasText("Red") {
		t.Error("Escape left the picker open")
	}
}

func TestTimeLayout(t *testing.T) {
	for _, c := range []struct {
		pattern, sep string
		periodFirst  bool
	}{
		{"h:mm a", ":", false},
		{"ah:mm", ":", true},
		{"H 'h' mm", " h ", false},
		{"HH.mm", ".", false},
		{"H'm' mm", "m ", false},
		{"a h'''h' mm", "'h ", true},
		{"B h:mm", ":", true},
	} {
		if sep, first := timeLayout(c.pattern); sep != c.sep || first != c.periodFirst {
			t.Errorf("timeLayout(%q) = %q, %v; want %q, %v", c.pattern, sep, first, c.sep, c.periodFirst)
		}
	}
	// French of Canada writes 07 h 30.
	alarm := time.Date(2026, 10, 4, 7, 30, 0, 0, time.UTC)
	tt := coreNewTester(func(c *context) { coreTimeInput(c, &alarm) }, 300, 100)
	tt.SetLocale("fr-CA")
	tt.SetRegion(i18n.Region{Locale: "fr-CA", FirstWeekday: time.Sunday, HourCycle: 24})
	if !tt.HasText(" h ") {
		t.Errorf("fr-CA: %q", tt.Texts())
	}
}
