package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCollectSnapshotIgnoresMetadataFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "index.html"), "<h1>Hello</h1>")
	mustWriteFile(t, filepath.Join(root, ".DS_Store"), "ignored")
	mustWriteFile(t, filepath.Join(root, "nested", "Thumbs.db"), "ignored")
	mustWriteFile(t, filepath.Join(root, "nested", "data.txt"), "content")

	snapshot, err := collectSnapshot(root)
	if err != nil {
		t.Fatalf("collectSnapshot() error = %v", err)
	}

	if _, ok := snapshot[".DS_Store"]; ok {
		t.Fatal("expected .DS_Store to be ignored")
	}
	if _, ok := snapshot["nested/Thumbs.db"]; ok {
		t.Fatal("expected nested/Thumbs.db to be ignored")
	}
	if _, ok := snapshot["index.html"]; !ok {
		t.Fatal("expected index.html in snapshot")
	}
	if _, ok := snapshot["nested/data.txt"]; !ok {
		t.Fatal("expected nested/data.txt in snapshot")
	}
}

func TestSnapshotsDifferDetectsChanges(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	file := filepath.Join(root, "index.html")
	mustWriteFile(t, file, "v1")

	s1, err := collectSnapshot(root)
	if err != nil {
		t.Fatalf("collectSnapshot() error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	mustWriteFile(t, file, "v2")
	s2, err := collectSnapshot(root)
	if err != nil {
		t.Fatalf("collectSnapshot() error = %v", err)
	}
	if !snapshotsDiffer(s1, s2) {
		t.Fatal("expected snapshots to differ after file content update")
	}

	mustWriteFile(t, filepath.Join(root, "new.txt"), "new")
	s3, err := collectSnapshot(root)
	if err != nil {
		t.Fatalf("collectSnapshot() error = %v", err)
	}
	if !snapshotsDiffer(s2, s3) {
		t.Fatal("expected snapshots to differ after file addition")
	}

	if err := os.Remove(file); err != nil {
		t.Fatalf("Remove(%q): %v", file, err)
	}
	s4, err := collectSnapshot(root)
	if err != nil {
		t.Fatalf("collectSnapshot() error = %v", err)
	}
	if !snapshotsDiffer(s3, s4) {
		t.Fatal("expected snapshots to differ after file deletion")
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
