package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// prefixWriter is an io.Writer that splits input on newlines and logs each
// line via slog.Info with a fixed prefix tag. It is used to capture build
// command output in a structured way.
type prefixWriter struct {
	prefix string
	buf    bytes.Buffer
}

// Write implements io.Writer. It buffers partial lines and flushes complete
// lines to slog.Info with the configured prefix.
func (pw *prefixWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	pw.buf.Write(p)

	for {
		line, readErr := pw.buf.ReadString('\n')
		if readErr != nil {
			// Incomplete line — put it back for next Write call.
			pw.buf.WriteString(line)
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			slog.Info(line, "src", pw.prefix)
		}
	}

	return n, nil
}

// flush writes any remaining buffered content that didn't end with a newline.
func (pw *prefixWriter) flush() {
	remaining := pw.buf.String()
	remaining = strings.TrimRight(remaining, "\r\n")
	if remaining != "" {
		slog.Info(remaining, "src", pw.prefix)
	}
	pw.buf.Reset()
}

// Build runs the given build command string and returns any error. Build
// output (stdout and stderr) is captured and logged line-by-line via slog
// with a [build] prefix.
//
// If the context is cancelled (e.g. by a newer build being triggered), Build
// returns the context error rather than a misleading build-failure error.
// If the command exits with a non-zero code, the returned error includes the
// exit code.
func Build(ctx context.Context, buildCmd string) error {
	args := strings.Fields(buildCmd)
	if len(args) == 0 {
		return fmt.Errorf("empty build command")
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	pw := &prefixWriter{prefix: "[build]"}
	cmd.Stdout = pw
	cmd.Stderr = pw

	slog.Info("building", "cmd", buildCmd)

	runErr := cmd.Run()
	pw.flush()

	if runErr != nil {
		// If the context was cancelled, report that instead of a confusing
		// exit-code error — this means a newer file change superseded us.
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return fmt.Errorf("build failed with exit code %d: %w", exitErr.ExitCode(), runErr)
		}

		return fmt.Errorf("build failed: %w", runErr)
	}

	slog.Info("build succeeded")
	return nil
}
