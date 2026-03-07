package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunTriggersInitialBuild(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	marker := filepath.Join(dir, "build.marker")
	buildCmd := "touch " + marker
	execCmd := "sleep 3600"

	cfg, err := NewConfig(dir, buildCmd, execCmd)
	if err != nil {
		t.Fatal(err)
	}

	eng, err := NewEngine(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- eng.Run(ctx)
	}()

	// Wait for the marker file to appear (initial build).
	deadline := time.After(5 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatal("initial build did not fire within timeout")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestRapidEventsDebounce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	counter := filepath.Join(dir, "build.counter")
	// Write a build script that appends a line to the counter file.
	buildScript := filepath.Join(dir, "build.sh")
	if err := os.WriteFile(buildScript, []byte("#!/bin/bash\necho 1 >> "+counter+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	buildCmd := buildScript
	execCmd := "sleep 3600"

	cfg, err := NewConfig(dir, buildCmd, execCmd)
	if err != nil {
		t.Fatal(err)
	}

	eng, err := NewEngine(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- eng.Run(ctx)
	}()

	// Wait for initial build to complete.
	deadline := time.After(5 * time.Second)
	for {
		if data, err := os.ReadFile(counter); err == nil && len(data) > 0 {
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatal("initial build did not fire within timeout")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Let the initial build cycle fully settle.
	time.Sleep(300 * time.Millisecond)

	// Create 5 .go files rapidly — should debounce into a single build.
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, fmt.Sprintf("rapid%d.go", i))
		os.WriteFile(name, []byte("package main\n"), 0644)
	}

	// Wait for debounce (150ms) + build time + buffer.
	time.Sleep(1 * time.Second)

	cancel()
	<-errCh

	data, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	builds := len(strings.Split(strings.TrimSpace(string(data)), "\n"))
	// Expect at most 3 builds: initial + at most 2 debounced batches.
	// Without debouncing we would see 6+ builds (initial + 5 file events).
	if builds > 3 {
		t.Errorf("expected at most 3 builds (debounce should coalesce), got %d", builds)
	}
	if builds < 2 {
		t.Errorf("expected at least 2 builds (initial + debounced file events), got %d", builds)
	}
}

func TestCancelledContextReturnsNil(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := NewConfig(dir, "echo ok", "sleep 3600")
	if err != nil {
		t.Fatal(err)
	}

	eng, err := NewEngine(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err = eng.Run(ctx)
	if err != nil {
		t.Fatalf("Run should return nil on cancelled context, got: %v", err)
	}
}
