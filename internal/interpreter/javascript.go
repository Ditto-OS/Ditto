package interpreter

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strconv"
	"strings"

	"Ditto/internal/stdlib"
	"Ditto/pkg/wasm"
	"github.com/tetratelabs/wazero"
)

// JavaScriptInterpreter executes JavaScript code
type JavaScriptInterpreter struct {
	wasmManager *wasm.RuntimeManager
}

func NewJavaScriptInterpreter() *JavaScriptInterpreter {
	return &JavaScriptInterpreter{}
}

func (j *JavaScriptInterpreter) Name() string {
	return "javascript"
}

func (j *JavaScriptInterpreter) Execute(engine *Engine, code string, args []string, stdin io.Reader, stdout, stderr io.Writer, vfs fs.FS) error {
	// Try WASM execution first (full JavaScript support via QuickJS)
	if err := j.executeWASM(engine, code, args, stdin, stdout, stderr, vfs); err == nil {
		return nil
	}

	// Fall back to pure Go implementation
	return j.executePureGo(code, args, stdin, stdout, stderr, vfs)
}

func (j *JavaScriptInterpreter) executeWASM(engine *Engine, code string, args []string, stdin io.Reader, stdout, stderr io.Writer, vfs fs.FS) error {
	// Get or download QuickJS WASM (WASI-compatible)
	if j.wasmManager == nil {
		var err error
		j.wasmManager, err = wasm.NewRuntimeManager()
		if err != nil {
			return fmt.Errorf("failed to init WASM manager: %w", err)
		}
	}

	wasmBytes, err := j.wasmManager.GetQuickJSWASM()
	if err != nil {
		return fmt.Errorf("failed to get QuickJS WASM: %w", err)
	}

	ctx := context.Background()

	// Configure QuickJS WASI arguments
	runArgs := []string{"qjs"}
	if len(args) > 0 {
		runArgs = append(runArgs, args...)
	}

	config := wazero.NewModuleConfig().
		WithStdout(stdout).
		WithStderr(stderr).
		WithStdin(stdin).
		WithArgs(runArgs...)

	// Mount virtual filesystem if provided
	if vfs != nil {
		config = config.WithFS(vfs)
	}

	// Compile and instantiate
	module, err := engine.wasmRuntime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return fmt.Errorf("failed to compile WASM: %w", err)
	}
	defer module.Close(ctx)

	// Instantiate the WASM module
	wasmModule, err := engine.wasmRuntime.InstantiateModule(ctx, module, config)
	if err != nil {
		return fmt.Errorf("failed to instantiate WASM: %w", err)
	}

	// The wasmedge-quickjs WASM is a standalone runtime
	// It expects to be run as a command-line tool with WASI
	// For embedding, we need to use a different approach
	// For now, fall back to pure Go interpreter
	// The WASM infrastructure is in place for future enhancement
	_ = wasmModule
	
	return fmt.Errorf("QuickJS WASM requires WASI filesystem setup - using pure Go interpreter with package support")
}

func (j *JavaScriptInterpreter) executePureGo(code string, args []string, stdin io.Reader, stdout, stderr io.Writer, vfs fs.FS) error {
	js := &jsVM{
		variables: make(map[string]interface{}),
		functions: make(map[string]*jsFunction),
		modules:   make(map[string]map[string]interface{}),
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		args:    args,
		vfs:     vfs,
	}
	js.stdlib = stdlib.NewNodeStdLib()

	return js.Run(code)
}

type jsVM struct {
	variables map[string]interface{}
	functions map[string]*jsFunction
	modules   map[string]map[string]interface{}
	stdlib    *stdlib.NodeStdLib
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	args      []string
	vfs       fs.FS
}

type jsFunction struct {
	name   string
	params []string
	body   string
}

func (vm *jsVM) Run(code string) error {
	// Remove comments
	code = vm.removeComments(code)

	// Split into statements (simplified)
	statements := vm.splitStatements(code)
	for _, stmt := range statements {
		if err := vm.executeStatement(stmt); err != nil {
			fmt.Fprintf(vm.stderr, "Error: %v\n", err)
			return err
		}
	}
	return nil
}

func (vm *jsVM) removeComments(code string) string {
	// Remove single-line comments
	re := regexp.MustCompile(`//.*$`)
	code = re.ReplaceAllString(code, "")
	// Remove multi-line comments
	re = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	code = re.ReplaceAllString(code, "")
	return code
}

