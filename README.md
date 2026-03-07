# hotreload

## Overview

hotreload is a CLI hot-reload tool for Go projects. It watches your source directory for file changes, debounces rapid edits, rebuilds your project, and restarts the server process automatically. It handles process group cleanup, crash loop detection with exponential backoff, and dynamic directory watching — all with zero configuration files.

## Requirements

- Go 1.22 or later
- Linux is the primary target (uses inotify via fsnotify). macOS should work via kqueue, but is not the primary test target.

## Installation

Install directly:

```sh
go install github.com/kunalsinghdadhwal/hotreload/cmd/hotreload@latest
```

Or build from source:

```sh
git clone https://github.com/kunalsinghdadhwal/hotreload.git
cd hotreload
make build
```

## Usage

```sh
hotreload --root <dir> --build "<build command>" --exec "<exec command>"
```

Flags:

- `--root` — root directory to watch (default: `.`)
- `--build` — build command to run on changes (required)
- `--exec` — command to execute after successful build (required)

Example using the included testserver:

```sh
hotreload --root ./testserver \
  --build "go build -o ./bin/testserver ./testserver" \
  --exec "./bin/testserver"
```

## Demo

Run the demo with make:

```sh
make demo
```

This will build hotreload, then start watching the `testserver/` directory. Edit `testserver/main.go` (for example, change `VERSION` from `"v1"` to `"v2"`), save, and watch the server rebuild and restart automatically. Visit http://localhost:8080 to see the new version.

## How it works

- **Watcher** — recursively monitors the project directory using fsnotify (inotify on Linux), dynamically adding watches for newly created directories
- **Debouncer** — coalesces rapid file system events (150ms window) into a single rebuild trigger
- **Builder** — executes the build command with context cancellation support so newer changes can pre-empt in-progress builds
- **Executor** — manages the server process lifecycle with process group isolation, ensuring clean SIGTERM → SIGKILL shutdown
- **Crash backoff** — detects crash loops (repeated exits within 10s) and applies exponential backoff (1s, 2s, 4s… up to 30s) before restarting

## inotify limits

Linux limits the number of inotify watches a user can hold. The default is often 8192, which may not be enough for large projects. hotreload reads the current limit from `/proc/sys/fs/inotify/max_user_watches` and uses 80% of it, logging a warning if the limit is reached. To increase the limit:

```sh
sudo sysctl fs.inotify.max_user_watches=524288
```

To make it permanent, add `fs.inotify.max_user_watches=524288` to `/etc/sysctl.conf`.

## Ground rules

- `fsnotify` is the only external dependency — everything else uses the Go standard library
- Logging uses `log/slog` throughout
- Process management uses OS-level process groups for reliable cleanup
- The tool is designed for Linux first, with macOS support via fsnotify's kqueue backend
