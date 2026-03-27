// Package stdlib provides embedded standard library implementations
package stdlib

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// PythonStdLib provides embedded Python standard library functions
type PythonStdLib struct {
	builtins map[string]interface{}
	modules  map[string]map[string]interface{}
}

// NewPythonStdLib creates a new Python standard library
func NewPythonStdLib() *PythonStdLib {
	return &PythonStdLib{
		builtins: make(map[string]interface{}),
		modules:  make(map[string]map[string]interface{}),
	}
}

// Init initializes standard library modules
func (p *PythonStdLib) Init() {
	p.initMath()
	p.initOS()
	p.initSys()
	p.initBuiltins()
}

// GetModule returns a module by name
func (p *PythonStdLib) GetModule(name string) map[string]interface{} {
	return p.modules[name]
}

// GetBuiltin returns a builtin function
func (p *PythonStdLib) GetBuiltin(name string) interface{} {
	return p.builtins[name]
}

func (p *PythonStdLib) initMath() {
	p.modules["math"] = map[string]interface{}{
		"sqrt": func(x float64) float64 { return math.Sqrt(x) },
		"pow":  func(x, y float64) float64 { return math.Pow(x, y) },
		"ceil": func(x float64) float64 { return math.Ceil(x) },
		"floor": func(x float64) float64 { return math.Floor(x) },
		"abs":  func(x float64) float64 { return math.Abs(x) },
		"sin":  func(x float64) float64 { return math.Sin(x) },
		"cos":  func(x float64) float64 { return math.Cos(x) },
		"tan":  func(x float64) float64 { return math.Tan(x) },
		"pi":   math.Pi,
		"e":    math.E,
	}
}

func (p *PythonStdLib) initOS() {
	p.modules["os"] = map[string]interface{}{
		"getcwd":   func() string { cwd, _ := os.Getwd(); return cwd },
		"chdir":    func(path string) error { return os.Chdir(path) },
		"mkdir":    func(path string) error { return os.Mkdir(path, 0755) },
		"makedirs": func(path string) error { return os.MkdirAll(path, 0755) },
		"remove":   func(path string) error { return os.Remove(path) },
		"rename":   func(old, new string) error { return os.Rename(old, new) },
		"exists":   func(path string) bool { _, err := os.Stat(path); return err == nil },
		"isfile":   func(path string) bool { info, err := os.Stat(path); return err == nil && !info.IsDir() },
		"isdir":    func(path string) bool { info, err := os.Stat(path); return err == nil && info.IsDir() },
		"listdir":  func(path string) []string { entries, _ := os.ReadDir(path); names := make([]string, len(entries)); for i, e := range entries { names[i] = e.Name() }; return names },
		"getenv":   func(name string) string { return os.Getenv(name) },
		"setenv":   func(name, value string) error { return os.Setenv(name, value) },
		"name":     runtime.GOOS,
		"sep":      string(filepath.Separator),
	}
}

func (p *PythonStdLib) initSys() {
	p.modules["sys"] = map[string]interface{}{
		"version":    runtime.Version(),
		"platform":   runtime.GOOS + "/" + runtime.GOARCH,
		"argv":       []string{},
		"exit":       func(code int) { os.Exit(code) },
		"maxsize":    math.MaxInt,
		"maxunicode": 0x10FFFF,
	}
}

