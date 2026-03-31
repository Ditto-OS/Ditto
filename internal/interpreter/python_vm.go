package interpreter

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"Ditto/internal/stdlib"
)

// pythonVM is a minimal Python virtual machine
type pythonVM struct {
	variables map[string]interface{}
	functions map[string]*pythonFunction
	classes   map[string]*pythonClass
	stdlib    *stdlib.PythonStdLib
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	args      []string
	scope     []map[string]interface{}
	vfs       fs.FS
}

type pythonFunction struct {
	name   string
	params []string
	body   string
}

type pythonClass struct {
	name    string
	attrs   map[string]interface{}
	methods map[string]*pythonFunction
}

// Run executes Python code
func (vm *pythonVM) Run(code string) error {
	fmt.Fprintf(os.Stderr, "[DEBUG] Run: %d lines\n", len(strings.Split(code, "\n")))
	lines := strings.Split(code, "\n")
	return vm.executeLines(lines, 0, 0)
}

// executeLines executes a block of lines
func (vm *pythonVM) executeLines(lines []string, start, indent int) error {
	i := start
	firstLine := true
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimLeft(line, " \t")
		currentIndent := len(line) - len(trimmed)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			i++
			continue
		}

		// Check indent
		if indent == 0 {
			// At top level, allow any indentation (effectively no indentation check)
			// But update indent if we are starting a block
			if firstLine {
				indent = currentIndent
				firstLine = false
			}
		} else {
			if firstLine {
				indent = currentIndent
				firstLine = false
			} else if currentIndent < indent {
				return nil // Return to parent scope
			} else if currentIndent > indent {
				return fmt.Errorf("unexpected indent at line %d: got %d, expected %d", i+1, currentIndent, indent)
			}
		}

		fmt.Fprintf(os.Stderr, "[DEBUG] Executing (%d): %s\n", currentIndent, trimmed)
		var err error
		i, err = vm.executeLine(trimmed, lines, i, currentIndent)
		if err != nil {
			fmt.Fprintf(vm.stderr, "Error: %v\n", err)
			return err
		}
	}
	return nil
}

// getBodyLines extracts indented body lines
func (vm *pythonVM) getBodyLines(lines []string, start, parentIndent int) []string {
	var body []string
	for i := start; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			body = append(body, line)
			continue
		}
		currentIndent := len(line) - len(trimmed)
		if currentIndent <= parentIndent {
			break
		}
		// Keep the original line to preserve relative indentation for nested blocks
		body = append(body, line)
	}
	return body
}

// formatValue formats a value for display
func (vm *pythonVM) formatValue(val interface{}) string {
	switch v := val.(type) {
	case int:
		return fmt.Sprintf("%d", v)
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
		// Check if it's a Response-like object
		if status, ok := v["status_code"]; ok {
			return fmt.Sprintf("<Response [%v]>", status)
		}
		return "<object>"
	case bool:
		if v {
			return "True"
		}
		return "False"
	case *pythonFunction:
		return fmt.Sprintf("<function %s>", v.name)
	case *pythonClass:
		return fmt.Sprintf("<class '%s'>", v.name)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// isTrue checks if a value is truthy
func isTrue(val interface{}) bool {
	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case string:
		return v != ""
	case []interface{}:
		return len(v) > 0
	default:
		return val != nil
	}
}
