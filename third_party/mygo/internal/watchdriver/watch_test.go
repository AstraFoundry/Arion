package watchdriver

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

type testDriver struct {
	wake    chan struct{}
	added   []uint64
	removed []uint64
	fail    chan error
	closed  bool
}

func (d *testDriver) Add(t platform.FileWatchTarget) error {
	d.added = append(d.added, t.ID)
	if t.ID == 0 {
		return platform.ErrUnsupported
	}
	return nil
}
func (d *testDriver) Remove(id uint64) { d.removed = append(d.removed, id) }
func (d *testDriver) Wait() ([]platform.FileWatchNotice, error) {
	select {
	case <-d.wake:
		return []platform.FileWatchNotice{{Target: 1}}, nil
	case err := <-d.fail:
		return nil, err
	}
}
func (d *testDriver) Wake() {
	select {
	case d.wake <- struct{}{}:
	default:
	}
}
func (d *testDriver) Close() { d.closed = true }

func TestWatch(t *testing.T) {
	d := &testDriver{wake: make(chan struct{}, 1), fail: make(chan error)}
	notices := make(chan platform.FileWatchNotice, 16)
	w := New(d, func(n platform.FileWatchNotice) { notices <- n })
	if err := w.Add(platform.FileWatchTarget{ID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := w.Add(platform.FileWatchTarget{ID: 0}); err != platform.ErrUnsupported {
		t.Fatalf("Add = %v", err)
	}
	w.Remove(1)
	// Commands run in order and Add returns once the driver has run it, so
	// Remove has too before the failure below, which the driver could
	// otherwise pick over the wake that Remove left.
	if err := w.Add(platform.FileWatchTarget{ID: 3}); err != nil {
		t.Fatal(err)
	}
	if n := <-notices; n.Target != 1 {
		t.Fatalf("notice %+v", n)
	}
	// An error from Wait ends the watch, and is the last notice.
	failure := errors.New("failure")
	d.fail <- failure
	for n := range notices {
		if n.Err != nil {
			if n.Err != failure {
				t.Fatalf("notice %+v", n)
			}
			break
		}
	}
	if err := w.Add(platform.FileWatchTarget{ID: 2}); err == nil {
		t.Fatal("added to a stopped watch")
	}
	w.Close()
	if !d.closed || len(d.added) != 3 || len(d.removed) != 1 {
		t.Fatalf("driver %+v", d)
	}
}
