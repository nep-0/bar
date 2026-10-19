# bar — build and run

Run a source file without remembering its interpreter or build command:

```sh
bar hello.py
bar main.go
bar hello.cpp
```

Bar uses tools already installed on your machine. If a tool is missing, it suggests what to install; it never installs anything itself.

## Install

With Go installed:

```sh
go install github.com/nep-0/bar@latest
```

Make sure your Go binary directory is on `PATH`. To install from a local checkout, use `go install .`. To build a local executable instead:

```sh
go build -o bar .
```

## Usage

```sh
bar <file> [--] [args...]
bar --help
```

Everything after the filename is passed to your program, including flags:

```sh
bar script.py --verbose "two words"
```

Your program uses the current working directory, environment, and terminal input/output. Bar preserves its exit status and adds no banners.

### When do I need `--`?

**Before the filename:** use `--` if the filename begins with `-`, so bar does not treat it as an option:

```sh
bar -- -script.py
# Or avoid the separator by using a path:
bar ./-script.py
```

**After the filename:** `--` is optional. Bar has already stopped parsing its own options, so these are equivalent:

```sh
bar script.py --verbose
bar script.py -- --verbose
```

Bar removes only the first `--` immediately after the filename. To pass a literal `--` as your program's first argument:

```sh
bar script.py -- --
```

## Supported languages

Without a shebang, bar detects the language from the filename and contents. It uses the first available tool in the order below.

| Language | Tools, in preference order |
| --- | --- |
| Python | `python3`, `python` |
| JavaScript | `bun`, `deno`, `node` |
| Ruby | `ruby` |
| PHP | `php` |
| Perl | `perl` |
| Shell | `bash` |
| Go | `go` |
| Rust | `rustc` |
| C | `cc`, `clang`, `gcc` |
| C++ | `c++`, `clang++`, `g++` |
| Java | Both `javac` and `java` |

Fallbacks are for missing tools, not failed programs. Once a tool starts, bar does not retry with another one. Deno runs with its normal permission restrictions; bar does not grant extra permissions.

Compiled programs are built in a temporary directory and removed afterward, not written beside your source.

### Single-file limits

- Go builds the named file, not its whole package.
- Java needs a default-package class matching the filename, such as `Main` in `Main.java`.
- Compiler defaults and your existing environment apply. Bar does not discover projects or manage dependencies.
- Unknown, ambiguous, binary, and unsupported files produce an error rather than a guess.

## Shebangs

If the file starts with `#!`, bar follows that command instead of detecting a language or choosing a configured runtime:

```python
#!/usr/bin/env python3
print("hello")
```

The source file does not need executable permission. Shebangs can also select languages outside the table above.

Interpreter options work too, for example `#!/usr/bin/python3 -u`. For multiple options through `env`, use `#!/usr/bin/env -S python3 -u`; bar passes the rest of the shebang as one argument, leaving splitting to `env`.

A missing interpreter is an error—bar does not fall back or rewrite the shebang using configured tool paths. Only run files you trust: their shebang chooses an executable to run.

## Configuration

Configuration is optional. Create a per-user `config.json` to change tool preferences or use tools outside `PATH`:

| Platform | Config location |
| --- | --- |
| Linux | `$XDG_CONFIG_HOME/bar/config.json`, or `~/.config/bar/config.json` if unset |
| macOS | `~/Library/Application Support/bar/config.json` |
| Windows | `%AppData%/bar/config.json` |

The location is independent of where bar is installed. Bar does not read project-local or system-wide config files.

```json
{
  "runtimes": {
    "JavaScript": ["node", "bun", "deno"],
    "C": ["clang", "gcc"]
  },
  "tools": {
    "node": "/opt/node/bin/node",
    "clang": "/opt/llvm/bin/clang"
  }
}
```

- **`runtimes`** replaces a language's preference order. Use language and tool names from the table above; for Java, use `["javac"]`. Languages you omit keep their defaults.
- **`tools`** assigns an absolute path to an interpreter or compiler. You can set `javac` and `java` separately. Paths may contain spaces, but `~` and environment variables are not expanded.
- **Missing config:** use defaults. Bar does not create a file automatically.
- **Invalid or unreadable config:** stop with an error naming the file, before running anything. Unknown names, empty or duplicate runtime lists, and relative tool paths are invalid.
- **Broken configured tool path:** stop when that tool is selected, rather than silently choosing another tool.

## Development

```sh
go test -v ./...
```

Real-toolchain tests skip when the required tools are absent. Linux execution has been verified with Python, Bun, Node.js, Ruby, Perl, Bash, Go, C, and C++. Real Deno, PHP, Rust, Java compilation, and Windows execution remain unverified locally.
