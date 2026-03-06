package internal

import (
	"sync/atomic"
	"testing"
	"time"
)

const testDebounceDuration = 50 * time.Millisecond

func TestDebouncerSingleTrigger(t *testing.T) {
	var count atomic.Int32
	d := NewDebouncer(testDebounceDuration)

	d.Trigger(func() {
		count.Add(1)
	})

	// Wait well past the debounce window.
	time.Sleep(testDebounceDuration * 3)

	got := count.Load()
	if got != 1 {
		t.Errorf("expected callback to fire exactly once, got %d", got)
	}
}

func TestDebouncerRapidTriggers(t *testing.T) {
	var count atomic.Int32
	d := NewDebouncer(testDebounceDuration)

	// Fire 5 rapid triggers within 10ms — only the last should survive.
	for i := 0; i < 5; i++ {
		d.Trigger(func() {
			count.Add(1)
		})
		time.Sleep(2 * time.Millisecond)
	}

	// Wait for the debounce to settle.
	time.Sleep(testDebounceDuration * 3)

	got := count.Load()
	if got != 1 {
		t.Errorf("expected callback to fire exactly once after rapid triggers, got %d", got)
	}
}

func TestDebouncerStop(t *testing.T) {
	var count atomic.Int32
	d := NewDebouncer(testDebounceDuration)

	d.Trigger(func() {
		count.Add(1)
	})

	// Stop before the timer fires.
	d.Stop()

	// Wait well past the debounce window.
	time.Sleep(testDebounceDuration * 3)

	got := count.Load()
	if got != 0 {
		t.Errorf("expected callback to never fire after Stop(), got %d", got)
	}
}
