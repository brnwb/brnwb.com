package serve

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"brnwb.com/internal/pathsafe"
)

type Config struct {
	Root    string
	Port    int
	Verbose bool
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.Port <= 0 {
		return fmt.Errorf("port must be greater than zero")
	}

	handler, err := NewHandler(cfg.Root)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	url := fmt.Sprintf("http://localhost:%d/", cfg.Port)
	fmt.Printf("sitegen: serving at %s\n", url)
	if cfg.Verbose {
		fmt.Printf("serve: root=%s\n", cfg.Root)
	}

	errCh := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("run server: %w", err)
		}
		return nil
	}
}

func NewHandler(root string) (http.Handler, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve server root: %w", err)
	}

	info, err := os.Stat(rootAbs)
	if err != nil {
		return nil, fmt.Errorf("stat server root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("server root is not a directory: %s", root)
	}

	rootResolved, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return nil, fmt.Errorf("resolve server root symlinks: %w", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetPath, err := resolveRequestPath(rootAbs, r.URL.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		info, err := os.Stat(targetPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if info.IsDir() {
			targetPath = filepath.Join(targetPath, "index.html")
			info, err = os.Stat(targetPath)
			if err != nil || info.IsDir() {
				http.NotFound(w, r)
				return
			}
		}

		resolvedTarget, err := filepath.EvalSymlinks(targetPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		ok, err := pathsafe.IsWithinRoot(rootResolved, resolvedTarget)
		if err != nil || !ok {
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, targetPath)
	}), nil
}

func resolveRequestPath(rootAbs, requestPath string) (string, error) {
	if requestPath == "" {
		requestPath = "/"
	}

	cleanURLPath := path.Clean("/" + requestPath)
	relativePath := strings.TrimPrefix(cleanURLPath, "/")
	targetPath := filepath.Join(rootAbs, filepath.FromSlash(relativePath))

	ok, err := pathsafe.IsWithinRoot(rootAbs, targetPath)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("request path resolves outside root: %s", requestPath)
	}

	return targetPath, nil
}
