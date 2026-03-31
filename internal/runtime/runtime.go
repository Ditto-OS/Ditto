package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"Ditto/internal/config"
)

// Manager handles WASM runtime downloads
type Manager struct {
	config *config.Config
}

// NewManager creates a new runtime manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{config: cfg}
}

// WASMRuntimeSource defines a WASM runtime to download
type WASMRuntimeSource struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	Sha256 string `json:"sha256"`
}

// GetWASMRuntimes returns embedded WASM runtime sources
func (m *Manager) GetWASMRuntimes() []WASMRuntimeSource {
	return []WASMRuntimeSource{
		{
			Name: "micropython",
			URL:  "https://github.com/lepture/micropython-wasm/releases/download/v1.0/micropython.wasm",
			Size: 2 * 1024 * 1024,
		},
		{
			Name: "quickjs",
			URL:  "https://github.com/jedisct1/webassemblyjs-quickjs/releases/download/2021-09-07/quickjs.wasm",
			Size: 1 * 1024 * 1024,
		},
		{
			Name: "mruby",
			URL:  "",
			Size: 1 * 1024 * 1024,
		},
		{
			Name: "yaegi",
			URL:  "",
			Size: 3 * 1024 * 1024,
		},
	}
}

// DownloadWASMRuntime downloads a WASM runtime
func (m *Manager) DownloadWASMRuntime(src WASMRuntimeSource) (string, error) {
	wasmPath := filepath.Join(m.config.RuntimesDir, src.Name+".wasm")
	
	if _, err := os.Stat(wasmPath); err == nil {
		return wasmPath, nil
	}

	if src.URL == "" {
		return "", fmt.Errorf("no download URL available")
	}

	resp, err := http.Get(src.URL)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	out, err := os.Create(wasmPath)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		return "", err
	}

	return wasmPath, nil
}

// LoadWASM loads a WASM runtime binary from disk
func (m *Manager) LoadWASM(name string) ([]byte, error) {
	wasmPath := filepath.Join(m.config.RuntimesDir, name+".wasm")
	if _, err := os.Stat(wasmPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("WASM runtime %s not found", name)
	}
	return os.ReadFile(wasmPath)
}

// SaveMetadata saves runtime metadata
func (m *Manager) SaveMetadata(info config.RuntimeInfo) error {
	metaPath := filepath.Join(m.config.RuntimesDir, "metadata.json")
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metaPath, data, 0644)
}

// LoadMetadata loads runtime metadata
func (m *Manager) LoadMetadata() (*config.RuntimeInfo, error) {
	metaPath := filepath.Join(m.config.RuntimesDir, "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}
	var info config.RuntimeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}
