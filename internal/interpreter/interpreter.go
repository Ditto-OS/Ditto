// Package interpreter provides embedded interpreters for multiple languages
// All interpreters are implemented in pure Go or use embedded WASM
package interpreter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Interpreter executes code in a specific language
type Interpreter interface {
	Name() string
	Execute(code string, args []string, stdin io.Reader, stdout, stderr io.Writer) error
}

// Engine manages all interpreters
type Engine struct {
	wasmRuntime   wazero.Runtime
	interpreters  map[string]Interpreter
}

// NewEngine creates a new execution engine
func NewEngine() *Engine {
	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.Instantiate(ctx, runtime)

	e := &Engine{
		wasmRuntime:  runtime,
		interpreters: make(map[string]Interpreter),
	}

	// Register built-in interpreters
	e.Register(&PythonInterpreter{})
	e.Register(&JavaScriptInterpreter{})
	e.Register(&LuaInterpreter{})
	e.Register(&SQLInterpreter{})
	e.Register(&CInterpreter{})
	e.Register(&RubyInterpreter{})
	e.Register(&GoInterpreter{})

	return e
}

// Close cleans up resources
func (e *Engine) Close() error {
	if e.wasmRuntime != nil {
		return e.wasmRuntime.Close(context.Background())
	}
	return nil
}

// Register adds an interpreter
func (e *Engine) Register(interp Interpreter) {
	e.interpreters[interp.Name()] = interp
}

// GetInterpreter returns an interpreter by name
func (e *Engine) GetInterpreter(name string) (Interpreter, error) {
	interp, ok := e.interpreters[name]
	if !ok {
		return nil, fmt.Errorf("interpreter not found: %s", name)
	}
	return interp, nil
}

// Execute runs code with the specified interpreter
func (e *Engine) Execute(lang, code string, args []string) error {
	interp, err := e.GetInterpreter(lang)
	if err != nil {
		return err
	}

	var stdout, stderr bytes.Buffer
	err = interp.Execute(code, args, os.Stdin, &stdout, &stderr)

	// Print output
	if stdout.Len() > 0 {
		fmt.Print(stdout.String())
	}
	if stderr.Len() > 0 {
		fmt.Fprint(os.Stderr, stderr.String())
	}

	return err
}

// executeWASM runs a WASM module with the given code
func (e *Engine) executeWASM(ctx context.Context, wasmBytes []byte, code string, args []string, stdout, stderr io.Writer) error {
	module, err := e.wasmRuntime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return fmt.Errorf("failed to compile WASM: %w", err)
	}
	defer module.Close(ctx)

	_, err = e.wasmRuntime.InstantiateModule(ctx, module,
		wazero.NewModuleConfig().
			WithName("interpreter").
			WithStdout(stdout).
			WithStderr(stderr).
			WithStdin(os.Stdin).
			WithArgs(append([]string{"interpreter"}, args...)...))

	return err
}
