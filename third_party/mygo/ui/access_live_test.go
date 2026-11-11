package ui

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestRegionSeparatorRowHeader checks the roles of landmarks, a rule and
// a row's header: a region is one only with a name, a search landmark
// holds its field and button, an article holds its post, named or not, a
// rule taller than wide runs up and down, and a row header is named by
// its content.
func TestRegionSeparatorRowHeader(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			coreColumn(c).Role(RoleRegion).Label("Activity").Children(func() {
				coreText(c, "Ann joined")
			})
			coreColumn(c).Role(RoleRegion).Children(func() {
				coreText(c, "Unnamed")
			})
			coreRow(c).Role(RoleSearch).Label("Site search").Children(func() {
				coreText(c, "Query").Role(RoleTextField)
				coreText(c, "Go").Role(RoleButton)
			})
			coreColumn(c).Role(RoleArticle).Label("Post by Ann").Children(func() {
				coreText(c, "Shipped the beta")
			})
			coreColumn(c).Role(RoleArticle).Children(func() {
				coreText(c, "An unnamed post")
			})
			coreRow(c).Role(RoleToolbar).Label("Format").Height(20).Children(func() {
				coreText(c, "Bold")
				coreDivider(c).Role(RoleSeparator)
				coreText(c, "Left")
			})
			coreDivider(c).Role(RoleSeparator).Width(100)
			coreColumn(c).Role(RoleTable).Label("Prices").Children(func() {
				coreRow(c).Role(RoleRow).Children(func() {
					coreText(c, "Apples").Role(RoleRowHeader)
					coreText(c, "3").Role(RoleCell)
				})
			})
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	activity := accessNode(t, tree, platform.RoleRegion, "Activity")
	if text := accessNode(t, tree, platform.RoleText, "Ann joined"); text.Parent < 0 || tree.Nodes[text.Parent].ID != activity.ID {
		t.Errorf("the text is not in the region")
	}
	search := accessNode(t, tree, platform.RoleSearch, "Site search")
	for _, name := range []string{"Query", "Go"} {
		for _, n := range tree.Nodes {
			if n.Label == name && (n.Parent < 0 || tree.Nodes[n.Parent].ID != search.ID) {
				t.Errorf("%s is not in the search landmark", name)
			}
		}
	}
	post := accessNode(t, tree, platform.RoleArticle, "Post by Ann")
	if text := accessNode(t, tree, platform.RoleText, "Shipped the beta"); text.Parent < 0 || tree.Nodes[text.Parent].ID != post.ID {
		t.Errorf("the post's text is not in the article")
	}
	if text := accessNode(t, tree, platform.RoleText, "An unnamed post"); text.Parent < 0 || tree.Nodes[text.Parent].Role != platform.RoleArticle {
		t.Errorf("an article without a name is not one")
	}
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleRegion && n.Label != "Activity" {
			t.Errorf("a region without a name is a landmark: %+v", n)
		}
	}
	var rules []platform.AccessNode
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleSeparator {
			rules = append(rules, n)
		}
	}
	if len(rules) != 2 {
		t.Fatalf("%d separators, want 2", len(rules))
	}
	if rules[0].States&platform.AccessVertical == 0 || tree.Nodes[rules[0].Parent].Role != platform.RoleToolbar {
		t.Errorf("the toolbar's rule: %+v", rules[0])
	}
	if rules[1].States&platform.AccessVertical != 0 {
		t.Errorf("the rule across runs up and down: %+v", rules[1])
	}
	accessNode(t, tree, platform.RoleRowHeader, "Apples")
}

