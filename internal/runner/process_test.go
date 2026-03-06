package runner

import (
	"os/exec"
	"testing"
	"time"
)

func TestIsRunningTrue(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	if !IsRunning(cmd) {
		t.Error("expected IsRunning to return true for a running process")
	}

	// Clean up.
	if err := KillProcessGroup(cmd); err != nil {
		t.Errorf("cleanup KillProcessGroup error: %v", err)
	}
}

func TestIsRunningFalseAfterExit(t *testing.T) {
	cmd := exec.Command("true")
	SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}

	if IsRunning(cmd) {
		t.Error("expected IsRunning to return false after process exited")
	}
}

func TestIsRunningNilCmd(t *testing.T) {
	if IsRunning(nil) {
		t.Error("expected IsRunning(nil) to return false")
	}
}

func TestIsRunningNilProcess(t *testing.T) {
	cmd := &exec.Cmd{}
	if IsRunning(cmd) {
		t.Error("expected IsRunning with nil Process to return false")
	}
}

func TestKillProcessGroupTerminates(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	start := time.Now()
	err := KillProcessGroup(cmd)
	elapsed := time.Since(start)

	if elapsed > 4*time.Second {
		t.Errorf("KillProcessGroup took %v, expected < 4s", elapsed)
	}

	// sleep responds to SIGTERM, so we expect a clean exit.
	if err != nil {
		t.Errorf("expected nil error for SIGTERM kill of sleep, got: %v", err)
	}

	if IsRunning(cmd) {
		t.Error("expected process to be dead after KillProcessGroup")
	}
}

func TestKillProcessGroupAlreadyExited(t *testing.T) {
	cmd := exec.Command("true")
	SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}

	// Killing an already-exited process should not panic.
	_ = KillProcessGroup(cmd)
}

func TestKillProcessGroupNil(t *testing.T) {
	if err := KillProcessGroup(nil); err != nil {
		t.Errorf("expected nil error for nil cmd, got: %v", err)
	}
}

func TestKillProcessGroupNilProcess(t *testing.T) {
	cmd := &exec.Cmd{}
	if err := KillProcessGroup(cmd); err != nil {
		t.Errorf("expected nil error for cmd with nil Process, got: %v", err)
	}
}

func TestSetProcessGroup(t *testing.T) {
	cmd := exec.Command("true")
	SetProcessGroup(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be set")
	}
	if !cmd.SysProcAttr.Setpgid {
		t.Error("expected Setpgid to be true")
	}
}

func TestKillProcessGroupStubbornProcess(t *testing.T) {
	// Start a process that traps SIGTERM and ignores it.
	// This forces the SIGKILL path after the 3-second timeout.
	cmd := exec.Command("sh", "-c", "trap '' TERM; sleep 30")
	SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start stubborn process: %v", err)
	}

	start := time.Now()
	err := KillProcessGroup(cmd)
	elapsed := time.Since(start)

	// Should take roughly 3 seconds (SIGTERM timeout) then SIGKILL.
	if elapsed < 2*time.Second {
		t.Errorf("expected ~3s for stubborn process, but took only %v", elapsed)
	}
	if elapsed > 5*time.Second {
		t.Errorf("KillProcessGroup took %v, expected < 5s", elapsed)
	}

	if IsRunning(cmd) {
		t.Error("expected stubborn process to be dead after SIGKILL")
	}

	// err may be non-nil (signal: killed) — that's expected.
	_ = err
}