func (p *PythonStdLib) initBuiltins() {
	p.builtins = map[string]interface{}{
		"len":     func(x interface{}) int { return 0 }, // Implemented in VM
		"print":   func(args ...interface{}) {},          // Implemented in VM
		"range":   func(start, end int) []int { r := []int{}; for i := start; i < end; i++ { r = append(r, i) }; return r },
		"str":     func(x interface{}) string { return fmt.Sprintf("%v", x) },
		"int":     func(x interface{}) int { return 0 },  // Implemented in VM
		"float":   func(x interface{}) float64 { return 0 },
		"bool":    func(x interface{}) bool { return x != nil && x != false && x != 0 && x != "" },
		"list":    func() []interface{} { return make([]interface{}, 0) },
		"dict":    func() map[string]interface{} { return make(map[string]interface{}) },
		"set":     func() map[interface{}]bool { return make(map[interface{}]bool) },
		"tuple":   func(args ...interface{}) []interface{} { return args },
		"sum":     func(x []interface{}) float64 { s := 0.0; for _, v := range x { if f, ok := v.(float64); ok { s += f } else if i, ok := v.(int); ok { s += float64(i) } }; return s },
		"min":     func(x ...interface{}) interface{} { if len(x) == 0 { return nil }; m := x[0]; for _, v := range x[1:] { if less(v, m) { m = v } }; return m },
		"max":     func(x ...interface{}) interface{} { if len(x) == 0 { return nil }; m := x[0]; for _, v := range x[1:] { if less(m, v) { m = v } }; return m },
		"abs":     func(x interface{}) interface{} { if i, ok := x.(int); ok { if i < 0 { return -i }; return i }; if f, ok := x.(float64); ok { return math.Abs(f) }; return x },
		"round":   func(x float64, n int) int { return int(math.Round(x)) },
		"sorted":  func(x []interface{}) []interface{} { r := make([]interface{}, len(x)); copy(r, x); sort(r); return r },
		"reversed": func(x []interface{}) []interface{} { r := make([]interface{}, len(x)); for i, v := range x { r[len(x)-1-i] = v }; return r },
		"enumerate": func(x []interface{}) [][2]interface{} { r := make([][2]interface{}, len(x)); for i, v := range x { r[i] = [2]interface{}{i, v} }; return r },
		"zip":     func(args ...[]interface{}) [][]interface{} { if len(args) == 0 { return nil }; min := len(args[0]); for _, a := range args { if len(a) < min { min = len(a) } }; r := make([][]interface{}, min); for i := 0; i < min; i++ { r[i] = make([]interface{}, len(args)); for j, a := range args { r[i][j] = a[i] } }; return r },
		"type":    func(x interface{}) string { return fmt.Sprintf("<class '%T'>", x) },
		"id":      func(x interface{}) int { return 0 }, // Placeholder
		"hash":    func(x interface{}) int { return 0 }, // Placeholder
		"repr":    func(x interface{}) string { return fmt.Sprintf("%v", x) },
		"input":   func(prompt string) string { fmt.Print(prompt); var s string; fmt.Scanln(&s); return s },
		"open":    func(filename, mode string) interface{} { return &File{filename: filename, mode: mode} },
		"exec":    func(code string) error { return fmt.Errorf("exec not supported") },
		"eval":    func(expr string) interface{} { return nil },
	}
}

// File represents an open file
type File struct {
	filename string
	mode     string
	content  string
	pos      int
}

func (f *File) Read(n int) string {
	data, _ := os.ReadFile(f.filename)
	if f.pos >= len(data) {
		return ""
	}
	end := f.pos + n
	if end > len(data) {
		end = len(data)
	}
	result := string(data[f.pos:end])
	f.pos = end
	return result
}

func (f *File) Write(data string) error {
	return os.WriteFile(f.filename, []byte(data), 0644)
}

func (f *File) Close() error {
	return nil
}

func (f *File) Readlines() []string {
	data, _ := os.ReadFile(f.filename)
	return strings.Split(string(data), "\n")
}

func less(a, b interface{}) bool {
	switch av := a.(type) {
	case int:
		if bv, ok := b.(int); ok {
			return av < bv
		}
	case float64:
		if bv, ok := b.(float64); ok {
			return av < bv
		}
	case string:
		if bv, ok := b.(string); ok {
			return av < bv
		}
	}
	return false
}

func sort(x []interface{}) {
	for i := 0; i < len(x)-1; i++ {
		for j := i + 1; j < len(x); j++ {
			if less(x[j], x[i]) {
				x[i], x[j] = x[j], x[i]
			}
		}
	}
}

// NodeStdLib provides embedded Node.js standard library functions
type NodeStdLib struct {
	modules map[string]map[string]interface{}
}

// NewNodeStdLib creates a new Node.js standard library
func NewNodeStdLib() *NodeStdLib {
	n := &NodeStdLib{
		modules: make(map[string]map[string]interface{}),
	}
	n.init()
	return n
}

