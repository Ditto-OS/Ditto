package interpreter

import (
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
)

// RubyInterpreter executes Ruby code
type RubyInterpreter struct{}

func (r *RubyInterpreter) Name() string {
	return "ruby"
}

func (r *RubyInterpreter) Execute(engine *Engine, code string, args []string, stdin io.Reader, stdout, stderr io.Writer, vfs fs.FS) error {
	ruby := &rubyVM{
		variables: make(map[string]interface{}),
		methods:   make(map[string]*rubyMethod),
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		args:    args,
	}

	return ruby.Run(code)
}

type rubyVM struct {
	variables map[string]interface{}
	methods   map[string]*rubyMethod
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	args      []string
}

type rubyMethod struct {
	name   string
	params []string
	body   string
}

func (vm *rubyVM) Run(code string) error {
	lines := strings.Split(code, "\n")
	return vm.executeLines(lines)
}

func (vm *rubyVM) executeLines(lines []string) error {
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			i++
			continue
		}

		var err error
		i, err = vm.executeLine(line, i)
		if err != nil {
			fmt.Fprintf(vm.stderr, "Error: %v\n", err)
			return err
		}
	}
	return nil
}

func (vm *rubyVM) executeLine(line string, lineNum int) (int, error) {
	// puts()
	if match := regexp.MustCompile(`^puts\((.+)\)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handlePuts(match[1])
	}
	if match := regexp.MustCompile(`^puts\s+(.+)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handlePuts(match[1])
	}

	// print()
	if match := regexp.MustCompile(`^print\((.+)\)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handlePrint(match[1])
	}

	// Variable assignment
	if match := regexp.MustCompile(`^(\w+)\s*=\s*(.+)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleAssignment(match[1], match[2])
	}

	// Method definition
	if match := regexp.MustCompile(`^def\s+(\w+)\(([^)]*)\)$`).FindStringSubmatch(line); match != nil {
		return vm.handleMethodDef(match[1], match[2], lineNum)
	}

	// Each loop
	if match := regexp.MustCompile(`^(\w+)\.each\s+do\s+\|(\w+)\|$`).FindStringSubmatch(line); match != nil {
		return vm.handleEachLoop(match[1], match[2], lineNum)
	}

	// Times loop
	if match := regexp.MustCompile(`^(\d+)\.times\s+do$`).FindStringSubmatch(line); match != nil {
		return vm.handleTimesLoop(match[1], lineNum)
	}

	// If statement (modifier)
	if match := regexp.MustCompile(`^(.+)\s+if\s+(.+)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleIfModifier(match[1], match[2])
	}

	// If statement
	if match := regexp.MustCompile(`^if\s+(.+)$`).FindStringSubmatch(line); match != nil {
		return vm.handleIfStatement(match[1], lineNum)
	}

	// Method call
	if match := regexp.MustCompile(`^(\w+)\((.*)\)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleMethodCall(match[1], match[2])
	}

	// end (block terminator)
	if line == "end" {
		return lineNum + 1, nil
	}

	return lineNum + 1, nil
}

func (vm *rubyVM) handlePuts(expr string) error {
	result, err := vm.evaluate(expr)
	if err != nil {
		return err
	}
	fmt.Fprintln(vm.stdout, vm.formatValue(result))
	return nil
}

func (vm *rubyVM) handlePrint(expr string) error {
	result, err := vm.evaluate(expr)
	if err != nil {
		return err
	}
	fmt.Fprint(vm.stdout, vm.formatValue(result))
	return nil
}

func (vm *rubyVM) handleAssignment(name, value string) error {
	val, err := vm.evaluate(value)
	if err != nil {
		return err
	}
	vm.variables[name] = val
	return nil
}

func (vm *rubyVM) handleMethodDef(name, params string, lineNum int) (int, error) {
	paramList := vm.parseParams(params)
	vm.methods[name] = &rubyMethod{
		name:   name,
		params: paramList,
		body:   "",
	}
	return lineNum + 1, nil
}

func (vm *rubyVM) handleEachLoop(arrName, varName string, lineNum int) (int, error) {
	if arr, ok := vm.variables[arrName]; ok {
		switch v := arr.(type) {
		case []interface{}:
			for _, item := range v {
				vm.variables[varName] = item
			}
		}
	}
	return lineNum + 1, nil
}

