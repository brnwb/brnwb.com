package serve

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandlerServesDirectoryIndex(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "index.html"), "home")
	mustWriteFile(t, filepath.Join(root, "docs", "index.html"), "docs-index")
	mustWriteFile(t, filepath.Join(root, "plain.txt"), "plain")

	handler, err := NewHandler(root)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	assertResponseBody(t, handler, "/docs/", http.StatusOK, "docs-index")
	assertResponseBody(t, handler, "/docs", http.StatusOK, "docs-index")
	assertResponseBody(t, handler, "/plain.txt", http.StatusOK, "plain")
	assertResponseBody(t, handler, "/missing", http.StatusNotFound, "404 page not found\n")
}

func TestResolveRequestPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	got, err := resolveRequestPath(root, "/docs/index.html")
	if err != nil {
		t.Fatalf("resolveRequestPath() error = %v", err)
	}
	want := filepath.Join(root, "docs", "index.html")
	if got != want {
		t.Fatalf("resolveRequestPath() got %q, want %q", got, want)
	}
}

func TestHandlerDoesNotServeSymlinkOutsideRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "index.html"), "home")
	mustWriteFile(t, filepath.Join(outside, "secret.txt"), "secret")

	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	handler, err := NewHandler(root)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	assertResponseBody(t, handler, "/link.txt", http.StatusNotFound, "404 page not found\n")
}

func assertResponseBody(t *testing.T, handler http.Handler, requestPath string, wantStatus int, wantBody string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, requestPath, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("ReadAll(): %v", err)
	}

	if res.StatusCode != wantStatus {
		t.Fatalf("status for %q: got %d, want %d", requestPath, res.StatusCode, wantStatus)
	}
	if string(body) != wantBody {
		t.Fatalf("body for %q: got %q, want %q", requestPath, string(body), wantBody)
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
