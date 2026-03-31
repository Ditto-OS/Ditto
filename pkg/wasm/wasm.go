// Package wasm provides embedded WASM runtimes for Ditto
// MicroPython and QuickJS WASM builds for full language support
package wasm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	// Pyodide WASM runtime for full Python support
	// Pyodide is CPython compiled to WebAssembly
	PyodideURL = "https://cdn.jsdelivr.net/pyodide/v0.25.0/full/pyodide.asm.wasm"
	
	// WasmEdge QuickJS WASI runtime for full JavaScript support
	QuickJSURL = "https://github.com/second-state/wasmedge-quickjs/releases/download/v0.5.0-alpha/wasmedge_quickjs.wasm"
	
	// Cache directory
	CacheDirName = "ditto-wasm-cache"
)

// WASMInfo contains information about a WASM runtime
type WASMInfo struct {
	Name       string
	Version    string
	URL        string
	SHA256     string
	Size       int64
	Downloaded time.Time
}

// RuntimeManager manages WASM runtime downloads and caching
type RuntimeManager struct {
	cacheDir   string
	httpClient *http.Client
}

// NewRuntimeManager creates a new WASM runtime manager
func NewRuntimeManager() (*RuntimeManager, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, err
	}

	return &RuntimeManager{
		cacheDir: cacheDir,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}, nil
}

// getCacheDir returns the cache directory for WASM runtimes
func getCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var cacheBase string
	switch runtime.GOOS {
	case "windows":
		cacheBase = os.Getenv("LOCALAPPDATA")
		if cacheBase == "" {
			cacheBase = homeDir
		}
	case "darwin":
		cacheBase = filepath.Join(homeDir, "Library", "Caches")
	default:
		cacheBase = os.Getenv("XDG_CACHE_HOME")
		if cacheBase == "" {
			cacheBase = filepath.Join(homeDir, ".cache")
		}
	}

	cacheDir := filepath.Join(cacheBase, CacheDirName)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	return cacheDir, nil
}

// GetPyodideWASM returns Pyodide WASM bytes (full Python support)
// Downloads and caches if not already present
func (m *RuntimeManager) GetPyodideWASM() ([]byte, error) {
	wasmPath := filepath.Join(m.cacheDir, "pyodide.wasm")
	
	// Try to load from cache
	if data, err := os.ReadFile(wasmPath); err == nil {
		return data, nil
	}

	// Download
	fmt.Println("Downloading Pyodide WASM runtime (full Python support)...")
	data, err := m.downloadWASM(PyodideURL, "pyodide.wasm")
	if err != nil {
		return nil, err
	}

	// Cache it
	if err := os.WriteFile(wasmPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to cache WASM: %w", err)
	}

	fmt.Println("✓ Pyodide WASM cached (", len(data)/1024, "KB )")
	return data, nil
}

// GetQuickJSWASM returns QuickJS WASM bytes (full JavaScript support)
// Downloads and caches if not already present
func (m *RuntimeManager) GetQuickJSWASM() ([]byte, error) {
	wasmPath := filepath.Join(m.cacheDir, "quickjs.wasm")
	
	// Try to load from cache
	if data, err := os.ReadFile(wasmPath); err == nil {
		return data, nil
	}

	// Download
	fmt.Println("Downloading QuickJS WASM runtime (full JavaScript support)...")
	data, err := m.downloadWASM(QuickJSURL, "quickjs.wasm")
	if err != nil {
		return nil, err
	}

	// Cache it
	if err := os.WriteFile(wasmPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to cache WASM: %w", err)
	}

	fmt.Println("✓ QuickJS WASM cached (", len(data)/1024, "KB )")
	return data, nil
}

// downloadWASM downloads a WASM file from URL
func (m *RuntimeManager) downloadWASM(url, filename string) ([]byte, error) {
	resp, err := m.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Verify it's a WASM file (magic number: 0x00 0x61 0x73 0x6D = "\0asm")
	if len(data) < 4 || data[0] != 0x00 || data[1] != 0x61 || data[2] != 0x73 || data[3] != 0x6D {
		return nil, fmt.Errorf("invalid WASM file")
	}

	return data, nil
}

// IsWASMAvailable checks if WASM runtimes are cached
func (m *RuntimeManager) IsWASMAvailable() (pyodide, quickjs bool) {
	pyPath := filepath.Join(m.cacheDir, "pyodide.wasm")
	qjsPath := filepath.Join(m.cacheDir, "quickjs.wasm")

	_, err1 := os.Stat(pyPath)
	_, err2 := os.Stat(qjsPath)

	return err1 == nil, err2 == nil
}

// ClearCache removes all cached WASM runtimes
func (m *RuntimeManager) ClearCache() error {
	return os.RemoveAll(m.cacheDir)
}

// GetCacheInfo returns information about cached runtimes
func (m *RuntimeManager) GetCacheInfo() (map[string]WASMInfo, error) {
	info := make(map[string]WASMInfo)

	files := map[string]string{
		"micropython.wasm": "MicroPython",
		"quickjs.wasm":     "QuickJS",
	}

	for filename, name := range files {
		path := filepath.Join(m.cacheDir, filename)
		stat, err := os.Stat(path)
		if err == nil {
			info[name] = WASMInfo{
				Name:       name,
				Size:       stat.Size(),
				Downloaded: stat.ModTime(),
			}
		}
	}

	return info, nil
}

// VerifySHA256 verifies the SHA256 hash of data
func VerifySHA256(data []byte, expected string) bool {
	hash := sha256.Sum256(data)
	actual := hex.EncodeToString(hash[:])
	return actual == expected
}

// DownloadProgress wraps an io.Reader with progress reporting
type DownloadProgress struct {
	reader   io.Reader
	total    int64
	current  int64
	lastPrint time.Time
}

func (p *DownloadProgress) Read(buf []byte) (int, error) {
	n, err := p.reader.Read(buf)
	p.current += int64(n)

	// Print progress every 500ms
	if time.Since(p.lastPrint) > 500*time.Millisecond {
		percent := float64(p.current) / float64(p.total) * 100
		fmt.Printf("\r  Downloading: %.1f%% (%d/%d bytes)", 
			percent, p.current, p.total)
		p.lastPrint = time.Now()
	}

	return n, err
}
