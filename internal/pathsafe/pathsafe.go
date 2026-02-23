package pathsafe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveWithinRoot(root, relativePath string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root path: %w", err)
	}

	resolved := filepath.Join(rootAbs, filepath.FromSlash(relativePath))
	resolvedAbs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	ok, err := IsWithinRoot(rootAbs, resolvedAbs)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("path resolves outside root: %q", relativePath)
	}

	return resolvedAbs, nil
}

func IsWithinRoot(root, target string) (bool, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false, fmt.Errorf("resolve root path: %w", err)
	}

	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false, fmt.Errorf("resolve target path: %w", err)
	}

	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false, fmt.Errorf("compute relative path: %w", err)
	}

	if rel == "." {
		return true, nil
	}

	if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return false, nil
	}

	return true, nil
}

func EnsureNoSymlinkTraversal(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root path: %w", err)
	}

	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}

	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return fmt.Errorf("compute relative path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("target is outside root")
	}
	if rel == "." {
		return nil
	}

	current := rootAbs
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		current = filepath.Join(current, part)

		info, err := os.Lstat(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// Remaining path does not exist yet; no symlink to traverse.
				return nil
			}
			return fmt.Errorf("lstat %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink traversal is not allowed: %s", current)
		}
	}

	return nil
}
