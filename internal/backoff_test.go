package internal

import (
	"testing"
	"time"
)

func TestFirstCrashReturnsZero(t *testing.T) {
	b := NewBackoff()
	now := time.Now()
	wait := b.recordCrash(now)
	if wait != 0 {
		t.Errorf("first crash: want 0, got %v", wait)
	}
}

func TestSecondCrashReturns1s(t *testing.T) {
	b := NewBackoff()
	now := time.Now()
	b.recordCrash(now)
	wait := b.recordCrash(now.Add(1 * time.Second))
	if wait != 1*time.Second {
		t.Errorf("second crash: want 1s, got %v", wait)
	}
}

func TestThirdCrashReturns2s(t *testing.T) {
	b := NewBackoff()
	now := time.Now()
	b.recordCrash(now)
	b.recordCrash(now.Add(1 * time.Second))
	wait := b.recordCrash(now.Add(2 * time.Second))
	if wait != 2*time.Second {
		t.Errorf("third crash: want 2s, got %v", wait)
	}
}

func TestCrashAfter10sResets(t *testing.T) {
	b := NewBackoff()
	now := time.Now()
	b.recordCrash(now)
	b.recordCrash(now.Add(1 * time.Second))
	// Crash more than 10s after the last crash — should reset counter.
	wait := b.recordCrash(now.Add(12 * time.Second))
	if wait != 0 {
		t.Errorf("crash after 10s silence: want 0, got %v", wait)
	}
	if b.CrashCount() != 1 {
		t.Errorf("crash count after reset: want 1, got %d", b.CrashCount())
	}
}

func TestResetZeroesCrashCount(t *testing.T) {
	b := NewBackoff()
	b.recordCrash(time.Now())
	b.recordCrash(time.Now())
	b.Reset()
	if b.CrashCount() != 0 {
		t.Errorf("after Reset: want 0, got %d", b.CrashCount())
	}
}

func TestBackoffCappedAt30s(t *testing.T) {
	b := NewBackoff()
	now := time.Now()
	var lastWait time.Duration
	for i := 0; i < 10; i++ {
		lastWait = b.recordCrash(now.Add(time.Duration(i) * time.Millisecond))
	}
	if lastWait != 30*time.Second {
		t.Errorf("expected 30s cap after many crashes, got %v", lastWait)
	}
	if lastWait > 30*time.Second {
		t.Errorf("backoff exceeded max: got %v, want <= 30s", lastWait)
	}
}
