package runner

import (
	"os/exec"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewExecutor(t *testing.T) {
	e := NewExecutor("echo hello", nil)
	if e.execCmd != "echo hello" {
		t.Errorf("expected execCmd %q, got %q", "echo hello", e.execCmd)
	}
}

func TestExecutorStartAndStop(t *testing.T) {
	var exited atomic.Bool
	e := NewExecutor("sleep 30", func(err error) {
		exited.Store(true)
	})

	if err := e.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	cmd := e.Current()
	if cmd == nil {
		t.Fatal("expected non-nil Current() after Start")
	}
	if !IsRunning(cmd) {
		t.Error("expected process to be running after Start")
	}

	if err := e.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Wait for onExit callback to fire.
	deadline := time.After(3 * time.Second)
	for !exited.Load() {
		select {
		case <-deadline:
			t.Fatal("onExit callback was never invoked")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	if e.Current() != nil {
		t.Error("expected Current() to be nil after Stop")
	}
}

func TestExecutorStopWhenNotRunning(t *testing.T) {
	e := NewExecutor("sleep 30", nil)
	// Stop should be safe when nothing is running.
	if err := e.Stop(); err != nil {
		t.Errorf("Stop on idle executor returned error: %v", err)
	}
}

func TestExecutorStartEmptyCommand(t *testing.T) {
	e := NewExecutor("", nil)
	err := e.Start()
	if err == nil {
		t.Fatal("expected error for empty exec command")
	}
}

func TestExecutorStartNonexistentBinary(t *testing.T) {
	e := NewExecutor("/nonexistent/binary", nil)
	err := e.Start()
	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}
}

func TestExecutorRestart(t *testing.T) {
	e := NewExecutor("sleep 30", nil)

	if err := e.Start(); err != nil {
		t.Fatalf("initial Start failed: %v", err)
	}
	firstPid := e.Current().Process.Pid

	if err := e.Restart(); err != nil {
		t.Fatalf("Restart failed: %v", err)
	}

	cmd := e.Current()
	if cmd == nil {
		t.Fatal("expected non-nil Current() after Restart")
	}
	if cmd.Process.Pid == firstPid {
		t.Error("expected different PID after Restart")
	}

	e.Stop()
}

func TestExecutorRestartWhenNotRunning(t *testing.T) {
	e := NewExecutor("sleep 30", nil)
	// Restart from idle — should just start.
	if err := e.Restart(); err != nil {
		t.Fatalf("Restart from idle failed: %v", err)
	}
	if e.Current() == nil {
		t.Error("expected process running after Restart from idle")
	}
	e.Stop()
}

func TestExecutorCurrentNil(t *testing.T) {
	e := NewExecutor("echo hi", nil)
	if e.Current() != nil {
		t.Error("expected nil Current() before Start")
	}
}

func TestExecutorOnExitCalledOnCrash(t *testing.T) {
	var exitErr atomic.Value
	e := NewExecutor("false", func(err error) {
		exitErr.Store(err)
	})

	if err := e.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// "false" exits immediately with code 1.
	deadline := time.After(3 * time.Second)
	for exitErr.Load() == nil {
		select {
		case <-deadline:
			t.Fatal("onExit was never called after process crash")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	err, _ := exitErr.Load().(*exec.ExitError)
	if err == nil {
		t.Error("expected ExitError from crashed process")
	}
}

func TestExecutorOnExitNilCallback(t *testing.T) {
	// Ensure no panic when onExit is nil.
	e := NewExecutor("true", nil)
	if err := e.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	// "true" exits immediately — just let the goroutine run.
	time.Sleep(100 * time.Millisecond)
}
