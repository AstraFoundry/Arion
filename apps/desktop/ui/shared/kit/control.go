package kit

import "github.com/egoist/mygo/ui"

// ControlFrame gives value controls the shared settings-field geometry and surface.
func ControlFrame(c *ui.Context, control ui.Element) ui.Element {
	t := c.Theme()
	return control.Padding(t.Space(1.5), t.Space(3.5)).Gap(t.Space(1.5)).Radius(t.Radius).
		AlignItems(ui.Center).MinWidth(t.Space(35)).Border(1, t.Border).TextColor(t.Text).Background(t.Surface)
}
