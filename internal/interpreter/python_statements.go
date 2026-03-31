package interpreter

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// executeLine executes a single line
func (vm *pythonVM) executeLine(line string, lines []string, lineNum, indent int) (int, error) {
	// Handle module method call (e.g., print(os.getcwd()))
	if match := regexp.MustCompile(`^print\((\w+)\.(\w+)\(([^)]*)\)\)$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handlePrintModuleCall(match[1], match[2], match[3])
	}

	// Handle import statement (including "import X.Y as Z")
	if match := regexp.MustCompile(`^import\s+(.+)$`).FindStringSubmatch(line); match != nil {
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

	// Handle function call (including dotted calls like math.sqrt(4))
	if match := regexp.MustCompile(`^([\w\.]+)\((.*)\)$`).FindStringSubmatch(line); match != nil {
		name := match[1]
		args := match[2]

		if strings.Contains(name, ".") {
			_, err := vm.evaluate(line)
			return lineNum + 1, err
		}
		return lineNum + 1, vm.handleFunctionCall(name, args)
	}

	// Handle list comprehension (simplified)
	if match := regexp.MustCompile(`^\[(.+)\]$`).FindStringSubmatch(line); match != nil {
		return lineNum + 1, vm.handleListLiteral(match[1])
	}

	return lineNum + 1, nil
}

// handlePrint handles print() statements
func (vm *pythonVM) handlePrint(expr string) error {
	// Handle multiple arguments separated by commas
	args := vm.splitPrintArgs(expr)
	var results []string
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		result, err := vm.evaluate(trimmed)
		if err != nil {
			// If evaluation fails, print the literal string
			results = append(results, trimmed)
		} else {
			results = append(results, vm.formatValue(result))
		}
	}
	fmt.Fprintln(vm.stdout, strings.Join(results, " "))
	// For debugging, print to stderr too
	fmt.Fprintf(os.Stderr, "[DEBUG] PRINT: %s\n", strings.Join(results, " "))
	return nil
}

// splitPrintArgs splits print arguments by comma (not inside strings)
func (vm *pythonVM) splitPrintArgs(expr string) []string {
	var args []string
	var current strings.Builder
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(expr); i++ {
		ch := expr[i]

		// Track string literals
		if ch == '"' || ch == '\'' {
			if !inString {
				inString = true
				stringChar = ch
			} else if ch == stringChar {
				inString = false
				stringChar = 0
			}
		}

		// Split on comma outside strings
		if ch == ',' && !inString {
			args = append(args, current.String())
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// handlePrintModuleCall handles print(module.func())
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
			arg, _ := vm.evaluate(strings.TrimSpace(argsStr))
			if argFloat, ok := arg.(float64); ok {
				result := f(argFloat)
				fmt.Fprintln(vm.stdout, result)
			}
		}
	case float64, int, string:
		fmt.Fprintln(vm.stdout, f)
	}

	return nil
}

// handleAssignment handles variable assignment
func (vm *pythonVM) handleAssignment(name, value string) error {
	val, err := vm.evaluate(value)
	if err != nil {
		return err
	}
	vm.variables[name] = val
	return nil
}

// handleForLoop handles for loops
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
	case []interface{}:
		for _, item := range v {
			vm.variables[varName] = item
			if err := vm.executeLines(bodyLines, 0, 0); err != nil {
				return lineNum + len(bodyLines) + 1, err
			}
		}
	}

	return lineNum + len(bodyLines) + 1, nil
}

// handleIfStatement handles if statements
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

// handleFunctionDef handles function definitions
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

// handleClassDef handles class definitions
func (vm *pythonVM) handleClassDef(name, parent string, lines []string, lineNum, indent int) (int, error) {
	bodyLines := vm.getBodyLines(lines, lineNum+1, indent)

	class := &pythonClass{
		name:    name,
		attrs:   make(map[string]interface{}),
		methods: make(map[string]*pythonFunction),
	}

	// Parse class body for methods and attributes
	i := 0
	for i < len(bodyLines) {
		line := strings.TrimSpace(bodyLines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			i++
			continue
		}

		// Method detection
		if match := regexp.MustCompile(`^def\s+(\w+)\(self(?:,\s*([^)]*)\))?:$`).FindStringSubmatch(line); match != nil {
			methodName := match[1]
			params := match[2]

			// Collect method body (indented lines)
			trimmedLine := strings.TrimLeft(bodyLines[i], " \t")
			methodIndent := len(bodyLines[i]) - len(trimmedLine)

			methodBody := []string{}
			i++
			for i < len(bodyLines) {
				methodLine := bodyLines[i]
				trimmed := strings.TrimLeft(methodLine, " \t")
				if trimmed == "" {
					methodBody = append(methodBody, methodLine)
					i++
					continue
				}
				currentIndent := len(methodLine) - len(trimmed)

				// Check if still indented (part of method body)
				if currentIndent > methodIndent {
					methodBody = append(methodBody, methodLine)
					i++
				} else {
					break
				}
			}

			// Store the method
			class.methods[methodName] = &pythonFunction{
				name:   methodName,
				params: strings.Split(params, ","),
				body:   strings.Join(methodBody, "\n"),
			}
			continue
		}

		// Attribute assignment (self.attr = value)
		if match := regexp.MustCompile(`^self\.(\w+)\s*=\s*(.+)$`).FindStringSubmatch(line); match != nil {
			class.attrs[match[1]] = match[2]
		}

		i++
	}

	vm.classes[name] = class
	return lineNum + len(bodyLines) + 1, nil
}

// handleListLiteral handles list literals
func (vm *pythonVM) handleListLiteral(expr string) error {
	// Parse list literal
	return nil
}
