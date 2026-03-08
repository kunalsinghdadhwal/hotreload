package internal

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kunalsinghdadhwal/hotreload/internal/runner"
)

// Engine orchestrates the hot-reload loop: it watches for file changes,
// debounces rapid edits, rebuilds the project, and restarts the server
// process. It is the central coordination point of the tool.
type Engine struct {
	cfg       *Config
	watcher   *Watcher
	debouncer *Debouncer
	executor  *runner.Executor
	backoff   *Backoff

	mu          sync.Mutex
	cancelBuild context.CancelFunc
	building    bool
	stopped     bool
}

// NewEngine creates an Engine from the given config. It initialises the
// file watcher, debouncer, executor, and backoff. The watcher begins
// monitoring cfg.Root recursively.
func NewEngine(cfg *Config) (*Engine, error) {
	w, err := New()
	if err != nil {
		return nil, err
	}

	e := &Engine{
		cfg:       cfg,
		watcher:   w,
		debouncer: NewDebouncer(150 * time.Millisecond),
		backoff:   NewBackoff(),
	}
	e.executor = runner.NewExecutor(cfg.ExecCmd, e.onProcessExit)

	if err := w.AddRecursive(cfg.Root); err != nil {
		w.Close()
		return nil, err
	}

	return e, nil
}

// Run is the main blocking event loop. It triggers an immediate build on
// startup and then reacts to file-system events, debouncing rapid changes
// into a single rebuild. It returns nil when ctx is cancelled.
func (e *Engine) Run(ctx context.Context) error {
	e.scheduleBuild()

	for {
		select {
		case <-ctx.Done():
			e.shutdown()
			return nil

		case event := <-e.watcher.Events():
			e.watcher.HandleCreateEvent(event)
			if ShouldIgnoreEvent(event) {
				continue
			}

			slog.Info("change detected", "file", event.Name, "op", event.Op.String())
			e.debouncer.Trigger(e.scheduleBuild)

		case err := <-e.watcher.Errors():
			slog.Warn("watcher error", "err", err)
		}
	}
}

// scheduleBuild cancels any in-progress build and launches a new one.
func (e *Engine) scheduleBuild() {
	e.mu.Lock()
	if e.cancelBuild != nil {
		e.cancelBuild()
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.cancelBuild = cancel
	e.building = true
	e.mu.Unlock()

	go e.runBuild(ctx)
}

// runBuild stops the running server, builds the project, and starts the
// new binary. It respects context cancellation so that a newer build can
// pre-empt it.
func (e *Engine) runBuild(ctx context.Context) {
	e.executor.Stop()

	slog.Info("building...", "cmd", e.cfg.BuildCmd)

	if err := runner.Build(ctx, e.cfg.BuildCmd); err != nil {
		if ctx.Err() != nil {
			slog.Info("build cancelled, newer change incoming")
			return
		}
		slog.Error("build failed", "err", err)
		e.mu.Lock()
		e.building = false
		e.mu.Unlock()
		return
	}

	slog.Info("build succeeded, starting server")
	if err := e.executor.Start(); err != nil {
		slog.Error("failed to start server", "err", err)
		e.mu.Lock()
		e.building = false
		e.mu.Unlock()
		return
	}

	e.mu.Lock()
	e.building = false
	e.mu.Unlock()
}

// onProcessExit is called by the executor when the server process exits.
// On a clean exit it does nothing; on a crash it applies backoff and
// schedules a rebuild.
func (e *Engine) onProcessExit(err error) {
	e.mu.Lock()
	stopped := e.stopped
	building := e.building
	e.mu.Unlock()

	// If stopped or a build is in flight (meaning we intentionally killed
	// the server for a rebuild), ignore this exit.
	if stopped || building {
		return
	}

	if err == nil {
		slog.Info("server exited cleanly")
		e.backoff.Reset()
		return
	}

	slog.Warn("server crashed", "err", err)

	wait := e.backoff.RecordCrash()
	if wait > 0 {
		slog.Warn("crash loop detected, backing off", "duration", wait)
		time.Sleep(wait)
	}

	e.scheduleBuild()
}

// shutdown tears down all engine subsystems.
func (e *Engine) shutdown() {
	e.mu.Lock()
	e.stopped = true
	if e.cancelBuild != nil {
		e.cancelBuild()
	}
	e.mu.Unlock()

	e.debouncer.Stop()
	e.executor.Stop()
	e.watcher.Close()

	slog.Info("hotreload shutting down")
}
