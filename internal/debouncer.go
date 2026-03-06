package internal

import (
	"sync"
	"time"
)

// Debouncer coalesces rapid successive calls into a single delayed invocation.
// It is goroutine-safe and does not spawn any goroutines of its own — the
// deferred function runs in the goroutine created by time.AfterFunc.
type Debouncer struct {
	duration time.Duration
	mu       sync.Mutex
	timer    *time.Timer
}

// NewDebouncer creates a Debouncer that waits for the given duration of
// inactivity before firing. A typical value for hot-reload is 150ms.
func NewDebouncer(duration time.Duration) *Debouncer {
	return &Debouncer{duration: duration}
}

// Trigger schedules fn to run after the debounce duration. If Trigger is
// called again before the duration elapses, the previous pending call is
// cancelled and the timer resets. The function fn will execute in its own
// goroutine (via time.AfterFunc).
func (d *Debouncer) Trigger(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.duration, fn)
}

// Stop cancels any pending debounced call. It is safe to call Stop multiple
// times or on a Debouncer that has no pending timer.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
}
