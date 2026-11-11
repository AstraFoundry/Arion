package ui

import (
	"testing"
	"time"
)

func TestCalendarMonthEnds(t *testing.T) {
	zone := time.FixedZone("UTC+08", 8*60*60)
	for _, tc := range []struct {
		name      string
		from, to  time.Time
		direction int
	}{
		{"next-February", time.Date(2026, 1, 31, 9, 41, 17, 123, time.UTC), time.Date(2026, 2, 28, 9, 41, 17, 123, time.UTC), 1},
		{"previous-February", time.Date(2026, 3, 31, 9, 41, 17, 123, time.UTC), time.Date(2026, 2, 28, 9, 41, 17, 123, time.UTC), -1},
		{"leap-February", time.Date(2028, 1, 31, 9, 41, 17, 123, time.UTC), time.Date(2028, 2, 29, 9, 41, 17, 123, time.UTC), 1},
		{"thirty-day-month", time.Date(2026, 5, 31, 9, 41, 17, 123, time.UTC), time.Date(2026, 4, 30, 9, 41, 17, 123, time.UTC), -1},
		{"next-year", time.Date(2026, 12, 31, 9, 41, 17, 123, time.UTC), time.Date(2027, 1, 31, 9, 41, 17, 123, time.UTC), 1},
		{"previous-year", time.Date(2026, 1, 31, 9, 41, 17, 123, time.UTC), time.Date(2025, 12, 31, 9, 41, 17, 123, time.UTC), -1},
		{"mid-month", time.Date(2026, 1, 15, 9, 41, 17, 123, zone), time.Date(2026, 2, 15, 9, 41, 17, 123, zone), 1},
		{"local-month-end", time.Date(2026, 1, 31, 9, 41, 17, 123, zone), time.Date(2026, 2, 28, 9, 41, 17, 123, zone), 1},
	} {
		for _, widget := range []string{"Calendar", "Browse", "DateInput"} {
			for _, input := range []string{"button", "keyboard"} {
				t.Run(tc.name+"/"+widget+"/"+input, func(t *testing.T) {
					date, changed := tc.from, 0
					tt := NewTester(func(c *Context) {
						var e Element
						if widget == "DateInput" {
							e = DateInput(c, &date).Label("Due date")
						} else {
							e = Calendar(c, &date).AutoFocus()
							if widget == "Browse" {
								e = e.Browse()
							}
						}
						if e.Changed() {
							changed++
						}
					}, 400, 500)
					tt.SetLocale("en-US")
					tt.Frame()
					if widget == "DateInput" {
						if err := tt.Click("Due date"); err != nil {
							t.Fatal(err)
						}
					}
					if input == "button" {
						label := "Next month"
						if tc.direction < 0 {
							label = "Previous month"
						}
						if err := tt.Click(label); err != nil {
							t.Fatal(err)
						}
					} else {
						key := KeyPageDown
						if tc.direction < 0 {
							key = KeyPageUp
						}
						tt.Key(0, key)
					}
					tt.Frame()
					if month := tc.to.Format("January 2006"); !tt.HasText(month) {
						t.Fatalf("paging from %s did not show %s", tc.from.Format("2006-01-02"), month)
					}
					if widget == "Calendar" {
						if date != tc.to || changed != 1 {
							t.Fatalf("date %v, changes %d; want %v, 1", date, changed, tc.to)
						}
					} else if date != tc.from || changed != 0 {
						t.Fatalf("browsing chose %v (%d changes)", date, changed)
					}
					if widget != "Calendar" && input == "keyboard" {
						tt.Key(0, KeyEnter)
						if date != tc.to || changed != 1 {
							t.Fatalf("choosing the cursor: date %v, changes %d; want %v, 1", date, changed, tc.to)
						}
					}
				})
			}
		}
	}
}
