package internal

import (
	"sync"
	"time"
)

const (
	backoffBase  = 1 * time.Second
	backoffMax   = 30 * time.Second
	backoffMulti = 2
	crashWindow  = 10 * time.Second
)

// Backoff implements exponential backoff for crash-loop detection.
// When a process crashes repeatedly within a short window, each
// successive restart waits progressively longer before retrying.
type Backoff struct {
	mu        sync.Mutex
	crashes   int
	lastCrash time.Time
}

// NewBackoff creates a Backoff with zero state.
func NewBackoff() *Backoff {
	return &Backoff{}
}

// RecordCrash records a crash event and returns the duration the caller
// should wait before restarting. If crashes are spaced far apart (outside
// the crash window), the counter resets and no wait is applied.
func (b *Backoff) RecordCrash() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if !b.lastCrash.IsZero() && now.Sub(b.lastCrash) > crashWindow {
		b.crashes = 0
	}
	b.lastCrash = now
	b.crashes++

	if b.crashes <= 1 {
		return 0
	}

	wait := backoffBase
	for i := 2; i < b.crashes; i++ {
		wait *= time.Duration(backoffMulti)
		if wait > backoffMax {
			wait = backoffMax
			break
		}
	}
	return wait
}

// Reset clears the crash counter, typically called after a successful start.
func (b *Backoff) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.crashes = 0
	b.lastCrash = time.Time{}
}
