package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func buildCLI(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "bar")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	return binary
}

func commandStatus(t *testing.T, cmd *exec.Cmd) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	if err == nil {
		return 0, out.String(), errOut.String()
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	return exitErr.ExitCode(), out.String(), errOut.String()
}

func TestExecutableBoundaries(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("required executable absent: python3")
	}
	binary := buildCLI(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "source with spaces.py")
	code := "import json, os, sys\nprint(json.dumps({'args': sys.argv[1:], 'stdin': sys.stdin.read(), 'cwd': os.getcwd()}))\nprint('child-stderr', file=sys.stderr)\nsys.exit(7)\n"
	if err := os.WriteFile(source, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	args := []string{"two words", "--flag", "$(touch should-not-exist)"}
	cmd := exec.Command(binary, append([]string{source, "--"}, args...)...)
	cmd.Dir = cwd
	cmd.Stdin = strings.NewReader("hello\n")
	status, out, errOut := commandStatus(t, cmd)
	var got struct {
		Args       []string
		Stdin, Cwd string
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode %q: %v (stderr %q)", out, err, errOut)
	}
	if status != 7 || !reflect.DeepEqual(got.Args, args) || got.Stdin != "hello\n" || got.Cwd != cwd || errOut != "child-stderr\n" {
		t.Fatalf("status=%d got=%+v stderr=%q", status, got, errOut)
	}
	if _, err := os.Stat(filepath.Join(cwd, "should-not-exist")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("shell side effect: %v", err)
	}
}

func TestExecutableFailures(t *testing.T) {
	binary := buildCLI(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "main.py")
	if err := os.WriteFile(source, []byte("print('must-not-run')\n"), 0600); err != nil {
		t.Fatal(err)
	}
	jsonSource := filepath.Join(dir, "data.json")
	if err := os.WriteFile(jsonSource, []byte("{\"answer\":42}"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, source, want string }{
		{"missing runtime", source, "required executables not found on PATH: python3, python (language: Python); install Python 3"},
		{"missing source", filepath.Join(dir, "missing.py"), "missing.py"},
		{"directory", dir, "not a regular file"},
		{"unsupported", jsonSource, "unsupported language: JSON"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binary, tc.source)
			for _, entry := range os.Environ() {
				if !strings.HasPrefix(entry, "PATH=") {
					cmd.Env = append(cmd.Env, entry)
				}
			}
			cmd.Env = append(cmd.Env, "PATH="+t.TempDir())
			status, out, errOut := commandStatus(t, cmd)
			if status != 1 || out != "" || !strings.HasPrefix(errOut, "bar: ") || !strings.Contains(errOut, tc.want) {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, out, errOut)
			}
		})
	}
	t.Run("unreadable", func(t *testing.T) {
		if err := os.Chmod(source, 0); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(source, 0600)
		if f, err := os.Open(source); err == nil {
			f.Close()
			t.Skip("current user can read permission-000 files")
		}
		status, out, errOut := commandStatus(t, exec.Command(binary, source))
		if status != 1 || out != "" || !strings.Contains(errOut, source) || !strings.HasPrefix(errOut, "bar: ") {
			t.Fatalf("status=%d stdout=%q stderr=%q", status, out, errOut)
		}
	})
}
