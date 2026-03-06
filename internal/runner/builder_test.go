package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestBuildSuccess(t *testing.T) {
	err := Build(context.Background(), "echo hello")
	if err != nil {
		t.Errorf("expected nil error for successful command, got: %v", err)
	}
}

func TestBuildFailure(t *testing.T) {
	err := Build(context.Background(), "sh -c exit 1")
	if err == nil {
		t.Fatal("expected error for failing command, got nil")
	}
	if !strings.Contains(err.Error(), "exit code") {
		t.Errorf("expected error to contain 'exit code', got: %v", err)
	}
}

func TestBuildCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := Build(ctx, "sleep 10")
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestBuildContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := Build(ctx, "sleep 30")
	if err == nil {
		t.Fatal("expected error for timed-out context, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestBuildExitCodes(t *testing.T) {
	tests := []struct {
		cmd      string
		wantCode string
	}{
		{"sh -c exit 1", "exit code 1"},
		{"sh -c exit 2", "exit code 2"},
		{"sh -c exit 42", "exit code 42"},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			err := Build(context.Background(), tt.cmd)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantCode) {
				t.Errorf("expected error containing %q, got: %v", tt.wantCode, err)
			}
		})
	}
}

func TestBuildEmptyCommand(t *testing.T) {
	err := Build(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty command, got nil")
	}
	if !strings.Contains(err.Error(), "empty build command") {
		t.Errorf("expected 'empty build command' error, got: %v", err)
	}
}

func TestBuildNonexistentBinary(t *testing.T) {
	err := Build(context.Background(), "/nonexistent/binary")
	if err == nil {
		t.Fatal("expected error for nonexistent binary, got nil")
	}
	if !strings.Contains(err.Error(), "build failed") {
		t.Errorf("expected 'build failed' in error, got: %v", err)
	}
}

func TestPrefixWriter(t *testing.T) {
	pw := &prefixWriter{prefix: "[build]"}

	input := "first line\nsecond line\n"
	n, err := pw.Write([]byte(input))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if n != len(input) {
		t.Errorf("expected n=%d, got n=%d", len(input), n)
	}
}

func TestPrefixWriterPartialLines(t *testing.T) {
	pw := &prefixWriter{prefix: "[test]"}

	// Write a partial line.
	partial := "partial"
	n1, err := pw.Write([]byte(partial))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n1 != len(partial) {
		t.Errorf("expected n=%d, got %d", len(partial), n1)
	}

	// The partial content should be buffered.
	if pw.buf.String() != "partial" {
		t.Errorf("expected buffer to contain 'partial', got %q", pw.buf.String())
	}

	// Complete the line.
	rest := " line\n"
	n2, err := pw.Write([]byte(rest))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n2 != len(rest) {
		t.Errorf("expected n=%d, got %d", len(rest), n2)
	}

	// After completing the line, buffer should be empty.
	if pw.buf.Len() != 0 {
		t.Errorf("expected empty buffer after complete line, got %q", pw.buf.String())
	}
}

func TestPrefixWriterFlush(t *testing.T) {
	pw := &prefixWriter{prefix: "[test]"}

	pw.Write([]byte("no newline"))
	if pw.buf.Len() == 0 {
		t.Fatal("expected buffered content before flush")
	}

	pw.flush()
	if pw.buf.Len() != 0 {
		t.Errorf("expected empty buffer after flush, got %q", pw.buf.String())
	}
}

func TestBuildContextCancellationMidRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- Build(ctx, "sleep 30")
	}()

	// Give the process a moment to start, then cancel.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Build did not return after context cancellation")
	}
}
