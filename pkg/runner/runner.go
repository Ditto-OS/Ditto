package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"Ditto/internal/config"
	"Ditto/internal/interpreter"
	"Ditto/pkg/archive"
	"Ditto/pkg/packager"
)

// RunConfig holds execution configuration
type RunConfig struct {
	SourceFile  string
	RuntimeName string
	Args        []string
	WorkingDir  string
	UsePackages bool // Whether to load installed packages
}

// Runner executes code with embedded interpreters
type Runner struct {
	config *config.Config
	engine *interpreter.Engine
}

// NewRunner creates a new runner instance
func NewRunner(cfg *config.Config) *Runner {
	return &Runner{
		config: cfg,
		engine: interpreter.NewEngine(),
	}
}

// Close cleans up resources
func (r *Runner) Close() error {
	if r.engine != nil {
		return r.engine.Close()
	}
	return nil
}

// Run executes a source file
func (r *Runner) Run(cfg RunConfig) error {
	// Detect if this is a bundled binary
	if strings.HasSuffix(cfg.SourceFile, ".bundle") {
		return r.runBundle(cfg)
	}

	// Detect runtime if not specified
	if cfg.RuntimeName == "" {
		cfg.RuntimeName = detectRuntime(cfg.SourceFile)
	}

	// Read source file
	sourceCode, err := os.ReadFile(cfg.SourceFile)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	// Create VFS with package support if enabled
	var vfs fs.FS
	if cfg.UsePackages {
		var err error
		vfs, err = r.createPackageVFS(cfg.WorkingDir, cfg.RuntimeName)
		if err != nil {
			// Log error but continue without packages
			fmt.Fprintf(os.Stderr, "Warning: failed to load packages: %v\n", err)
		}
	}

	// Execute with embedded interpreter
	return r.engine.Execute(cfg.RuntimeName, string(sourceCode), cfg.Args, vfs)
}

func (r *Runner) runBundle(cfg RunConfig) error {
	// Extract bundle to temporary directory
	tmpDir, err := os.MkdirTemp(r.config.TempDir, "Ditto-run-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := archive.ExtractBundle(cfg.SourceFile, tmpDir); err != nil {
		return fmt.Errorf("failed to extract bundle: %w", err)
	}

	// Read metadata
	metaBytes, err := os.ReadFile(filepath.Join(tmpDir, "metadata.json"))
	if err != nil {
		return fmt.Errorf("failed to read bundle metadata: %w", err)
	}
	var meta map[string]string
	json.Unmarshal(metaBytes, &meta)
	lang := meta["runtime"]

	// Read source code from bundle
	// Bundler adds it as "source.ext"
	files, _ := os.ReadDir(tmpDir)
	var sourcePath string
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "source.") {
			sourcePath = filepath.Join(tmpDir, f.Name())
			break
		}
	}

	if sourcePath == "" {
		return fmt.Errorf("source file not found in bundle")
	}

	sourceCode, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	// Create a VFS from the extracted directory
	vfs := os.DirFS(tmpDir)
	
	return r.engine.Execute(lang, string(sourceCode), cfg.Args, vfs)
}

// RunCode executes code directly (for testing)
func (r *Runner) RunCode(lang, code string, args []string) error {
	return r.engine.Execute(lang, code, args, nil)
}

