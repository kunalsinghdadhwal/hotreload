package runner

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Executor manages the lifecycle of the server process. It handles starting,
// stopping, and restarting the process, and notifies the caller via onExit
// when the process terminates unexpectedly.
type Executor struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	execCmd  string
	onExit   func(err error)
	waitDone chan struct{} // closed when cmd.Wait returns in the Start goroutine
}

// NewExecutor creates an Executor that will run the given command string.
// The onExit callback is invoked in a separate goroutine whenever the
// managed process exits — this allows the engine to detect crashes and
// trigger a restart with backoff.
func NewExecutor(execCmd string, onExit func(err error)) *Executor {
	return &Executor{
		execCmd: execCmd,
		onExit:  onExit,
	}
}

// Start launches the server process. The process inherits os.Stdout and
// os.Stderr directly for zero-latency output streaming. It runs in its own
// process group so that KillProcessGroup can terminate the entire tree.
//
// Start does not block — it calls cmd.Start and spawns a goroutine to
// wait for the process to exit and invoke the onExit callback.
func (e *Executor) Start() error {
	args := strings.Fields(e.execCmd)
	if len(args) == 0 {
		return fmt.Errorf("empty exec command")
	}

	cmd := exec.Command(args[0], args[1:]...)
	SetProcessGroup(cmd)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	slog.Info("server started", "pid", cmd.Process.Pid, "cmd", e.execCmd)

	waitDone := make(chan struct{})

	e.mu.Lock()
	e.cmd = cmd
	e.waitDone = waitDone
	e.mu.Unlock()

	go func() {
		waitErr := cmd.Wait()
		close(waitDone)
		if e.onExit != nil {
			e.onExit(waitErr)
		}
	}()

	return nil
}

// Stop terminates the running server process by sending SIGTERM to the
// process group, escalating to SIGKILL if necessary. It waits for the
// Start goroutine's cmd.Wait to complete rather than calling Wait itself,
// avoiding data races on the exec.Cmd. It is safe to call Stop when no
// process is running.
func (e *Executor) Stop() error {
	e.mu.Lock()
	cmd := e.cmd
	waitDone := e.waitDone
	e.cmd = nil
	e.waitDone = nil
	e.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pid := cmd.Process.Pid
	slog.Info("stopping server", "pid", pid)

	// Send SIGTERM to the entire process group.
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		slog.Warn("SIGTERM failed", "pid", pid, "error", err)
	}

	// Wait for the Start goroutine's cmd.Wait to return, with a timeout.
	select {
	case <-waitDone:
		slog.Info("process exited after SIGTERM", "pid", pid)
		return nil
	case <-time.After(killTimeout):
		slog.Info("sending SIGKILL to process group", "pid", pid)
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
			slog.Warn("SIGKILL failed", "pid", pid, "error", err)
		}
		// Wait for the Start goroutine to finish reaping.
		<-waitDone
		return nil
	}
}

// Restart stops the current process (if any) and starts a new one.
func (e *Executor) Restart() error {
	if err := e.Stop(); err != nil {
		slog.Warn("error stopping server during restart", "error", err)
	}
	return e.Start()
}

// Current returns the currently running command, or nil if no process is
// active. The caller must not modify the returned *exec.Cmd.
func (e *Executor) Current() *exec.Cmd {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cmd
}
