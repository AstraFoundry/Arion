package ui

import (
	"testing"
	"time"

	transfer "github.com/egoist/mygo/transfer"
)

func followTester(s *ListState, n *int, last *float32, below *float32) (*Tester, *time.Time) {
	now := time.Unix(1000, 0)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Grow(1).Children(func() {
			coreList(c, s, *n, func(i int) {
				h := float32(40)
				if i == *n-1 {
					h = *last
				}
				coreBox(c).Height(h).Children(func() { coreTextf(c, "Message %d", i) })
			}).Grow(1)
			coreBox(c).Height(*below)
		})
	}, 300, 400)
	tt.rt.clock = func() time.Time { return now }
	return tt, &now
}

// The list gets shorter (a composer below grows): no rows were added, yet
// the end glides.
func TestReviewFollowGlideOnShrink(t *testing.T) {
	n, last, below := 30, float32(40), float32(0)
	s := ListState{FollowEnd: true, FollowAnimation: &Smooth}
	tt, _ := followTester(&s, &n, &last, &below)
	tt.Frame()
	tt.Frame()
	if r, _ := rowBox(tt, &s, n-1); r.Y+r.H != 400 {
		t.Fatalf("at first the last row ends at %v", r.Y+r.H)
	}
	below = 100
	tt.Frame()
	if r, _ := rowBox(tt, &s, n-1); r.Y+r.H != 300 {
		t.Errorf("list 100 shorter: last row ends at %v, want 300 (it is hidden under the composer)", r.Y+r.H)
	}
}

// The last row shrinks back while it glides: the rows above it stay where
// they show, then go down to their place.
func TestReviewFollowShrinkDuringGlide(t *testing.T) {
	n, last, below := 30, float32(40), float32(0)
	s := ListState{FollowEnd: true, FollowAnimation: &Smooth}
	tt, now := followTester(&s, &n, &last, &below)
	tt.Frame()
	tt.Frame()
	last = 300
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	before, _ := rowBox(tt, &s, n-2)
	last = 40
	*now = now.Add(1 * time.Millisecond)
	tt.Frame()
	after, _ := rowBox(tt, &s, n-2)
	if after.Y > before.Y+1 {
		t.Errorf("the last row shrank back: the row above it jumped down from %v to %v (lag %v)", before.Y, after.Y, s.followLag.value)
	}
	*now = now.Add(2 * time.Second)
	tt.Frame()
	if r, _ := rowBox(tt, &s, n-1); r.Y+r.H != 400 {
		t.Errorf("after the glide, the last row ends at %v", r.Y+r.H)
	}
}

// The last row is removed while the list glides (a typing indicator goes).
func TestReviewFollowRemoveDuringGlide(t *testing.T) {
	n, last, below := 30, float32(40), float32(0)
	s := ListState{FollowEnd: true, FollowAnimation: &Smooth}
	tt, now := followTester(&s, &n, &last, &below)
	tt.Frame()
	tt.Frame()
	n++
	last = 200
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	before, _ := rowBox(tt, &s, n-2)
	n--
	last = 40
	*now = now.Add(1 * time.Millisecond)
	tt.Frame()
	after, _ := rowBox(tt, &s, n-1)
	// With the row it measured from gone, the list shows its rows in place,
	// not below it by what was left of the glide.
	if after.Y+after.H != 400 {
		t.Errorf("the last row went: the row above it moved from %v to %v, ending at %v (lag %v)", before.Y, after.Y, after.Y+after.H, s.followLag.value)
	}
}

func TestReviewFollowStaticContainers(t *testing.T) {
	for _, kind := range []string{"row", "maxh", "grid", "wrap"} {
		t.Run(kind, func(t *testing.T) {
			n := 30
			s := ListState{FollowEnd: true, FollowAnimation: &Smooth}
			now := time.Unix(1000, 0)
			calls := 0
			tt := coreNewTester(func(c *context) {
				list := func() *node {
					return coreList(c, &s, n, func(i int) {
						coreBox(c).Height(40).Children(func() { coreTextf(c, "M %d", i) })
					})
				}
				switch kind {
				case "row":
					coreRow(c).Grow(1).Children(func() { list().Grow(1) })
				case "maxh":
					coreColumn(c).Children(func() { list().MaxHeight(300) })
				case "grid":
					coreGrid(c).Grow(1).Children(func() { list().Height(300) })
				case "wrap":
					coreRow(c).Wrap().Children(func() { list().Height(300).Width(200); coreBox(c).Size(150, 50) })
				}
				calls++
			}, 300, 400)
			tt.rt.clock = func() time.Time { return now }
			for f := 0; f < 5; f++ {
				now = now.Add(16 * time.Millisecond)
				tt.Frame()
				if s.followLag.value != 0 {
					t.Errorf("frame %d: static list glides, lag %v", f, s.followLag.value)
				}
			}
		})
	}
}

