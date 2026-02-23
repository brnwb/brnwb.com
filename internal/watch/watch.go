package watch

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Root     string
	Interval time.Duration
	Debounce time.Duration
	Verbose  bool
}

type fileState struct {
	ModTimeUnixNano int64
	Size            int64
	Mode            fs.FileMode
}

func Run(ctx context.Context, cfg Config, onDebouncedChange func() error) error {
	if onDebouncedChange == nil {
		return fmt.Errorf("onDebouncedChange callback is required")
	}

	rootAbs, err := filepath.Abs(cfg.Root)
	if err != nil {
		return fmt.Errorf("resolve watch root: %w", err)
	}

	info, err := os.Stat(rootAbs)
	if err != nil {
		return fmt.Errorf("stat watch root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("watch root is not a directory: %s", cfg.Root)
	}

	if cfg.Interval <= 0 {
		cfg.Interval = 250 * time.Millisecond
	}
	if cfg.Debounce <= 0 {
		cfg.Debounce = 200 * time.Millisecond
	}

	lastSnapshot, err := collectSnapshot(rootAbs)
	if err != nil {
		return err
	}

	if cfg.Verbose {
		logf(cfg, "watching %s", rootAbs)
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	var pending bool
	var dueAt time.Time

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			currentSnapshot, err := collectSnapshot(rootAbs)
			if err != nil {
				return err
			}

			if snapshotsDiffer(lastSnapshot, currentSnapshot) {
				lastSnapshot = currentSnapshot
				pending = true
				dueAt = time.Now().Add(cfg.Debounce)
				if cfg.Verbose {
					logf(cfg, "change detected; waiting for debounce")
				}
			}

			if pending && !time.Now().Before(dueAt) {
				if cfg.Verbose {
					logf(cfg, "running change callback")
				}

				if err := onDebouncedChange(); err != nil {
					fmt.Fprintf(os.Stderr, "sitegen: rebuild error: %v\n", err)
				}

				latestSnapshot, err := collectSnapshot(rootAbs)
				if err != nil {
					return err
				}
				lastSnapshot = latestSnapshot
				pending = false
			}
		}
	}
}

func collectSnapshot(root string) (map[string]fileState, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve snapshot root: %w", err)
	}

	snapshot := make(map[string]fileState)

	err = filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if shouldIgnore(d.Name(), d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return fmt.Errorf("compute relative path for %s: %w", path, err)
		}
		if rel == "." || d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		rel = filepath.ToSlash(rel)
		snapshot[rel] = fileState{
			ModTimeUnixNano: info.ModTime().UnixNano(),
			Size:            info.Size(),
			Mode:            info.Mode(),
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk snapshot: %w", err)
	}

	return snapshot, nil
}

func snapshotsDiffer(a, b map[string]fileState) bool {
	if len(a) != len(b) {
		return true
	}

	for path, stateA := range a {
		stateB, ok := b[path]
		if !ok {
			return true
		}
		if stateA != stateB {
			return true
		}
	}

	return false
}

func shouldIgnore(name string, isDir bool) bool {
	if name == "" {
		return false
	}

	if !isDir {
		if name == ".DS_Store" || name == "Thumbs.db" || strings.HasPrefix(name, "._") {
			return true
		}
	}

	return false
}

func logf(cfg Config, format string, args ...any) {
	if !cfg.Verbose {
		return
	}
	fmt.Printf("watch: "+format+"\n", args...)
}
