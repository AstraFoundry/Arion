package clipboard

import (
	"math"
	"strings"

	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

const PreviewWidth = 320
const PreviewHeight = 220

func previewText(item runtime.Clip) string {
	if item.Payload.Kind == runtime.ClipFiles {
		return strings.Join(item.Payload.Files, "\n")
	}
	return item.Payload.Text
}

// previewHeight sizes short content without leaving a large empty card.
func previewHeight(c *ui.Context, item runtime.Clip, bitmap *ui.Bitmap) int {
	if item.Payload.Kind == runtime.ClipImage && bitmap != nil {
		w, h := bitmap.Size()
		return min(PreviewHeight, max(64, (PreviewWidth-26)*h/w+26))
	}
	_, height := c.MeasureText(PreviewWidth-26, ui.Span{Text: previewText(item), Size: theme.Body})
	return min(PreviewHeight, max(44, int(math.Ceil(float64(height)))+26))
}

// Preview shows the original content without adding delivery or management actions.
func Preview(c *ui.Context, item runtime.Clip, bitmap *ui.Bitmap) {
	tokens := theme.Apply(c)
	ui.Column(c).Fill().Padding(12).Background(tokens.Card).Border(1, tokens.CardBorder).Radius(theme.RadiusLG).Children(func() {
		if item.Payload.Kind == runtime.ClipImage && bitmap != nil {
			ui.Image(c, bitmap).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Fit(ui.Contain).Radius(theme.RadiusSM)
			return
		}
		text := previewText(item)
		ui.Text(c, text).FontSize(theme.Body).TextColor(tokens.Text).MaxLines(12)
	})
}
