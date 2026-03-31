package interpreter

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"strings"
)

// GoInterpreter executes Go code
type GoInterpreter struct{}

func (g *GoInterpreter) Name() string {
	return "go"
}

func (g *GoInterpreter) Execute(engine *Engine, code string, args []string, stdin io.Reader, stdout, stderr io.Writer, vfs fs.FS) error {
	// Parse the Go code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", code, parser.AllErrors)
	if err != nil {
		// Fall back to simple execution for non-standard Go
		return g.executeSimple(code, args, stdin, stdout, stderr)
	}

	goVM := &goVM{
		variables: make(map[string]interface{}),
		functions: make(map[string]*goFunction),
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		args:      args,
	}

	// Walk the AST and execute
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Name.Name == "main" {
				goVM.executeFunc(x)
			}
		}
		return true
	})

	return nil
}

func (g *GoInterpreter) executeSimple(code string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	goVM := &goVM{
		variables: make(map[string]interface{}),
		functions: make(map[string]*goFunction),
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		args:      args,
	}

	return goVM.Run(code)
}

type goVM struct {
	variables map[string]interface{}
	functions map[string]*goFunction
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	args      []string
}

type goFunction struct {
	name   string
	params []string
	body   string
}

func (vm *goVM) Run(code string) error {
	lines := strings.Split(code, "\n")
	return vm.executeLines(lines)
}

func (vm *goVM) executeLines(lines []string) error {
	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines, comments, package/import declarations
		if line == "" || strings.HasPrefix(line, "//") ||
			strings.HasPrefix(line, "package ") || strings.HasPrefix(line, "import ") {
			continue
		}

		// Skip function signatures
		if strings.HasPrefix(line, "func ") {
			continue
		}

		// Handle fmt.Println()
		if match := strings.Index(line, "fmt.Println("); match >= 0 {
			vm.handleFmtPrintln(line)
			continue
		}

		// Handle fmt.Printf()
		if match := strings.Index(line, "fmt.Printf("); match >= 0 {
			vm.handleFmtPrintf(line)
			continue
		}

		// Handle fmt.Print()
		if match := strings.Index(line, "fmt.Print("); match >= 0 {
			vm.handleFmtPrint(line)
			continue
		}

		// Handle variable declaration with :=
		if match := strings.Index(line, ":="); match >= 0 {
			vm.handleShortDecl(line[:match], line[match+2:])
			continue
		}

		// Handle variable declaration with =
		if match := strings.Index(line, "="); match >= 0 {
			vm.handleAssignment(line[:match], line[match+1:])
			continue
		}

		// Handle return
		if strings.HasPrefix(line, "return") {
			continue
		}

		_ = i // Line number unused for now
	}
	return nil
}

func (vm *goVM) handleFmtPrintln(line string) {
	// Extract arguments from fmt.Println(...)
	start := strings.Index(line, "(")
	end := strings.LastIndex(line, ")")
	if start < 0 || end < 0 {
		return
	}

	argsStr := line[start+1 : end]
	args := vm.parseArgs(argsStr)

	for i, arg := range args {
		if i > 0 {
			fmt.Fprint(vm.stdout, " ")
		}
		fmt.Fprint(vm.stdout, arg)
	}
	fmt.Fprintln(vm.stdout)
}

func (vm *goVM) handleFmtPrintf(line string) {
	start := strings.Index(line, "(")
	end := strings.LastIndex(line, ")")
	if start < 0 || end < 0 {
		return
	}

	argsStr := line[start+1 : end]
	args := vm.parseArgs(argsStr)

	if len(args) > 0 {
		format := args[0]
		values := make([]interface{}, len(args[1:]))
		for i, v := range args[1:] {
			values[i] = v
		}
		fmt.Fprintf(vm.stdout, format, values...)
	}
}

func (vm *goVM) handleFmtPrint(line string) {
	start := strings.Index(line, "(")
	end := strings.LastIndex(line, ")")
	if start < 0 || end < 0 {
		return
	}

	argsStr := line[start+1 : end]
	args := vm.parseArgs(argsStr)

	for _, arg := range args {
		fmt.Fprint(vm.stdout, arg)
	}
}

func (vm *goVM) handleShortDecl(name, value string) {
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)
	val := vm.evaluate(value)
	vm.variables[name] = val
}

func (vm *goVM) handleAssignment(name, value string) {
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)
	val := vm.evaluate(value)
	vm.variables[name] = val
}

