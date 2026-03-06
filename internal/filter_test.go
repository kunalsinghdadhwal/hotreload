package internal

import "testing"

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
