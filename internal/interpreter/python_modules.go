package interpreter

import (
	"fmt"
	"io"
	"strings"
)

// handleImport handles import statements
func (vm *pythonVM) handleImport(moduleName string) error {
	// Clean up the module name by trimming whitespace
	moduleName = strings.TrimSpace(moduleName)

	// Handle "import X as Y" or "import X.Y"
	// Check for "as" alias
	asAlias := ""
	baseModule := moduleName

	if idx := strings.Index(moduleName, " as "); idx > 0 {
		baseModule = strings.TrimSpace(moduleName[:idx])
		asAlias = strings.TrimSpace(moduleName[idx+4:])
	}

	// Handle dotted module names (e.g., "requests.api")
	modulePath := strings.ReplaceAll(baseModule, ".", "/")

	// First check stdlib (only for non-dotted names)
	if !strings.Contains(baseModule, ".") {
		module := vm.stdlib.GetModule(baseModule)
		if module != nil {
			targetName := asAlias
			if targetName == "" {
				targetName = baseModule
			}
			vm.variables[targetName] = module
			return nil
		}
	}

	// Check VFS for installed packages
	if vm.vfs != nil {
		// Try to find module package (__init__.py)
		fullPath := modulePath + "/__init__.py"

		if file, err := vm.vfs.Open(fullPath); err == nil {
			file.Close()
			targetName := asAlias
			if targetName == "" {
				// Use last part of dotted name
				parts := strings.Split(baseModule, ".")
				targetName = parts[len(parts)-1]
			}
			return vm.loadModuleFromVFSWithTarget(targetName, fullPath)
		}

		// Try single file module
		fullPath = modulePath + ".py"
		if file, err := vm.vfs.Open(fullPath); err == nil {
			file.Close()
			targetName := asAlias
			if targetName == "" {
				parts := strings.Split(baseModule, ".")
				targetName = parts[len(parts)-1]
			}
			return vm.loadModuleFromVFSWithTarget(targetName, fullPath)
		}
	}

	return fmt.Errorf("ModuleNotFoundError: No module named '%s'", moduleName)
}

// loadModuleFromVFS loads a module from the virtual filesystem
func (vm *pythonVM) loadModuleFromVFS(moduleName, modulePath string) error {
	return vm.loadModuleFromVFSWithTarget(moduleName, modulePath)
}

// loadModuleFromVFSWithTarget loads a module with a specific target variable name
func (vm *pythonVM) loadModuleFromVFSWithTarget(targetName, modulePath string) error {
	// Open and read the module file
	file, err := vm.vfs.Open(modulePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info for size
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	// Read file content
	content := make([]byte, stat.Size())
	n, err := file.Read(content)
	if err != nil && err != io.EOF {
		return err
	}
	content = content[:n]

	// Create a new VM scope for the module
	moduleScope := &pythonVM{
		variables: make(map[string]interface{}),
		functions: make(map[string]*pythonFunction),
		classes:   make(map[string]*pythonClass),
		stdlib:    vm.stdlib,
		stdin:     vm.stdin,
		stdout:    vm.stdout,
		stderr:    vm.stderr,
		vfs:       vm.vfs,
	}

	// Execute the module code
	moduleCode := string(content)
	if err := moduleScope.Run(moduleCode); err != nil {
		// If execution fails, still register basic module
		vm.variables[targetName] = make(map[string]interface{})
		return nil
	}

	// Export the module's public symbols (those not starting with _)
	// Exception: keep _api for re-exports to work
	exportedModule := make(map[string]interface{})
	for name, value := range moduleScope.variables {
		if !strings.HasPrefix(name, "_") || name == "_api" {
			exportedModule[name] = value
		}
	}

	// Also export functions
	for name, fn := range moduleScope.functions {
		if !strings.HasPrefix(name, "_") || name == "_api" {
			exportedModule[name] = fn
		}
	}

	// Also export classes
	for name, class := range moduleScope.classes {
		if !strings.HasPrefix(name, "_") || name == "_api" {
			exportedModule[name] = class
		}
	}

	vm.variables[targetName] = exportedModule
	return nil
}

// handleFromImport handles from X import Y statements
func (vm *pythonVM) handleFromImport(moduleName, imports string) error {
	// First check stdlib
	module := vm.stdlib.GetModule(moduleName)
	if module != nil {
		// Import specific items
		for _, item := range strings.Split(imports, ",") {
			item = strings.TrimSpace(item)
			if val, ok := module[item]; ok {
				vm.variables[item] = val
			}
		}
		return nil
	}

	// Check VFS for installed packages
	if vm.vfs != nil {
		modulePath := moduleName + "/__init__.py"
		if file, err := vm.vfs.Open(modulePath); err == nil {
			file.Close()
			// Module found - load it first
			if err := vm.loadModuleFromVFS(moduleName, modulePath); err != nil {
				return err
			}
			// Then import specific items
			if mod, ok := vm.variables[moduleName].(map[string]interface{}); ok {
				for _, item := range strings.Split(imports, ",") {
					item = strings.TrimSpace(item)
					if item == "*" {
						// Import all public symbols
						for k, v := range mod {
							if !strings.HasPrefix(k, "_") {
								vm.variables[k] = v
							}
						}
					} else if val, ok := mod[item]; ok {
						vm.variables[item] = val
					}
				}
			}
			return nil
		}

		modulePath = moduleName + ".py"
		if file, err := vm.vfs.Open(modulePath); err == nil {
			file.Close()
			return nil
		}
	}

	return fmt.Errorf("ModuleNotFoundError: No module named '%s'", moduleName)
}
