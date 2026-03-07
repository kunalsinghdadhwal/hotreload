package internal

import (
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// skipDirs contains directory names that should always be ignored by the watcher.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".cache":       true,
	"dist":         true,
	"build":        true,
	"bin":          true,
	"__pycache__":  true,
}

// skipSuffixes contains file suffixes that indicate temporary or editor files
// which should be ignored by the watcher.
var skipSuffixes = []string{
	".swp",
	".swx",
	"~",
	".tmp",
	".DS_Store",
}

// ShouldSkip reports whether the given path should be ignored by the watcher.
// It returns true for known noisy directories (e.g. .git, node_modules),
// any path whose base name starts with a dot, and files with temporary-file
// suffixes such as .swp or .tmp.
func ShouldSkip(path string) bool {
	base := filepath.Base(path)

	// Check explicit directory skip list.
	if skipDirs[base] {
		return true
	}

	// Skip any entry whose base name starts with a dot (hidden files/dirs).
	if strings.HasPrefix(base, ".") {
		return true
	}

	// Skip files matching known temporary/editor suffixes.
	for _, suffix := range skipSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}

	return false
}

// allowedExts is the set of file extensions that should trigger a rebuild.
// Files with extensions not in this set are ignored. Files with no extension
// are always allowed through.
var allowedExts = map[string]bool{
	".go":    true,
	".env":   true,
	".json":  true,
	".yaml":  true,
	".yml":   true,
	".toml":  true,
	".sql":   true,
	".html":  true,
	".proto": true,
}

// ShouldIgnoreEvent reports whether an fsnotify event should be suppressed
// and not trigger a rebuild. It filters out CHMOD-only events, editor
// temporary files (Emacs lock/autosave, Vim 4913), known noisy paths
// (via ShouldSkip), and files whose extension is not in the allowed set.
func ShouldIgnoreEvent(event fsnotify.Event) bool {
	// CHMOD-only events fire constantly from editors and never mean real
	// content changed.
	if event.Op == fsnotify.Chmod {
		return true
	}

	base := filepath.Base(event.Name)

	// Delegate to the existing path-based filter (covers hidden files,
	// temp suffixes, noisy directories).
	if ShouldSkip(event.Name) {
		return true
	}

	// Emacs lock files: .#main.go
	if strings.HasPrefix(base, ".#") {
		return true
	}

	// Emacs autosave files: #main.go#
	if strings.HasPrefix(base, "#") && strings.HasSuffix(base, "#") {
		return true
	}

	// Vim atomic-write temp file.
	if base == "4913" {
		return true
	}

	// Extension allowlist. Files with no extension (e.g. Makefile,
	// Dockerfile, compiled binaries without suffix) are allowed through.
	ext := filepath.Ext(base)
	if ext == "" {
		return false
	}
	return !allowedExts[ext]
}
