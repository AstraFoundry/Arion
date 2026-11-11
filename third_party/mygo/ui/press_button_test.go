package ui

import "testing"

// A press tells its button (PressedButton), and Pressed, Dragged,
// DragVelocity and DragEnded follow the primary button's alone: a drag
// with the secondary or middle button moves nothing, as on the web, where
// a carousel takes a press only with event.button === 0.
func TestPressedButton(t *testing.T) {
	type seen struct {
		pressed, dragged, ended, right, clicked bool
		button                                  int
		pressing                                bool
		dx                                      float32
	}
	var s seen
	tt := NewTester(func(c *Context) {
		card := Box(c).Size(100, 100).Label("card")
		s.pressed = card.Pressed()
		var dx float32
		dx, _, s.dragged = card.Dragged()
		s.dx += dx
		if _, _, ok := card.DragEnded(); ok {
			s.ended = true
		}
		s.button, s.pressing = card.PressedButton()
		s.right = s.right || card.RightClicked()
		s.clicked = s.clicked || card.Clicked()
	}, 200, 200)
	tt.Frame()
	for _, b := range []int{0, 1, 2} {
		s = seen{}
		tt.PressButton(b, 50, 50)
		tt.Frame()
		if s.button != b || !s.pressing {
			t.Fatalf("button %d: PressedButton() = %d, %v", b, s.button, s.pressing)
		}
		primary := b == 0
		if s.pressed != primary || s.dragged != primary {
			t.Errorf("button %d: Pressed %v, Dragged %v; want %v", b, s.pressed, s.dragged, primary)
		}
		tt.Move(70, 50)
		tt.Frame()
		if want := map[bool]float32{true: 20}[primary]; s.dx != want {
			t.Errorf("button %d: dragged by %v, want %v", b, s.dx, want)
		}
		tt.ReleaseButton(b, 70, 50)
		tt.Frame()
		tt.Frame()
		if s.pressing || s.pressed || s.dragged {
			t.Errorf("button %d: let go, PressedButton %v, Pressed %v, Dragged %v", b, s.pressing, s.pressed, s.dragged)
		}
		if s.ended != primary {
			t.Errorf("button %d: DragEnded %v, want %v", b, s.ended, primary)
		}
		// The secondary button still right-clicks, and the primary clicks.
		if s.right != (b == 1) || s.clicked != primary {
			t.Errorf("button %d: RightClicked %v, Clicked %v", b, s.right, s.clicked)
		}
	}
}

// The controls move with the primary button alone: a right or middle
// drag over a slider, a split's handle or a Draggable leaves them as they
// were, and the secondary button still opens their context menus.
func TestControlsTakeThePrimaryButton(t *testing.T) {
	v, size := 50.0, float32(100)
	var off Point
	menus := 0
	tt := NewTester(func(c *Context) {
		Column(c).Fill().Children(func() {
			Slider(c, &v, 0, 100).Width(200).Label("slider")
			Box(c).Size(40, 40).Label("card").Draggable(&off, DraggableOptions{}).ContextMenu(func(m *Menu) {
				menus++
				m.Item("Remove")
			})
			Split(c, &size, func() { Box(c).Fill() }, func() { Box(c).Fill() }).Height(100).Width(300)
		})
	}, 400, 400)
	tt.Frame()
	slider, _ := tt.Find("slider")
	card, _ := tt.Find("card")
	for _, b := range []int{1, 2} {
		drag := func(x0, y0, x1, y1 float32) {
			tt.Move(x0, y0)
			tt.PressButton(b, x0, y0)
			tt.Frame()
			tt.Move((x0+x1)/2, (y0+y1)/2)
			tt.Frame()
			tt.Move(x1, y1)
			tt.Frame()
			tt.ReleaseButton(b, x1, y1)
			tt.Frame()
			tt.CloseMenu()
		}
		drag(slider.X+20, slider.Y+slider.H/2, slider.X+180, slider.Y+slider.H/2)
		if v != 50 {
			t.Errorf("button %d dragged the slider to %v", b, v)
		}
		drag(card.X+20, card.Y+20, card.X+120, card.Y+80)
		if off != (Point{}) {
			t.Errorf("button %d dragged the card to %v", b, off)
		}
		// The split's handle sits at size, below the slider and the card.
		y := card.Y + card.H + 50
		drag(size+1, y, size+101, y)
		if size != 100 {
			t.Errorf("button %d dragged the split to %v", b, size)
		}
	}
	if menus == 0 {
		t.Error("the secondary button opened no context menu over the card")
	}
	// The primary button still drags them.
	tt.Move(slider.X+20, slider.Y+slider.H/2)
	tt.Press(slider.X+20, slider.Y+slider.H/2)
	tt.Frame()
	tt.Move(slider.X+180, slider.Y+slider.H/2)
	tt.Frame()
	tt.Release(slider.X+180, slider.Y+slider.H/2)
	tt.Frame()
	if v <= 50 {
		t.Errorf("the primary button left the slider at %v", v)
	}
}

// A disabled element tells no button.
func TestPressedButtonDisabled(t *testing.T) {
	ok := true
	tt := NewTester(func(c *Context) {
		_, ok = Box(c).Size(100, 100).Disabled(true).PressedButton()
	}, 200, 200)
	tt.Frame()
	tt.PressButton(1, 50, 50)
	tt.Frame()
	if ok {
		t.Error("a disabled element tells the button pressing it")
	}
}
