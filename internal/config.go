package internal

import (
	"fmt"
	"os"
)

// Config holds the runtime configuration for the hot-reload engine.
type Config struct {
	Root     string
	BuildCmd string
	ExecCmd  string
}

// NewConfig creates and validates a Config. It returns an error if root
// does not exist on disk, or if buildCmd or execCmd are empty.
func NewConfig(root, buildCmd, execCmd string) (*Config, error) {
	if root == "" {
		return nil, fmt.Errorf("root directory must not be empty")
	}
	if _, err := os.Stat(root); err != nil {
		return nil, fmt.Errorf("root directory %q does not exist: %w", root, err)
	}
	if buildCmd == "" {
		return nil, fmt.Errorf("build command must not be empty")
	}
	if execCmd == "" {
		return nil, fmt.Errorf("exec command must not be empty")
	}
	return &Config{
		Root:     root,
		BuildCmd: buildCmd,
		ExecCmd:  execCmd,
	}, nil
}
