package interpreter

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// handleFunctionCall handles function calls
func (vm *pythonVM) handleFunctionCall(name string, argsStr string) error {
	if name == "range" {
		return vm.handleRange(argsStr)
	}
	if name == "len" {
		return vm.handleLen(argsStr)
	}
	if name == "str" {
		return vm.handleStr(argsStr)
	}
	if name == "int" {
		return vm.handleInt(argsStr)
	}
	if name == "input" {
		return vm.handleInput()
	}

	// User-defined function
	if fn, ok := vm.functions[name]; ok {
		return vm.callFunction(fn, argsStr)
	}

	return fmt.Errorf("undefined function: %s", name)
}

// handleRange handles range()
func (vm *pythonVM) handleRange(argsStr string) error {
	// range is usually called in for loop context
	return nil
}

// handleLen handles len()
func (vm *pythonVM) handleLen(argsStr string) error {
	val, err := vm.evaluate(argsStr)
	if err != nil {
		return err
	}
	switch v := val.(type) {
	case []int:
		fmt.Fprintln(vm.stdout, len(v))
	case []string:
		fmt.Fprintln(vm.stdout, len(v))
	case string:
		fmt.Fprintln(vm.stdout, len(v))
	case []interface{}:
		fmt.Fprintln(vm.stdout, len(v))
	}
	return nil
}

// handleStr handles str()
func (vm *pythonVM) handleStr(argsStr string) error {
	val, err := vm.evaluate(argsStr)
	if err != nil {
		return err
	}
	fmt.Fprintln(vm.stdout, vm.formatValue(val))
	return nil
}

// handleInt handles int()
func (vm *pythonVM) handleInt(argsStr string) error {
	val, err := vm.evaluate(argsStr)
	if err != nil {
		return err
	}
	switch v := val.(type) {
	case int:
		fmt.Fprintln(vm.stdout, v)
	case float64:
		fmt.Fprintln(vm.stdout, int(v))
	case string:
		i, _ := strconv.Atoi(v)
		fmt.Fprintln(vm.stdout, i)
	}
	return nil
}

// handleInput handles input()
func (vm *pythonVM) handleInput() error {
	buf := make([]byte, 1024)
	n, err := vm.stdin.Read(buf)
	if err == nil && n > 0 {
		fmt.Fprintln(vm.stdout, strings.TrimSpace(string(buf[:n])))
	}
	return nil
}

// callFunction calls a user-defined function
func (vm *pythonVM) callFunction(fn *pythonFunction, argsStr string) error {
	result, err := vm.callPythonFunction(fn, argsStr)
	if err != nil {
		return err
	}
	if result != nil {
		fmt.Fprintln(vm.stdout, vm.formatValue(result))
	}
	return nil
}

// callPythonFunction calls a pythonFunction and returns its result
func (vm *pythonVM) callPythonFunction(fn *pythonFunction, argsStr string) (interface{}, error) {
	// Parse arguments
	args := vm.parseArgs(argsStr)

	// Create a new scope for the function
	funcVM := &pythonVM{
		variables: make(map[string]interface{}),
		functions: vm.functions,
		classes:   vm.classes,
		stdlib:    vm.stdlib,
		stdin:     vm.stdin,
		stdout:    vm.stdout,
		stderr:    vm.stderr,
		vfs:       vm.vfs,
	}

	// Bind arguments to parameters (simplified - just bind by position)
	for i, param := range fn.params {
		param = strings.TrimSpace(param)
		// Skip **kwargs and *args for now
		if strings.HasPrefix(param, "*") {
			continue
		}
		// Handle default values (param=value)
		if idx := strings.Index(param, "="); idx > 0 {
			param = param[:idx]
		}
		if i < len(args) {
			funcVM.variables[param] = args[i]
		}
	}

	// Execute function body line by line
	lines := strings.Split(fn.body, "\n")
	var lastResult interface{}

	fmt.Fprintf(os.Stderr, "[DEBUG] callPythonFunction: %s, lines: %d\n", fn.name, len(lines))

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fmt.Fprintf(os.Stderr, "[DEBUG] func line (%d): %s\n", i, line)

		// Handle return statement
		if strings.HasPrefix(line, "return ") {
			returnExpr := strings.TrimPrefix(line, "return ")
			return funcVM.evaluate(returnExpr)
		}

		// Handle if statement
		if match := regexp.MustCompile(`^if\s+(.+):$`).FindStringSubmatch(line); match != nil {
			nextIdx, err := funcVM.handleIfStatement(match[1], lines, i, 0)
			if err == nil {
				i = nextIdx - 1 // Subtract 1 because loop will add 1
			}
			continue
		}

		// Handle for loop
		if match := regexp.MustCompile(`^for\s+(\w+)\s+in\s+(.+):$`).FindStringSubmatch(line); match != nil {
			nextIdx, err := funcVM.handleForLoop(match[1], match[2], lines, i, 0)
			if err == nil {
				i = nextIdx - 1
			}
			continue
		}

		// Handle assignment
		if match := regexp.MustCompile(`^(\w+)\s*=\s*(.+)$`).FindStringSubmatch(line); match != nil {
			if err := funcVM.handleAssignment(match[1], match[2]); err != nil {
				// Continue on error
			}
			continue
		}

		// Handle direct function call
		if match := regexp.MustCompile(`^(\w+)\((.*)\)$`).FindStringSubmatch(line); match != nil {
			funcVM.handleFunctionCall(match[1], match[2])
			continue
		}
	}

	// Return nil for functions without explicit return
	return lastResult, nil
}
