package build

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"brnwb.com/internal/assets"
	"brnwb.com/internal/pathsafe"
)

type Config struct {
	InDir   string
	OutDir  string
	Clean   bool
	Verbose bool
}

func Run(cfg Config) error {
	if cfg.InDir == "" {
		return fmt.Errorf("input directory is required")
	}
	if cfg.OutDir == "" {
		return fmt.Errorf("output directory is required")
	}

	inAbs, err := filepath.Abs(cfg.InDir)
	if err != nil {
		return fmt.Errorf("resolve input directory: %w", err)
	}

	outAbs, err := filepath.Abs(cfg.OutDir)
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}

	if inAbs == outAbs {
		return fmt.Errorf("input and output directories must differ")
	}

	inStat, err := os.Stat(inAbs)
	if err != nil {
		return fmt.Errorf("stat input directory: %w", err)
	}
	if !inStat.IsDir() {
		return fmt.Errorf("input path is not a directory: %s", cfg.InDir)
	}

	if strings.HasPrefix(outAbs+string(os.PathSeparator), inAbs+string(os.PathSeparator)) {
		return fmt.Errorf("output directory cannot be nested inside input directory")
	}

	if cfg.Clean {
		logf(cfg, "clean %s", outAbs)
		if err := os.RemoveAll(outAbs); err != nil {
			return fmt.Errorf("clean output directory: %w", err)
		}
	}

	if err := os.MkdirAll(outAbs, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	logf(cfg, "build %s -> %s", inAbs, outAbs)

	manifest, err := assets.Build(inAbs, outAbs, cfg.Verbose)
	if err != nil {
		return fmt.Errorf("build assets: %w", err)
	}

	err = filepath.WalkDir(inAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if isIgnored(d.Name(), d.IsDir()) {
			logf(cfg, "skip %s", path)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(inAbs, path)
		if err != nil {
			return fmt.Errorf("compute relative path for %s: %w", path, err)
		}

		if rel == "." {
			return nil
		}

		if isPipelineSource(rel) {
			logf(cfg, "skip pipeline source %s", path)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relSlash := filepath.ToSlash(rel)
		if _, exists := manifest.Assets[relSlash]; exists {
			logf(cfg, "skip source shadowed by bundle output %s", path)
			return nil
		}

		outPath := filepath.Join(outAbs, rel)

		if d.IsDir() {
			logf(cfg, "mkdir %s", outPath)
			return os.MkdirAll(outPath, 0o755)
		}

		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not supported: %s", path)
		}

		if isTemplateFile(path) {
			logf(cfg, "render %s -> %s", path, outPath)
			return renderFile(inAbs, path, outPath, manifest)
		}

		logf(cfg, "copy %s -> %s", path, outPath)

		return copyFile(path, outPath)
	})
	if err != nil {
		return fmt.Errorf("walk source: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create output parent directory: %w", err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer in.Close()

	mode := info.Mode() & os.ModePerm
	if mode == 0 {
		mode = 0o644
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}

	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("copy file data: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close output file: %w", closeErr)
	}

	return nil
}

func renderFile(srcRoot, src, dst string, manifest assets.Manifest) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source template: %w", err)
	}

	renderer := &templateRenderer{
		srcRoot: srcRoot,
		assets:  manifest.Assets,
	}

	rendered, err := renderer.render(src, nil)
	if err != nil {
		return err
	}

	return writeFileWithMode(dst, []byte(rendered), info.Mode()&os.ModePerm)
}

type templateRenderer struct {
	srcRoot string
	assets  map[string]string
}

func (r *templateRenderer) render(templatePath string, stack []string) (string, error) {
	templateAbs, err := filepath.Abs(templatePath)
	if err != nil {
		return "", fmt.Errorf("resolve template path %q: %w", templatePath, err)
	}

	ok, err := pathsafe.IsWithinRoot(r.srcRoot, templateAbs)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("template path %q is outside source root", templateAbs)
	}

	for _, parent := range stack {
		if parent == templateAbs {
			cycle := append(append([]string{}, stack...), templateAbs)
			return "", fmt.Errorf("render include cycle detected: %s", strings.Join(cycle, " -> "))
		}
	}

	content, err := os.ReadFile(templateAbs)
	if err != nil {
		return "", fmt.Errorf("read template %q: %w", templateAbs, err)
	}

	nextStack := append(append([]string{}, stack...), templateAbs)

	tmpl := template.New(filepath.ToSlash(templateAbs)).Funcs(template.FuncMap{
		"render": func(includePath string) (string, error) {
			resolved, err := resolveIncludePath(r.srcRoot, templateAbs, includePath)
			if err != nil {
				return "", err
			}
			return r.render(resolved, nextStack)
		},
		"asset": func(name string) (string, error) {
			if strings.TrimSpace(name) == "" {
				return "", fmt.Errorf("asset name cannot be empty")
			}
			path, ok := r.assets[name]
			if !ok {
				return "", fmt.Errorf("asset %q not found in manifest", name)
			}
			return path, nil
		},
	})

	tmpl, err = tmpl.Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", templateAbs, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return "", fmt.Errorf("execute template %q: %w", templateAbs, err)
	}

	return buf.String(), nil
}

func resolveIncludePath(srcRoot, currentTemplate, includePath string) (string, error) {
	if includePath == "" {
		return "", fmt.Errorf("include path cannot be empty")
	}

	cleanInclude := filepath.Clean(includePath)
	resolved := filepath.Join(filepath.Dir(currentTemplate), cleanInclude)
	resolvedAbs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve include path %q: %w", includePath, err)
	}

	ok, err := pathsafe.IsWithinRoot(srcRoot, resolvedAbs)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("include path %q resolves outside source root", includePath)
	}

	info, err := os.Stat(resolvedAbs)
	if err != nil {
		return "", fmt.Errorf("stat include %q: %w", includePath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("include path %q resolves to a directory", includePath)
	}
	if err := pathsafe.EnsureNoSymlinkTraversal(srcRoot, resolvedAbs); err != nil {
		return "", fmt.Errorf("include path %q is invalid: %w", includePath, err)
	}

	return resolvedAbs, nil
}

func isTemplateFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".html"
}

var pipelineSourcePrefixes = []string{"_assets", "_css", "_js"}

func isPipelineSource(relPath string) bool {
	relPath = filepath.ToSlash(relPath)
	if relPath == "" {
		return false
	}

	for _, prefix := range pipelineSourcePrefixes {
		if relPath == prefix || strings.HasPrefix(relPath, prefix+"/") {
			return true
		}
	}

	return false
}

func writeFileWithMode(dst string, content []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create output parent directory: %w", err)
	}

	if mode == 0 {
		mode = 0o644
	}

	if err := os.WriteFile(dst, content, mode); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	return nil
}

func isIgnored(name string, isDir bool) bool {
	if name == "" {
		return false
	}

	// Ignore common cross-platform metadata files.
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
	fmt.Printf(format+"\n", args...)
}
