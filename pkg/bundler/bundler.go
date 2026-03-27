package bundler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"Ditto/internal/config"
	rt "Ditto/internal/runtime"
	"Ditto/pkg/archive"
)

// BundleConfig holds bundling configuration
type BundleConfig struct {
	SourceFile    string
	OutputFile    string
	RuntimeName   string
	IncludeFiles  []string
	EmbedRuntimes bool
}

// Bundler creates self-contained executables
type Bundler struct {
	config  *config.Config
	manager *rt.Manager
}

// NewBundler creates a new bundler instance
func NewBundler(cfg *config.Config) *Bundler {
	return &Bundler{
		config:  cfg,
		manager: rt.NewManager(cfg),
	}
}

// Bundle creates a standalone executable from source
func (b *Bundler) Bundle(cfg BundleConfig) error {
	// Read source file
	sourceCode, err := os.ReadFile(cfg.SourceFile)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	// Determine runtime
	rtName := cfg.RuntimeName
	if rtName == "" {
		rtName = detectRuntime(cfg.SourceFile)
	}

	// Download WASM runtime if embedding
	var wasmPath string
	if cfg.EmbedRuntimes {
		wasmPath, err = b.downloadWASMRuntime(rtName)
		if err != nil {
			fmt.Printf("Warning: Could not download WASM runtime: %v\n", err)
			fmt.Println("Bundle will use system runtime at execution time")
		}
	}

	// Create bundle directory
	bundleDir, err := os.MkdirTemp(b.config.TempDir, "Ditto-bundle-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(bundleDir)

	// Collect files to bundle
	files := make(map[string][]byte)

	// Add source code
	files["source"+filepath.Ext(cfg.SourceFile)] = sourceCode

	// Add WASM runtime if downloaded
	if wasmPath != "" {
		wasmBytes, err := os.ReadFile(wasmPath)
		if err != nil {
			return fmt.Errorf("failed to read WASM runtime: %w", err)
		}
		files["runtime/"+rtName+".wasm"] = wasmBytes
	}

	// Add additional files
	for _, f := range cfg.IncludeFiles {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read include file %s: %w", f, err)
		}
		files[filepath.Base(f)] = content
	}

	// Add metadata
	meta := map[string]interface{}{
		"runtime": rtName,
		"version": "0.1.0",
	}
	metaBytes, _ := json.Marshal(meta)
	files["metadata.json"] = metaBytes

	// Create the bundle
	bundlePath := cfg.OutputFile + ".bundle"
	if err := archive.CreateBundle(bundlePath, files); err != nil {
		return fmt.Errorf("failed to create bundle: %w", err)
	}

	// Create launcher (for now, just rename)
	if err := os.Rename(bundlePath, cfg.OutputFile); err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}

	fmt.Printf("✓ Created standalone executable: %s\n", cfg.OutputFile)
	return nil
}

func (b *Bundler) downloadWASMRuntime(rtName string) (string, error) {
	runtimes := b.manager.GetWASMRuntimes()
	for _, rt := range runtimes {
		if rt.Name == rtName {
			return b.manager.DownloadWASMRuntime(rt)
		}
	}
	return "", fmt.Errorf("unknown runtime: %s", rtName)
}

func detectRuntime(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".py":
		return "micropython"
	case ".js", ".ts":
		return "quickjs"
	case ".rb":
		return "mruby"
	case ".go":
		return "yaegi"
	default:
		return "micropython"
	}
}

// SaveMetadata saves bundle metadata
func (b *Bundler) SaveMetadata(outputPath string, info config.RuntimeInfo) error {
	metaPath := outputPath + ".meta.json"
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, data, 0644)
}