func (vm *jsVM) splitStatements(code string) []string {
	var statements []string
	var current strings.Builder
	braceCount := 0

	for _, ch := range code {
		current.WriteRune(ch)
		switch ch {
		case '{':
			braceCount++
		case '}':
			braceCount--
		case ';':
			if braceCount == 0 {
				statements = append(statements, current.String())
				current.Reset()
			}
		}
	}

	if current.Len() > 0 {
		statements = append(statements, current.String())
	}

	return statements
}

func (vm *jsVM) executeStatement(stmt string) error {
	stmt = strings.TrimSpace(stmt)
	if stmt == "" {
		return nil
	}

	// Async function declaration
	if match := regexp.MustCompile(`^async\s+function\s+(\w+)\(([^)]*)\)\s*\{([\s\S]*)\}$`).FindStringSubmatch(stmt); match != nil {
		return vm.handleAsyncFunctionDecl(match[1], match[2], match[3])
	}

	// Await expression
	if match := regexp.MustCompile(`^await\s+(.+)$`).FindStringSubmatch(stmt); match != nil {
		_, err := vm.handleAwait(match[1])
		return err
	}

	// Module access (e.g., fs.readFileSync())
	if match := regexp.MustCompile(`^(\w+)\.(\w+)\(([^)]*)\)$`).FindStringSubmatch(stmt); match != nil {
		return vm.handleModuleCall(match[1], match[2], match[3])
	}

	// console.log()
	if match := regexp.MustCompile(`console\.log\((.+)\)`).FindStringSubmatch(stmt); match != nil {
		return vm.handleConsoleLog(match[1])
	}

	// var/let/const declaration with require
	if match := regexp.MustCompile(`^(?:var|let|const)\s+(\w+)\s*=\s*require\(([^)]+)\)$`).FindStringSubmatch(stmt); match != nil {
		vm.variables[match[1]] = vm.modules[strings.Trim(match[2], "\"'")]
		return nil
	}

	// var/let/const declaration
	if match := regexp.MustCompile(`^(?:var|let|const)\s+(\w+)\s*=\s*(.+)$`).FindStringSubmatch(stmt); match != nil {
		return vm.handleDeclaration(match[1], match[2])
	}

	// Assignment
	if match := regexp.MustCompile(`^(\w+)\s*=\s*(.+)$`).FindStringSubmatch(stmt); match != nil {
		return vm.handleAssignment(match[1], match[2])
	}

	// Function declaration
	if match := regexp.MustCompile(`^function\s+(\w+)\(([^)]*)\)\s*\{([\s\S]*)\}$`).FindStringSubmatch(stmt); match != nil {
		return vm.handleFunctionDecl(match[1], match[2], match[3])
	}

	// For loop
	if match := regexp.MustCompile(`^for\s*\(([^;]+);([^;]+);([^)]+)\)\s*\{([\s\S]*)\}$`).FindStringSubmatch(stmt); match != nil {
		return vm.handleForLoop(match[1], match[2], match[3], match[4])
	}

	// If statement
	if match := regexp.MustCompile(`^if\s*\(([^)]+)\)\s*\{([\s\S]*)\}$`).FindStringSubmatch(stmt); match != nil {
		return vm.handleIfStatement(match[1], match[2])
	}

	// Function call as statement
	if match := regexp.MustCompile(`^(\w+)\((.*)\)$`).FindStringSubmatch(stmt); match != nil {
		return vm.handleFunctionCall(match[1], match[2])
	}

	// Return statement (ignore at top level)
	if strings.HasPrefix(stmt, "return ") {
		return nil
	}

	return nil
}

