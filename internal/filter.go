package internal

import (
	"path/filepath"
	"strings"
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
