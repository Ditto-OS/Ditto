package interpreter

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// LuaInterpreter executes Lua code
type LuaInterpreter struct{}

func (l *LuaInterpreter) Name() string {
	return "lua"
}

func (l *LuaInterpreter) Execute(code string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	lua := &luaVM{
		variables: make(map[string]interface{}),
		functions: make(map[string]*luaFunction),
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		args:    args,
	}

	return lua.Run(code)
}

type luaVM struct {
	variables map[string]interface{}
	functions map[string]*luaFunction
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	args      []string
}

type luaFunction struct {
	name string
	params []string
	body string
}

func (vm *luaVM) Run(code string) error {
	lines := strings.Split(code, "\n")
	return vm.executeLines(lines)
}

func (vm *luaVM) executeLines(lines []string) error {
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "--") {
			i++
			continue
		}

		// Handle multi-line strings and functions
		if strings.Contains(line, "[[") {
			var multiLine strings.Builder
			multiLine.WriteString(line)
			for !strings.Contains(line, "]]") && i < len(lines)-1 {
				i++
				line = lines[i]
				multiLine.WriteString("\n")
				multiLine.WriteString(line)
			}
			line = multiLine.String()
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

func (vm *luaVM) executeLine(line string, lineNum int) (int, error) {
	// print()
	if match := regexp.MustCompile(`^print\((.+)\)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handlePrint(match[1])
	}

	// io.write()
	if match := regexp.MustCompile(`^io\.write\((.+)\)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleIOWrite(match[1])
	}

	// Variable assignment
	if match := regexp.MustCompile(`^local\s+(\w+)\s*=\s*(.+)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleLocalDecl(match[1], match[2])
	}
	if match := regexp.MustCompile(`^(\w+)\s*=\s*(.+)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleAssignment(match[1], match[2])
	}

	// Function definition
	if match := regexp.MustCompile(`^function\s+(\w+)\(([^)]*)\)$`).FindStringSubmatch(line); match != nil {
		return vm.handleFunctionDef(match[1], match[2], lineNum)
	}

	// For loop (numeric)
	if match := regexp.MustCompile(`^for\s+(\w+)\s*=\s*(\d+),\s*(\d+)\s+do$`).FindStringSubmatch(line); match != nil {
		return vm.handleForNumeric(match[1], match[2], match[3], lineNum)
	}

	// For loop (generic)
	if match := regexp.MustCompile(`^for\s+(\w+)\s+in\s+(.+)\s+do$`).FindStringSubmatch(line); match != nil {
		return vm.handleForGeneric(match[1], match[2], lineNum)
	}

	// If statement
	if match := regexp.MustCompile(`^if\s+(.+)\s+then$`).FindStringSubmatch(line); match != nil {
		return vm.handleIfStatement(match[1], lineNum)
	}

	// While loop
	if match := regexp.MustCompile(`^while\s+(.+)\s+do$`).FindStringSubmatch(line); match != nil {
		return vm.handleWhileLoop(match[1], lineNum)
	}

	// Function call
	if match := regexp.MustCompile(`^(\w+)\((.*)\)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleFunctionCall(match[1], match[2])
	}

	// Return
	if strings.HasPrefix(line, "return ") {
		return lineNum + 1, nil
	}

	// End (block terminator)
	if line == "end" {
		return lineNum + 1, nil
	}

	return lineNum + 1, nil
}

func (vm *luaVM) handlePrint(expr string) error {
	result, err := vm.evaluate(expr)
	if err != nil {
		return err
	}
	fmt.Fprintln(vm.stdout, vm.formatValue(result))
	return nil
}

func (vm *luaVM) handleIOWrite(expr string) error {
	result, err := vm.evaluate(expr)
	if err != nil {
		return err
	}
	fmt.Fprint(vm.stdout, vm.formatValue(result))
	return nil
}

func (vm *luaVM) handleLocalDecl(name, value string) error {
	val, err := vm.evaluate(value)
	if err != nil {
		return err
	}
	vm.variables[name] = val
	return nil
}

func (vm *luaVM) handleAssignment(name, value string) error {
	val, err := vm.evaluate(value)
	if err != nil {
		return err
	}
	vm.variables[name] = val
	return nil
}

func (vm *luaVM) handleFunctionDef(name, params string, lineNum int) (int, error) {
	// Simplified - would need to collect body until "end"
	paramList := vm.parseParams(params)
	vm.functions[name] = &luaFunction{
		name: name,
		params: paramList,
		body: "",
	}
	return lineNum + 1, nil
}

func (vm *luaVM) handleForNumeric(varName, startStr, endStr string, lineNum int) (int, error) {
	start, _ := strconv.Atoi(startStr)
	end, _ := strconv.Atoi(endStr)

	for i := start; i <= end; i++ {
		vm.variables[varName] = i
	}

	return lineNum + 1, nil
}

func (vm *luaVM) handleForGeneric(varName, iterable string, lineNum int) (int, error) {
	// Handle ipairs/pairs
	if strings.HasPrefix(iterable, "ipairs(") {
		arrName := strings.TrimSuffix(iterable[7:], ")")
		if arr, ok := vm.variables[arrName]; ok {
			switch v := arr.(type) {
			case []interface{}:
				for _, item := range v {
					vm.variables[varName] = item
				}
			}
		}
	}
	return lineNum + 1, nil
}

func (vm *luaVM) handleIfStatement(condition string, lineNum int) (int, error) {
	result, err := vm.evaluate(condition)
	if err != nil {
		return lineNum + 1, err
	}

	if isTrueLua(result) {
		// Would execute body until "then"
	}

	return lineNum + 1, nil
}

func (vm *luaVM) handleWhileLoop(condition string, lineNum int) (int, error) {
	// Simplified
	return lineNum + 1, nil
}

func (vm *luaVM) handleFunctionCall(name, argsStr string) error {
	if name == "tonumber" {
		return vm.handleToNumber(argsStr)
	}
	if name == "tostring" {
		return vm.handleToString(argsStr)
	}
	if name == "type" {
		return vm.handleType(argsStr)
	}
	if name == "ipairs" || name == "pairs" {
		return nil
	}

	// User-defined function
	if fn, ok := vm.functions[name]; ok {
		return vm.callFunction(fn, argsStr)
	}

	return nil
}

func (vm *luaVM) handleToNumber(argsStr string) error {
	val, err := vm.evaluate(argsStr)
	if err != nil {
		return err
	}
	switch v := val.(type) {
	case string:
		i, _ := strconv.Atoi(v)
		fmt.Fprintln(vm.stdout, i)
	case int:
		fmt.Fprintln(vm.stdout, v)
	case float64:
		fmt.Fprintln(vm.stdout, int(v))
	}
	return nil
}

func (vm *luaVM) handleToString(argsStr string) error {
	val, err := vm.evaluate(argsStr)
	if err != nil {
		return err
	}
	fmt.Fprintln(vm.stdout, vm.formatValue(val))
	return nil
}

func (vm *luaVM) handleType(argsStr string) error {
	val, err := vm.evaluate(argsStr)
	if err != nil {
		return err
	}
	fmt.Fprintln(vm.stdout, vm.typeOf(val))
	return nil
}

func (vm *luaVM) callFunction(fn *luaFunction, argsStr string) error {
	return nil
}

func (vm *luaVM) evaluate(expr string) (interface{}, error) {
	expr = strings.TrimSpace(expr)

	// Variable
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

	// Table literal
	if strings.HasPrefix(expr, "{") && strings.HasSuffix(expr, "}") {
		return vm.parseTableLiteral(expr)
	}

	// Arithmetic
	if strings.ContainsAny(expr, "+-*/") {
		return vm.evalArithmetic(expr)
	}

	// Concatenation
	if strings.Contains(expr, "..") {
		parts := strings.Split(expr, "..")
		var result strings.Builder
		for _, p := range parts {
			val, _ := vm.evaluate(strings.TrimSpace(p))
			result.WriteString(vm.formatValue(val))
		}
		return result.String(), nil
	}

	// Comparison
	if strings.Contains(expr, "==") {
		parts := strings.Split(expr, "==")
		if len(parts) == 2 {
			left, _ := vm.evaluate(strings.TrimSpace(parts[0]))
			right, _ := vm.evaluate(strings.TrimSpace(parts[1]))
			return left == right, nil
		}
	}

	return expr, nil
}

func (vm *luaVM) parseTableLiteral(expr string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	return result, nil
}

func (vm *luaVM) evalArithmetic(expr string) (interface{}, error) {
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
		result = left / right
	}

	if result == float64(int(result)) {
		return int(result), nil
	}
	return result, nil
}

func (vm *luaVM) formatValue(val interface{}) string {
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
		return "table: " + strconv.Itoa(len(v)) + " items"
	case map[string]interface{}:
		return "table: " + strconv.Itoa(len(v)) + " items"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "nil"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (vm *luaVM) typeOf(val interface{}) string {
	switch val.(type) {
	case nil:
		return "nil"
	case bool:
		return "boolean"
	case int, float64:
		return "number"
	case string:
		return "string"
	case []interface{}, map[string]interface{}:
		return "table"
	default:
		return "userdata"
	}
}

func (vm *luaVM) parseParams(params string) []string {
	if params == "" {
		return []string{}
	}
	return strings.Split(params, ",")
}

func isTrueLua(val interface{}) bool {
	if val == nil {
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
