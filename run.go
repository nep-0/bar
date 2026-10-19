package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type recipe struct {
	tools       []string
	compileArgs []string
	runArgs     []string
}

var recipes = map[string][]recipe{
	"Python":     {{tools: []string{"python3"}}, {tools: []string{"python"}}},
	"JavaScript": {{tools: []string{"bun"}}, {tools: []string{"deno"}, runArgs: []string{"run"}}, {tools: []string{"node"}}},
	"Ruby":       {{tools: []string{"ruby"}}},
	"PHP":        {{tools: []string{"php"}}},
	"Perl":       {{tools: []string{"perl"}}},
	"Shell":      {{tools: []string{"bash"}}},
	"Go":         {{tools: []string{"go"}, compileArgs: []string{"build", "-o"}}},
	"Rust":       {{tools: []string{"rustc"}, compileArgs: []string{"-o"}}},
	"C":          {{tools: []string{"cc"}, compileArgs: []string{"-o"}}, {tools: []string{"clang"}, compileArgs: []string{"-o"}}, {tools: []string{"gcc"}, compileArgs: []string{"-o"}}},
	"C++":        {{tools: []string{"c++"}, compileArgs: []string{"-o"}}, {tools: []string{"clang++"}, compileArgs: []string{"-o"}}, {tools: []string{"g++"}, compileArgs: []string{"-o"}}},
	"Java":       {{tools: []string{"javac", "java"}, compileArgs: []string{"-d"}}},
}

func findRecipe(options []recipe, name string) (recipe, bool) {
	for _, option := range options {
		if option.tools[0] == name {
			return option, true
		}
	}
	return recipe{}, false
}

func selectRecipe(language string, cfg config) (recipe, []string, error) {
	options, ok := recipes[language]
	if !ok {
		return recipe{}, nil, fmt.Errorf("unsupported language: %s", language)
	}
	if chain, ok := cfg.Runtimes[language]; ok {
		options = make([]recipe, 0, len(chain))
		for _, name := range chain {
			option, _ := findRecipe(recipes[language], name)
			options = append(options, option)
		}
	}
	var missing []string
	for _, option := range options {
		tools := make([]string, len(option.tools))
		available := true
		for i, name := range option.tools {
			target := name
			if path, ok := cfg.Tools[name]; ok {
				target = path
			}
			path, err := exec.LookPath(target)
			if err != nil {
				if target != name {
					return recipe{}, nil, fmt.Errorf("configured executable %q (%s): %w", target, name, err)
				}
				missing = append(missing, name)
				available = false
				break
			}
			tools[i] = path
		}
		if available {
			return option, tools, nil
		}
	}
	return recipe{}, nil, fmt.Errorf("required executables not found on PATH: %s (language: %s); install %s or set absolute tool paths in bar/config.json", strings.Join(missing, ", "), language, installationSuggestion(language))
}

func installationSuggestion(language string) string {
	switch language {
	case "JavaScript":
		return "Bun, Deno, or Node.js"
	case "Python":
		return "Python 3"
	case "Java":
		return "a JDK (javac and java)"
	case "C", "C++":
		return "a C/C++ compiler toolchain (GCC or Clang)"
	case "Go":
		return "the Go toolchain"
	case "Rust":
		return "Rust (rustc, via rustup)"
	default:
		return language + " (" + recipes[language][0].tools[0] + ")"
	}
}

func runFile(language, source string, args []string, cfg config, stdin io.Reader, stdout, stderr io.Writer) (status int, err error) {
	recipe, tools, err := selectRecipe(language, cfg)
	if err != nil {
		return 1, err
	}
	if len(recipe.compileArgs) == 0 {
		commandArgs := append(append(append([]string{}, recipe.runArgs...), source), args...)
		return runCommand(exec.Command(tools[0], commandArgs...), stdin, stdout, stderr)
	}
	build, err := os.MkdirTemp("", "bar-*")
	if err != nil {
		return 1, fmt.Errorf("create build directory: %w", err)
	}
	cleanupPath := build
	defer func() {
		if cleanupErr := os.RemoveAll(cleanupPath); cleanupErr != nil && status == 0 && err == nil {
			status, err = 1, fmt.Errorf("remove build directory %q: %w", cleanupPath, cleanupErr)
		}
	}()
	build, err = filepath.Abs(build)
	if err != nil {
		return 1, fmt.Errorf("resolve build directory: %w", err)
	}
	executable := filepath.Join(build, "program")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	target := executable
	if language == "Java" {
		target = build
	}
	compileArgs := append(append([]string{}, recipe.compileArgs...), target, source)
	status, err = runCommand(exec.Command(tools[0], compileArgs...), stdin, stdout, stderr)
	if status != 0 || err != nil {
		return status, err
	}
	if language == "Java" {
		class := strings.TrimSuffix(filepath.Base(source), ".java")
		return runCommand(exec.Command(tools[1], append([]string{"-cp", build, class}, args...)...), stdin, stdout, stderr)
	}
	return runCommand(exec.Command(executable, args...), stdin, stdout, stderr)
}

func runCommand(cmd *exec.Cmd, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		status := exitErr.ExitCode()
		if status < 0 {
			status = 1
		}
		return status, nil
	}
	return 1, fmt.Errorf("launch %q: %w", cmd.Path, err)
}
