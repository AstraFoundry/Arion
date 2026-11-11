package ui

import "testing"

func TestPlainCallbackNil(t *testing.T) {
	for _, tc := range []struct {
		name    string
		install func(Element)
	}{
		{"Draw", func(e Element) { e.Draw(nil) }},
		{"DrawOver", func(e Element) { e.DrawOver(nil) }},
		{"DrawObjects", func(e Element) { e.DrawObjects(nil) }},
		{"HandleInput", func(e Element) { e.HandleInput(nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := NewTester(func(c *Context) {
				element := Box(c).Size(10, 10)
				tc.install(element)
				n := element.node()
				if n.paintFn != nil || n.paintAfterFn != nil || n.objectFn != nil || n.inputFn != nil {
					t.Error("nil callback became a non-nil handler")
				}
			}, 20, 20)
			defer tt.rt.close()
			tt.Frame()
			tt.Move(5, 5)
		})
	}
}

func TestPlainCallbackRegistrationAllocs(t *testing.T) {
	draw := func(*Painter, Rect) {}
	input := func(InputEvent) bool { return false }
	tt := NewTester(func(c *Context) {
		element := Box(c).Size(10, 10)
		allocs := testing.AllocsPerRun(100, func() { element.Draw(draw).DrawOver(draw).HandleInput(input) })
		if allocs != 0 {
			t.Errorf("registering retained callbacks allocated %g times", allocs)
		}
	}, 20, 20)
	defer tt.rt.close()
}

// BenchmarkPlainCallbackRegistration registers three handlers retained by an
// existing element during its build pass, isolating facade wrapping from rendering.
func BenchmarkPlainCallbackRegistration(b *testing.B) {
	draw := func(*Painter, Rect) {}
	input := func(InputEvent) bool { return false }
	tt := NewTester(func(c *Context) {
		element := Box(c).Size(10, 10)
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			element.Draw(draw).DrawOver(draw).HandleInput(input)
		}
	}, 20, 20)
	defer tt.rt.close()
}

// BenchmarkPlainCallbackFrame builds and renders a box with three retained
// callbacks, or the same box without them, to measure whole-frame overhead.
func BenchmarkPlainCallbackFrame(b *testing.B) {
	for _, callbacks := range []bool{false, true} {
		name := "Box"
		if callbacks {
			name = "Callbacks"
		}
		b.Run(name, func(b *testing.B) {
			draw := func(*Painter, Rect) {}
			input := func(InputEvent) bool { return false }
			tt := NewTester(func(c *Context) {
				e := Box(c).Size(10, 10)
				if callbacks {
					e.Draw(draw).DrawOver(draw).HandleInput(input)
				}
			}, 20, 20)
			defer tt.rt.close()
			for range 10 {
				tt.Frame()
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				tt.Frame()
			}
		})
	}
}
