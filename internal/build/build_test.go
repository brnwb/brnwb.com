package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCopiesTreeAndIgnoresMetadataFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inDir := filepath.Join(root, "src")
	outDir := filepath.Join(root, "html")

	mustWriteFile(t, filepath.Join(inDir, "index.html"), "<h1>Hello</h1>")
	mustWriteFile(t, filepath.Join(inDir, "nested", "about.txt"), "about")
	mustWriteFile(t, filepath.Join(inDir, ".DS_Store"), "metadata")
	mustWriteFile(t, filepath.Join(inDir, "nested", "Thumbs.db"), "metadata")

	err := Run(Config{
		InDir:  inDir,
		OutDir: outDir,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertFileContent(t, filepath.Join(outDir, "index.html"), "<h1>Hello</h1>")
	assertFileContent(t, filepath.Join(outDir, "nested", "about.txt"), "about")

	assertNotExists(t, filepath.Join(outDir, ".DS_Store"))
	assertNotExists(t, filepath.Join(outDir, "nested", "Thumbs.db"))
}

func TestRunBuildsAssetsAndRendersAssetHelper(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inDir := filepath.Join(root, "src")
	outDir := filepath.Join(root, "html")

	mustWriteFile(t, filepath.Join(inDir, "_assets", "bundles.json"), `{
  "css_bundles": [
    {
      "name": "style.css",
      "inputs": ["_css/colors.css", "_css/elements.css"]
    }
  ],
  "js_bundles": [
    {
      "name": "app/main.js",
      "inputs": ["_js/part-a.js", "_js/part-b.js"]
    }
  ]
}`)
	mustWriteFile(t, filepath.Join(inDir, "_css", "colors.css"), ":root{--bg:#fff;}")
	mustWriteFile(t, filepath.Join(inDir, "_css", "elements.css"), "body{color:#111;}")
	mustWriteFile(t, filepath.Join(inDir, "_js", "part-a.js"), "const a = 1;")
	mustWriteFile(t, filepath.Join(inDir, "_js", "part-b.js"), "const b = 2;")

	mustWriteFile(t, filepath.Join(inDir, "index.html"), `<link rel="stylesheet" href='{{ asset "style.css" }}' />`)
	mustWriteFile(t, filepath.Join(inDir, "app.html"), `<script src='{{ asset "app/main.js" }}'></script>`)

	err := Run(Config{
		InDir:  inDir,
		OutDir: outDir,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertFileContains(t, filepath.Join(outDir, "index.html"), `href='style.css'`)
	assertFileContains(t, filepath.Join(outDir, "app.html"), `src='app/main.js'`)

	assertFileContent(t, filepath.Join(outDir, "style.css"), ":root{--bg:#fff;}\nbody{color:#111;}\n")
	assertFileContent(t, filepath.Join(outDir, "app", "main.js"), "const a = 1;\nconst b = 2;\n")
	assertFileContains(t, filepath.Join(outDir, "assets-manifest.json"), `"style.css": "style.css"`)
	assertFileContains(t, filepath.Join(outDir, "assets-manifest.json"), `"app/main.js": "app/main.js"`)

	assertNotExists(t, filepath.Join(outDir, "_assets"))
	assertNotExists(t, filepath.Join(outDir, "_css"))
	assertNotExists(t, filepath.Join(outDir, "_js"))
}

func TestRunRejectsMissingAssetReference(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inDir := filepath.Join(root, "src")
	outDir := filepath.Join(root, "html")
	mustWriteFile(t, filepath.Join(inDir, "index.html"), `<link href='{{ asset "missing.css" }}' />`)

	err := Run(Config{
		InDir:  inDir,
		OutDir: outDir,
	})
	assertErrorContains(t, err, "not found in manifest")
}

func TestRunRejectsOutputNestedInInput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inDir := filepath.Join(root, "src")
	outDir := filepath.Join(inDir, "public")
	mustWriteFile(t, filepath.Join(inDir, "index.html"), "hello")

	err := Run(Config{
		InDir:  inDir,
		OutDir: outDir,
	})
	if err == nil {
		t.Fatal("expected error for nested output directory, got nil")
	}
}

func TestRunCleanRemovesStaleOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inDir := filepath.Join(root, "src")
	outDir := filepath.Join(root, "html")

	mustWriteFile(t, filepath.Join(inDir, "index.html"), "new")
	mustWriteFile(t, filepath.Join(outDir, "stale.txt"), "stale")

	err := Run(Config{
		InDir:  inDir,
		OutDir: outDir,
		Clean:  true,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertNotExists(t, filepath.Join(outDir, "stale.txt"))
	assertFileContent(t, filepath.Join(outDir, "index.html"), "new")
}

func TestRunRendersTemplatesWithNestedIncludes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inDir := filepath.Join(root, "src")
	outDir := filepath.Join(root, "html")

	mustWriteFile(t, filepath.Join(inDir, "fragments", "header.html"), "<header>Header</header>")
	mustWriteFile(t, filepath.Join(inDir, "index.html"), `<main>{{ render "./fragments/header.html" }}</main>`)

	err := Run(Config{
		InDir:  inDir,
		OutDir: outDir,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertFileContent(t, filepath.Join(outDir, "index.html"), "<main><header>Header</header></main>")
}

func TestRunRejectsIncludeOutsideSourceRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inDir := filepath.Join(root, "src")
	outDir := filepath.Join(root, "html")
	mustWriteFile(t, filepath.Join(root, "secret.html"), "secret")
	mustWriteFile(t, filepath.Join(inDir, "index.html"), `{{ render "../secret.html" }}`)

	err := Run(Config{
		InDir:  inDir,
		OutDir: outDir,
	})
	assertErrorContains(t, err, "outside source root")
}

func TestRunRejectsIncludeSymlinkTraversal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inDir := filepath.Join(root, "src")
	outDir := filepath.Join(root, "html")
	mustWriteFile(t, filepath.Join(root, "secret.html"), "secret")
	if err := os.MkdirAll(filepath.Join(inDir, "_css"), 0o755); err != nil {
		t.Fatalf("MkdirAll(_css): %v", err)
	}
	if err := os.Symlink("../../secret.html", filepath.Join(inDir, "_css", "link.html")); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}
	mustWriteFile(t, filepath.Join(inDir, "index.html"), `{{ render "./_css/link.html" }}`)

	err := Run(Config{
		InDir:  inDir,
		OutDir: outDir,
	})
	assertErrorContains(t, err, "symlink traversal")
}

func TestRunRejectsIncludeCycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	inDir := filepath.Join(root, "src")
	outDir := filepath.Join(root, "html")

	mustWriteFile(t, filepath.Join(inDir, "index.html"), `{{ render "./a.html" }}`)
	mustWriteFile(t, filepath.Join(inDir, "a.html"), `A{{ render "./b.html" }}`)
	mustWriteFile(t, filepath.Join(inDir, "b.html"), `B{{ render "./a.html" }}`)

	err := Run(Config{
		InDir:  inDir,
		OutDir: outDir,
	})
	assertErrorContains(t, err, "cycle")
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

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("file %q content mismatch: got %q, want %q", path, string(got), want)
	}
}

func assertFileContains(t *testing.T, path, wantFragment string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if !strings.Contains(string(got), wantFragment) {
		t.Fatalf("file %q does not contain %q; got %q", path, wantFragment, string(got))
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		t.Fatalf("expected %q to not exist, got err=%v", path, err)
	}
}

func assertErrorContains(t *testing.T, err error, fragment string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q, got nil", fragment)
	}
	if !strings.Contains(err.Error(), fragment) {
		t.Fatalf("error mismatch: got %q, want fragment %q", err.Error(), fragment)
	}
}
