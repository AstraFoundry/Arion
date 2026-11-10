package bootstrap

import "github.com/egoist/mygo"

type clipboardAccessWindow interface {
	Bounds() mygo.Rectangle
	SetBounds(mygo.Rectangle)
	SetOpacity(float64)
	Focus()
}

// The first visible frame must already have the measured content height.
// A resize schedules another layout; reveal only after that layout agrees.
type clipboardAccessLayout struct {
	height   int
	revealed bool
}

func (s *clipboardAccessLayout) apply(window clipboardAccessWindow, height int) {
	if height <= 0 {
		return
	}
	s.height = height
	bounds := window.Bounds()
	if bounds.Height != height {
		bounds.Y += (bounds.Height - height) / 2
		bounds.Height = height
		window.SetBounds(bounds)
		return
	}
	if !s.revealed {
		s.revealed = true
		window.SetOpacity(1)
		window.Focus()
	}
}