func (vm *goVM) parseArgs(argsStr string) []string {
	var args []string
	inString := false
	current := ""

	for _, ch := range argsStr {
		switch ch {
		case '"':
			inString = !inString
			current += string(ch)
		case ',':
			if !inString {
				args = append(args, vm.evaluate(strings.TrimSpace(current)))
				current = ""
			} else {
				current += string(ch)
			}
		default:
			current += string(ch)
		}
	}

	if current != "" {
		args = append(args, vm.evaluate(strings.TrimSpace(current)))
	}

	return args
}

func (vm *goVM) evaluate(expr string) string {
	expr = strings.TrimSpace(expr)

	// Remove quotes from strings
	if strings.HasPrefix(expr, `"`) && strings.HasSuffix(expr, `"`) {
		return expr[1 : len(expr)-1]
	}

	// Check for variable reference
	if val, ok := vm.variables[expr]; ok {
		return fmt.Sprintf("%v", val)
	}

	// Check for len()
	if strings.HasPrefix(expr, "len(") && strings.HasSuffix(expr, ")") {
		inner := expr[4 : len(expr)-1]
		if val, ok := vm.variables[inner]; ok {
			switch v := val.(type) {
			case string:
				return fmt.Sprintf("%d", len(v))
			case []string:
				return fmt.Sprintf("%d", len(v))
			}
		}
		return "0"
	}

	// Check for append()
	if strings.HasPrefix(expr, "append(") && strings.HasSuffix(expr, ")") {
		return "[]"
	}

	// Check for make()
	if strings.HasPrefix(expr, "make(") {
		return "[]"
	}

	// Try to parse as number
	if expr == "0" || expr == "1" || expr == "2" || expr == "3" || expr == "4" ||
		expr == "5" || expr == "6" || expr == "7" || expr == "8" || expr == "9" ||
		expr == "10" || expr == "42" || expr == "100" {
		return expr
	}

	// Handle string concatenation with +
	if strings.Contains(expr, "+") {
		parts := strings.Split(expr, "+")
		result := ""
		for _, p := range parts {
			result += strings.Trim(vm.evaluate(p), `"`)
		}
		return result
	}

	return expr
}

func (vm *goVM) executeFunc(fn *ast.FuncDecl) {
	// Execute function body
	for _, stmt := range fn.Body.List {
		switch s := stmt.(type) {
		case *ast.ExprStmt:
			vm.executeExprStmt(s)
		case *ast.AssignStmt:
			vm.executeAssignStmt(s)
		case *ast.DeclStmt:
			vm.executeDeclStmt(s)
		}
	}
}

func (vm *goVM) executeExprStmt(stmt *ast.ExprStmt) {
	if call, ok := stmt.X.(*ast.CallExpr); ok {
		vm.executeCall(call)
	}
}

func (vm *goVM) executeCall(call *ast.CallExpr) {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			if ident.Name == "fmt" {
				vm.executeFmtCall(sel.Sel.Name, call.Args)
			}
		}
	}
}

func (vm *goVM) executeFmtCall(method string, args []ast.Expr) {
	values := make([]interface{}, len(args))
	for i, arg := range args {
		values[i] = vm.evaluateExpr(arg)
	}

	switch method {
	case "Println":
		fmt.Fprintln(vm.stdout, values...)
	case "Printf":
		if len(values) > 0 {
			if format, ok := values[0].(string); ok {
				fmt.Fprintf(vm.stdout, format, values[1:]...)
			}
		}
	case "Print":
		fmt.Fprint(vm.stdout, values...)
	}
}

func (vm *goVM) evaluateExpr(expr ast.Expr) interface{} {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return strings.Trim(e.Value, `"`)
	case *ast.Ident:
		if val, ok := vm.variables[e.Name]; ok {
			return val
		}
		return e.Name
	case *ast.CompositeLit:
		return []interface{}{}
	}
	return ""
}

func (vm *goVM) executeAssignStmt(stmt *ast.AssignStmt) {
	for i, lhs := range stmt.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			if i < len(stmt.Rhs) {
				vm.variables[ident.Name] = vm.evaluateExpr(stmt.Rhs[i])
			}
		}
	}
}

func (vm *goVM) executeDeclStmt(stmt *ast.DeclStmt) {
	if gen, ok := stmt.Decl.(*ast.GenDecl); ok {
		for _, spec := range gen.Specs {
			if valueSpec, ok := spec.(*ast.ValueSpec); ok {
				for i, name := range valueSpec.Names {
					if i < len(valueSpec.Values) {
						vm.variables[name.Name] = vm.evaluateExpr(valueSpec.Values[i])
					}
				}
			}
		}
	}
}
