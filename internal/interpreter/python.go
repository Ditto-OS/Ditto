package interpreter

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"Ditto/internal/stdlib"
)

// PythonInterpreter executes Python code
// Uses embedded MicroPython WASM when available, falls back to pure Go parser
type PythonInterpreter struct{}

func (p *PythonInterpreter) Name() string {
	return "python"
}

func (p *PythonInterpreter) Execute(code string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Try WASM execution first
	if err := p.executeWASM(code, args, stdin, stdout, stderr); err == nil {
		return nil
	}

	// Fall back to pure Go implementation
	return p.executePureGo(code, args, stdin, stdout, stderr)
}

func (p *PythonInterpreter) executeWASM(code string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Load embedded MicroPython WASM (when available)
	wasmBytes := getMicroPythonWASM()
	if wasmBytes == nil {
		return fmt.Errorf("WASM not available")
	}
	// Would execute WASM here
	return fmt.Errorf("WASM execution not fully implemented")
}

func (p *PythonInterpreter) executePureGo(code string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Parse and execute Python code in pure Go
	// This is a simplified Python interpreter supporting common operations

	py := &pythonVM{
		variables: make(map[string]interface{}),
		functions: make(map[string]*pythonFunction),
		classes:   make(map[string]*pythonClass),
		stdlib:    stdlib.NewPythonStdLib(),
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		args:    args,
	}
	py.stdlib.Init()

	return py.Run(code)
}

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

func (vm *pythonVM) Run(code string) error {
	lines := strings.Split(code, "\n")
	return vm.executeLines(lines, 0, 0)
}

func (vm *pythonVM) executeLines(lines []string, start, indent int) error {
	i := start
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
		if currentIndent < indent {
			return nil // Return to parent scope
		}
		if currentIndent > indent && i > start {
			return fmt.Errorf("unexpected indent")
		}

		var err error
		i, err = vm.executeLine(trimmed, lines, i, currentIndent)
		if err != nil {
			fmt.Fprintf(vm.stderr, "Error: %v\n", err)
			return err
		}
	}
	return nil
}