func (n *NodeStdLib) init() {
	n.modules["fs"] = map[string]interface{}{
		"readFileSync": func(path string) string { data, _ := os.ReadFile(path); return string(data) },
		"writeFileSync": func(path, data string) error { return os.WriteFile(path, []byte(data), 0644) },
		"existsSync":   func(path string) bool { _, err := os.Stat(path); return err == nil },
		"mkdirSync":    func(path string) error { return os.Mkdir(path, 0755) },
		"rmSync":       func(path string) error { return os.Remove(path) },
		"renameSync":   func(old, new string) error { return os.Rename(old, new) },
		"statSync":     func(path string) map[string]interface{} { info, err := os.Stat(path); if err != nil { return nil }; return map[string]interface{}{"isFile": !info.IsDir(), "isDirectory": info.IsDir(), "size": info.Size(), "mtime": info.ModTime().Unix()} },
		"readdirSync":  func(path string) []string { entries, _ := os.ReadDir(path); names := make([]string, len(entries)); for i, e := range entries { names[i] = e.Name() }; return names },
	}

	n.modules["path"] = map[string]interface{}{
		"join":     func(parts ...string) string { return filepath.Join(parts...) },
		"resolve":  func(parts ...string) string { p, _ := filepath.Abs(filepath.Join(parts...)); return p },
		"dirname":  func(p string) string { return filepath.Dir(p) },
		"basename": func(p string) string { return filepath.Base(p) },
		"extname":  func(p string) string { return filepath.Ext(p) },
		"isAbsolute": func(p string) bool { return filepath.IsAbs(p) },
	}

	n.modules["os"] = map[string]interface{}{
		"platform":    func() string { return runtime.GOOS },
		"arch":        func() string { return runtime.GOARCH },
		"homedir":     func() string { d, _ := os.UserHomeDir(); return d },
		"tmpdir":      func() string { return os.TempDir() },
		"hostname":    func() string { h, _ := os.Hostname(); return h },
		"uptime":      func() int64 { return int64(time.Since(time.Unix(0, 0)).Seconds()) },
		"freemem":     func() int64 { return 0 }, // Placeholder
		"totalmem":    func() int64 { return 0 }, // Placeholder
		"cpus":        func() []map[string]interface{} { return []map[string]interface{}{} },
		"networkInterfaces": func() map[string]interface{} { return map[string]interface{}{} },
	}

	n.modules["process"] = map[string]interface{}{
		"argv":       []string{},
		"env":        make(map[string]string),
		"cwd":        func() string { d, _ := os.Getwd(); return d },
		"chdir":      func(d string) error { return os.Chdir(d) },
		"exit":       func(code int) { os.Exit(code) },
		"pid":        os.Getpid(),
		"ppid":       0,
		"version":    runtime.Version(),
		"platform":   runtime.GOOS,
		"arch":       runtime.GOARCH,
		"uptime":     func() int64 { return int64(time.Since(time.Unix(0, 0)).Seconds()) },
	}

	n.modules["util"] = map[string]interface{}{
		"log":       func(args ...interface{}) { fmt.Println(args...) },
		"format":    func(format string, args ...interface{}) string { return fmt.Sprintf(format, args...) },
		"inspect":   func(x interface{}) string { return fmt.Sprintf("%#v", x) },
		"isArray":   func(x interface{}) bool { _, ok := x.([]interface{}); return ok },
		"isFunction": func(x interface{}) bool { _, ok := x.(func(...interface{}) interface{}); return ok },
	}

	n.modules["console"] = map[string]interface{}{
		"log":    func(args ...interface{}) { fmt.Println(args...) },
		"info":   func(args ...interface{}) { fmt.Println(args...) },
		"warn":   func(args ...interface{}) { fmt.Fprintln(os.Stderr, args...) },
		"error":  func(args ...interface{}) { fmt.Fprintln(os.Stderr, args...) },
		"debug":  func(args ...interface{}) { fmt.Println(args...) },
		"time":   func(label string) { /* Timer start */ },
		"timeEnd": func(label string) { /* Timer end */ },
		"trace":  func(args ...interface{}) { /* Stack trace */ },
	}

	n.modules["buffer"] = map[string]interface{}{
		"from": func(data string) []byte { return []byte(data) },
		"alloc": func(size int) []byte { return make([]byte, size) },
	}

	n.modules["events"] = map[string]interface{}{
		"EventEmitter": func() *EventEmitter { return &EventEmitter{handlers: make(map[string][]func(...interface{}))} },
	}

	n.modules["stream"] = map[string]interface{}{
		"Readable":   func() interface{} { return nil },
		"Writable":   func() interface{} { return nil },
		"Transform":  func() interface{} { return nil },
	}

	n.modules["http"] = map[string]interface{}{
		"createServer": func(handler func(interface{}, interface{})) *HTTPServer { return &HTTPServer{handler: handler} },
		"get":          func(url string, callback func(interface{})) { /* HTTP GET */ },
		"request":      func(options interface{}, callback func(interface{})) { /* HTTP request */ },
	}

	n.modules["child_process"] = map[string]interface{}{
		"exec":   func(cmd string, callback func(error, string, string)) { /* Execute command */ },
		"spawn":  func(cmd string, args []string) interface{} { return nil },
		"fork":   func(module string) interface{} { return nil },
	}

	n.modules["crypto"] = map[string]interface{}{
		"createHash": func(algo string) *Hash { return &Hash{algo: algo} },
		"randomBytes": func(size int) []byte { b := make([]byte, size); for i := range b { b[i] = byte(time.Now().UnixNano() % 256) }; return b },
	}

	n.modules["url"] = map[string]interface{}{
		"parse":    func(u string) map[string]string { return map[string]string{"href": u} },
		"format":   func(parts map[string]string) string { return parts["href"] },
		"URL":      func(u string) map[string]string { return map[string]string{"href": u} },
	}

	n.modules["querystring"] = map[string]interface{}{
		"parse":  func(s string) map[string]string { return map[string]string{} },
		"stringify": func(obj map[string]string) string { return "" },
	}

	n.modules["assert"] = map[string]interface{}{
		"ok":       func(x interface{}, msg string) { if x == nil || x == false { panic(msg) } },
		"equal":    func(a, b interface{}, msg string) { if a != b { panic(msg) } },
		"deepEqual": func(a, b interface{}, msg string) { if fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b) { panic(msg) } },
	}

	n.modules["timers"] = map[string]interface{}{
		"setTimeout":  func(fn func(), delay int) int { return 0 },
		"setInterval": func(fn func(), delay int) int { return 0 },
		"clearTimeout": func(id int) {},
		"clearInterval": func(id int) {},
		"setImmediate": func(fn func()) {},
	}

	n.modules["module"] = map[string]interface{}{
		"exports":    make(map[string]interface{}),
		"require":    func(name string) map[string]interface{} { return n.GetModule(name) },
		"filename":   "",
		"dirname":    "",
	}
}

