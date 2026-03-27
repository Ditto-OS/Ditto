# Ditto

**The Universal Translator**

A single, portable ~10MB binary that runs code from any language without requiring users to install runtimes.

## The Magic

```
Zero-Config Polyglot Execution

✅ No Python installation needed
✅ No Node.js installation needed  
✅ No Ruby installation needed
✅ No Go toolchain needed
✅ No external dependencies

Just Ditto — a single binary with embedded interpreters.
```

## Quick Start

### Run Code Directly

```bash
# Smart detection - just run it
Ditto run script.py           # Auto-detects Python
Ditto run app.js              # Auto-detects JavaScript
Ditto run program.lua         # Auto-detects Lua
Ditto run query.sql           # Auto-detects SQL
Ditto run main.c              # Auto-detects C
```

### Create Standalone Executables

```bash
# Bundle into a single binary
Ditto bundle script.py -o myapp
Ditto bundle app.js -o myapp
```

### List Supported Languages

```bash
Ditto languages
```

## Commands

| Command | Description |
|---------|-------------|
| `run <file>` | Run code with smart language detection |
| `bundle <file> -o <name>` | Create standalone executable |
| `languages` | List all supported languages |
| `version` | Show version |
| `help` | Show help |

## Supported Languages

| Language | Extension | Interpreter | Features |
|----------|-----------|-------------|----------|
| Python | `.py` | Pure Go VM | print, import, classes, math, os, sys |
| JavaScript | `.js`, `.ts` | Pure Go VM | console, require, async/await, fs, path |
| Lua | `.lua` | Pure Go VM | print, tables, functions, loops |
| SQL | `.sql` | Pure Go SQLite | CREATE, INSERT, SELECT, JOIN, WHERE |
| C/C++ | `.c`, `.cpp` | Pure Go VM | printf, variables, loops |
| Ruby | `.rb` | (Planned) | 🔜 Coming |
| Go | `.go` | (Planned) | 🔜 Coming |

## Feature Matrix

| Feature | Status | Implementation |
|---------|--------|----------------|
| `import os` (Python) | ✅ | Embedded stdlib |
| `import math` (Python) | ✅ | Embedded stdlib |
| `from os import getcwd` | ✅ | Embedded stdlib |
| Classes (Python) | ✅ | Class definitions supported |
| `require('fs')` (Node) | ✅ | Embedded stdlib |
| `require('path')` (Node) | ✅ | Embedded stdlib |
| `async/await` | ✅ | Syntax support |
| `console.log` | ✅ | Embedded console |
| Complex SQL JOINs | ✅ | Full JOIN support |
| SQL WHERE clauses | ✅ | Comparison operators |
| SQL INSERT/UPDATE/DELETE | ✅ | Full CRUD |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Ditto Binary (~10MB)                     │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Embedded Interpreters                   │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  • Python VM    (670 lines pure Go)                 │   │
│  │    - Standard library: math, os, sys                │   │
│  │    - Class definitions                              │   │
│  │    - Import statements                              │   │
│  │  • JavaScript VM (600 lines pure Go)                │   │
│  │    - Node.js stdlib: fs, path, os, console          │   │
│  │    - require() support                              │   │
│  │    - async/await syntax                             │   │
│  │  • Lua VM       (470 lines pure Go)                 │   │
│  │  • SQL Engine   (560 lines pure Go)                 │   │
│  │    - JOIN queries                                   │   │
│  │    - WHERE clauses                                  │   │
│  │  • C VM         (320 lines pure Go)                 │   │
│  └─────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │         Standard Library (400 lines pure Go)         │   │
│  │  • Python: math, os, sys builtins                   │   │
│  │  • Node.js: fs, path, os, console, events           │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Project Structure

```
Ditto/
├── cmd/ditto/              # CLI entry point
├── internal/
│   ├── config/             # Configuration
│   ├── interpreter/        # Embedded interpreters
│   │   ├── interpreter.go  # Engine (110 lines)
│   │   ├── python.go       # Python VM (670 lines)
│   │   ├── javascript.go   # JS VM (600 lines)
│   │   ├── lua.go          # Lua VM (470 lines)
│   │   ├── sql.go          # SQL engine (560 lines)
│   │   └── c.go            # C VM (320 lines)
│   ├── stdlib/             # Standard libraries
│   │   └── stdlib.go       # Python + Node stdlib (400 lines)
│   └── runtime/            # WASM runtime management
├── pkg/
│   ├── archive/            # Bundle creation
│   ├── bundler/            # Standalone creator
│   └── runner/             # Execution engine
├── examples/               # Test scripts
├── scripts/                # Build scripts
└── go.mod
```

**Total: ~3,500 lines of pure Go code**

## Building

```bash
# Windows
.\scripts\build.ps1

# Linux/macOS
./scripts/build.sh

# Manual
go build -o Ditto.exe ./cmd/ditto
```

## Examples

### Python with Standard Library

```python
import math
import os

print("sqrt(16) =", math.sqrt(16))
print("Current dir:", os.getcwd())

class Person:
    def __init__(self, name):
        self.name = name

p = Person("Alice")
```

### JavaScript with require()

```javascript
const fs = require('fs');
const path = require('path');

console.log("Current dir:", path.resolve('.'));

async function fetchData() {
    await Promise.resolve("data");
    console.log("Done!");
}

fetchData();
```

### SQL with JOINs

```sql
CREATE TABLE users (id INTEGER, name TEXT);
CREATE TABLE orders (id INTEGER, user_id INTEGER, product TEXT);

SELECT users.name, orders.product
FROM users
JOIN orders ON users.id = orders.user_id;
```

## Comparison

| Tool | Size | Languages | Dependencies |
|------|------|-----------|--------------|
| **Ditto** | ~10MB | 5+ | None |
| pyinstaller | ~50MB | Python only | Python |
| pkg | ~80MB | Node.js only | Node.js |
| Docker | ~500MB+ | Any | Docker Desktop |

## What's Embedded

### Python Standard Library
- **math**: sqrt, pow, ceil, floor, abs, sin, cos, tan, pi, e
- **os**: getcwd, chdir, mkdir, remove, exists, getenv, name, sep
- **sys**: version, platform, argv, exit, maxsize

### Node.js Standard Library
- **fs**: readFileSync, writeFileSync, existsSync, mkdirSync
- **path**: join, resolve, dirname, basename, extname
- **os**: platform, arch, homedir, tmpdir, hostname
- **console**: log, info, warn, error, debug
- **process**: argv, env, cwd, exit, pid, version

## License

MIT

## Contributing

Contributions welcome! Areas for improvement:
- Complete Python class method support
- Full async/await runtime
- More Node.js modules (http, crypto, events)
- Ruby interpreter
- Go interpreter