func (vm *rubyVM) handleTimesLoop(countStr string, lineNum int) (int, error) {
	count, _ := strconv.Atoi(countStr)
	for i := 0; i < count; i++ {
		// Would execute loop body
	}
	return lineNum + 1, nil
}

func (vm *rubyVM) handleIfModifier(expr, condition string) error {
	condResult, err := vm.evaluate(condition)
	if err != nil {
		return err
	}
	if isTrueRuby(condResult) {
		return vm.handlePuts(expr)
	}
	return nil
}

func (vm *rubyVM) handleIfStatement(condition string, lineNum int) (int, error) {
	condResult, err := vm.evaluate(condition)
	if err != nil {
		return lineNum + 1, err
	}
	if isTrueRuby(condResult) {
		// Would execute body
	}
	return lineNum + 1, nil
}

func (vm *rubyVM) handleMethodCall(name, argsStr string) error {
	// Built-in methods
	switch name {
	case "puts":
		return vm.handlePuts(argsStr)
	case "print":
		return vm.handlePrint(argsStr)
	case "gets":
		return vm.handleGets()
	case "chomp":
		return vm.handleChomp()
	}

	// User-defined method
	if _, ok := vm.methods[name]; ok {
		return nil
	}

	return nil
}

func (vm *rubyVM) handleGets() error {
	buf := make([]byte, 1024)
	n, err := vm.stdin.Read(buf)
	if err == nil && n > 0 {
		vm.variables["_"] = strings.TrimSpace(string(buf[:n]))
	}
	return nil
}

func (vm *rubyVM) handleChomp() error {
	if val, ok := vm.variables["_"]; ok {
		if s, ok := val.(string); ok {
			vm.variables["_"] = strings.TrimRight(s, "\r\n")
		}
	}
	return nil
}

func (vm *rubyVM) evaluate(expr string) (interface{}, error) {
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
	if expr == "nil" {
		return nil, nil
	}

	// Array literal
	if strings.HasPrefix(expr, "[") && strings.HasSuffix(expr, "]") {
		return vm.parseArrayLiteral(expr)
	}

	// String interpolation
	if strings.HasPrefix(expr, `"`) && strings.Contains(expr, "#{") {
		return vm.evalInterpolation(expr)
	}

	// Arithmetic
	if strings.ContainsAny(expr, "+-*/") {
		return vm.evalArithmetic(expr)
	}

	// String concatenation
	if strings.Contains(expr, "+") && strings.Contains(expr, `"`) {
		return vm.evalStringConcat(expr)
	}

	return expr, nil
}

func (vm *rubyVM) parseArrayLiteral(expr string) ([]interface{}, error) {
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

func (vm *rubyVM) evalInterpolation(expr string) (string, error) {
	re := regexp.MustCompile(`#\{(\w+)\}`)
	return re.ReplaceAllStringFunc(expr, func(match string) string {
		varName := match[2 : len(match)-1]
		if val, ok := vm.variables[varName]; ok {
			return vm.formatValue(val)
		}
		return match
	}), nil
}

func (vm *rubyVM) evalArithmetic(expr string) (interface{}, error) {
	re := regexp.MustCompile(`(-?\d+(?:\.\d+)?)\s*([+\-*/])\s*(-?\d+(?:\.\d+)?)`)
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
		if right != 0 {
			result = left / right
		}
	}

	if result == float64(int(result)) {
		return int(result), nil
	}
	return result, nil
}

func (vm *rubyVM) evalStringConcat(expr string) (string, error) {
	parts := strings.Split(expr, "+")
	var result strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, `"`) && strings.HasSuffix(p, `"`) {
			result.WriteString(p[1 : len(p)-1])
		} else if val, _ := vm.evaluate(p); val != nil {
			result.WriteString(vm.formatValue(val))
		}
	}
	return result.String(), nil
}

func (vm *rubyVM) formatValue(val interface{}) string {
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
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (vm *rubyVM) parseParams(params string) []string {
	if params == "" {
		return []string{}
	}
	return strings.Split(params, ",")
}

func isTrueRuby(val interface{}) bool {
	if val == nil || val == false {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case string:
		return v != ""
	default:
		return true
	}
}