// EventEmitter implements Node.js EventEmitter
type EventEmitter struct {
	handlers map[string][]func(...interface{})
}

func (e *EventEmitter) On(event string, handler func(...interface{})) {
	e.handlers[event] = append(e.handlers[event], handler)
}

func (e *EventEmitter) Emit(event string, args ...interface{}) {
	for _, handler := range e.handlers[event] {
		handler(args...)
	}
}

func (e *EventEmitter) Once(event string, handler func(...interface{})) {
	var wrapper func(...interface{})
	wrapper = func(args ...interface{}) {
		handler(args...)
		e.Off(event, wrapper)
	}
	e.On(event, wrapper)
}

func (e *EventEmitter) Off(event string, handler func(...interface{})) {
	handlers := e.handlers[event]
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			e.handlers[event] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// HTTPServer implements basic HTTP server
type HTTPServer struct {
	handler func(interface{}, interface{})
}

func (s *HTTPServer) Listen(port int, callback func()) {
	// Would start HTTP server
	callback()
}

// Hash implements crypto.Hash
type Hash struct {
	algo string
	data string
}

func (h *Hash) Update(data string) {
	h.data += data
}

func (h *Hash) Digest(encoding string) string {
	// Would compute actual hash
	return fmt.Sprintf("hash(%s, %s)", h.algo, h.data)
}

// GetModule returns a module by name
func (n *NodeStdLib) GetModule(name string) map[string]interface{} {
	return n.modules[name]
}