// TestLiveRegions checks what logs and alerts read out: what a log gains
// once it showed, politely, whole for what is added with content of its
// own, and an alert's text as it appears and as it changes, at once.
// Timers and what was there as assistive technology came read nothing.
func TestLiveRegions(t *testing.T) {
	messages := []string{"Hi"}
	var errText, clock string
	clock = "5:00"
	show := true
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			if show {
				coreColumn(c).Role(RoleLog).Label("Messages").Children(func() {
					for i, m := range messages {
						coreRow(c).Key(i).Gap(4).Children(func() {
							coreText(c, "Ann:")
							coreText(c, m)
						})
					}
				})
			}
			coreRow(c).Role(RoleTimer).Label("Time left").Children(func() {
				coreText(c, clock)
			})
			if errText != "" {
				coreText(c, errText).Role(RoleAlert)
			}
			coreColumn(c).Role(RoleAlert).Label("Saved")
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tt.Frame()
	check := func(step string, announced, alerted []string) {
		t.Helper()
		tree := tt.h.access
		if !slices.Equal(tree.Announcements, announced) {
			t.Errorf("%s: announcements %q, want %q", step, tree.Announcements, announced)
		}
		if !slices.Equal(tree.Alerts, alerted) {
			t.Errorf("%s: alerts %q, want %q", step, tree.Alerts, alerted)
		}
	}
	check("as it came", nil, nil)
	accessNode(t, tt.h.access, platform.RoleLog, "Messages")
	accessNode(t, tt.h.access, platform.RoleTimer, "Time left")

	messages = append(messages, "How are you?")
	clock = "4:59"
	tt.Frame()
	check("a message", []string{"Ann: How are you?"}, nil)
	tt.Frame()
	check("nothing new", nil, nil)

	errText = "Name is required"
	tt.Frame()
	check("an error", nil, []string{"Name is required"})
	tt.Frame()
	check("the same error", nil, nil)
	errText = "Name is too long"
	tt.Frame()
	check("another error", nil, []string{"Name is too long"})

	// A log shown anew reads nothing of what it shows.
	show = false
	tt.Frame()
	show = true
	messages = append(messages, "Bye")
	tt.Frame()
	check("the log again", nil, nil)
}

// TestRowHeaderKeepsChildren checks that a row header shows what is
// inside it, as a cell does (ARIA's rowheader is not
// children-presentational): a button in it stays reachable, while the
// header is still named by its content.
func TestRowHeaderKeepsChildren(t *testing.T) {
	pressed := 0
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Role(RoleTable).Label("Prices").Children(func() {
			coreRow(c).Role(RoleRow).Children(func() {
				coreRow(c).Role(RoleRowHeader).Gap(4).Children(func() {
					coreText(c, "Apples")
					if coreButton(c, "Pin").Clicked() {
						pressed++
					}
				})
				coreText(c, "3").Role(RoleCell)
			})
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	header := accessNode(t, tree, platform.RoleRowHeader, "Apples Pin")
	pin := accessNode(t, tree, platform.RoleButton, "Pin")
	if pin.Parent < 0 || tree.Nodes[pin.Parent].ID != header.ID {
		t.Errorf("the button is not in the row header")
	}
	if pin.Actions&platform.ActionPress == 0 {
		t.Errorf("the button can't be pressed: %+v", pin)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: pin.ID, Action: platform.AccessPress})
	tt.Frame()
	if pressed != 1 {
		t.Errorf("pressed %d times through assistive technology, want 1", pressed)
	}
}

// TestTableRowHeaders checks that the cells of a Table's RowHeader column
// are row headers, named by their text, and the others cells.
func TestTableRowHeaders(t *testing.T) {
	s := &ListState{}
	cols := []TableColumn{{Title: "#", RowHeader: true, Width: 40}, {Title: "Name"}}
	tt := NewTester(func(c *Context) {
		Table(c, s, cols, 2, func(row, col int) {
			if col == 0 {
				Textf(c, "%d", row+1)
				return
			}
			Textf(c, "file %d", row)
		}).Grow(1)
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	accessNode(t, tree, platform.RoleRowHeader, "2")
	for _, n := range tree.Nodes {
		if n.Role == platform.RoleRowHeader && n.Label != "1" && n.Label != "2" {
			t.Errorf("a row header %q", n.Label)
		}
	}
	accessNode(t, tree, platform.RoleText, "file 1")
}