func (vm *pythonVM) executeLine(line string, lines []string, lineNum, indent int) (int, error) {
	// Handle module method call (e.g., print(os.getcwd()))
	if match := regexp.MustCompile(`^print\((\w+)\.(\w+)\(([^)]*)\)\)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handlePrintModuleCall(match[1], match[2], match[3])
	}

	// Handle import statement
	if match := regexp.MustCompile(`^import\s+(\w+)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleImport(match[1])
	}
	if match := regexp.MustCompile(`^from\s+(\w+)\s+import\s+(.+)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleFromImport(match[1], match[2])
	}

	// Handle class definition
	if match := regexp.MustCompile(`^class\s+(\w+)(?:\(([^)]*)\))?:$`).FindStringSubmatch(line); match != nil {
		return vm.handleClassDef(match[1], match[2], lines, lineNum, indent)
	}

	// Handle print()
	if match := regexp.MustCompile(`^print\((.*)\)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handlePrint(match[1])
	}

	// Handle variable assignment
	if match := regexp.MustCompile(`^(\w+)\s*=\s*(.+)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleAssignment(match[1], match[2])
	}

	// Handle for loop
	if match := regexp.MustCompile(`^for\s+(\w+)\s+in\s+(.+):$`).FindStringSubmatch(line); match != nil {
		return vm.handleForLoop(match[1], match[2], lines, lineNum, indent)
	}

	// Handle if statement
	if match := regexp.MustCompile(`^if\s+(.+):$`).FindStringSubmatch(line); match != nil {
		return vm.handleIfStatement(match[1], lines, lineNum, indent)
	}

	// Handle function definition
	if match := regexp.MustCompile(`^def\s+(\w+)\(([^)]*)\):$`).FindStringSubmatch(line); match != nil {
		return vm.handleFunctionDef(match[1], match[2], lines, lineNum, indent)
	}

	// Handle function call
	if match := regexp.MustCompile(`^(\w+)\((.*)\)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleFunctionCall(match[1], match[2])
	}

	// Handle list comprehension (simplified)
	if match := regexp.MustCompile(`^\[(.+)\]$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleListLiteral(match[1])
	}

	return lineNum + 1, nil
}

func (vm *pythonVM) handlePrint(expr string) error {
	result, err := vm.evaluate(expr)
	if err != nil {
		return err
	}
	fmt.Fprintln(vm.stdout, vm.formatValue(result))
	return nil
}

func (vm *pythonVM) handlePrintModuleCall(moduleName, method, argsStr string) error {
	module, ok := vm.variables[moduleName].(map[string]interface{})
	if !ok {
		return fmt.Errorf("ModuleNotFoundError: No module named '%s'", moduleName)
	}

	fn, ok := module[method]
	if !ok {
		return fmt.Errorf("AttributeError: module '%s' has no attribute '%s'", moduleName, method)
	}

	// Call the function and print result
	switch f := fn.(type) {
	case func() string:
		result := f()
		fmt.Fprintln(vm.stdout, result)
	case func() int:
		result := f()
		fmt.Fprintln(vm.stdout, result)
	case func(string) string:
		if argsStr != "" {
			result := f(strings.Trim(argsStr, "\"'"))
			fmt.Fprintln(vm.stdout, result)
		}
	case func(string) bool:
		if argsStr != "" {
			result := f(strings.Trim(argsStr, "\"'"))
			fmt.Fprintln(vm.stdout, result)
		}
	case func(string) []string:
		if argsStr != "" {
			result := f(strings.Trim(argsStr, "\"'"))
			fmt.Fprintln(vm.stdout, vm.formatValue(result))
		}
	case func(float64) float64:
		if argsStr != "" {
			arg, _ := strconv.ParseFloat(strings.TrimSpace(argsStr), 64)
			result := f(arg)
			fmt.Fprintln(vm.stdout, result)
		}
	case float64:
		fmt.Fprintln(vm.stdout, f)
	case int:
		fmt.Fprintln(vm.stdout, f)
	case string:
		fmt.Fprintln(vm.stdout, f)
	}

	return nil
}

func (vm *pythonVM) handleAssignment(name, value string) error {
	val, err := vm.evaluate(value)
	if err != nil {
		return err
	}
	vm.variables[name] = val
	return nil
}

func (vm *pythonVM) handleForLoop(varName, iterable string, lines []string, lineNum, indent int) (int, error) {
	iterValue, err := vm.evaluate(iterable)
	if err != nil {
		return lineNum + 1, err
	}

	// Get loop body (indented lines)
	bodyLines := vm.getBodyLines(lines, lineNum+1, indent)

	// Iterate
	switch v := iterValue.(type) {
	case []int:
		for _, item := range v {
			vm.variables[varName] = item
			if err := vm.executeLines(bodyLines, 0, 0); err != nil {
				return lineNum + len(bodyLines) + 1, err
			}
		}
	case []string:
		for _, item := range v {
			vm.variables[varName] = item
			if err := vm.executeLines(bodyLines, 0, 0); err != nil {
				return lineNum + len(bodyLines) + 1, err
			}
		}
	}

	return lineNum + len(bodyLines) + 1, nil
}

func (vm *pythonVM) handleIfStatement(condition string, lines []string, lineNum, indent int) (int, error) {
	result, err := vm.evaluate(condition)
	if err != nil {
		return lineNum + 1, err
	}

	bodyLines := vm.getBodyLines(lines, lineNum+1, indent)

	if isTrue(result) {
		if err := vm.executeLines(bodyLines, 0, 0); err != nil {
			return lineNum + len(bodyLines) + 1, err
		}
	}

	return lineNum + len(bodyLines) + 1, nil
}

func (vm *pythonVM) handleFunctionDef(name, params string, lines []string, lineNum, indent int) (int, error) {
	bodyLines := vm.getBodyLines(lines, lineNum+1, indent)
	paramList := vm.parseParams(params)

	vm.functions[name] = &pythonFunction{
		name:   name,
		params: paramList,
		body:   strings.Join(bodyLines, "\n"),
	}

	return lineNum + len(bodyLines) + 1, nil
}

func (vm *pythonVM) handleImport(moduleName string) error {
	module := vm.stdlib.GetModule(moduleName)
	if module != nil {
		vm.variables[moduleName] = module
		return nil
	}
	return fmt.Errorf("ModuleNotFoundError: No module named '%s'", moduleName)
}

func (vm *pythonVM) handleFromImport(moduleName, imports string) error {
	module := vm.stdlib.GetModule(moduleName)
	if module == nil {
		return fmt.Errorf("ModuleNotFoundError: No module named '%s'", moduleName)
	}

	// Import specific items
	for _, item := range strings.Split(imports, ",") {
		item = strings.TrimSpace(item)
		if val, ok := module[item]; ok {
			vm.variables[item] = val
		}
	}
	return nil
}

func (vm *pythonVM) handleClassDef(name, parent string, lines []string, lineNum, indent int) (int, error) {
	bodyLines := vm.getBodyLines(lines, lineNum+1, indent)
	
	class := &pythonClass{
		name:    name,
		attrs:   make(map[string]interface{}),
		methods: make(map[string]*pythonFunction),
	}

	// Parse class body for methods and attributes
	for _, line := range bodyLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Simple method detection
		if match := regexp.MustCompile(`^def\s+(\w+)\(self(?:,\s*([^)]*)\))?:$`).FindStringSubmatch(line); match != nil {
			// Would need to collect method body
			continue
		}
		// Attribute assignment
		if match := regexp.MustCompile(`^self\.(\w+)\s*=\s*(.+)$`).FindStringSubmatch(line); match != nil {
			class.attrs[match[1]] = match[2] // Store as expression
		}
	}

	vm.classes[name] = class
	return lineNum + len(bodyLines) + 1, nil
}

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

func (vm *pythonVM) handleRange(argsStr string) error {
	// range is usually called in for loop context
	return nil
}

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
	}
	return nil
}

func (vm *pythonVM) handleStr(argsStr string) error {
	val, err := vm.evaluate(argsStr)
	if err != nil {
		return err
	}
	fmt.Fprintln(vm.stdout, vm.formatValue(val))
	return nil
}

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

func (vm *pythonVM) handleInput() error {
	buf := make([]byte, 1024)
	n, err := vm.stdin.Read(buf)
	if err == nil && n > 0 {
		fmt.Fprintln(vm.stdout, strings.TrimSpace(string(buf[:n])))
	}
	return nil
}

func (vm *pythonVM) callFunction(fn *pythonFunction, argsStr string) error {
	// Simplified function call
	return nil
}

func (vm *pythonVM) handleListLiteral(expr string) error {
	// Parse list literal
	return nil
}

func (vm *pythonVM) evaluate(expr string) (interface{}, error) {
	expr = strings.TrimSpace(expr)

	// Check for module attribute access (e.g., os.getcwd())
	if match := regexp.MustCompile(`^(\w+)\.(\w+)(?:\(([^)]*)\))?$`).FindStringSubmatch(expr); match != nil {
		moduleName := match[1]
		attrName := match[2]
		args := match[3]

		if module, ok := vm.variables[moduleName].(map[string]interface{}); ok {
			if attr, ok := module[attrName]; ok {
				// It's a function call
				if args != "" {
					switch fn := attr.(type) {
					case func(string) string:
						return fn(args), nil
					case func(string) error:
						return nil, fn(args)
					case func() string:
						return fn(), nil
					case func() int:
						return fn(), nil
					case func(string) bool:
						return fn(args), nil
					case func(string, string) error:
						return nil, fn(args, "")
					case func(string) []string:
						return fn(args), nil
					case func(string) interface{}:
						return fn(args), nil
					case float64, int:
						return attr, nil
					case string:
						return attr, nil
					}
				}
				return attr, nil
			}
		}
	}

	// Check for class instantiation (e.g., MyClass())
	if match := regexp.MustCompile(`^(\w+)\(\)$`).FindStringSubmatch(expr); match != nil {
		className := match[1]
		if class, ok := vm.classes[className]; ok {
			// Return instance with class attributes
			instance := make(map[string]interface{})
			for k, v := range class.attrs {
				instance[k] = v
			}
			instance["__class__"] = class
			return instance, nil
		}
	}

	// Check for variable reference
	if val, ok := vm.variables[expr]; ok {
		return val, nil
	}

	// Check for list comprehension
	if strings.Contains(expr, "for") {
		return vm.evalListComprehension(expr)
	}

	// Check for string literal
	if strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`) {
		return expr[1 : len(expr)-1], nil
	}
	if strings.HasPrefix(expr, `'`) && strings.HasSuffix(expr, `'`) {
		return expr[1 : len(expr)-1], nil
	}

	// Check for integer
	if i, err := strconv.Atoi(expr); err == nil {
		return i, nil
	}

	// Check for float
	if f, err := strconv.ParseFloat(expr, 64); err == nil {
		return f, nil
	}

	// Check for list literal
	if strings.HasPrefix(expr, "[") && strings.HasSuffix(expr, "]") {
		return vm.parseListLiteral(expr)
	}

	// Check for dict literal
	if strings.HasPrefix(expr, "{") && strings.HasSuffix(expr, "}") {
		return vm.parseDictLiteral(expr)
	}

	// Check for f-string
	if strings.HasPrefix(expr, "f\"") {
		return vm.evalFString(expr[2 : len(expr)-1])
	}

	// Check for arithmetic
	if strings.ContainsAny(expr, "+-*/") {
		return vm.evalArithmetic(expr)
	}

	// Check for comparison
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

