package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"Ditto/internal/config"
	"Ditto/internal/interpreter"
)

// RunConfig holds execution configuration
type RunConfig struct {
	SourceFile string
	RuntimeName string
	Args       []string
	WorkingDir string
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
	// Detect runtime if not specified
	if cfg.RuntimeName == "" {
		cfg.RuntimeName = detectRuntime(cfg.SourceFile)
	}

	// Read source file
	sourceCode, err := os.ReadFile(cfg.SourceFile)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	// Execute with embedded interpreter
	return r.engine.Execute(cfg.RuntimeName, string(sourceCode), cfg.Args)
}

// RunCode executes code directly (for testing)
func (r *Runner) RunCode(lang, code string, args []string) error {
	return r.engine.Execute(lang, code, args)
}

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
