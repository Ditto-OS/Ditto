package interpreter

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// CInterpreter executes C code (simplified - uses tcc or gcc if available)
type CInterpreter struct{}

func (c *CInterpreter) Name() string {
	return "c"
}

func (c *CInterpreter) Execute(code string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Try to find a C compiler
	compiler := findCCompiler()
	if compiler == "" {
		// Fall back to pure Go simulation
		return c.executePureGo(code, args, stdin, stdout, stderr)
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "ditto-c-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Write source file
	srcPath := filepath.Join(tmpDir, "program.c")
	if err := os.WriteFile(srcPath, []byte(code), 0644); err != nil {
		return err
	}

	// Compile
	outPath := filepath.Join(tmpDir, "program")
	if os.PathSeparator == '\\' {
		outPath += ".exe"
	}

	cmd := exec.Command(compiler, "-o", outPath, srcPath)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}

	// Run
	runCmd := exec.Command(outPath, args...)
	runCmd.Stdin = stdin
	runCmd.Stdout = stdout
	runCmd.Stderr = stderr
	return runCmd.Run()
}

func (c *CInterpreter) executePureGo(code string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Simplified C-like execution for basic programs
	cvm := &cVM{
		variables: make(map[string]interface{}),
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		args:    args,
	}

	return cvm.Run(code)
}

type cVM struct {
	variables map[string]interface{}
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	args      []string
}

func (vm *cVM) Run(code string) error {
	// Find main function
	mainMatch := regexp.MustCompile(`int\s+main\s*\(([^)]*)\)\s*\{([\s\S]*)\}`).FindStringSubmatch(code)
	if mainMatch == nil {
		return fmt.Errorf("no main function found")
	}

	body := mainMatch[2]
	return vm.executeBody(body)
}

func (vm *cVM) executeBody(body string) error {
	statements := vm.splitStatements(body)
	for _, stmt := range statements {
		if err := vm.executeStatement(stmt); err != nil {
			fmt.Fprintf(vm.stderr, "Error: %v\n", err)
			return err
		}
	}
	return nil
}

func (vm *cVM) splitStatements(body string) []string {
	var statements []string
	var current strings.Builder
	braceCount := 0
	inString := false

	for _, ch := range body {
		current.WriteRune(ch)
		switch ch {
		case '"':
			inString = !inString
		case '{':
			if !inString {
				braceCount++
			}
		case '}':
			if !inString {
				braceCount--
			}
		case ';':
			if !inString && braceCount == 0 {
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

func (vm *cVM) executeStatement(stmt string) error {
	stmt = strings.TrimSpace(stmt)
	if stmt == "" {
		return nil
	}

	// printf()
	if match := regexp.MustCompile(`printf\("([^"]+)"(?:,\s*(.+))?\)`).FindStringSubmatch(stmt); match != nil {
		return vm.handlePrintf(match[1], match[2])
	}

	// scanf()
	if match := regexp.MustCompile(`scanf\("([^"]+)",\s*&(\w+)\)`).FindStringSubmatch(stmt); match != nil {
		return vm.handleScanf(match[1], match[2])
	}

	// Variable declaration with initialization
	if match := regexp.MustCompile(`^(?:int|float|double|char\s*\*|char)\s+(\w+)\s*=\s*(.+)$`).FindStringSubmatch(stmt); match != nil {
		return vm.handleDeclaration(match[1], match[2])
	}

	// Variable declaration without initialization
	if match := regexp.MustCompile(`^(?:int|float|double|char\s*\*|char)\s+(\w+)$`).FindStringSubmatch(stmt); match != nil {
		vm.variables[match[1]] = 0
		return nil
	}

	// Assignment
	if match := regexp.MustCompile(`^(\w+)\s*=\s*(.+)$`).FindStringSubmatch(stmt); match != nil {
		return vm.handleAssignment(match[1], match[2])
	}

	// Return
	if strings.HasPrefix(stmt, "return ") {
		return nil
	}

	return nil
}

func (vm *cVM) handlePrintf(format string, argsStr string) error {
	var args []interface{}
	if argsStr != "" {
		for _, arg := range strings.Split(argsStr, ",") {
			val, _ := vm.evaluate(strings.TrimSpace(arg))
			args = append(args, val)
		}
	}

	// Convert format specifiers
	result := format
	for _, arg := range args {
		switch v := arg.(type) {
		case int:
			result = strings.Replace(result, "%d", strconv.Itoa(v), 1)
			result = strings.Replace(result, "%i", strconv.Itoa(v), 1)
		case float64:
			result = strings.Replace(result, "%f", fmt.Sprintf("%.6f", v), 1)
		case string:
			result = strings.Replace(result, "%s", v, 1)
		}
	}

	// Handle escape sequences
	result = strings.ReplaceAll(result, "\\n", "\n")
	result = strings.ReplaceAll(result, "\\t", "\t")

	fmt.Fprint(vm.stdout, result)
	return nil
}

func (vm *cVM) handleScanf(format string, varName string) error {
	// Simplified - would need actual input handling
	return nil
}

func (vm *cVM) handleDeclaration(name, value string) error {
	val, err := vm.evaluate(value)
	if err != nil {
		return err
	}
	vm.variables[name] = val
	return nil
}

func (vm *cVM) handleAssignment(name, value string) error {
	val, err := vm.evaluate(value)
	if err != nil {
		return err
	}
	vm.variables[name] = val
	return nil
}

func (vm *cVM) evaluate(expr string) (interface{}, error) {
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
		return string(expr[1]), nil
	}

	// Number
	if i, err := strconv.Atoi(expr); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(expr, 64); err == nil {
		return f, nil
	}

	// Character literal
	if strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") && len(expr) == 3 {
		return int(expr[1]), nil
	}

	// Arithmetic
	if strings.ContainsAny(expr, "+-*/%") {
		return vm.evalArithmetic(expr)
	}

	return expr, nil
}

func (vm *cVM) evalArithmetic(expr string) (interface{}, error) {
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
		if right != 0 {
			result = left / right
		}
	case "%":
		result = float64(int(left) % int(right))
	}

	if result == float64(int(result)) {
		return int(result), nil
	}
	return result, nil
}

func findCCompiler() string {
	compilers := []string{"tcc", "gcc", "clang", "cc"}
	for _, compiler := range compilers {
		if path, err := exec.LookPath(compiler); err == nil {
			return path
		}
	}
	return ""
}
