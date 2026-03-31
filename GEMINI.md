# GEMINI.md - Ditto Project Context

## Project Overview
**Ditto** is a universal polyglot interpreter and bundler written in Go. Its primary goal is to provide a single, portable binary (~10MB) that can execute code from multiple languages (Python, JavaScript, Lua, SQL, C, etc.) without requiring the user to install any external runtimes or toolchains.

### Key Features
- **Zero-Config Polyglot Execution**: No need to install Python, Node.js, Ruby, or GCC.
- **Embedded Interpreters**: Custom "pure Go" VM implementations and WASM-based interpreters (via `wazero`).
- **Embedded Standard Libraries**: Go-native implementations of core language modules (e.g., Python's `os`, `math`, `sys`; Node.js's `fs`, `path`, `os`, `console`).
- **Standalone Bundling**: Ability to bundle source files into independent executables by embedding the interpreter.

## Architecture & Structure
The project is organized into the following key components:

- **`cmd/ditto/`**: CLI entry point and subcommand handling (`run`, `bundle`, `languages`).
- **`internal/interpreter/`**: Language-specific interpreter logic.
  - `interpreter.go`: Defines the `Interpreter` interface and the central execution `Engine`.
  - Individual implementations: `python.go`, `javascript.go`, `lua.go`, `sql.go`, `c.go`, etc.
- **`internal/stdlib/`**: Go-native implementations of language standard libraries.
- **`internal/runtime/`**: Management of the WASM runtime (`wazero`).
- **`pkg/runner/`**: Orchestration logic for detecting languages and executing files.
- **`pkg/bundler/`**: Logic for creating standalone executables from source files.
- **`pkg/engine/`**: (Internal) core execution engine components.
- **`examples/`**: Sample scripts in various supported languages for testing and demonstration.
- **`scripts/`**: Platform-specific build scripts (`.sh` for Linux/macOS, `.ps1` for Windows).

## Technical Stack
- **Language**: Go 1.21+
- **WASM Runtime**: [wazero](https://github.com/tetratelabs/wazero) (for future/partial WASM-based interpreters).
- **Interpreters**: Combination of custom Go parsers/VMs and embedded WASM modules.

## Building and Running

### Building the Project
You can use the provided scripts or standard Go commands:
- **Windows**: `.\scripts\build.ps1`
- **Linux/macOS**: `./scripts/build.sh`
- **Manual**: `go build -o Ditto.exe ./cmd/ditto`

### Running Code
```bash
./Ditto run examples/hello.py
./Ditto run examples/hello.js
```

### Bundling
```bash
./Ditto bundle examples/hello.py -o myapp
```

### Testing
- **Standard**: `go test ./...`
- **Manual**: Run the scripts in the `examples/` directory using the built `Ditto` binary.

## Development Conventions
- **Interpreters**: All new language support should implement the `Interpreter` interface found in `internal/interpreter/interpreter.go`.
- **Standard Library**: Extensions to language capabilities should be added to `internal/stdlib/stdlib.go` by mapping Go functions to the respective language's module structure.
- **Language Detection**: Logic for auto-detecting languages resides in `pkg/runner/runner.go`.
- **Pure Go VM**: Current implementations (like Python) use regex-based parsing for simplicity. More robust implementations should transition towards proper AST parsing or WASM-based execution.

## Key Files
- `README.md`: Comprehensive overview and quick start guide.
- `go.mod`: Project dependencies (primarily `wazero`).
- `internal/interpreter/interpreter.go`: Core execution interface.
- `internal/stdlib/stdlib.go`: Implementation of embedded language modules.
- `pkg/runner/runner.go`: Language detection and execution flow.