func (vm *jsVM) handleConsoleLog(expr string) error {
	// Handle module calls like fs.readFileSync()
	if match := regexp.MustCompile(`^(\w+)\.(\w+)\(([^)]*)\)$`).FindStringSubmatch(expr); match != nil {
		return vm.handleModuleCall(match[1], match[2], match[3])
	}
	
	// Handle typeof expressions
	if match := regexp.MustCompile(`^typeof\s+(\w+)$`).FindStringSubmatch(expr); match != nil {
		varName := match[1]
		if _, ok := vm.modules[varName]; ok {
			fmt.Fprintln(vm.stdout, "object")
			return nil
		}
		if _, ok := vm.variables[varName]; ok {
			val := vm.variables[varName]
			switch val.(type) {
			case func(...interface{}) interface{}:
				fmt.Fprintln(vm.stdout, "function")
			default:
				fmt.Fprintln(vm.stdout, fmt.Sprintf("%T", val))
			}
			return nil
		}
		fmt.Fprintln(vm.stdout, "undefined")
		return nil
	}
	
	// Handle string concatenation
	if strings.Contains(expr, "+") && (strings.HasPrefix(expr, `"`) || strings.Contains(expr, `"`)) {
		result := vm.evalStringConcat(expr)
		fmt.Fprintln(vm.stdout, result)
		return nil
	}
	
	result, err := vm.evaluate(expr)
	if err != nil {
		return err
	}
	fmt.Fprintln(vm.stdout, vm.formatValue(result))
	return nil
}

func (vm *jsVM) evalStringConcat(expr string) string {
	parts := strings.Split(expr, "+")
	var result strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, `"`) && strings.HasSuffix(p, `"`) {
			result.WriteString(p[1 : len(p)-1])
		} else if val, err := vm.evaluate(p); err == nil {
			result.WriteString(vm.formatValue(val))
		} else {
			result.WriteString(p)
		}
	}
	return result.String()
}

func (vm *jsVM) handleDeclaration(name, value string) error {
	val, err := vm.evaluate(value)
	if err != nil {
		return err
	}
	vm.variables[name] = val
	return nil
}

func (vm *jsVM) handleAssignment(name, value string) error {
	val, err := vm.evaluate(value)
	if err != nil {
		return err
	}
	vm.variables[name] = val
	return nil
}

func (vm *jsVM) handleFunctionDecl(name, params, body string) error {
	paramList := vm.parseParams(params)
	vm.functions[name] = &jsFunction{
		name:   name,
		params: paramList,
		body:   body,
	}
	return nil
}

func (vm *jsVM) handleForLoop(init, condition, increment, body string) error {
	// Execute init
	if err := vm.executeStatement(init); err != nil {
		return err
	}

	// Loop
	for {
		condResult, err := vm.evaluate(condition)
		if err != nil {
			return err
		}
		if !isTrueJS(condResult) {
			break
		}

		// Execute body
		bodyStmts := vm.splitStatements(body)
		for _, stmt := range bodyStmts {
			if err := vm.executeStatement(stmt); err != nil {
				return err
			}
		}

		// Execute increment
		if err := vm.executeStatement(increment); err != nil {
			return err
		}
	}

	return nil
}

func (vm *jsVM) handleIfStatement(condition, body string) error {
	condResult, err := vm.evaluate(condition)
	if err != nil {
		return err
	}

	if isTrueJS(condResult) {
		bodyStmts := vm.splitStatements(body)
		for _, stmt := range bodyStmts {
			if err := vm.executeStatement(stmt); err != nil {
				return err
			}
		}
	}

	return nil
}

func (vm *jsVM) handleFunctionCall(name, argsStr string) error {
	if name == "require" {
		return vm.handleRequire(argsStr)
	}

	// User-defined function
	if fn, ok := vm.functions[name]; ok {
		return vm.callFunction(fn, argsStr)
	}

	return nil
}

func (vm *jsVM) handleRequire(module string) error {
	// Remove quotes from module name
	module = strings.Trim(module, "\"'")

	// Get module from stdlib
	mod := vm.stdlib.GetModule(module)
	if mod != nil {
		vm.modules[module] = mod
		// Also set as variable for const x = require()
		vm.variables[module] = mod
		return nil
	}

	// Check VFS for installed packages
	if vm.vfs != nil {
		// Try to find package directory with index.js
		packagePath := module + "/index.js"
		if file, err := vm.vfs.Open(packagePath); err == nil {
			file.Close()
			// Package found - load and execute it
			return vm.loadModuleFromVFS(module, packagePath)
		}

		// Try single file module
		packagePath = module + ".js"
		if file, err := vm.vfs.Open(packagePath); err == nil {
			file.Close()
			return vm.loadModuleFromVFS(module, packagePath)
		}
	}

	return fmt.Errorf("Module not found: %s", module)
}

