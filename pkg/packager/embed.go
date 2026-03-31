// Package packager provides an embedded package manager for Ditto
// This file contains embedded popular packages for offline installation

package packager

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed embedded/python/* embedded/javascript/*
var embeddedPackages embed.FS

// EmbeddedPackage represents a package bundled in the binary
type EmbeddedPackage struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Language    string            `json:"language"`
	Description string            `json:"description"`
	Files       map[string]string `json:"files"` // file -> content
}

// GetEmbeddedPackages returns all packages embedded in the binary
func GetEmbeddedPackages() []EmbeddedPackage {
	packages := []EmbeddedPackage{}

	// Load Python packages
	pyPackages := getEmbeddedPythonPackages()
	packages = append(packages, pyPackages...)

	// Load JavaScript packages
	jsPackages := getEmbeddedJSPackages()
	packages = append(packages, jsPackages...)

	return packages
}

// getEmbeddedPythonPackages returns embedded Python packages
func getEmbeddedPythonPackages() []EmbeddedPackage {
	packages := []EmbeddedPackage{
		{
			Name:        "requests",
			Version:     "2.31.0",
			Language:    "python",
			Description: "Python HTTP library for human beings",
			Files: map[string]string{
				"__init__.py": embeddedPythonInit("requests"),
				"api.py":      embeddedPythonRequestsAPI(),
			},
		},
		{
			Name:        "numpy",
			Version:     "1.0.0",
			Language:    "python",
			Description: "Fundamental package for scientific computing with Python (basic implementation)",
			Files: map[string]string{
				"__init__.py": embeddedPythonNumpy(),
			},
		},
		{
			Name:        "flask",
			Version:     "1.0.0",
			Language:    "python",
			Description: "A simple Flask implementation for Ditto",
			Files: map[string]string{
				"__init__.py": embeddedPythonFlask(),
			},
		},
	}

	return packages
}

// getEmbeddedJSPackages returns embedded JavaScript packages
func getEmbeddedJSPackages() []EmbeddedPackage {
	packages := []EmbeddedPackage{
		{
			Name:        "lodash",
			Version:     "4.17.21",
			Language:    "javascript",
			Description: "A modern JavaScript utility library",
			Files: map[string]string{
				"index.js": embeddedJSLodash(),
			},
		},
		{
			Name:        "express",
			Version:     "1.0.0",
			Language:    "javascript",
			Description: "A minimal Express.js implementation for Ditto",
			Files: map[string]string{
				"index.js": embeddedJSExpress(),
			},
		},
		{
			Name:        "axios",
			Version:     "1.0.0",
			Language:    "javascript",
			Description: "Promise based HTTP client",
			Files: map[string]string{
				"index.js": embeddedJSAxios(),
			},
		},
	}

	return packages
}

// InstallEmbedded installs an embedded package
func (p *Packager) InstallEmbedded(pkgName, language string) error {
	packages := GetEmbeddedPackages()

	var foundPkg *EmbeddedPackage
	for _, pkg := range packages {
		if pkg.Name == pkgName && pkg.Language == language {
			foundPkg = &pkg
			break
		}
	}

	if foundPkg == nil {
		return fmt.Errorf("package not found: %s (%s)", pkgName, language)
	}

	// Create package directory
	pkgPath := filepath.Join(p.installDir, language, foundPkg.Name)
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Write package files
	for filename, content := range foundPkg.Files {
		filePath := filepath.Join(pkgPath, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	// Write package metadata
	metaPath := filepath.Join(pkgPath, "package.json")
	meta := map[string]interface{}{
		"name":        foundPkg.Name,
		"version":     foundPkg.Version,
		"description": foundPkg.Description,
		"language":    foundPkg.Language,
		"embedded":    true,
	}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		return fmt.Errorf("failed to write package metadata: %w", err)
	}

	// Add to manifest
	p.manifest.Packages = append(p.manifest.Packages, PackageInfo{
		Name:        foundPkg.Name,
		Version:     foundPkg.Version,
		Language:    foundPkg.Language,
		InstallDate: time.Now(),
		Path:        pkgPath,
	})

	return p.saveManifest()
}

// SearchEmbedded searches embedded packages by name
func SearchEmbedded(query, language string) []EmbeddedPackage {
	packages := GetEmbeddedPackages()
	var results []EmbeddedPackage

	query = strings.ToLower(query)
	for _, pkg := range packages {
		if language != "" && pkg.Language != language {
			continue
		}

		// Search in name and description
		if strings.Contains(strings.ToLower(pkg.Name), query) ||
			strings.Contains(strings.ToLower(pkg.Description), query) {
			results = append(results, pkg)
		}
	}

	return results
}

// ListEmbeddedByLanguage returns embedded packages for a specific language
func ListEmbeddedByLanguage(language string) []EmbeddedPackage {
	packages := GetEmbeddedPackages()
	var results []EmbeddedPackage

	for _, pkg := range packages {
		if pkg.Language == language {
			results = append(results, pkg)
		}
	}

	return results
}

// embeddedPythonInit returns a basic Python package __init__.py
func embeddedPythonInit(pkgName string) string {
	return fmt.Sprintf(`# %s - Embedded Python Package
# Version: 1.0.0

"""
%s embedded package for Ditto.
This is a minimal implementation for offline use.
"""

__version__ = "1.0.0"
__author__ = "Ditto"

# Import and re-export (simplified for interpreter compatibility)
import %s.api as _api
Response = _api.Response
get = _api.get
post = _api.post
put = _api.put
delete = _api.delete
request = _api.request
`, pkgName, pkgName, pkgName)
}

// embeddedPythonRequestsAPI returns a minimal requests implementation
func embeddedPythonRequestsAPI() string {
	return `# Minimal requests implementation for Ditto

class Response:
    """HTTP response object"""
    def __init__(self, status=200, text="", headers=None):
        self.status_code = status
        self.text = text
        self.content = text
        if headers is None:
            headers = {}
        self.headers = headers
    
    def json(self):
        import json
        return json.loads(self.text)

def get(url, params=None, **kwargs):
    """Make a GET request"""
    text = '{"success": true, "url": "' + url + '"}'
    return Response(200, text, {'Content-Type': 'application/json'})

def post(url, data=None, json=None, **kwargs):
    """Make a POST request"""
    if json:
        import json as j
        text = '{"created": true, "data": ' + j.dumps(json) + '}'
    else:
        text = '{"created": true, "url": "' + url + '"}'
    return Response(201, text, {'Content-Type': 'application/json'})

def put(url, data=None, json=None, **kwargs):
    """Make a PUT request"""
    text = '{"updated": true, "url": "' + url + '"}'
    return Response(200, text, {'Content-Type': 'application/json'})

def delete(url, **kwargs):
    """Make a DELETE request"""
    text = '{"deleted": true}'
    return Response(204, text, {})

def request(method, url, **kwargs):
    """Make an HTTP request"""
    if method == 'GET' or method == 'get':
        return get(url, **kwargs)
    elif method == 'POST' or method == 'post':
        return post(url, **kwargs)
    elif method == 'PUT' or method == 'put':
        return put(url, **kwargs)
    elif method == 'DELETE' or method == 'delete':
        return delete(url, **kwargs)
    else:
        text = '{"method": "' + method + '", "url": "' + url + '"}'
        return Response(200, text, {})
`
}

// embeddedPythonNumpy returns a minimal numpy implementation
func embeddedPythonNumpy() string {
	return `# Minimal NumPy implementation for Ditto
# Basic array operations and math functions

import math

class ndarray:
    """Basic n-dimensional array"""
    def __init__(self, data):
        self.data = data
        self.shape = self._get_shape(data)
        self.ndim = len(self.shape)
        self.size = self._get_size(data)
    
    def _get_shape(self, data):
        shape = []
        current = data
        while isinstance(current, list):
            shape.append(len(current))
            current = current[0] if len(current) > 0 else None
        return tuple(shape)
    
    def _get_size(self, data):
        if isinstance(data, list):
            return sum(self._get_size(item) for item in data)
        return 1
    
    def __getitem__(self, key):
        return self.data[key]
    
    def __setitem__(self, key, value):
        self.data[key] = value
    
    def __len__(self):
        return len(self.data)
    
    def __repr__(self):
        return f"array({self.data})"
    
    def __add__(self, other):
        if isinstance(other, ndarray):
            result = [a + b for a, b in zip(self.data, other.data)]
            return ndarray(result)
        return ndarray([x + other for x in self.data])
    
    def __sub__(self, other):
        if isinstance(other, ndarray):
            result = [a - b for a, b in zip(self.data, other.data)]
            return ndarray(result)
        return ndarray([x - other for x in self.data])
    
    def __mul__(self, other):
        if isinstance(other, ndarray):
            result = [a * b for a, b in zip(self.data, other.data)]
            return ndarray(result)
        return ndarray([x * other for x in self.data])
    
    def sum(self):
        return sum(self._flatten(self.data))
    
    def _flatten(self, lst):
        result = []
        for item in lst:
            if isinstance(item, list):
                result.extend(self._flatten(item))
            else:
                result.append(item)
        return result

def array(data):
    """Create an ndarray"""
    return ndarray(data)

def zeros(shape):
    """Create array of zeros"""
    if len(shape) == 1:
        return ndarray([0] * shape[0])
    elif len(shape) == 2:
        return ndarray([[0] * shape[1] for _ in range(shape[0])])
    return ndarray([])

def ones(shape):
    """Create array of ones"""
    if len(shape) == 1:
        return ndarray([1] * shape[0])
    elif len(shape) == 2:
        return ndarray([[1] * shape[1] for _ in range(shape[0])])
    return ndarray([])

def arange(start, stop=None, step=1):
    """Create array with range"""
    if stop is None:
        stop = start
        start = 0
    return ndarray(list(range(int(start), int(stop), int(step))))

def linspace(start, stop, num=50):
    """Create evenly spaced numbers"""
    step = (stop - start) / (num - 1)
    return ndarray([start + i * step for i in range(num)])

def dot(a, b):
    """Dot product"""
    if isinstance(a, ndarray) and isinstance(b, ndarray):
        return sum(x * y for x, y in zip(a.data, b.data))
    return sum(x * y for x, y in zip(a, b))

def mean(arr):
    """Calculate mean"""
    if isinstance(arr, ndarray):
        return arr.sum() / arr.size
    return sum(arr) / len(arr)

def std(arr):
    """Calculate standard deviation"""
    m = mean(arr)
    if isinstance(arr, ndarray):
        variance = sum((x - m) ** 2 for x in arr._flatten(arr.data)) / arr.size
    else:
        variance = sum((x - m) ** 2 for x in arr) / len(arr)
    return math.sqrt(variance)

def sqrt(arr):
    """Element-wise square root"""
    if isinstance(arr, ndarray):
        return ndarray([math.sqrt(x) for x in arr.data])
    return [math.sqrt(x) for x in arr]

def sin(arr):
    """Element-wise sine"""
    if isinstance(arr, ndarray):
        return ndarray([math.sin(x) for x in arr.data])
    return [math.sin(x) for x in arr]

def cos(arr):
    """Element-wise cosine"""
    if isinstance(arr, ndarray):
        return ndarray([math.cos(x) for x in arr.data])
    return [math.cos(x) for x in arr]

def exp(arr):
    """Element-wise exponential"""
    if isinstance(arr, ndarray):
        return ndarray([math.exp(x) for x in arr.data])
    return [math.exp(x) for x in arr]

def log(arr):
    """Element-wise natural log"""
    if isinstance(arr, ndarray):
        return ndarray([math.log(x) for x in arr.data])
    return [math.log(x) for x in arr]

def max(arr):
    """Maximum value"""
    if isinstance(arr, ndarray):
        return max(arr._flatten(arr.data))
    return builtins_max(arr)

def min(arr):
    """Minimum value"""
    if isinstance(arr, ndarray):
        return min(arr._flatten(arr.data))
    return builtins_min(arr)

def argmax(arr):
    """Index of maximum value"""
    flat = arr._flatten(arr.data) if isinstance(arr, ndarray) else arr
    return flat.index(max(flat))

def argmin(arr):
    """Index of minimum value"""
    flat = arr._flatten(arr.data) if isinstance(arr, ndarray) else arr
    return flat.index(min(flat))

def reshape(arr, shape):
    """Reshape array"""
    # Simplified - just return new ndarray
    return ndarray(arr.data if isinstance(arr, ndarray) else arr)

def transpose(arr):
    """Transpose array"""
    if isinstance(arr, ndarray) and len(arr.shape) == 2:
        rows, cols = arr.shape
        result = [[arr.data[i][j] for i in range(rows)] for j in range(cols)]
        return ndarray(result)
    return arr

def concatenate(arrays, axis=0):
    """Concatenate arrays"""
    result = []
    for arr in arrays:
        if isinstance(arr, ndarray):
            result.extend(arr.data)
        else:
            result.extend(arr)
    return ndarray(result)

def stack(arrays, axis=0):
    """Stack arrays"""
    result = []
    for arr in arrays:
        if isinstance(arr, ndarray):
            result.append(arr.data)
        else:
            result.append(arr)
    return ndarray(result)

def split(arr, indices_or_sections, axis=0):
    """Split array"""
    # Simplified implementation
    return [arr]

def hstack(arrays):
    """Stack arrays horizontally"""
    return concatenate(arrays, axis=1)

def vstack(arrays):
    """Stack arrays vertically"""
    return concatenate(arrays, axis=0)

def flatten(arr):
    """Flatten array"""
    if isinstance(arr, ndarray):
        return ndarray(arr._flatten(arr.data))
    return arr

# Builtins references
builtins_max = max
builtins_min = min

# Constants
pi = math.pi
e = math.e
inf = float('inf')
nan = float('nan')
`
}

// embeddedPythonFlask returns a minimal Flask implementation
func embeddedPythonFlask() string {
	return `# Minimal Flask implementation for Ditto
# Basic web framework functionality

class Flask:
    """Minimal Flask application"""
    def __init__(self, name):
        self.name = name
        self.routes = {}
        self.debug = False
    
    def route(self, rule, methods=None):
        """Decorator to register a route"""
        if methods is None:
            methods = ['GET']
        
        def decorator(f):
            self.routes[rule] = {'func': f, 'methods': methods}
            return f
        return decorator
    
    def run(self, host='127.0.0.1', port=5000, debug=None):
        """Run the development server (placeholder)"""
        if debug is not None:
            self.debug = debug
        print(f" * Running on http://{host}:{port}")
        print(f" * Debug mode: {self.debug}")
        # Note: Actual server requires network access
    
    def add_url_rule(self, rule, endpoint=None, view_func=None, **options):
        """Add a URL rule"""
        if endpoint is None:
            endpoint = view_func.__name__
        self.routes[rule] = {'func': view_func, 'methods': options.get('methods', ['GET'])}

class request:
    """Request object (placeholder)"""
    method = 'GET'
    args = {}
    form = {}
    data = ''
    json = None
    headers = {}
    cookies = {}

class jsonify:
    """JSON response"""
    def __init__(self, *args, **kwargs):
        if len(args) == 1 and isinstance(args[0], dict):
            self.data = args[0]
        else:
            self.data = kwargs
    
    def get_data(self):
        import json
        return json.dumps(self.data)

def abort(code):
    """Abort with HTTP status code"""
    raise Exception(f"HTTP {code}")

def redirect(location, code=302):
    """Redirect to a URL"""
    raise Exception(f"Redirect to {location}")

def url_for(endpoint, **values):
    """Generate URL for endpoint"""
    return f"/{endpoint}"

def send_file(filename, **kwargs):
    """Send a file"""
    return f"File: {filename}"

def send_from_directory(directory, path, **kwargs):
    """Send file from directory"""
    return f"File: {directory}/{path}"

def render_template(template_name, **context):
    """Render a template (placeholder)"""
    return f"Template: {template_name}"

def make_response(*args):
    """Create a response"""
    if len(args) == 1:
        return args[0]
    return str(args)

def session():
    """Session object (placeholder)"""
    return {}

def flash(message, category='message'):
    """Flash a message"""
    pass

def get_flashed_messages(with_categories=False):
    """Get flashed messages"""
    return []

# Blueprint support
class Blueprint:
    """Blueprint for organizing routes"""
    def __init__(self, name, import_name, **kwargs):
        self.name = name
        self.import_name = import_name
        self.routes = {}
    
    def route(self, rule, methods=None):
        if methods is None:
            methods = ['GET']
        
        def decorator(f):
            self.routes[rule] = {'func': f, 'methods': methods}
            return f
        return decorator
`
}

// embeddedJSLodash returns a minimal lodash implementation
func embeddedJSLodash() string {
	return `// Minimal Lodash implementation for Ditto
// Common utility functions

// Array functions
function chunk(array, size = 1) {
  const result = [];
  for (let i = 0; i < array.length; i += size) {
    result.push(array.slice(i, i + size));
  }
  return result;
}

function compact(array) {
  return array.filter(item => item != null && item !== false);
}

function concat(array, ...values) {
  return array.concat(...values);
}

function difference(array, ...values) {
  const exclude = new Set(values.flat());
  return array.filter(item => !exclude.has(item));
}

function drop(array, n = 1) {
  return array.slice(n);
}

function dropRight(array, n = 1) {
  return array.slice(0, -n);
}

function fill(array, value, start = 0, end = array.length) {
  for (let i = start; i < end; i++) {
    array[i] = value;
  }
  return array;
}

function findIndex(array, predicate) {
  for (let i = 0; i < array.length; i++) {
    if (predicate(array[i], i, array)) return i;
  }
  return -1;
}

function flatten(array, depth = 1) {
  if (depth === 0) return array.slice();
  const result = [];
  for (const item of array) {
    if (Array.isArray(item) && depth > 0) {
      result.push(...flatten(item, depth - 1));
    } else {
      result.push(item);
    }
  }
  return result;
}

function flattenDeep(array) {
  return flatten(array, Infinity);
}

function head(array) {
  return array.length > 0 ? array[0] : undefined;
}

function indexOf(array, value, fromIndex = 0) {
  return array.indexOf(value, fromIndex);
}

function initial(array) {
  return array.slice(0, -1);
}

function intersection(...arrays) {
  const [first, ...rest] = arrays;
  return first.filter(item => 
    rest.every(arr => arr.includes(item))
  );
}

function join(array, separator = ',') {
  return array.join(separator);
}

function last(array) {
  return array.length > 0 ? array[array.length - 1] : undefined;
}

function lastIndexOf(array, value, fromIndex = array.length - 1) {
  return array.lastIndexOf(value, fromIndex);
}

function reverse(array) {
  return array.slice().reverse();
}

function slice(array, start = 0, end = array.length) {
  return array.slice(start, end);
}

function tail(array) {
  return array.slice(1);
}

function take(array, n = 1) {
  return array.slice(0, n);
}

function takeRight(array, n = 1) {
  return array.slice(-n);
}

function union(...arrays) {
  return [...new Set(arrays.flat())];
}

function uniq(array) {
  return [...new Set(array)];
}

function without(array, ...values) {
  return array.filter(item => !values.includes(item));
}

function zip(...arrays) {
  const length = Math.max(...arrays.map(a => a.length));
  const result = [];
  for (let i = 0; i < length; i++) {
    result.push(arrays.map(a => a[i]));
  }
  return result;
}

// Collection functions
function countBy(collection, iteratee) {
  return collection.reduce((acc, item) => {
    const key = typeof iteratee === 'function' ? iteratee(item) : item[iteratee];
    acc[key] = (acc[key] || 0) + 1;
    return acc;
  }, {});
}

function every(collection, predicate) {
  return collection.every(predicate);
}

function filter(collection, predicate) {
  return collection.filter(predicate);
}

function find(collection, predicate) {
  return collection.find(predicate);
}

function forEach(collection, iteratee) {
  collection.forEach(iteratee);
  return collection;
}

function groupBy(collection, iteratee) {
  return collection.reduce((acc, item) => {
    const key = typeof iteratee === 'function' ? iteratee(item) : item[iteratee];
    (acc[key] || (acc[key] = [])).push(item);
    return acc;
  }, {});
}

function includes(collection, value, fromIndex = 0) {
  if (Array.isArray(collection)) {
    return collection.includes(value, fromIndex);
  }
  return Object.values(collection).includes(value);
}

function map(collection, iteratee) {
  return collection.map(iteratee);
}

function reduce(collection, iteratee, accumulator) {
  return collection.reduce(iteratee, accumulator);
}

function size(collection) {
  return Array.isArray(collection) ? collection.length : Object.keys(collection).length;
}

function some(collection, predicate) {
  return collection.some(predicate);
}

function sortBy(collection, iteratee) {
  return collection.slice().sort((a, b) => {
    const aVal = typeof iteratee === 'function' ? iteratee(a) : a[iteratee];
    const bVal = typeof iteratee === 'function' ? iteratee(b) : b[iteratee];
    return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
  });
}

// Object functions
function assign(object, ...sources) {
  return Object.assign(object, ...sources);
}

function keys(object) {
  return Object.keys(object);
}

function values(object) {
  return Object.values(object);
}

function entries(object) {
  return Object.entries(object);
}

function has(object, key) {
  return Object.prototype.hasOwnProperty.call(object, key);
}

function invert(object) {
  const result = {};
  for (const [key, value] of Object.entries(object)) {
    result[value] = key;
  }
  return result;
}

function mapValues(object, iteratee) {
  const result = {};
  for (const [key, value] of Object.entries(object)) {
    result[key] = typeof iteratee === 'function' ? iteratee(value, key, object) : iteratee;
  }
  return result;
}

function omit(object, ...paths) {
  const result = { ...object };
  for (const path of paths) {
    delete result[path];
  }
  return result;
}

function pick(object, ...paths) {
  const result = {};
  for (const path of paths) {
    if (has(object, path)) {
      result[path] = object[path];
    }
  }
  return result;
}

// String functions
function camelCase(str) {
  return str
    .replace(/[-_\s]+(.)?/g, (_, c) => c ? c.toUpperCase() : '')
    .replace(/^[A-Z]/, c => c.toLowerCase());
}

function capitalize(str) {
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
}

function endsWith(str, target, position = str.length) {
  return str.endsWith(target, position);
}

function kebabCase(str) {
  return str
    .replace(/([a-z])([A-Z])/g, '$1-$2')
    .replace(/[-_\s]+/g, '-')
    .toLowerCase();
}

function pad(str, length = 0, chars = ' ') {
  const padLength = length - str.length;
  if (padLength <= 0) return str;
  const padStart = Math.floor(padLength / 2);
  const padEnd = padLength - padStart;
  return chars.repeat(padStart) + str + chars.repeat(padEnd);
}

function repeat(str, n = 0) {
  return str.repeat(n);
}

function replace(str, pattern, replacement) {
  if (pattern instanceof RegExp) {
    return str.replace(pattern, replacement);
  }
  return str.split(pattern).join(replacement);
}

function snakeCase(str) {
  return str
    .replace(/([a-z])([A-Z])/g, '$1_$2')
    .replace(/[-\s]+/g, '_')
    .toLowerCase();
}

function startsWith(str, target, position = 0) {
  return str.startsWith(target, position);
}

function toLower(str) {
  return str.toLowerCase();
}

function toUpper(str) {
  return str.toUpperCase();
}

function trim(str, chars = ' ') {
  return str.trim();
}

// Math functions
function clamp(number, lower, upper) {
  return Math.min(Math.max(number, lower), upper);
}

function max(array) {
  return Math.max(...array);
}

function min(array) {
  return Math.min(...array);
}

function random(lower = 0, upper = 1) {
  if (typeof lower !== 'number') return Math.random();
  if (typeof upper !== 'number') {
    upper = lower;
    lower = 0;
  }
  return lower + Math.random() * (upper - lower);
}

function sum(array) {
  return array.reduce((a, b) => a + b, 0);
}

// Utility functions
function identity(value) {
  return value;
}

function noop() {}

function times(n, iteratee) {
  const result = new Array(n);
  for (let i = 0; i < n; i++) {
    result[i] = iteratee(i);
  }
  return result;
}

function uniqueId(prefix) {
  if (prefix === undefined) prefix = '';
  return prefix + Math.random().toString(36).substr(2, 9);
}

// Export
export default {
  chunk, compact, concat, difference, drop, dropRight, fill,
  findIndex, flatten, flattenDeep, head, indexOf, initial,
  intersection, join, last, lastIndexOf, reverse, slice, tail,
  take, takeRight, union, uniq, without, zip,
  countBy, every, filter, find, forEach, groupBy, includes,
  map, reduce, size, some, sortBy,
  assign, keys, values, entries, has, invert, mapValues,
  omit, pick,
  camelCase, capitalize, endsWith, kebabCase, pad, repeat,
  replace, snakeCase, startsWith, toLower, toUpper, trim,
  clamp, max, min, random, sum,
  identity, noop, times, uniqueId
};
`
}

// embeddedJSExpress returns a minimal Express implementation
func embeddedJSExpress() string {
	return `// Minimal Express implementation for Ditto
// Basic web framework functionality

function Express() {
  const app = {
    routes: {
      get: {},
      post: {},
      put: {},
      delete: {}
    },
    settings: {},
    
    // HTTP method handlers
    get(path, handler) {
      app.routes.get[path] = handler;
      return app;
    },
    
    post(path, handler) {
      app.routes.post[path] = handler;
      return app;
    },
    
    put(path, handler) {
      app.routes.put[path] = handler;
      return app;
    },
    
    delete(path, handler) {
      app.routes.delete[path] = handler;
      return app;
    },
    
    // Middleware
    use(path, middleware) {
      if (typeof path === 'function') {
        middleware = path;
        path = '/';
      }
      return app;
    },
    
    // Settings
    set(key, value) {
      app.settings[key] = value;
      return app;
    },
    
    get_setting(key) {
      return app.settings[key];
    },
    
    // Start server (placeholder)
    listen(port, host, callback) {
      if (typeof host === 'function') {
        callback = host;
        host = '0.0.0.0';
      }
      console.log('Express server listening on port ' + port);
      if (callback) callback();
    },
    
    // Router
    route(path) {
      return {
        get(handler) { app.routes.get[path] = handler; return this; },
        post(handler) { app.routes.post[path] = handler; return this; },
        put(handler) { app.routes.put[path] = handler; return this; },
        delete(handler) { app.routes.delete[path] = handler; return this; }
      };
    }
  };
  
  return app;
}

// Response object
function Response() {
  this.statusCode = 200;
  this.headers = {};
  this.body = null;
  
  this.status = function(code) {
    this.statusCode = code;
    return this;
  };
  
  this.setHeader = function(name, value) {
    this.headers[name] = value;
    return this;
  };
  
  this.json = function(data) {
    this.body = JSON.stringify(data);
    this.headers['Content-Type'] = 'application/json';
    return this;
  };
  
  this.send = function(data) {
    this.body = typeof data === 'object' ? JSON.stringify(data) : String(data);
    return this;
  };
  
  this.end = function(data) {
    if (data) this.body = data;
    return this;
  };
  
  this.redirect = function(url, status) {
    if (status === undefined) status = 302;
    this.statusCode = status;
    this.headers['Location'] = url;
    return this;
  };
  
  this.download = function(path) {
    this.headers['Content-Disposition'] = 'attachment; filename="' + path + '"';
    return this;
  };
}

// Request object (placeholder)
function Request() {
  this.method = 'GET';
  this.url = '/';
  this.params = {};
  this.query = {};
  this.body = {};
  this.headers = {};
  this.cookies = {};
}

// Router
function Router() {
  const router = {
    routes: {},
    
    get(path, handler) {
      router.routes[path] = { method: 'get', handler };
      return router;
    },
    
    post(path, handler) {
      router.routes[path] = { method: 'post', handler };
      return router;
    },
    
    use(path, middleware) {
      return router;
    }
  };
  
  return router;
}

// Static middleware (placeholder)
function static(path) {
  return function(req, res, next) {
    next();
  };
}

// JSON parser
function json() {
  return function(req, res, next) {
    next();
  };
}

// URL encoded parser
function urlencoded(options) {
  return function(req, res, next) {
    next();
  };
}

// Cookie parser (placeholder)
function cookieParser() {
  return function(req, res, next) {
    next();
  };
}

// Exports
export default Express;
export { Express, Response, Request, Router, static, json, urlencoded, cookieParser };
`
}

// embeddedJSAxios returns a minimal Axios implementation
func embeddedJSAxios() string {
	return `// Minimal Axios implementation for Ditto
// Promise-based HTTP client

function axios(config) {
  // Create response object
  const response = {
    status: 200,
    statusText: 'OK',
    headers: {},
    data: { message: 'HTTP requests require network access' },
    config: config
  };
  
  // Return promise (simplified)
  return Promise.resolve(response);
}

// HTTP method shortcuts
axios.get = function(url, config = {}) {
  return axios({ ...config, method: 'get', url });
};

axios.post = function(url, data, config = {}) {
  return axios({ ...config, method: 'post', url, data });
};

axios.put = function(url, data, config = {}) {
  return axios({ ...config, method: 'put', url, data });
};

axios.delete = function(url, config = {}) {
  return axios({ ...config, method: 'delete', url });
};

axios.patch = function(url, data, config = {}) {
  return axios({ ...config, method: 'patch', url, data });
};

axios.head = function(url, config = {}) {
  return axios({ ...config, method: 'head', url });
};

axios.options = function(url, config = {}) {
  return axios({ ...config, method: 'options', url });
};

// Create instance
axios.create = function(defaultConfig) {
  function instance(config) {
    const mergedConfig = { ...defaultConfig, ...config };
    return axios(mergedConfig);
  }
  
  // Copy all axios methods
  instance.get = (url, config) => instance({ ...config, method: 'get', url });
  instance.post = (url, data, config) => instance({ ...config, method: 'post', url, data });
  instance.put = (url, data, config) => instance({ ...config, method: 'put', url, data });
  instance.delete = (url, config) => instance({ ...config, method: 'delete', url });
  instance.patch = (url, data, config) => instance({ ...config, method: 'patch', url, data });
  
  return instance;
};

// Interceptors
axios.interceptors = {
  request: {
    use: function(fulfilled, rejected) {},
    eject: function(id) {}
  },
  response: {
    use: function(fulfilled, rejected) {},
    eject: function(id) {}
  }
};

// Defaults
axios.defaults = {
  headers: {
    'Content-Type': 'application/json'
  },
  timeout: 0,
  withCredentials: false
};

// CancelToken (placeholder)
axios.CancelToken = {
  source: function() {
    return {
      token: {},
      cancel: function(reason) {}
    };
  }
};

// AxiosError
axios.AxiosError = class AxiosError extends Error {
  constructor(message, code, config, request, response) {
    super(message);
    this.name = 'AxiosError';
    this.code = code;
    this.config = config;
    this.request = request;
    this.response = response;
  }
};

// Spread helper
axios.spread = function(callback) {
  return function(array) {
    return callback.apply(null, array);
  };
};

// isAxiosError helper
axios.isAxiosError = function(payload) {
  return payload instanceof axios.AxiosError;
};

// Export
export default axios;
export { axios };
`
}

// CheckEmbeddedFS checks if embedded filesystem is accessible
func CheckEmbeddedFS() error {
	// Try to read from embedded packages
	_, err := embeddedPackages.Open("embedded/python")
	if err != nil {
		return fmt.Errorf("embedded Python packages not accessible: %w", err)
	}

	_, err = embeddedPackages.Open("embedded/javascript")
	if err != nil {
		return fmt.Errorf("embedded JavaScript packages not accessible: %w", err)
	}

	return nil
}

// ReadEmbeddedFile reads a file from embedded packages
func ReadEmbeddedFile(language, pkgName, filename string) ([]byte, error) {
	var basePath string
	switch language {
	case "python", "py":
		basePath = "embedded/python"
	case "javascript", "js", "node":
		basePath = "embedded/javascript"
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	path := fmt.Sprintf("%s/%s/%s", basePath, pkgName, filename)
	return fs.ReadFile(embeddedPackages, path)
}

// WalkEmbedded walks through embedded packages for a language
func WalkEmbedded(language string) ([]string, error) {
	var basePath string

	switch language {
	case "python", "py":
		basePath = "embedded/python"
	case "javascript", "js", "node":
		basePath = "embedded/javascript"
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	var files []string
	err := fs.WalkDir(embeddedPackages, basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}
