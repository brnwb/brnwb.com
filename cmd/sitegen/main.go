package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"brnwb.com/internal/build"
	"brnwb.com/internal/serve"
	"brnwb.com/internal/watch"
)

func main() {
	inDir := flag.String("in", "src", "input source directory")
	outDir := flag.String("out", "html", "output directory")
	clean := flag.Bool("clean", false, "remove output directory before build")
	verbose := flag.Bool("verbose", false, "print build progress")
	watchEnabled := flag.Bool("watch", false, "watch source directory for changes")
	servePort := flag.Int("serve", 0, "serve output directory on a port")
	flag.Parse()

	cfg := build.Config{
		InDir:   *inDir,
		OutDir:  *outDir,
		Clean:   *clean,
		Verbose: *verbose,
	}

	if err := build.Run(cfg); err != nil {
		exitErr(err)
	}

	if !*watchEnabled && *servePort == 0 {
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)
	workers := 0

	if *servePort != 0 {
		workers++
		go func() {
			errCh <- serve.Run(ctx, serve.Config{
				Root:    cfg.OutDir,
				Port:    *servePort,
				Verbose: cfg.Verbose,
			})
		}()
	}

	if *watchEnabled {
		workers++
		watchBuildCfg := cfg
		// In watch mode, rebuild from a clean output tree to avoid stale files after deletes.
		watchBuildCfg.Clean = true

		go func() {
			errCh <- watch.Run(ctx, watch.Config{
				Root:     cfg.InDir,
				Interval: 250 * time.Millisecond,
				Debounce: 200 * time.Millisecond,
				Verbose:  cfg.Verbose,
			}, func() error {
				if cfg.Verbose {
					fmt.Fprintln(os.Stderr, "sitegen: rebuilding...")
				}
				return build.Run(watchBuildCfg)
			})
		}()
	}

	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			exitErr(err)
		}
	}
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "sitegen:", err)
	os.Exit(1)
}