func (vm *pythonVM) evalListComprehension(expr string) ([]interface{}, error) {
	// Simplified list comprehension: [x**2 for x in numbers]
	re := regexp.MustCompile(`(.+)\s+for\s+(\w+)\s+in\s+(\w+)`)
	match := re.FindStringSubmatch(expr)
	if match == nil {
		return nil, fmt.Errorf("invalid list comprehension")
	}

	resultExpr := match[1]
	varName := match[2]
	listName := match[3]

	listVal, ok := vm.variables[listName]
	if !ok {
		return nil, fmt.Errorf("undefined variable: %s", listName)
	}

	var result []interface{}
	switch v := listVal.(type) {
	case []int:
		for _, item := range v {
			vm.variables[varName] = item
			val, _ := vm.evaluate(resultExpr)
			result = append(result, val)
		}
	}

	return result, nil
}

func (vm *pythonVM) parseListLiteral(expr string) ([]interface{}, error) {
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

func (vm *pythonVM) parseDictLiteral(expr string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	return result, nil
}

func (vm *pythonVM) evalFString(expr string) (string, error) {
	re := regexp.MustCompile(`\{(\w+)\}`)
	return re.ReplaceAllStringFunc(expr, func(match string) string {
		varName := match[1 : len(match)-1]
		if val, ok := vm.variables[varName]; ok {
			return vm.formatValue(val)
		}
		return match
	}), nil
}

func (vm *pythonVM) evalArithmetic(expr string) (interface{}, error) {
	// Simple arithmetic evaluation
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

func (vm *pythonVM) formatValue(val interface{}) string {
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
			return "True"
		}
		return "False"
	default:
		return fmt.Sprintf("%v", v)
	}
}

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
		body = append(body, trimmed)
	}
	return body
}

func (vm *pythonVM) parseParams(params string) []string {
	if params == "" {
		return []string{}
	}
	return strings.Split(params, ",")
}

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

// getMicroPythonWASM returns embedded MicroPython WASM bytes
func getMicroPythonWASM() []byte {
	// Would return embedded WASM when available
	return nil
}
