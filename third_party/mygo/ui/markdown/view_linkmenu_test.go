package markdown

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestViewLinkMenu checks that Options.LinkMenu gives each link with a
// destination a context menu of its own, told which link it is, in View
// and in Stream, while the text around it, links to #a-heading, footnote
// references and a link still being written keep the menu of the
// Selectable container around the view.
func TestViewLinkMenu(t *testing.T) {
	const src = "# Top\n\nIntro text.\n\nSee [plan.md](file:///proj/plan.md), [notes](notes.md) and [up](#top).[^1]\n\n[^1]: A note.\n\nNow [half](file:///pr"
	linkMenu := []string{"Copy Path", "Reveal in Finder", "-", "Copy", "-", "Select All"}
	outer := []string{"Quote", "-", "Copy", "-", "Select All"}
	for _, stream := range []bool{false, true} {
		var asked, copied []string
		o := &Options{
			ShareSelection: true,
			Reveal:         &Reveal{Unit: AtOnce},
			LinkMenu: func(m *ui.Menu, url string) {
				asked = append(asked, url)
				if m.Item("Copy Path").Chosen() {
					copied = append(copied, url)
				}
				m.Item("Reveal in Finder")
				m.Separator()
				m.EditItems()
			},
		}
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(10).Selectable().ContextMenu(func(m *ui.Menu) {
				m.Item("Quote")
				m.Separator()
				m.EditItems()
			}).Children(func() {
				if stream {
					Stream(c, src, o)
				} else {
					View(c, src, o)
				}
			})
		}, 600, 400)
		for _, s := range []string{"plan.md", "notes"} {
			if err := tt.RightClick(s); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(tt.Menu(), linkMenu) {
				t.Errorf("stream %v: menu of %q %q, want %q", stream, s, tt.Menu(), linkMenu)
			}
			if err := tt.ChooseMenuItem("Copy Path"); err != nil {
				t.Fatal(err)
			}
		}
		if want := []string{"file:///proj/plan.md", "notes.md"}; !slices.Equal(copied, want) {
			t.Errorf("stream %v: Copy Path got %q, want %q", stream, copied, want)
		}
		for _, s := range []string{"Intro text.", "up", "[1]", "half"} {
			if s == "half" && !stream {
				continue // only Stream has a link still being written
			}
			if err := tt.RightClick(s); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(tt.Menu(), outer) {
				t.Errorf("stream %v: menu of %q %q, want %q", stream, s, tt.Menu(), outer)
			}
			tt.CloseMenu()
		}
		for _, url := range asked {
			if url != "file:///proj/plan.md" && url != "notes.md" {
				t.Errorf("stream %v: LinkMenu asked for %q", stream, url)
			}
		}
	}
}
