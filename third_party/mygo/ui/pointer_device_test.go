package ui

import (
	"errors"
	"testing"
)

// Presses and moves tell their device and pressure, as Pointer Events'
// pointerType and pressure: a mouse presses 0.5 while a button is down,
// a pen as hard as the system tells, and a pen pressing harder where it
// stands moves.
func TestPointerDevice(t *testing.T) {
	var events []InputEvent
	var pointer PointerType
	var pressure float32
	var pressing bool
	tt := NewTester(func(c *Context) {
		pad := Box(c).Size(100, 100).Label("pad").HandleInput(func(ev InputEvent) bool {
			switch ev.Kind {
			case InputPointerDown, InputPointerMove, InputPointerUp:
				events = append(events, ev)
				return true
			}
			return false
		})
		pointer, pressure, pressing = pad.PressedPointer()
	}, 200, 200)
	tt.Frame()

	// The mouse.
	tt.Move(20, 20)
	tt.Press(20, 20)
	tt.Move(30, 20)
	tt.Release(30, 20)
	want := []struct {
		kind     InputKind
		pressure float32
	}{{InputPointerMove, 0}, {InputPointerDown, 0.5}, {InputPointerMove, 0.5}, {InputPointerUp, 0}}
	if len(events) != len(want) {
		t.Fatalf("mouse: %d events, want %d: %+v", len(events), len(want), events)
	}
	for i, w := range want {
		if ev := events[i]; ev.Kind != w.kind || ev.Pointer != PointerMouse || ev.Pressure != w.pressure {
			t.Errorf("mouse event %d: kind %d, pointer %d, pressure %v; want kind %d, pressure %v", i, ev.Kind, ev.Pointer, ev.Pressure, w.kind, w.pressure)
		}
	}

	// A pen, pressing harder where it stands, then moving.
	events = nil
	tt.UsePointer(PointerPen, 0.25)
	tt.Press(40, 40)
	tt.Frame()
	if !pressing || pointer != PointerPen || pressure != 0.25 {
		t.Errorf("pen down: PressedPointer() = %d, %v, %v", pointer, pressure, pressing)
	}
	tt.UsePointer(PointerPen, 0.75)
	tt.Move(40, 40)
	tt.Frame()
	if !pressing || pressure != 0.75 {
		t.Errorf("pen pressing harder: PressedPointer() = %d, %v, %v", pointer, pressure, pressing)
	}
	tt.Move(40, 40) // the same pressure, where it stood: no move
	tt.Release(40, 40)
	tt.Frame()
	if pressing {
		t.Error("the pen let go, and PressedPointer still reports a press")
	}
	// It comes over where it presses, pressing nothing yet.
	kinds := []InputKind{InputPointerMove, InputPointerDown, InputPointerMove, InputPointerUp}
	pressures := []float32{0, 0.25, 0.75, 0}
	if len(events) != len(kinds) {
		t.Fatalf("pen: %d events, want %d: %+v", len(events), len(kinds), events)
	}
	for i, ev := range events {
		if ev.Kind != kinds[i] || ev.Pointer != PointerPen || ev.Pressure != pressures[i] {
			t.Errorf("pen event %d: kind %d, pointer %d, pressure %v", i, ev.Kind, ev.Pointer, ev.Pressure)
		}
	}

	// A finger, whose touchscreen tells no pressure: 0.5, as a mouse's.
	events = nil
	tt.UsePointer(PointerTouch, 0)
	tt.Press(60, 60)
	tt.Frame()
	if n := len(events); pointer != PointerTouch || pressure != 0.5 || n == 0 || events[n-1].Pointer != PointerTouch || events[n-1].Pressure != 0.5 {
		t.Errorf("touch: PressedPointer() = %d, %v; events %+v", pointer, pressure, events)
	}
	tt.Release(60, 60)
}

// WriteClipboardThen tells whether the text got on the clipboard, a
// moment later, and the view builds anew with it.
func TestWriteClipboardThen(t *testing.T) {
	var outcome error
	calls := 0
	tt := NewTester(func(c *Context) {
		if Button(c, "Copy").Clicked() {
			c.WriteClipboardThen("the link", func(err error) {
				outcome = err
				calls++
			})
		}
		switch {
		case calls > 0 && outcome == nil:
			Text(c, "Copied")
		case calls > 0:
			Text(c, "Failed")
		}
	}, 200, 100)
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || outcome != nil || tt.Clipboard() != "the link" || !tt.HasText("Copied") {
		t.Fatalf("copied: %d calls, error %v, clipboard %q, texts %q", calls, outcome, tt.Clipboard(), tt.Texts())
	}
	tt.SetClipboard("")
	refused := errors.New("refused")
	tt.FailClipboard(refused)
	tt.Click("Copy")
	if calls != 2 || outcome != refused || tt.Clipboard() != "" || !tt.HasText("Failed") {
		t.Errorf("refused: %d calls, error %v, clipboard %q, texts %q", calls, outcome, tt.Clipboard(), tt.Texts())
	}
}
