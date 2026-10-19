package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const usage = "Usage: bar <file> [--] [args...]"
const supportedLanguages = "Supported languages: Python, JavaScript, Ruby, PHP, Perl, Shell, Go, Rust, C, C++, Java"

func parseInput(args []string, stdout, stderr io.Writer) (string, []string, int) {
	flags := flag.NewFlagSet("bar", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { fmt.Fprintln(stderr, usage) }
	// Handle help ourselves so flag's ErrHelp path writes to stdout.
	flags.Bool("h", false, "show help")
	flags.Bool("help", false, "show help")
	if err := flags.Parse(args); err != nil {
		return "", nil, 2
	}
	if flags.Lookup("h").Value.String() == "true" || flags.Lookup("help").Value.String() == "true" {
		fmt.Fprintln(stdout, usage)
		fmt.Fprintln(stdout, supportedLanguages)
		return "", nil, 0
	}
	operands := flags.Args()
	if len(operands) == 0 {
		fmt.Fprintln(stderr, usage)
		return "", nil, 2
	}
	childArgs := operands[1:]
	if len(childArgs) > 0 && childArgs[0] == "--" {
		childArgs = childArgs[1:]
	}
	return operands[0], childArgs, 0
}

func readSource(name string) (string, []byte, error) {
	source, err := filepath.Abs(name)
	if err != nil {
		return "", nil, fmt.Errorf("resolve source %q: %w", name, err)
	}
	file, err := os.Open(source)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", nil, err
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("source %q is not a regular file", source)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return "", nil, fmt.Errorf("read source %q: %w", source, err)
	}
	return source, content, nil
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func runCLI(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	name, childArgs, status := parseInput(args, stdout, stderr)
	if name == "" {
		return status
	}
	source, content, err := readSource(name)
	if err == nil {
		var cfg config
		cfg, err = loadConfig()
		if err == nil {
			if bytes.HasPrefix(content, []byte("#!")) {
				status, err = runShebang(content, source, childArgs, stdin, stdout, stderr)
			} else {
				var language string
				language, err = detectLanguage(source, content)
				if err == nil {
					status, err = runFile(language, source, childArgs, cfg, stdin, stdout, stderr)
				}
			}
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "bar: %v\n", err)
		return 1
	}
	return status
}
