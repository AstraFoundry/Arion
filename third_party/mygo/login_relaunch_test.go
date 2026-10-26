//go:build !js

package mygo

import (
	"testing"
	"time"
)

func TestRelaunchReportsQuitVeto(t *testing.T) {
	off := App.OnBeforeQuit(func(e *QuitEvent) { e.PreventDefault() })
	defer off()
	result := make(chan bool, 1)
	App.RelaunchWithResult(func(accepted bool) {
		if !isMainThread() {
			t.Error("relaunch result did not run on the main thread")
		}
		result <- accepted
	})
	select {
	case accepted := <-result:
		if accepted {
			t.Fatal("relaunch accepted a vetoed quit")
		}
	case <-time.After(time.Second):
		t.Fatal("relaunch did not report its veto")
	}
	if onMainValue(func() bool { return App.relaunch || App.quitting }) {
		t.Fatal("vetoed relaunch retained quit state")
	}
	select {
	case <-result:
		t.Fatal("relaunch reported twice")
	default:
	}
}