func TestReviewFollowStreamInvariants(t *testing.T) {
	for _, tc := range []struct {
		name    string
		n       int
		justify Align
		header  bool
	}{{"many", 30, Start, false}, {"few-end", 2, End, false}, {"few-start", 2, Start, false}, {"headers", 30, Start, true}} {
		t.Run(tc.name, func(t *testing.T) {
			n := tc.n
			last := float32(20)
			header := func(i int) bool { return i%7 == 0 }
			s := ListState{FollowEnd: true, FollowAnimation: &Smooth}
			if tc.header {
				s.Header = header
			}
			now := time.Unix(1000, 0)
			tt := coreNewTester(func(c *context) {
				coreList(c, &s, n, func(i int) {
					h := float32(40)
					if tc.header && header(i) {
						h = 24
					}
					if i == n-1 {
						h = last
					}
					coreBox(c).Height(h).Children(func() { coreTextf(c, "M %d", i) })
				}).Grow(1).Justify(tc.justify)
			}, 300, 400)
			tt.rt.clock = func() time.Time { return now }
			tt.Frame()
			for f := 0; f < 200; f++ {
				now = now.Add(16 * time.Millisecond)
				if f%25 == 24 {
					n++
					last = 20
				} else if f < 150 {
					last += 7
				}
				tt.Frame()
				if !s.AtEnd() || !s.following {
					t.Fatalf("frame %d: AtEnd %v following %v", f, s.AtEnd(), s.following)
				}
				first, lastV := s.Visible()
				if false {
					var rs []Rect
					for i := max(0, n-4); i < n; i++ {
						r, _ := rowBox(tt, &s, i)
						rs = append(rs, r)
					}
					t.Fatalf("frame %d: visible %d..%d of %d: %v lag %v", f, first, lastV, n, rs, s.followLag.value)
				}
				if !tc.header {
					prev, _ := rowBox(tt, &s, first)
					for i := first + 1; i <= lastV; i++ {
						r, ok := rowBox(tt, &s, i)
						if d := r.Y - (prev.Y + prev.H); !ok || d > 0.01 || d < -0.01 {
							t.Fatalf("frame %d: row %d at %v after %v", f, i, r, prev)
						}
						prev = r
					}
				}
			}
			now = now.Add(3 * time.Second)
			tt.Frame()
			if r, _ := rowBox(tt, &s, n-1); r.Y+r.H != 400 && (tc.justify == End || tc.n > 8) {
				t.Errorf("rested: last row ends at %v", r.Y+r.H)
			}
			if s.followLag.moving || tt.rt.animating {
				t.Errorf("still animating after rest")
			}
		})
	}
}

func TestReviewSelectionGrowAndCopyData(t *testing.T) {
	a, b := "Alpha beta", "Gamma"
	var parts []SelectedText
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Selectable().Padding(12).Gap(8).CopyData(func(p []SelectedText) transfer.Data {
			parts = p
			s := ""
			for _, x := range p {
				s += string([]rune(x.Text)[x.Start:x.End]) + "|"
			}
			return transfer.TextData(s)
		}).Children(func() {
			coreText(c, a)
			coreText(c, b)
		})
	}, 300, 150)
	selectBetween(t, tt, a, 6, b, 3)
	b = "Gamma delta"
	tt.Frame()
	tt.Key(Cmd, KeyC)
	if got := tt.Clipboard(); got != "beta|Gam|" {
		t.Errorf("after growth copied %q (%+v)", got, parts)
	}
	a = "Alpha beta and more"
	tt.Frame()
	tt.Command("copy")
	if got := tt.Clipboard(); got != "beta and more|Gam|" {
		t.Errorf("after first grew copied %q", got)
	}
	x, y := selectableAt(t, tt, b, 1)
	tt.RightClickAt(x, y)
	tt.SetClipboard("")
	if err := tt.ChooseMenuItem("Copy"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != "beta and more|Gam|" {
		t.Errorf("menu copied %q", got)
	}
}

