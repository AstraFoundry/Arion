package iso

import (
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Ink is a figure's paper and its one lit part, from the Sunbeam tokens:
// paper from Card, shading toward Shade, lines from Faint and CardBorder,
// the lit part from Accent and AccentInk.
type Ink struct {
	Body, Lit Palette
	// Top is the paper of faces looking up and Edge the outline ink.
	Top, Edge ui.Color
	// Halo is the soft shadow under a solid standing on the floor.
	Halo ui.Color
	// Shade and Well are how far sides and recesses darken toward Shade.
	Shade, Well float32
}

// InkOf returns the inks for t's appearance.
func InkOf(t theme.Tokens) Ink {
	shade, well, top, edge, halo := float32(0.16), float32(0.1), t.Card, t.Faint, float32(0.08)
	if t.Dark {
		shade, well, top, edge, halo = 0.5, 0.5, t.Card.Mix(t.Text, 0.05), t.Faint.Mix(t.Card, 0.35), 0.22
	}
	return Ink{
		Body: Palette{Top: top, Shadow: t.Card.Mix(t.Shade, shade), Lit: t.Card.Mix(t.Shade, shade/4), Edge: edge, Crease: t.CardBorder},
		Lit:  Palette{Top: top.Mix(t.Accent, 0.24), Shadow: t.Card.Mix(t.Accent, 0.3).Mix(t.Shade, shade), Lit: t.Card.Mix(t.Accent, 0.3), Edge: t.AccentInk, Crease: t.AccentInk.Alpha(0.5)},
		Top:  top, Edge: edge, Halo: t.Shade.Alpha(halo),
		Shade: shade, Well: well,
	}
}

// Mix blends every tone of p toward q by t.
func (p Palette) Mix(q Palette, t float32) Palette {
	return Palette{Top: p.Top.Mix(q.Top, t), Shadow: p.Shadow.Mix(q.Shadow, t), Lit: p.Lit.Mix(q.Lit, t), Edge: p.Edge.Mix(q.Edge, t), Crease: p.Crease.Mix(q.Crease, t)}
}

// Alpha fades every tone of p to a.
func (p Palette) Alpha(a float32) Palette {
	return Palette{Top: p.Top.Alpha(a), Shadow: p.Shadow.Alpha(a), Lit: p.Lit.Alpha(a), Edge: p.Edge.Alpha(a), Crease: p.Crease.Alpha(a)}
}
