package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kunalsinghdadhwal/hotreload/internal"
)

func main() {
	slog.SetDefault(slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				switch a.Key {

				case slog.TimeKey:
					return slog.Attr{}

				case slog.SourceKey:
					src := a.Value.Any().(*slog.Source)
					src.File = filepath.Base(src.File)
				}

				return a
			},
		}),
	))

	root := flag.String("root", ".", "root directory to watch")
	build := flag.String("build", "", "build command (required)")
	exec := flag.String("exec", "", "exec command (required)")
	flag.Parse()

	if *build == "" || *exec == "" {
		fmt.Fprintln(os.Stderr, "both --build and --exec flags are required")
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := internal.NewConfig(*root, *build, *exec)
	if err != nil {
		slog.Error("invalid config", "err", err)
		os.Exit(1)
	}

	eng, err := internal.NewEngine(cfg)
	if err != nil {
		slog.Error("failed to create engine", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := eng.Run(ctx); err != nil {
		slog.Error("engine error", "err", err)
		os.Exit(1)
	}

	slog.Info("goodbye")
}
