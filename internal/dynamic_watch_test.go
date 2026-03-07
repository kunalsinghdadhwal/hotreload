package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestDynamicDirectoryWatching(t *testing.T) {
	dir := t.TempDir()

	w, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if err := w.AddRecursive(dir); err != nil {
		t.Fatal(err)
	}

	// Create nested directory structure a/b/c.
	aDir := filepath.Join(dir, "a")
	bDir := filepath.Join(dir, "a", "b")
	cDir := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(cDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Simulate the CREATE event for 'a' — AddRecursive will walk into
	// b and c because they already exist (handles the mkdir -p race).
	w.HandleCreateEvent(fsnotify.Event{
		Name: aDir,
		Op:   fsnotify.Create,
	})

	// Drain any buffered events from the MkdirAll + AddRecursive.
	drainDeadline := time.After(200 * time.Millisecond)
drain:
	for {
		select {
		case <-w.Events():
		case <-drainDeadline:
			break drain
		}
	}

	// Assert that b and c are now watched.
	watchList := w.WatchList()
	watched := make(map[string]bool, len(watchList))
	for _, p := range watchList {
		watched[p] = true
	}

	if !watched[bDir] {
		t.Errorf("expected %s to be watched, watch list: %v", bDir, watchList)
	}
	if !watched[cDir] {
		t.Errorf("expected %s to be watched, watch list: %v", cDir, watchList)
	}

	// Write a file inside c and verify we receive an event within 500ms.
	testFile := filepath.Join(cDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-w.Events():
		if event.Name != testFile {
			t.Errorf("expected event for %s, got %s", testFile, event.Name)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("timed out waiting for event from nested directory c")
	}
}
