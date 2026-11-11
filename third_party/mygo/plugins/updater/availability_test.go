package updater

import (
	"context"
	"errors"
	"testing"
	"time"
)

func activateAvailabilityHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t, Options{})
	active.Store(h.u)
	t.Cleanup(func() { active.Store(nil) })
	return h
}

func TestAvailableVersionChangesNotify(t *testing.T) {
	h := activateAvailabilityHarness(t)
	if AvailableVersion() != "" {
		t.Fatal("a new updater already has an available version")
	}
	originalNow := now
	now = func() time.Time { return time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = originalNow })
	changes := make(chan string, 8)
	off := OnChange(func() { changes <- AvailableVersion() })
	defer off()
	wantChange := func(want string) {
		t.Helper()
		select {
		case got := <-changes:
			if got != want {
				t.Fatalf("available version = %q, want %q", got, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("availability change was not reported")
		}
	}
	h.u.checked(nil, "2.0.0", false)
	wantChange("2.0.0")
	h.u.checked(errors.New("offline"), "", false)
	wantChange("")
	h.u.checked(nil, "2.0.0", false)
	wantChange("2.0.0") // LastCheck is unchanged; availability must still notify.
	h.u.checked(nil, "", false)
	wantChange("")
}

func TestAvailableVersionFollowsSkipAndLater(t *testing.T) {
	h := activateAvailabilityHarness(t)
	h.result = func(context.Context) (*release, error) { return newRelease("2.0.0", nil), nil }
	done := h.begin(true)
	s := h.shown()
	v := waitView(t, s, "A new version of")
	if AvailableVersion() != "2.0.0" {
		t.Fatal("a detected update did not expose its version")
	}
	s.Respond(v.Prompt, actionSkip, false)
	waitDone(t, done)
	if AvailableVersion() != "" {
		t.Fatal("a skipped update remains available")
	}
	waitDone(t, h.begin(false))
	h.noWindow()
	if AvailableVersion() != "" {
		t.Fatal("the automatic check exposed a skipped update")
	}
	h.result = func(context.Context) (*release, error) { return newRelease("3.0.0", nil), nil }
	done = h.begin(false)
	s = h.shown()
	v = waitView(t, s, "A new version of")
	s.Respond(v.Prompt, actionLater, false)
	waitDone(t, done)
	if AvailableVersion() != "3.0.0" {
		t.Fatal("reminding later lost the detected version")
	}
	h.result = func(context.Context) (*release, error) { return nil, nil }
	waitDone(t, h.begin(false))
	if AvailableVersion() != "" {
		t.Fatal("an up-to-date result retained the previous version")
	}
}

func TestAvailableVersionRetainedAfterInstallation(t *testing.T) {
	h := activateAvailabilityHarness(t)
	h.result = func(context.Context) (*release, error) { return newRelease("2.0.0", nil), nil }
	done := h.begin(true)
	s := h.shown()
	v := waitView(t, s, "A new version of")
	s.Respond(v.Prompt, actionInstall, false)
	v = waitView(t, s, "Ready to Relaunch")
	s.Respond(v.Prompt, actionLater, false)
	waitDone(t, done)
	if AvailableVersion() != "2.0.0" {
		t.Fatal("the installed update needs to remain discoverable until relaunch")
	}
}