func TestReviewNestedInlineSpans(t *testing.T) {
	inner := "c"
	tt := coreNewTester(func(c *context) {
		coreRichText(c).Selectable().Children(func() {
			coreText(c, "a ")
			coreRichText(c, Span{Text: "b "}).Children(func() {
				coreText(c, inner).Bold()
			})
			coreText(c, " d")
		})
	}, 300, 100)
	for i := 0; i < 3; i++ {
		inner += "x"
		tt.Frame()
		for _, s := range tt.rt.texts {
			want := "a b " + inner + " d"
			if s.editor.source != want {
				t.Fatalf("frame %d: paragraph %q want %q", i, s.editor.source, want)
			}
			sc := s.spanCache()
			got := ""
			for _, sp := range sc.inline {
				got += sp.Text
			}
			if got != want {
				t.Fatalf("frame %d: spans %q want %q", i, got, want)
			}
		}
	}
}

func TestReviewFollowGlideOnWindowResize(t *testing.T) {
	n, last, below := 30, float32(40), float32(0)
	s := ListState{FollowEnd: true, FollowAnimation: &Smooth}
	tt, _ := followTester(&s, &n, &last, &below)
	tt.Frame()
	tt.SetSize(300, 340)
	tt.Frame()
	if r, _ := rowBox(tt, &s, n-1); r.Y+r.H != 340 {
		t.Errorf("window 60 shorter: last row ends at %v, want 340 (lag %v)", r.Y+r.H, s.followLag.value)
	}
}

// The user scrolls down (toward the end) during a glide: following stops.
func TestReviewFollowScrollDownDuringGlide(t *testing.T) {
	n, last, below := 30, float32(40), float32(0)
	s := ListState{FollowEnd: true, FollowAnimation: &Smooth}
	tt, now := followTester(&s, &n, &last, &below)
	tt.Frame()
	tt.Frame()
	n++
	last = 200
	tt.Frame()
	*now = now.Add(16 * time.Millisecond)
	tt.Move(100, 100)
	tt.Scroll(100, 100, 0, 20) // toward the end, less than the lag
	tt.Frame()
	if !s.following {
		t.Errorf("scrolling toward the end during the glide stopped following (AtEnd %v)", s.AtEnd())
	}
	*now = now.Add(2 * time.Second)
	tt.Frame()
	n++
	tt.Frame()
	*now = now.Add(2 * time.Second)
	tt.Frame()
	if r, _ := rowBox(tt, &s, n-1); r.Y+r.H != 400 {
		t.Errorf("after a row came, the last row ends at %v, want 400", r.Y+r.H)
	}
}

