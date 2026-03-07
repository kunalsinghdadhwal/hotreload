package internal

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// Watcher wraps an fsnotify.Watcher to provide recursive directory watching
// with automatic inotify limit awareness. It does not start any goroutines
// internally; the caller is responsible for driving the event loop.
type Watcher struct {
	fsw        *fsnotify.Watcher
	maxWatches int
	watchCount int
	limitHit   bool
}

// New creates a new Watcher backed by fsnotify. It reads the inotify watch
// limit from /proc/sys/fs/inotify/max_user_watches and reserves 80% of it.
// If the limit cannot be read, it defaults to 8192.
func New() (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}

	limit := 8192
	data, err := os.ReadFile("/proc/sys/fs/inotify/max_user_watches")
	if err == nil {
		parsed, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
		if parseErr == nil {
			limit = parsed
		}
	}

	maxWatches := limit * 80 / 100

	slog.Debug("inotify watch limit configured",
		"os_limit", limit,
		"max_watches", maxWatches,
	)

	return &Watcher{
		fsw:        fsw,
		maxWatches: maxWatches,
	}, nil
}

// AddRecursive walks the directory tree rooted at root and adds an inotify
// watch for every directory that passes the ShouldSkip filter. It stops
// adding watches when the inotify limit is approached or hit.
func (w *Watcher) AddRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Warn("walk error", "path", path, "error", err)
			return filepath.SkipDir
		}

		// Only watch directories; skip filtered paths.
		if ShouldSkip(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		// Stop adding watches if the limit has been hit.
		if w.limitHit || w.watchCount >= w.maxWatches {
			return filepath.SkipDir
		}

		addErr := w.fsw.Add(path)
		if addErr != nil {
			errMsg := addErr.Error()
			if strings.Contains(errMsg, "no space left on device") ||
				strings.Contains(errMsg, "too many open files") {
				slog.Warn("inotify watch limit reached",
					"path", path,
					"hint", "run: sudo sysctl fs.inotify.max_user_watches=524288",
				)
				w.limitHit = true
				return filepath.SkipDir
			}
			return fmt.Errorf("watch %s: %w", path, addErr)
		}

		w.watchCount++
		return nil
	})
}

// Events returns the channel of file system events from the underlying
// fsnotify watcher.
func (w *Watcher) Events() <-chan fsnotify.Event {
	return w.fsw.Events
}

// Errors returns the channel of errors from the underlying fsnotify watcher.
func (w *Watcher) Errors() <-chan error {
	return w.fsw.Errors
}

// HandleCreateEvent processes a file system event. For Create events it stats
// the path and, if it is a directory, recursively adds watches for the new
// subtree. This handles the race where mkdir -p creates a/b/c/d faster than
// watches can be registered — by the time we stat 'a' and call AddRecursive,
// subdirectories b/c/d already exist and WalkDir catches them all.
// For Remove and Rename events it removes the watch (no-op if the path was
// not watched). All other event types are ignored.
func (w *Watcher) HandleCreateEvent(event fsnotify.Event) {
	switch {
	case event.Has(fsnotify.Create):
		info, err := os.Stat(event.Name)
		if err != nil {
			// Path was deleted before we could stat it — a real race.
			return
		}
		if info.IsDir() {
			if err := w.AddRecursive(event.Name); err != nil {
				slog.Warn("failed to watch new directory", "path", event.Name, "error", err)
				return
			}
			slog.Info("new directory detected, watching", "path", event.Name)
		}

	case event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename):
		_ = w.fsw.Remove(event.Name)
	}
}

// WatchList returns the list of paths currently being watched.
func (w *Watcher) WatchList() []string {
	return w.fsw.WatchList()
}

// Close closes the underlying fsnotify watcher and releases all inotify
// resources.
func (w *Watcher) Close() error {
	return w.fsw.Close()
}