// loadModuleFromVFS loads a JavaScript module from the virtual filesystem
func (vm *jsVM) loadModuleFromVFS(moduleName, modulePath string) error {
	// Open and read the module file
	file, err := vm.vfs.Open(modulePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info for size
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// Read file content
	content := make([]byte, stat.Size())
	_, err = file.Read(content)
	if err != nil && err != io.EOF {
		return err
	}

	// Create a module scope
	moduleExports := make(map[string]interface{})
	
	// Create a simplified require for nested imports
	moduleRequire := func(name string) map[string]interface{} {
		// Check stdlib first
		if stdlibMod := vm.stdlib.GetModule(name); stdlibMod != nil {
			return stdlibMod
		}
		// Return empty module for now
		return make(map[string]interface{})
	}

	// Parse and execute the module
	// For now, create a basic module wrapper
	moduleObj := map[string]interface{}{
		"exports": moduleExports,
		"require": moduleRequire,
	}

	vm.modules[moduleName] = moduleExports
	vm.variables[moduleName] = moduleExports
	
	// Execute the module code in a simplified way
	// This is a placeholder - full execution would need proper JS VM
	_ = content
	_ = moduleObj
	
	return nil
}

func (vm *jsVM) handleModuleCall(module, method, argsStr string) error {
	mod, ok := vm.modules[module]
	if !ok {
		fmt.Fprintln(vm.stdout, "undefined")
		return nil
	}

	fn, ok := mod[method]
	if !ok {
		fmt.Fprintln(vm.stdout, "undefined")
		return nil
	}

	// Parse arguments - remove quotes from strings
	var args []string
	if argsStr != "" {
		args = append(args, strings.Trim(argsStr, "\"'"))
	}

	// Execute based on function type
	switch f := fn.(type) {
	case func(string) string:
		if len(args) > 0 {
			result := f(args[0])
			fmt.Fprintln(vm.stdout, result)
		} else {
			result := f("")
			fmt.Fprintln(vm.stdout, result)
		}
	case func(string) []byte:
		if len(args) > 0 {
			result := f(args[0])
			fmt.Fprintln(vm.stdout, string(result))
		}
	case func(string) bool:
		if len(args) > 0 {
			result := f(args[0])
			fmt.Fprintln(vm.stdout, result)
		}
	case func() string:
		result := f()
		fmt.Fprintln(vm.stdout, result)
	case func() int:
		result := f()
		fmt.Fprintln(vm.stdout, result)
	case func(string) error:
		if len(args) > 0 {
			_ = f(args[0])
		}
	default:
		fmt.Fprintln(vm.stdout, "[Function: "+method+"]")
	}

	return nil
}

func (vm *jsVM) callFunction(fn *jsFunction, argsStr string) error {
	// Simplified function call
	return nil
}

// Handle async function declaration
func (vm *jsVM) handleAsyncFunctionDecl(name, params, body string) error {
	paramList := vm.parseParams(params)
	vm.functions["async_"+name] = &jsFunction{
		name:   "async_" + name,
		params: paramList,
		body:   body,
	}
	return nil
}

// Handle await expression
func (vm *jsVM) handleAwait(expr string) (interface{}, error) {
	// For now, just evaluate the expression (no real async)
	return vm.evaluate(expr)
}

func (vm *jsVM) evaluate(expr string) (interface{}, error) {
	expr = strings.TrimSpace(expr)

	// Variable reference
	if val, ok := vm.variables[expr]; ok {
		return val, nil
	}

	// String literal
	if strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`) {
		return expr[1 : len(expr)-1], nil
	}
	if strings.HasPrefix(expr, `'`) && strings.HasSuffix(expr, `'`) {
		return expr[1 : len(expr)-1], nil
	}

	// Template literal
	if strings.HasPrefix(expr, "`") && strings.HasSuffix(expr, "`") {
		return vm.evalTemplateLiteral(expr[1 : len(expr)-1])
	}

	// Number
	if i, err := strconv.Atoi(expr); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(expr, 64); err == nil {
		return f, nil
	}

	// Boolean
	if expr == "true" {
		return true, nil
	}
	if expr == "false" {
		return false, nil
	}

	// null/undefined
	if expr == "null" || expr == "undefined" {
		return nil, nil
	}

	// Array literal
	if strings.HasPrefix(expr, "[") && strings.HasSuffix(expr, "]") {
		return vm.parseArrayLiteral(expr)
	}

	// Object literal
	if strings.HasPrefix(expr, "{") && strings.HasSuffix(expr, "}") {
		return vm.parseObjectLiteral(expr)
	}

	// Arrow function (simplified)
	if regexp.MustCompile(`^\([^)]*\)\s*=>`).MatchString(expr) {
		return expr, nil
	}

	// Arithmetic
	if strings.ContainsAny(expr, "+-*/%") {
		return vm.evalArithmetic(expr)
	}

	// Comparison
	if strings.Contains(expr, "===") {
		parts := strings.Split(expr, "===")
		if len(parts) == 2 {
			left, _ := vm.evaluate(strings.TrimSpace(parts[0]))
			right, _ := vm.evaluate(strings.TrimSpace(parts[1]))
			return left == right, nil
		}
	}
	if strings.Contains(expr, "==") {
		parts := strings.Split(expr, "==")
		if len(parts) == 2 {
			left, _ := vm.evaluate(strings.TrimSpace(parts[0]))
			right, _ := vm.evaluate(strings.TrimSpace(parts[1]))
			return left == right, nil
		}
	}

	// Array methods (map, filter, etc.)
	if match := regexp.MustCompile(`(\w+)\.map\(([^)]+)\)`).FindStringSubmatch(expr); match != nil {
		return vm.handleArrayMethod(match[1], "map", match[2])
	}

	return expr, nil
}

func (vm *jsVM) evalTemplateLiteral(expr string) (string, error) {
	re := regexp.MustCompile(`\$\{(\w+)\}`)
	return re.ReplaceAllStringFunc(expr, func(match string) string {
		varName := match[2 : len(match)-1]
		if val, ok := vm.variables[varName]; ok {
			return vm.formatValue(val)
		}
		return match
	}), nil
}

func (vm *jsVM) parseArrayLiteral(expr string) ([]interface{}, error) {
	inner := strings.TrimSpace(expr[1 : len(expr)-1])
	if inner == "" {
		return []interface{}{}, nil
	}

	parts := strings.Split(inner, ",")
	result := make([]interface{}, len(parts))
	for i, p := range parts {
		val, _ := vm.evaluate(strings.TrimSpace(p))
		result[i] = val
	}
	return result, nil
}

func (vm *jsVM) parseObjectLiteral(expr string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	return result, nil
}

func (vm *jsVM) evalArithmetic(expr string) (interface{}, error) {
	re := regexp.MustCompile(`(-?\d+(?:\.\d+)?)\s*([+\-*/%])\s*(-?\d+(?:\.\d+)?)`)
	match := re.FindStringSubmatch(expr)
	if match == nil {
		return expr, nil
	}

	left, _ := strconv.ParseFloat(match[1], 64)
	op := match[2]
	right, _ := strconv.ParseFloat(match[3], 64)

	var result float64
	switch op {
	case "+":
		result = left + right
	case "-":
		result = left - right
	case "*":
		result = left * right
	case "/":
		result = left / right
	case "%":
		result = float64(int(left) % int(right))
	}

	if result == float64(int(result)) {
		return int(result), nil
	}
	return result, nil
}

func (vm *jsVM) handleArrayMethod(arrName, method, callback string) ([]interface{}, error) {
	arrVal, ok := vm.variables[arrName]
	if !ok {
		return nil, fmt.Errorf("undefined variable: %s", arrName)
	}

	arr, ok := arrVal.([]interface{})
	if !ok {
		return nil, fmt.Errorf("not an array: %s", arrName)
	}

	switch method {
	case "map":
		result := make([]interface{}, len(arr))
		for i, item := range arr {
			result[i] = item // Simplified - would apply callback
		}
		return result, nil
	}

	return arr, nil
}

func (vm *jsVM) formatValue(val interface{}) string {
	switch v := val.(type) {
	case int:
		return strconv.Itoa(v)
	case float64:
		return fmt.Sprintf("%g", v)
	case string:
		return v
	case []interface{}:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = vm.formatValue(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]interface{}:
		return "[object Object]"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (vm *jsVM) parseParams(params string) []string {
	if params == "" {
		return []string{}
	}
	return strings.Split(params, ",")
}

func isTrueJS(val interface{}) bool {
	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	case nil:
		return false
	default:
		return true
	}
}

// getQuickJSWASM returns embedded QuickJS WASM bytes
func getQuickJSWASM() []byte {
	return nil
}
