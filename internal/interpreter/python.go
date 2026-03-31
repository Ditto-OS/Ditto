package interpreter

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"Ditto/internal/stdlib"
	"Ditto/pkg/wasm"
)

// PythonInterpreter executes Python code
// Uses embedded MicroPython WASM when available, falls back to pure Go parser
type PythonInterpreter struct {
	wasmManager *wasm.RuntimeManager
}

func NewPythonInterpreter() *PythonInterpreter {
	return &PythonInterpreter{}
}

func (p *PythonInterpreter) Name() string {
	return "python"
}

func (p *PythonInterpreter) Execute(engine *Engine, code string, args []string, stdin io.Reader, stdout, stderr io.Writer, vfs fs.FS) error {
	// Try WASM execution first (full Python support)
	err := p.executeWASM(engine, code, args, stdin, stdout, stderr, vfs)
	if err == nil {
		return nil
	}

	// Log WASM failure for debugging
	fmt.Fprintf(stderr, "WASM execution failed: %v\nFalling back to pure Go interpreter...\n", err)

	// Fall back to pure Go implementation
	return p.executePureGo(code, args, stdin, stdout, stderr, vfs)
}

func (p *PythonInterpreter) executeWASM(engine *Engine, code string, args []string, stdin io.Reader, stdout, stderr io.Writer, vfs fs.FS) error {
	// Get or download Pyodide WASM (full CPython support)
	if p.wasmManager == nil {
		var err error
		p.wasmManager, err = wasm.NewRuntimeManager()
		if err != nil {
			return fmt.Errorf("failed to init WASM manager: %w", err)
		}
	}

	wasmBytes, err := p.wasmManager.GetPyodideWASM()
	if err != nil {
		return fmt.Errorf("failed to get Pyodide WASM: %w", err)
	}

	ctx := context.Background()

	// Pyodide requires a complex initialization with JavaScript glue code
	// For pure Go + WASI execution, we use the pure Go interpreter
	// Full Pyodide support would require the Pyodide JS loader
	_ = ctx
	_ = wasmBytes
	_ = args
	_ = stdin
	_ = stdout
	_ = stderr
	_ = vfs

	// For now, use the pure Go interpreter which supports package loading
	// The infrastructure is in place for when we add full Pyodide support
	return fmt.Errorf("Pyodide requires JavaScript loader - using pure Go interpreter with package support")
}

func (p *PythonInterpreter) executePureGo(code string, args []string, stdin io.Reader, stdout, stderr io.Writer, vfs fs.FS) error {
	// Parse and execute Python code in pure Go
	// This is a simplified Python interpreter supporting common operations

	py := &pythonVM{
		variables: make(map[string]interface{}),
		functions: make(map[string]*pythonFunction),
		classes:   make(map[string]*pythonClass),
		stdlib:    stdlib.NewPythonStdLib(),
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		args:     args,
		vfs:      vfs,
	}
	py.stdlib.Init()

	return py.Run(code)
}

// getMicroPythonWASM returns embedded MicroPython WASM bytes
func getMicroPythonWASM() []byte {
	// Would return embedded WASM when available
	return nil
}
