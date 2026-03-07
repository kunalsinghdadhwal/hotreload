package internal

import (
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestShouldSkip(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "git directory",
			path: ".git",
			want: true,
		},
		{
			name: "node_modules directory",
			path: "project/node_modules",
			want: true,
		},
		{
			name: "hidden directory like .vscode",
			path: "src/.vscode",
			want: true,
		},
		{
			name: "swap file",
			path: "main.go.swp",
			want: true,
		},
		{
			name: "tilde backup file",
			path: "config.yaml~",
			want: true,
		},
		{
			name: "tmp file",
			path: "data.tmp",
			want: true,
		},
		{
			name: "DS_Store file",
			path: "project/.DS_Store",
			want: true,
		},
		{
			name: "vendor directory",
			path: "vendor",
			want: true,
		},
		{
			name: "__pycache__ directory",
			path: "scripts/__pycache__",
			want: true,
		},
		{
			name: "normal go file",
			path: "cmd/main.go",
			want: false,
		},
		{
			name: "normal subdirectory",
			path: "internal/handler",
			want: false,
		},
		{
			name: "normal nested file",
			path: "internal/runner/builder.go",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkip(tt.path)
			if got != tt.want {
				t.Errorf("ShouldSkip(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestShouldIgnoreEvent(t *testing.T) {
	tests := []struct {
		name  string
		event fsnotify.Event
		want  bool
	}{
		{
			name:  "CHMOD event is ignored",
			event: fsnotify.Event{Name: "main.go", Op: fsnotify.Chmod},
			want:  true,
		},
		{
			name:  "Emacs lock file",
			event: fsnotify.Event{Name: "src/.#main.go", Op: fsnotify.Create},
			want:  true,
		},
		{
			name:  "Emacs autosave file",
			event: fsnotify.Event{Name: "src/#main.go#", Op: fsnotify.Write},
			want:  true,
		},
		{
			name:  "Vim temp file 4913",
			event: fsnotify.Event{Name: "src/4913", Op: fsnotify.Create},
			want:  true,
		},
		{
			name:  "swap file",
			event: fsnotify.Event{Name: "main.go.swp", Op: fsnotify.Write},
			want:  true,
		},
		{
			name:  "valid .go WRITE event",
			event: fsnotify.Event{Name: "cmd/main.go", Op: fsnotify.Write},
			want:  false,
		},
		{
			name:  ".out binary is ignored",
			event: fsnotify.Event{Name: "app.out", Op: fsnotify.Write},
			want:  true,
		},
		{
			name:  "file with no extension is not ignored",
			event: fsnotify.Event{Name: "Makefile", Op: fsnotify.Write},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldIgnoreEvent(tt.event)
			if got != tt.want {
				t.Errorf("ShouldIgnoreEvent(%v) = %v, want %v", tt.event, got, tt.want)
			}
		})
	}
}
