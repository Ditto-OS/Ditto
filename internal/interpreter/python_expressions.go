package interpreter

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// evaluate evaluates an expression and returns its value
func (vm *pythonVM) evaluate(expr string) (interface{}, error) {
	expr = strings.TrimSpace(expr)
	fmt.Fprintf(os.Stderr, "[DEBUG] evaluate: %s\n", expr)

	// Check for module function call (e.g., requests.get("url"))
	callMatch := regexp.MustCompile(`^(\w+)\.(\w+)\(([^)]*)\)$`).FindStringSubmatch(expr)
	if callMatch != nil {
		moduleName := callMatch[1]
		funcName := callMatch[2]
		argsStr := callMatch[3]

		if module, ok := vm.variables[moduleName].(map[string]interface{}); ok {
			if fn, ok := module[funcName]; ok {
				// Handle pythonFunction objects
				if pyFn, ok := fn.(*pythonFunction); ok {
					return vm.callPythonFunction(pyFn, argsStr)
				}
				
				// Call the function with arguments
				switch f := fn.(type) {
				case func(string) interface{}:
					return f(strings.Trim(argsStr, "\"'")), nil
				case func(string, map[string]interface{}) interface{}:
					return f(strings.Trim(argsStr, "\"'"), nil), nil
				default:
					return fn, nil
				}
			}
		}
	}

	// Check for object attribute access (e.g., resp.status_code, resp.json())
	attrMatch := regexp.MustCompile(`^(\w+)\.(\w+)(?:\(([^)]*)\))?$`).FindStringSubmatch(expr)
	if attrMatch != nil {
		objName := attrMatch[1]
		attrName := attrMatch[2]
		args := attrMatch[3]

		// Check if it's an object in variables
		if obj, ok := vm.variables[objName]; ok {
			switch o := obj.(type) {
			case map[string]interface{}:
				// First check instance attributes
				if attr, ok := o[attrName]; ok {
					// It's a function call if there are args
					if args != "" {
						switch fn := attr.(type) {
						case func() string:
							return fn(), nil
						case func() interface{}:
							return fn(), nil
						case func() map[string]interface{}:
							return fn(), nil
						case func() int:
							return fn(), nil
						default:
							return attr, nil
						}
					}
					// Return the attribute value directly
					return attr, nil
				}

				// Check class methods if not found in instance
				if class, ok := o["__class__"].(*pythonClass); ok {
					if method, ok := class.methods[attrName]; ok {
						// It's a method - return it
						return method, nil
					}
				}
			}
		}
	}

	// Check for class instantiation (e.g., MyClass() or MyClass(arg1, arg2))
	if match := regexp.MustCompile(`^(\w+)\((.*)\)$`).FindStringSubmatch(expr); match != nil {
		className := match[1]

		if class, ok := vm.classes[className]; ok {
			// Create instance
			instance := make(map[string]interface{})
			instance["__class__"] = class

			// First copy class-level attributes
			for k, v := range class.attrs {
				if val, err := vm.evaluate(fmt.Sprintf("%v", v)); err == nil {
					instance[k] = val
				} else {
					instance[k] = v
				}
			}

			// Call __init__ if it exists
			if initFn, ok := class.methods["__init__"]; ok {
				// Set up self with the instance
				tempVars := map[string]interface{}{"self": instance}

				// Execute __init__ body line by line
				initLines := strings.Split(initFn.body, "\n")
				for _, line := range initLines {
					line = strings.TrimSpace(line)
					if line == "" || strings.HasPrefix(line, "#") {
						continue
					}

					// Handle self.attr = value
					if m := regexp.MustCompile(`^self\.(\w+)\s*=\s*(.+)$`).FindStringSubmatch(line); m != nil {
						attrName := m[1]
						attrValue := m[2]

						// Evaluate the value expression
						val, err := vm.evaluateWithVars(attrValue, tempVars)
						if err != nil {
							instance[attrName] = attrValue
						} else {
							instance[attrName] = val
						}
					}
				}
			}

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

// evaluateWithVars evaluates an expression with additional variables
func (vm *pythonVM) evaluateWithVars(expr string, extraVars map[string]interface{}) (interface{}, error) {
	// Temporarily merge extra variables
	originalVars := make(map[string]interface{})
	for k, v := range vm.variables {
		originalVars[k] = v
	}
	for k, v := range extraVars {
		vm.variables[k] = v
	}

	result, err := vm.evaluate(expr)

	// Restore original variables
	vm.variables = originalVars

	return result, err
}

// evalListComprehension evaluates list comprehensions
func (vm *pythonVM) evalListComprehension(expr string) ([]interface{}, error) {
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

// parseListLiteral parses a list literal
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

// parseDictLiteral parses a dict literal
func (vm *pythonVM) parseDictLiteral(expr string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	inner := strings.TrimSpace(expr[1 : len(expr)-1])
	if inner == "" {
		return result, nil
	}

	// Simple comma-separated key:value parsing
	// Note: this doesn't handle nested dicts or commas in strings well
	parts := vm.splitPrintArgs(inner) // Reuse split logic that handles strings
	for _, p := range parts {
		kv := strings.SplitN(p, ":", 2)
		if len(kv) == 2 {
			key, _ := vm.evaluate(strings.TrimSpace(kv[0]))
			val, _ := vm.evaluate(strings.TrimSpace(kv[1]))
			if keyStr, ok := key.(string); ok {
				result[keyStr] = val
			} else {
				result[fmt.Sprintf("%v", key)] = val
			}
		}
	}
	return result, nil
}

// evalFString evaluates an f-string
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

// evalArithmetic evaluates arithmetic expressions
func (vm *pythonVM) evalArithmetic(expr string) (interface{}, error) {
	// Support string concatenation
	if strings.Contains(expr, "+") {
		parts := strings.Split(expr, "+")
		var result strings.Builder
		isStringConcat := false

		// Check if it's string concat (if any part is a string literal or variable containing string)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			val, _ := vm.evaluate(p)
			if _, ok := val.(string); ok {
				isStringConcat = true
				break
			}
		}

		if isStringConcat {
			for _, p := range parts {
				p = strings.TrimSpace(p)
				val, _ := vm.evaluate(p)
				result.WriteString(vm.formatValue(val))
			}
			return result.String(), nil
		}
	}

	re := regexp.MustCompile(`(-?\d+(?:\.\d+)?)\s*([+\-*/])\s*(-?\d+(?:\.\d+)?)`)
	match := re.FindStringSubmatch(expr)
	if match == nil {
		// Try to evaluate parts if they are variables
		reVar := regexp.MustCompile(`(\w+)\s*([+\-*/])\s*(\w+)`)
		matchVar := reVar.FindStringSubmatch(expr)
		if matchVar != nil {
			leftVal, _ := vm.evaluate(matchVar[1])
			op := matchVar[2]
			rightVal, _ := vm.evaluate(matchVar[3])

			left, ok1 := toFloat(leftVal)
			right, ok2 := toFloat(rightVal)

			if ok1 && ok2 {
				return calcFloat(left, op, right), nil
			}
		}
		return expr, nil
	}

	left, _ := strconv.ParseFloat(match[1], 64)
	op := match[2]
	right, _ := strconv.ParseFloat(match[3], 64)

	return calcFloat(left, op, right), nil
}

func toFloat(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case int:
		return float64(v), true
	case float64:
		return v, true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func calcFloat(left float64, op string, right float64) interface{} {
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
		return int(result)
	}
	return result
}

// parseParams parses function parameters
func (vm *pythonVM) parseParams(params string) []string {
	if params == "" {
		return []string{}
	}
	return strings.Split(params, ",")
}

// parseArgs parses function call arguments
func (vm *pythonVM) parseArgs(argsStr string) []interface{} {
	if argsStr == "" {
		return []interface{}{}
	}

	var args []interface{}
	parts := strings.Split(argsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if val, err := vm.evaluate(p); err == nil {
			args = append(args, val)
		} else {
			args = append(args, p)
		}
	}
	return args
}
