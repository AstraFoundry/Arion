package ui

import (
	"image/color"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// accessLabels returns the labels of the nodes of the tree.
func accessLabels(tree *platform.AccessTree) []string {
	var out []string
	for _, n := range tree.Nodes {
		out = append(out, n.Label)
	}
	return out
}

// A card behind the front one paints, and takes the pointer and the
// keyboard, but neither it nor what is inside it reads to assistive
// technology, which cannot act on them either; a copy of a text fading
// out does not name its button a second time.
func TestAccessHidden(t *testing.T) {
	hide := false
	var opens int
	tt := coreNewTester(func(c *context) {
		coreColumn(c).AlignItems(Start).Gap(4).Children(func() {
			coreText(c, "Front")
			coreColumn(c).Size(120, 80).Background(RGB(255, 0, 0)).Label("card").AccessHidden(hide).Children(func() {
				coreText(c, "Behind")
				coreBox(c).Children(func() {
					if coreButton(c, "Open").Clicked() {
						opens++
					}
				})
			})
			coreButtonBase(c).Children(func() {
				coreText(c, "Save")
				coreText(c, "Saved").Absolute().Opacity(0.5).AccessHidden(hide)
			})
		})
	}, 300, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	accessNode(t, tt.h.access, platform.RoleGroup, "card")
	accessNode(t, tt.h.access, platform.RoleText, "Behind")
	open := accessNode(t, tt.h.access, platform.RoleButton, "Open")
	if n := accessNode(t, tt.h.access, platform.RoleButton, "Save"); n.Label != "Save Saved" {
		t.Errorf("shown, the copy does not name its button with the other: %q", n.Label)
	}

	hide = true
	tt.Frame()
	for _, n := range tt.h.access.Nodes {
		switch n.Label {
		case "card", "Behind", "Open":
			t.Errorf("%q, inside the element hidden from assistive technology, is in its tree: %q", n.Label, accessLabels(tt.h.access))
		}
		if n.ID == open.ID {
			t.Errorf("the button inside it is in the tree as %+v", n)
		}
	}
	accessNode(t, tt.h.access, platform.RoleText, "Front")
	if n := accessNode(t, tt.h.access, platform.RoleButton, "Save"); n.Label != "Save" {
		t.Errorf("the button is named %q with the copy hidden", n.Label)
	}
	// It cannot act on what it does not see, as an action sent before the
	// tree went would.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: open.ID, Action: platform.AccessPress})
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: open.ID, Action: platform.AccessFocus})
	if opens != 0 || tt.rt.focused != 0 {
		t.Errorf("assistive technology acted on the hidden button: %d presses, focus %d", opens, tt.rt.focused)
	}

	// It still paints, shows its texts, and takes the pointer and Tab.
	card, ok := tt.Find("card")
	if !ok || !tt.HasText("Behind") {
		t.Fatalf("the hidden card does not show: %q", tt.Texts())
	}
	if px := tt.Image().RGBAAt(int(card.X+card.W-4), int(card.Y+card.H-4)); px != (color.RGBA{255, 0, 0, 255}) {
		t.Errorf("the hidden card paints %v, not its background", px)
	}
	tt.Click("Open")
	if opens != 1 {
		t.Errorf("a click on the hidden button: %d presses", opens)
	}
	tt.rt.focused = 0
	tt.Key(0, KeyTab)
	if !tt.Focused("Open") {
		t.Error("Tab does not stop at the hidden button, as it does without IgnoreInput")
	}
	if tt.h.access.Focus == open.ID {
		t.Error("the tree's focus is on a button it does not have")
	}

	// Shown again, it is back, with the same node.
	hide = false
	tt.Frame()
	if n := accessNode(t, tt.h.access, platform.RoleButton, "Open"); n.ID != open.ID {
		t.Errorf("the button came back as node %d, not %d", n.ID, open.ID)
	}
}

// An inline text hidden from assistive technology, and the texts inside
// it, give nothing to the name of its paragraph or of what the paragraph
// names, as a node hidden with aria-hidden gives nothing to a name
// (accname 1.2, step 2A), while the paragraph still shows them.
func TestAccessHiddenInline(t *testing.T) {
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreText(c, "Total ").Children(func() {
				coreText(c, "(icon)").AccessHidden(true)
				coreText(c, " 42")
			})
			coreBox(c).Role(RoleButton).Focusable().Children(func() {
				coreText(c, "Save").Children(func() {
					coreText(c, " ⌘S").AccessHidden(true)
				})
			})
			coreText(c, "See ").Children(func() {
				coreText(c, "docs").Focusable().AccessHidden(true).Children(func() {
					coreText(c, " here")
				})
				coreText(c, " below")
			})
			coreText(c, "A ").Children(func() {
				coreText(c, "b").Children(func() {
					coreText(c, " c").AccessHidden(true)
				})
				coreText(c, " d")
			})
		})
	}, 300, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	for _, name := range []string{"Total  42", "See  below", "A b d"} {
		if _, ok := tt.AccessByRole(RoleText, name); !ok {
			t.Errorf("no text named %q among %q", name, accessLabels(tt.h.access))
		}
	}
	if _, ok := tt.AccessByRole(RoleButton, "Save"); !ok {
		t.Errorf("the button is not named by its text alone: %q", accessLabels(tt.h.access))
	}
	for _, n := range tt.h.access.Nodes {
		if strings.Contains(n.Label, "docs") || strings.Contains(n.Label, "here") {
			t.Errorf("the hidden link, or the text inside it, is in the tree: %q", accessLabels(tt.h.access))
			break
		}
	}
	for _, s := range []string{"Total (icon) 42", "Save ⌘S", "See docs here below", "A b c d"} {
		if !tt.HasText(s) {
			t.Errorf("the paragraph %q does not show: %q", s, tt.Texts())
		}
	}
}

// With IgnoreInput, an element hidden from assistive technology is inert,
// as HTML's inert: it paints, but nothing reaches it or what is inside it,
// nor selects their text. Hidden from assistive technology alone, its
// text is still selected, as aria-hidden's is.
func TestAccessHiddenInert(t *testing.T) {
	ignore := true
	var clicks int
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Selectable().Children(func() {
			coreText(c, "Current")
			coreColumn(c).IgnoreInput(ignore).AccessHidden(true).Children(func() {
				coreText(c, "Leaving")
				if coreButton(c, "Back").Clicked() {
					clicks++
				}
			})
		})
	}, 300, 200)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	for _, n := range tt.h.access.Nodes {
		if n.Label == "Leaving" || n.Label == "Back" {
			t.Errorf("%q, inert, is in the tree", n.Label)
		}
	}
	selectable := func(text string) bool {
		for _, s := range tt.rt.texts {
			if s.editor != nil && s.editor.source == text {
				return true
			}
		}
		return false
	}
	if !selectable("Current") || selectable("Leaving") {
		t.Error("the inert text is selectable, or the other is not")
	}
	if !tt.HasText("Leaving") {
		t.Error("the inert text does not show")
	}
	tt.Click("Back")
	tt.Key(0, KeyTab)
	if clicks != 0 || tt.Focused("Back") {
		t.Errorf("the inert button took %d clicks, or the focus", clicks)
	}

	ignore = false
	tt.Frame()
	if !selectable("Leaving") {
		t.Error("the text hidden from assistive technology alone is not selectable")
	}
}
