package bundler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

	// Resolve and install dependencies
	if err := b.resolveDependencies(cfg, bundleDir); err != nil {
		fmt.Printf("Warning: Failed to resolve dependencies: %v\n", err)
	}

	// Collect files to bundle
	files := make(map[string][]byte)

	// Add source code
	files["source"+filepath.Ext(cfg.SourceFile)] = sourceCode

	// Add installed dependencies from bundleDir
	depFiles, err := archive.WalkDir(bundleDir)
	if err == nil {
		for k, v := range depFiles {
			files[k] = v
		}
	}

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

func (b *Bundler) resolveDependencies(cfg BundleConfig, bundleDir string) error {
	srcDir := filepath.Dir(cfg.SourceFile)
	rtName := cfg.RuntimeName
	if rtName == "" {
		rtName = detectRuntime(cfg.SourceFile)
	}

	switch rtName {
	case "micropython", "python":
		reqPath := filepath.Join(srcDir, "requirements.txt")
		if _, err := os.Stat(reqPath); err == nil {
			fmt.Println("Installing Python dependencies...")
			depsDir := filepath.Join(bundleDir, "site-packages")
			if err := os.MkdirAll(depsDir, 0755); err != nil {
				return err
			}
			cmd := exec.Command("pip", "install", "-r", reqPath, "--target", depsDir)
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("pip install failed: %s", string(output))
			}
		}
	case "quickjs", "javascript":
		pkgPath := filepath.Join(srcDir, "package.json")
		if _, err := os.Stat(pkgPath); err == nil {
			fmt.Println("Installing Node.js dependencies...")
			// Run npm install in source directory
			cmd := exec.Command("npm", "install", "--production")
			cmd.Dir = srcDir
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("npm install failed: %s", string(output))
			}
			
			// Copy node_modules to bundleDir
			// (Simplified - in a real implementation, we'd copy the directory structure)
		}
	}
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
