package assets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"brnwb.com/internal/pathsafe"
)

const (
	ConfigPath       = "_assets/bundles.json"
	ManifestFilename = "assets-manifest.json"
)

type BundleConfig struct {
	CSSBundles []Bundle `json:"css_bundles"`
	JSBundles  []Bundle `json:"js_bundles"`
}

type Bundle struct {
	Name   string   `json:"name"`
	Inputs []string `json:"inputs"`
}

type Manifest struct {
	Assets map[string]string `json:"assets"`
}

func Build(srcRoot, outRoot string, verbose bool) (Manifest, error) {
	manifest := Manifest{
		Assets: map[string]string{},
	}

	config, err := loadConfig(srcRoot)
	if err != nil {
		return manifest, err
	}

	if config == nil {
		if err := writeManifest(outRoot, manifest); err != nil {
			return manifest, err
		}
		return manifest, nil
	}

	for _, bundle := range config.CSSBundles {
		if err := buildBundle(srcRoot, outRoot, bundle, manifest.Assets, verbose); err != nil {
			return manifest, fmt.Errorf("build css bundle %q: %w", bundle.Name, err)
		}
	}

	for _, bundle := range config.JSBundles {
		if err := buildBundle(srcRoot, outRoot, bundle, manifest.Assets, verbose); err != nil {
			return manifest, fmt.Errorf("build js bundle %q: %w", bundle.Name, err)
		}
	}

	if err := writeManifest(outRoot, manifest); err != nil {
		return manifest, err
	}

	return manifest, nil
}

func loadConfig(srcRoot string) (*BundleConfig, error) {
	configPath := filepath.Join(srcRoot, filepath.FromSlash(ConfigPath))
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read asset config %q: %w", ConfigPath, err)
	}

	var cfg BundleConfig
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode asset config %q: %w", ConfigPath, err)
	}

	return &cfg, nil
}

func buildBundle(srcRoot, outRoot string, bundle Bundle, manifest map[string]string, verbose bool) error {
	name := filepath.ToSlash(strings.TrimSpace(bundle.Name))
	if name == "" {
		return fmt.Errorf("bundle name is required")
	}
	if strings.HasPrefix(name, "/") {
		return fmt.Errorf("bundle name must be relative: %q", name)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("bundle name cannot contain '..': %q", name)
	}
	if len(bundle.Inputs) == 0 {
		return fmt.Errorf("bundle %q must include at least one input", name)
	}

	content, err := concatInputs(srcRoot, bundle.Inputs)
	if err != nil {
		return fmt.Errorf("concat bundle inputs: %w", err)
	}

	outPath, err := pathsafe.ResolveWithinRoot(outRoot, name)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	if err := pathsafe.EnsureNoSymlinkTraversal(outRoot, outPath); err != nil {
		return fmt.Errorf("validate output path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(outPath, content, 0o644); err != nil {
		return fmt.Errorf("write bundle output: %w", err)
	}

	manifest[name] = name

	if verbose {
		fmt.Printf("assets: bundle %s (%d inputs)\n", name, len(bundle.Inputs))
	}

	return nil
}

func concatInputs(srcRoot string, inputs []string) ([]byte, error) {
	var out bytes.Buffer

	for i, input := range inputs {
		rel := filepath.ToSlash(strings.TrimSpace(input))
		if rel == "" {
			return nil, fmt.Errorf("bundle input cannot be empty")
		}

		inputPath, err := pathsafe.ResolveWithinRoot(srcRoot, rel)
		if err != nil {
			return nil, fmt.Errorf("resolve input %q: %w", rel, err)
		}
		if err := pathsafe.EnsureNoSymlinkTraversal(srcRoot, inputPath); err != nil {
			return nil, fmt.Errorf("reject symlink input %q: %w", rel, err)
		}

		info, err := os.Stat(inputPath)
		if err != nil {
			return nil, fmt.Errorf("stat input %q: %w", rel, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("input %q is a directory", rel)
		}

		data, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, fmt.Errorf("read input %q: %w", rel, err)
		}

		if i > 0 && out.Len() > 0 && out.Bytes()[out.Len()-1] != '\n' {
			out.WriteByte('\n')
		}
		out.Write(data)
		if len(data) > 0 && data[len(data)-1] != '\n' {
			out.WriteByte('\n')
		}
	}

	return out.Bytes(), nil
}

func writeManifest(outRoot string, manifest Manifest) error {
	path := filepath.Join(outRoot, ManifestFilename)

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write asset manifest: %w", err)
	}

	return nil
}
