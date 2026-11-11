package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestChat(t *testing.T) {
	app := &chat{}
	tt := ui.NewTester(app.view, 820, 720)
	tt.Type("How do I cache?")
	if err := tt.Click("Send"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("How do I cache?") {
		t.Fatalf("no message in %q", tt.Texts())
	}
	// Without a window, the reply comes whole.
	if !tt.HasText("A few things to note:") || !tt.HasText("Copy") {
		t.Errorf("no reply in %q", tt.Texts())
	}
}
