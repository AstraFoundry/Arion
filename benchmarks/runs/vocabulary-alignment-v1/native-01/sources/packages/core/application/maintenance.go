package application

import "sync"

// TryMaintenance atomically excludes new sessions only after input and its
// asynchronous teardown have finished. The caller must release on failure;
// a successful process restart retains the lease until exit.
func (c *Coordinator) TryMaintenance() (release func(), acquired bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.maintenance || c.workers.Load() != 0 || (c.current != nil && !c.current.session.Terminal()) {
		return nil, false
	}
	c.maintenance = true
	var once sync.Once
	return func() { once.Do(func() { c.mu.Lock(); c.maintenance = false; c.mu.Unlock() }) }, true
}

// spawn retains resource ownership through asynchronous session teardown.
// Calls originate from an active session or a worker that already owns a count.
func (c *Coordinator) spawn(fn func()) {
	c.workers.Add(1)
	go func() { defer c.workers.Add(-1); fn() }()
}
