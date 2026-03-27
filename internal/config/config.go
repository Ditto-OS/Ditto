package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// Config holds the application configuration
type Config struct {
	TempDir       string
	RuntimesDir   string
	CacheDir      string
	TargetOS      string
	TargetArch    string
}

// RuntimeInfo describes a bundled runtime
type RuntimeInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Path    string `json:"path"`
}

// Default returns a configuration with default values
func Default() *Config {
	cacheDir := filepath.Join(os.TempDir(), "Ditto-cache")
	runtimesDir := filepath.Join(os.TempDir(), "Ditto-runtimes")

	return &Config{
		TempDir:      filepath.Join(os.TempDir(), "Ditto"),
		RuntimesDir:  runtimesDir,
		CacheDir:     cacheDir,
		TargetOS:     runtime.GOOS,
		TargetArch:   runtime.GOARCH,
	}
}

// EnsureDirs creates all necessary directories
func (c *Config) EnsureDirs() error {
	dirs := []string{c.TempDir, c.RuntimesDir, c.CacheDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
