package runner

import (
	"log/slog"
	"os/exec"
	"syscall"
	"time"
)

// killTimeout is the duration to wait for a process to exit after SIGTERM
// before escalating to SIGKILL.
const killTimeout = 3 * time.Second

// SetProcessGroup configures the command to run in its own process group.
// This ensures that KillProcessGroup can terminate the process and all of
// its children as a unit.
func SetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// KillProcessGroup sends SIGTERM to the entire process group of cmd, waits
// up to 3 seconds for a clean exit, and escalates to SIGKILL if the process
// is still alive. It always calls cmd.Wait to reap the zombie process.
//
// Returns nil if the process was not running or exited cleanly after SIGTERM.
func KillProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pid := cmd.Process.Pid

	slog.Info("sending SIGTERM to process group", "pid", pid)
	termErr := syscall.Kill(-pid, syscall.SIGTERM)
	if termErr != nil {
		slog.Warn("SIGTERM failed", "pid", pid, "error", termErr)
	}

	// Wait for the process to exit in a separate goroutine so we can
	// enforce a timeout.
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		slog.Info("process exited cleanly", "pid", pid)
		return nil

	case <-time.After(killTimeout):
		slog.Info("sending SIGKILL to process group", "pid", pid)
		killErr := syscall.Kill(-pid, syscall.SIGKILL)
		if killErr != nil {
			slog.Warn("SIGKILL failed", "pid", pid, "error", killErr)
		}

		// Reap the zombie regardless.
		waitErr := <-done
		return waitErr
	}
}

// IsRunning reports whether cmd represents a currently running process.
// It checks by sending signal 0 to the process, which validates existence
// without actually delivering a signal.
func IsRunning(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	return syscall.Kill(cmd.Process.Pid, 0) == nil
}
