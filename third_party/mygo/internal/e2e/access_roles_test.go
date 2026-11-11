//go:build !js

package e2e

import (
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// landmarksAndLiveRegions shows a region, a search landmark, an article,
// a log, a timer, an alert, a toolbar's rule and a table's row header, labelled to be found but for
// the rule, which is the toolbar's only child but its buttons.
func landmarksAndLiveRegions(frames *atomic.Int32) func(c *ui.Context) {
	return func(c *ui.Context) {
		frames.Add(1)
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			ui.Column(c).Role(ui.RoleRegion).Label("Activity").Children(func() {
				ui.Text(c, "Ann joined")
			})
			ui.Row(c).Role(ui.RoleSearch).Label("Site search").Children(func() {
				ui.Button(c, "Go")
			})
			ui.Column(c).Role(ui.RoleArticle).Label("Post by Ann").Children(func() {
				ui.Text(c, "Shipped the beta")
			})
			ui.Column(c).Role(ui.RoleLog).Label("Messages").Children(func() {
				ui.Text(c, "Hi")
			})
			ui.Row(c).Role(ui.RoleTimer).Label("Time left").Children(func() {
				ui.Text(c, "5:00")
			})
			ui.Column(c).Role(ui.RoleAlert).Label("Error").Children(func() {
				ui.Text(c, "Name is required")
			})
			ui.Row(c).Role(ui.RoleToolbar).Label("Format").Height(24).Gap(4).Children(func() {
				ui.Button(c, "Bold")
				ui.Divider(c).Role(ui.RoleSeparator)
				ui.Button(c, "Left")
			})
			ui.Column(c).Role(ui.RoleTable).Label("Prices").Children(func() {
				ui.Row(c).Role(ui.RoleRow).Children(func() {
					ui.Text(c, "Apples").Role(ui.RoleRowHeader)
					ui.Text(c, "3").Role(ui.RoleCell)
				})
			})
		})
	}
}

// TestAccessLandmarksAndLiveRegions reads, as screen readers do, the
// roles WAI-ARIA's Core Accessibility API Mappings give a region, a
// search landmark, an article, a log, a timer, an alert, a separator and
// a row header on each platform.
func TestAccessLandmarksAndLiveRegions(t *testing.T) {
	var frames atomic.Int32
	w := newWindow(t, mygo.WindowOptions{Title: "Landmarks and live regions", Width: 400, Height: 400, Content: ui.View(landmarksAndLiveRegions(&frames))})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	if _, ok := accessibility(w); !ok {
		t.Skip("accessibility automation not available on this platform")
	}
	// The roles by label; the separator, unlabelled, by "".
	want := map[string]map[string]string{
		"darwin": {"Activity": "AXGroup/AXLandmarkRegion", "Site search": "AXGroup/AXLandmarkSearch",
			"Post by Ann": "AXGroup/AXDocumentArticle", "Messages": "AXGroup/AXApplicationLog", "Time left": "AXGroup/AXApplicationTimer",
			"Error": "AXGroup/AXApplicationAlert", "": "AXSplitter", "Apples": "AXCell"},
		"linux": {"Activity": "landmark", "Site search": "landmark", "Post by Ann": "article", "Messages": "log", "Time left": "timer", "Error": "notification", "": "separator",
			"Apples": "row header"},
		"windows": {"Activity": "Group", "Site search": "Group", "Post by Ann": "Group", "Messages": "Group", "Time left": "Group", "Error": "Group", "": "Separator",
			"Apples": "HeaderItem"},
	}[runtime.GOOS]
	if want == nil {
		t.Skipf("no roles to expect on %s", runtime.GOOS)
	}
	for label, role := range want {
		eventually(t, label+" as "+role, func() bool {
			nodes, _ := accessibility(w)
			for _, n := range nodes {
				if n.label == label && n.role == role {
					return true
				}
			}
			return false
		})
	}
}

// TestAccessDeletionInsertion reads, as screen readers do, the roles
// Core-AAM gives ARIA's deletion and insertion, an old price struck out
// and the new one, on each platform.
func TestAccessDeletionInsertion(t *testing.T) {
	var frames atomic.Int32
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Row(c).Fill().Padding(20).Gap(6).Children(func() {
			ui.Text(c, "$40").Strikethrough().Role(ui.RoleDeletion)
			ui.Text(c, "$30").Role(ui.RoleInsertion)
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Deletion and insertion", Width: 300, Height: 200, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	if _, ok := accessibility(w); !ok {
		t.Skip("accessibility automation not available on this platform")
	}
	want := map[string]map[string]string{
		"darwin":  {"$40": "AXGroup/AXDeleteStyleGroup", "$30": "AXGroup/AXInsertStyleGroup"},
		"linux":   {"$40": "content deletion", "$30": "content insertion"},
		"windows": {"$40": "Text", "$30": "Text"},
	}[runtime.GOOS]
	if want == nil {
		t.Skipf("no roles to expect on %s", runtime.GOOS)
	}
	for label, role := range want {
		eventually(t, label+" as "+role, func() bool {
			nodes, _ := accessibility(w)
			for _, n := range nodes {
				if n.label == label && n.role == role {
					return true
				}
			}
			return false
		})
	}
}