// createPackageVFS creates a virtual filesystem that includes installed packages
func (r *Runner) createPackageVFS(workingDir, lang string) (fs.FS, error) {
	installDir, err := packager.GetSystemPackageDir()
	if err != nil {
		return nil, err
	}

	cacheDir, err := packager.GetSystemCacheDir()
	if err != nil {
		return nil, err
	}

	// Get package manager to find installed packages
	p, err := packager.NewPackager(installDir, cacheDir)
	if err != nil {
		return nil, err
	}

	// Map language to package language
	pkgLang := lang
	if lang == "python" || lang == "py" {
		pkgLang = "python"
	} else if lang == "javascript" || lang == "js" || lang == "node" {
		pkgLang = "javascript"
	}

	// Get all installed packages for this language
	var packageDirs []string
	for _, pkg := range p.ListPackages() {
		if pkg.Language == pkgLang {
			// Add the parent directory so module names resolve correctly
			// e.g., for package 'requests' at /path/python/requests, add /path/python
			// so that requests/__init__.py can be found
			packageDirs = append(packageDirs, filepath.Dir(pkg.Path))
		}
	}

	// Create a multi-root filesystem
	// Working directory takes precedence, then packages
	roots := []string{workingDir}
	roots = append(roots, packageDirs...)

	return newMultiRootFS(roots...), nil
}

// multiRootFS combines multiple directories into a single filesystem
type multiRootFS struct {
	roots []fs.FS
}

func newMultiRootFS(roots ...string) *multiRootFS {
	var fss []fs.FS
	for _, root := range roots {
		if root != "" {
			fss = append(fss, os.DirFS(root))
		}
	}
	return &multiRootFS{roots: fss}
}

func (m *multiRootFS) Open(name string) (fs.File, error) {
	// Try each root in order, first match wins
	for _, f := range m.roots {
		if file, err := f.Open(name); err == nil {
			return file, nil
		}
	}
	return nil, fs.ErrNotExist
}

// detectRuntime detects language from file extension
func detectRuntime(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".py":
		return "python"
	case ".js", ".ts":
		return "javascript"
	case ".lua":
		return "lua"
	case ".sql":
		return "sql"
	case ".c", ".cpp":
		return "c"
	case ".rb":
		return "ruby"
	case ".go":
		return "go"
	default:
		return "python"
	}
}

// GetSupportedLanguages returns list of supported languages
func GetSupportedLanguages() []string {
	return []string{
		"python",
		"javascript",
		"lua",
		"sql",
		"c",
		"ruby",
		"go",
	}
}

// DetectLanguage auto-detects language from file content and extension
func DetectLanguage(filename string, content []byte) string {
	// First check extension
	ext := strings.ToLower(filepath.Ext(filename))
	baseLang := map[string]string{
		".py":   "python",
		".js":   "javascript",
		".ts":   "javascript",
		".lua":  "lua",
		".sql":  "sql",
		".c":    "c",
		".cpp":  "c",
		".h":    "c",
		".rb":   "ruby",
		".go":   "go",
		".rs":   "rust",
		".php":  "php",
		".java": "java",
	}

	if lang, ok := baseLang[ext]; ok {
		return lang
	}

	// Check shebang
	contentStr := string(content)
	if strings.HasPrefix(contentStr, "#!") {
		lines := strings.SplitN(contentStr, "\n", 2)
		shebang := lines[0]
		if strings.Contains(shebang, "python") {
			return "python"
		}
		if strings.Contains(shebang, "node") {
			return "javascript"
		}
		if strings.Contains(shebang, "ruby") {
			return "ruby"
		}
		if strings.Contains(shebang, "lua") {
			return "lua"
		}
	}

	// Check for SQL-like content
	if strings.Contains(strings.ToUpper(contentStr), "SELECT ") &&
		strings.Contains(strings.ToUpper(contentStr), "FROM ") {
		return "sql"
	}

	return "unknown"
}

// FormatOutput formats interpreter output for display
func FormatOutput(output string, lang string) string {
	lines := strings.Split(output, "\n")
	var formatted strings.Builder

	for i, line := range lines {
		if i > 0 {
			formatted.WriteString("\n")
		}
		formatted.WriteString(line)
	}

	return formatted.String()
}

// CreatePipe creates a bidirectional pipe for I/O
func CreatePipe() (io.ReadWriteCloser, io.ReadWriteCloser, error) {
	// Simplified pipe creation
	return nil, nil, fmt.Errorf("not implemented")
}
