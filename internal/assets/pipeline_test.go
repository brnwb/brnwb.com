package assets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildBundlesCSSAndJSAndWritesManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	srcRoot := filepath.Join(root, "src")
	outRoot := filepath.Join(root, "html")

	mustWriteFile(t, filepath.Join(srcRoot, "_assets", "bundles.json"), `{
  "css_bundles": [
    {
      "name": "style.css",
      "inputs": ["_css/colors.css", "_css/elements.css"]
    }
  ],
  "js_bundles": [
    {
      "name": "zepbound/chart.js",
      "inputs": ["_js/zepbound/part-a.js", "_js/zepbound/part-b.js"]
    }
  ]
}`)
	mustWriteFile(t, filepath.Join(srcRoot, "_css", "colors.css"), ":root{--bg:#fff;}")
	mustWriteFile(t, filepath.Join(srcRoot, "_css", "elements.css"), "body{color:#111;}")
	mustWriteFile(t, filepath.Join(srcRoot, "_js", "zepbound", "part-a.js"), "const a = 1;")
	mustWriteFile(t, filepath.Join(srcRoot, "_js", "zepbound", "part-b.js"), "const b = 2;")

	manifest, err := Build(srcRoot, outRoot, false)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	assertFileContains(t, filepath.Join(outRoot, "style.css"), ":root{--bg:#fff;}\nbody{color:#111;}\n")
	assertFileContains(t, filepath.Join(outRoot, "zepbound", "chart.js"), "const a = 1;\nconst b = 2;\n")

	if got := manifest.Assets["style.css"]; got != "style.css" {
		t.Fatalf("manifest style.css mismatch: got %q", got)
	}
	if got := manifest.Assets["zepbound/chart.js"]; got != "zepbound/chart.js" {
		t.Fatalf("manifest zepbound/chart.js mismatch: got %q", got)
	}

	manifestPath := filepath.Join(outRoot, ManifestFilename)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", manifestPath, err)
	}

	var decoded Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal(manifest): %v", err)
	}
	if decoded.Assets["style.css"] != "style.css" {
		t.Fatalf("decoded manifest style.css mismatch: got %q", decoded.Assets["style.css"])
	}
}

func TestBuildRejectsTraversalInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	srcRoot := filepath.Join(root, "src")
	outRoot := filepath.Join(root, "html")

	mustWriteFile(t, filepath.Join(root, "secret.css"), "secret")
	mustWriteFile(t, filepath.Join(srcRoot, "_assets", "bundles.json"), `{
  "css_bundles": [
    {
      "name": "style.css",
      "inputs": ["../secret.css"]
    }
  ],
  "js_bundles": []
}`)

	_, err := Build(srcRoot, outRoot, false)
	if err == nil {
		t.Fatal("expected traversal error, got nil")
	}
	if !strings.Contains(err.Error(), "outside root") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsSymlinkTraversalInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	srcRoot := filepath.Join(root, "src")
	outRoot := filepath.Join(root, "html")

	mustWriteFile(t, filepath.Join(root, "secret.css"), "secret")
	mustWriteFile(t, filepath.Join(srcRoot, "_assets", "bundles.json"), `{
  "css_bundles": [
    {
      "name": "style.css",
      "inputs": ["_css/link.css"]
    }
  ],
  "js_bundles": []
}`)
	if err := os.MkdirAll(filepath.Join(srcRoot, "_css"), 0o755); err != nil {
		t.Fatalf("MkdirAll(_css): %v", err)
	}
	if err := os.Symlink("../../secret.css", filepath.Join(srcRoot, "_css", "link.css")); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	_, err := Build(srcRoot, outRoot, false)
	if err == nil {
		t.Fatal("expected symlink traversal error, got nil")
	}
	if !strings.Contains(err.Error(), "symlink traversal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRejectsSymlinkTraversalOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	srcRoot := filepath.Join(root, "src")
	outRoot := filepath.Join(root, "html")
	outside := filepath.Join(root, "outside")

	mustWriteFile(t, filepath.Join(srcRoot, "_assets", "bundles.json"), `{
  "css_bundles": [
    {
      "name": "style.css",
      "inputs": ["_css/colors.css"]
    }
  ],
  "js_bundles": []
}`)
	mustWriteFile(t, filepath.Join(srcRoot, "_css", "colors.css"), ":root{--bg:#fff;}")
	if err := os.MkdirAll(outRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(outRoot): %v", err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll(outside): %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "style.css"), filepath.Join(outRoot, "style.css")); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	_, err := Build(srcRoot, outRoot, false)
	if err == nil {
		t.Fatal("expected symlink traversal error, got nil")
	}
	if !strings.Contains(err.Error(), "symlink traversal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildWritesEmptyManifestWithoutConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	srcRoot := filepath.Join(root, "src")
	outRoot := filepath.Join(root, "html")
	if err := os.MkdirAll(srcRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(srcRoot): %v", err)
	}
	if err := os.MkdirAll(outRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(outRoot): %v", err)
	}

	manifest, err := Build(srcRoot, outRoot, false)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(manifest.Assets) != 0 {
		t.Fatalf("expected empty manifest, got %d assets", len(manifest.Assets))
	}

	assertFileContains(t, filepath.Join(outRoot, ManifestFilename), "\"assets\": {}")
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

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	got := string(data)
	if !strings.Contains(got, want) {
		t.Fatalf("file %q does not contain expected content %q; got %q", path, want, got)
	}
}