// TestListFollowGlideKeepsRowsThatStay checks that a list following its
// end with a FollowAnimation leaves the rows that do not change where
// they show on the frame its content changes, then glides them to their
// place: the rows fit it, or overflow it, or stop overflowing it, and
// the list is as tall as its content.
func TestListFollowGlideKeepsRowsThatStay(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rows     int
		from, to float32 // the last row's height before and after
		add      bool    // a row of height to comes after the last
		justify  Align
		glide    float32 // how far the row that stays moves, once settled
		// hug, when set, makes the list as tall as its content, up to hug
		// when it is above 0, in a column under a title.
		hug float32
	}{
		{"fit start grow", 3, 40, 100, false, Start, 0, 0},
		{"fit start add", 3, 40, 40, true, Start, 0, 0},
		{"fit end grow", 3, 40, 100, false, End, -60, 0},
		{"fit center grow", 3, 40, 100, false, Center, -30, 0},
		{"start to overflow", 9, 40, 100, false, Start, -20, 0},
		{"overflow grow", 20, 40, 100, false, Start, -60, 0},
		{"overflow add", 20, 40, 40, true, Start, -40, 0},
		{"overflow to fit", 10, 300, 40, false, Start, 260, 0},
		{"overflow to fit, far", 5, 500, 40, false, Start, 260, 0},
		{"hug start grow", 3, 44, 66, false, Start, 0, -1},
		{"hug start add", 3, 44, 44, true, Start, 0, -1},
		{"hug end grow", 3, 44, 66, false, End, 0, -1},
		{"hug center grow", 3, 44, 66, false, Center, 0, -1},
		{"hug start shrink", 3, 88, 44, false, Start, 0, -1},
		{"hug end shrink", 3, 88, 44, false, End, 0, -1},
		{"hug to its max height", 3, 44, 110, false, Start, -34, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, last := tc.rows, tc.from
			s := ListState{FollowEnd: true, FollowAnimation: &Smooth}
			now := time.Unix(1000, 0)
			list := func(c *context) *node {
				return coreList(c, &s, n, func(i int) {
					h := float32(40)
					if tc.hug != 0 {
						h = 44
					}
					if i == n-1 {
						h = last
					}
					coreBox(c).Height(h).Children(func() { coreTextf(c, "Message %d", i) })
				}).Justify(tc.justify)
			}
			tt := coreNewTester(func(c *context) {
				if tc.hug == 0 {
					list(c).Grow(1)
					return
				}
				coreColumn(c).Grow(1).Padding(14).Gap(10).Children(func() {
					coreTextf(c, "Chat")
					coreBox(c).Children(func() {
						l := list(c).Gap(8).Padding(10)
						if tc.hug > 0 {
							l.MaxHeight(tc.hug)
						}
					})
				})
			}, 300, 400)
			tt.rt.clock = func() time.Time { return now }
			tt.Frame()
			stay := tc.rows - 2
			before, _ := rowBox(tt, &s, stay)
			if tc.add {
				n++
			}
			last = tc.to
			tt.Frame()
			if r, _ := rowBox(tt, &s, stay); r.Y != before.Y {
				t.Errorf("on the frame the content changed, row %d moved from %v to %v (lag %v, offset %v)",
					stay, before.Y, r.Y, s.followLag.value, listStateOf(&s).scrollY)
			}
			now = now.Add(50 * time.Millisecond)
			tt.Frame()
			mid, _ := rowBox(tt, &s, stay)
			if moved := mid.Y - before.Y; tc.glide == 0 && moved != 0 || tc.glide != 0 && (moved*tc.glide <= 0 || abs(moved) >= abs(tc.glide)) {
				t.Errorf("50 ms into the glide, row %d is at %v: want it on its way from %v by %v", stay, mid.Y, before.Y, tc.glide)
			}
			now = now.Add(2 * time.Second)
			tt.Frame()
			if r, _ := rowBox(tt, &s, stay); r.Y != before.Y+tc.glide {
				t.Errorf("after the glide, row %d is at %v, want %v", stay, r.Y, before.Y+tc.glide)
			}
			if r, _ := rowBox(tt, &s, n-1); tc.rows > 9 && r.Y+r.H != 400 {
				t.Errorf("after the glide, the last row ends at %v, want 400", r.Y+r.H)
			}
			if !s.following {
				t.Error("the list stopped following its end")
			}
		})
	}
}

// TestListFollowGlideHugStreams checks that a list as tall as its content,
// whose last row grows in every frame, as a reply streams in, keeps the
// rows above it where they are: its height follows its content a frame
// late.
func TestListFollowGlideHugStreams(t *testing.T) {
	last := float32(44)
	s := ListState{FollowEnd: true, FollowAnimation: &Smooth}
	now := time.Unix(1000, 0)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Grow(1).Padding(14).Gap(10).Children(func() {
			coreTextf(c, "Chat")
			coreBox(c).Children(func() {
				coreList(c, &s, 3, func(i int) {
					h := float32(44)
					if i == 2 {
						h = last
					}
					coreBox(c).Height(h).Children(func() { coreTextf(c, "Message %d", i) })
				}).Gap(8).Padding(10)
			})
		})
	}, 300, 400)
	tt.rt.clock = func() time.Time { return now }
	tt.Frame()
	before, _ := rowBox(tt, &s, 1)
	for k := range 8 {
		last += 22
		now = now.Add(16 * time.Millisecond)
		tt.Frame()
		if r, _ := rowBox(tt, &s, 1); r.Y != before.Y {
			t.Fatalf("line %d: row 1 moved from %v to %v (lag %v)", k+1, before.Y, r.Y, s.followLag.value)
		}
	}
}
