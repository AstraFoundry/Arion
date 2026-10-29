package application

import (
	"arion.local/core/domain"
	"testing"
	"time"
)

func TestMaintenanceExcludesNewInput(t *testing.T) {
	c, _, events := setup(t, &fakeRecognizer{}, nil, Options{})
	release, ok := c.TryMaintenance()
	if !ok {
		t.Fatal("idle lease refused")
	}
	c.Start()
	c.Toggle()
	if c.Snapshot().Active {
		t.Fatal("input started during maintenance")
	}
	if _, ok = c.TryMaintenance(); ok {
		t.Fatal("overlapping maintenance lease")
	}
	release()
	release()
	c.Start()
	events.wait(t, stateIs(domain.Capturing))
	if _, ok = c.TryMaintenance(); ok {
		t.Fatal("lease acquired during capture")
	}
	c.Cancel()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if release, ok = c.TryMaintenance(); ok {
			release()
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("cancelled input did not release resources")
}

func TestMaintenanceWaitsForCancelledWorker(t *testing.T) {
	c, _, _ := setup(t, &fakeRecognizer{}, nil, Options{})
	done := make(chan struct{})
	entered := make(chan struct{})
	c.spawn(func() { close(entered); <-done })
	<-entered
	if _, ok := c.TryMaintenance(); ok {
		t.Fatal("lease acquired while worker owns resources")
	}
	close(done)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if release, ok := c.TryMaintenance(); ok {
			release()
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("worker did not release maintenance ownership")
}
