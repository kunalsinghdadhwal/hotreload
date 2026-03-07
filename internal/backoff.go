package internal

import (
	"log/slog"
	"sync"
	"time"
)

// Backoff implements exponential backoff for crash-loop detection.
// When a process crashes repeatedly within a short window (10s), each
// successive restart waits progressively longer (1s, 2s, 4s … 30s)
// before retrying.
type Backoff struct {
	mu         sync.Mutex
	crashes    int
	lastCrash  time.Time
	maxBackoff time.Duration
}

// NewBackoff creates a Backoff with a maximum backoff of 30 seconds.
func NewBackoff() *Backoff {
	return &Backoff{maxBackoff: 30 * time.Second}
}

// RecordCrash records a crash event and returns the duration the caller
// should wait before restarting. The first crash in a series never incurs
// a wait; repeated crashes within 10 seconds trigger exponential backoff.
func (b *Backoff) RecordCrash() time.Duration {
	return b.recordCrash(time.Now())
}

// recordCrash is the internal implementation of RecordCrash, accepting an
// explicit timestamp to allow deterministic testing without real clocks.
func (b *Backoff) recordCrash(now time.Time) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.lastCrash.IsZero() && now.Sub(b.lastCrash) > 10*time.Second {
		b.crashes = 0
	}
	b.crashes++
	b.lastCrash = now

	var wait time.Duration
	if b.crashes > 1 {
		wait = time.Duration(1<<uint(b.crashes-2)) * time.Second
		if wait > b.maxBackoff {
			wait = b.maxBackoff
		}
	}

	slog.Warn("crash recorded", "crash_count", b.crashes, "backoff", wait)
	return wait
}

// Reset clears the crash counter, typically called after a successful start.
func (b *Backoff) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.crashes = 0
	b.lastCrash = time.Time{}
	slog.Info("backoff reset \u2014 server started successfully")
}

// CrashCount returns the current crash count under mutex.
func (b *Backoff) CrashCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.crashes
}
