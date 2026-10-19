package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRealToolchains(t *testing.T) {
	for _, tc := range []struct {
		language, name, code string
		tools                []string
	}{
		{"Python", "main.py", "import sys\nprint('recipe-ok:' + sys.argv[1])\nsys.exit(7)\n", []string{"python3"}},
		{"JavaScript", "main.js", "console.log('recipe-ok:' + process.argv[2]); process.exit(7);", []string{"node"}},
		{"Ruby", "main.rb", "puts 'recipe-ok:' + ARGV[0]\nexit 7\n", []string{"ruby"}},
		{"PHP", "main.php", "<?php echo 'recipe-ok:' . $argv[1] . \"\\n\"; exit(7);", []string{"php"}},
		{"Perl", "main.pl", "#!/usr/bin/env perl\n# -*- mode: perl -*-\nprint \"recipe-ok:$ARGV[0]\\n\"; exit 7;", []string{"perl"}},
		{"Shell", "main.sh", "printf 'recipe-ok:%s\\n' \"$1\"\nexit 7\n", []string{"bash"}},
		{"Go", "main.go", "package main\nimport (\"fmt\"; \"os\")\nfunc main(){fmt.Println(\"recipe-ok:\"+os.Args[1]);os.Exit(7)}", []string{"go"}},
		{"Rust", "main.rs", "fn main(){println!(\"recipe-ok:{}\",std::env::args().nth(1).unwrap());std::process::exit(7);}", []string{"rustc"}},
		{"C", "main.c", "#include <stdio.h>\nint main(int argc,char **argv){printf(\"recipe-ok:%s\\n\",argv[1]);return 7;}\n", []string{"cc"}},
		{"C++", "main.cpp", "#include <iostream>\nint main(int argc,char **argv){std::cout << \"recipe-ok:\" << argv[1] << std::endl;return 7;}\n", []string{"c++"}},
		{"Java", "Main.java", "public class Main { public static void main(String[] args) { System.out.println(\"recipe-ok:\" + args[0]); System.exit(7); } }", []string{"javac", "java"}},
	} {
		t.Run(tc.language, func(t *testing.T) {
			for _, tool := range tc.tools {
				if _, err := exec.LookPath(tool); err != nil {
					t.Skipf("required executable absent: %s", tool)
				}
			}
			dir := t.TempDir()
			source := filepath.Join(dir, tc.name)
			if err := os.WriteFile(source, []byte(tc.code), 0600); err != nil {
				t.Fatal(err)
			}
			buildRoot := t.TempDir()
			t.Setenv("TMPDIR", buildRoot)
			t.Setenv("TMP", buildRoot)
			t.Setenv("TEMP", buildRoot)
			var out, errOut bytes.Buffer
			status := runCLI([]string{source, "--", "two words"}, strings.NewReader(""), &out, &errOut)
			if status != 7 || out.String() != "recipe-ok:two words\n" || errOut.Len() != 0 {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, out.String(), errOut.String())
			}
			assertEmpty(t, buildRoot)
		})
	}
}

func assertEmpty(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("artifacts left in %s: %v", dir, entries)
	}
}

func TestCompileFailureCleanup(t *testing.T) {
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("required executable absent: cc")
	}
	source := filepath.Join(t.TempDir(), "invalid.c")
	if err := os.WriteFile(source, []byte("#error deliberate compile failure\n"), 0600); err != nil {
		t.Fatal(err)
	}
	buildRoot := t.TempDir()
	t.Setenv("TMPDIR", buildRoot)
	var out, errOut bytes.Buffer
	status := runCLI([]string{source}, nil, &out, &errOut)
	if status == 0 || out.Len() != 0 || !strings.Contains(errOut.String(), "deliberate compile failure") || strings.Contains(errOut.String(), "bar:") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, out.String(), errOut.String())
	}
	assertEmpty(t, buildRoot)
}

func TestCommandLaunchFailure(t *testing.T) {
	var out, errOut bytes.Buffer
	name := filepath.Join(t.TempDir(), "missing-command")
	status, err := runCommand(exec.Command(name), nil, &out, &errOut)
	if status != 1 || err == nil || !strings.Contains(err.Error(), name) {
		t.Fatalf("status=%d err=%v", status, err)
	}
}

func TestSignalStatus(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("required executable absent: bash")
	}
	status, err := runCommand(exec.Command(bash, "-c", "kill -TERM $$"), nil, &bytes.Buffer{}, &bytes.Buffer{})
	if status != 1 || err != nil {
		t.Fatalf("status=%d err=%v", status, err)
	}
}
