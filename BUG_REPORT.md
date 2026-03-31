# Bug Report: Python Package Execution Incomplete

**Date:** 2026-03-29  
**Status:** Partially Fixed  
**Priority:** High

---

## Summary

Python packages can be **imported and loaded**, but **function execution is incomplete**. The interpreter successfully:
- ✅ Loads modules from VFS
- ✅ Handles dotted imports (`import requests.api as _api`)
- ✅ Finds functions in modules
- ✅ Calls Python functions

But **function bodies don't execute properly** - assignments and string concatenation inside functions don't work.

---

## Current Behavior

### What Works

```python
import requests  # ✅ Module loads
resp = requests.get("http://test.com")  # ✅ Function is called
print(resp)  # ⚠️ Returns function call expression, not result
```

### What Doesn't Work

```python
# Inside function body:
def get(url, params=None, **kwargs):
    text = '{"success": true, "url": "' + url + '"}'  # ❌ String concatenation fails
    return Response(200, text, {...})  # ❌ Return value not properly constructed
```

**Output:**
```
Response: Response(200, text, {'Content-Type': 'application/json'})
```

**Expected:**
```
Response: <Response [200]>
```

---

## Root Causes Identified

### 1. String Concatenation Not Implemented

The expression evaluator doesn't handle `+` for string concatenation:

```go
// In evaluate():
if strings.ContainsAny(expr, "+-*/") {
    return vm.evalArithmetic(expr)  // Only handles numbers!
}
```

**Fix needed:** Add string concatenation to `evalArithmetic()` or create separate `evalStringConcat()`.

### 2. Function Body Execution Incomplete

When `callPythonFunction()` executes a function body:
- It only handles assignments (`x = y`)
- It doesn't execute string operations
- It doesn't handle `if` statements inside functions
- It doesn't call class constructors properly

### 3. Class Constructor Calls Don't Work

```python
return Response(200, text, {'Content-Type': 'application/json'})
```

The `Response(...)` call inside a function:
- Creates the class instance
- But `text` variable isn't resolved (it's a local variable)
- Dict literals `{...}` aren't parsed

---

## Debug Evidence

From test run with debug logging:

```
[DEBUG] callPythonFunction: get("http://test.com")
[DEBUG] evaluate: "\"http://test.com\""           # ✅ Argument parsed
[DEBUG] evaluate: "'{\"success\": true, \"url\": \"' + url + '\"}'"  # ❌ String concat fails
[DEBUG] evaluate: "Response(200, text, {'Content-Type': 'application/json'})"  # ❌ Constructor fails
```

---

## Files Affected

| File | Issue |
|------|-------|
| `internal/interpreter/python_expressions.go` | `evalArithmetic()` doesn't handle strings |
| `internal/interpreter/python_builtins.go` | `callPythonFunction()` doesn't execute all statement types |
| `internal/interpreter/python_statements.go` | Class instantiation regex doesn't handle dict arguments |

---

## Required Fixes

### Priority 1: String Concatenation

```go
// Add to python_expressions.go
func (vm *pythonVM) evalStringConcat(expr string) (interface{}, error) {
    // Split by + and concatenate strings
    // Handle mixed string/variable expressions
}
```

### Priority 2: Function Body Execution

Extend `callPythonFunction()` to handle:
- `if` statements
- String operations in assignments
- Function calls within functions

### Priority 3: Dict Literal Parsing

```go
// Fix parseDictLiteral() to actually parse key-value pairs
func (vm *pythonVM) parseDictLiteral(expr string) (map[string]interface{}, error) {
    // Parse {"key": value, ...}
}
```

---

## Test Cases

### Test 1: Basic String Concatenation
```python
def test():
    url = "http://test.com"
    text = "URL: " + url
    return text
```
**Expected:** `"URL: http://test.com"`  
**Actual:** `"URL: " + url` (literal string)

### Test 2: requests.get()
```python
import requests
resp = requests.get("http://test.com")
print(resp.status_code)
```
**Expected:** `200`  
**Actual:** `resp.status_code` (literal)

### Test 3: Class with Constructor
```python
class Foo:
    def __init__(self, value):
        self.value = value

f = Foo("test")
print(f.value)
```
**Expected:** `test`  
**Actual:** `f.value` (literal)

---

## Workaround

For now, the embedded packages work with **Go-native implementations** instead of Python code. The requests package could be implemented as Go functions that return proper objects.

---

## Next Steps

1. **Implement string concatenation** in expression evaluator
2. **Fix dict literal parsing** 
3. **Extend function body execution** to handle if statements
4. **Add comprehensive test suite** for Python interpreter

---

## Related

- Issue: Python interpreter is ~1000 lines but needs ~3000+ for full compatibility
- Alternative: Use WASM runtime (Pyodide) for full Python support
- See: `TODO.md` for implementation roadmap
