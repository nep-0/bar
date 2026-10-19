package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "bar-test-config-*")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_CONFIG_HOME", dir)
	status := m.Run()
	os.RemoveAll(dir)
	os.Exit(status)
}

func TestConfigValidation(t *testing.T) {
	for _, input := range []string{`null`, `{`, `{} {}`, `{"unknown":true}`, `{"tools":{"node":"relative"}}`, `{"tools":{"typo":"/bin/tool"}}`, `{"runtimes":{"JavaScript":[]}}`, `{"runtimes":{"JavaScript":["node","node"]}}`, `{"runtimes":{"JavaScript":["python3"]}}`, `{"runtimes":{"Typo":["node"]}}`} {
		t.Run(input, func(t *testing.T) {
			if _, err := parseConfig([]byte(input)); err == nil {
				t.Fatal("accepted invalid config")
			}
		})
	}
	if _, err := parseConfig([]byte(`{"runtimes":{"JavaScript":["node","bun"]},"tools":{"node":"/opt/node"}}`)); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeSelectionAndShebang(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 needed for executable argument probe")
	}
	binary := buildCLI(t)
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(filepath.Join(cfgDir, "bar"), 0700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(cfgDir, "bar", "config.json")
	source := filepath.Join(dir, "source.js")
	if err := os.WriteFile(source, []byte("console.log('source');"), 0600); err != nil {
		t.Fatal(err)
	}
	probe := func(name string) string {
		path := filepath.Join(bin, name)
		code := "#!" + python + "\nimport json,sys\nprint(json.dumps(sys.argv))\nsys.exit(7)\n"
		if err := os.WriteFile(path, []byte(code), 0700); err != nil {
			t.Fatal(err)
		}
		return path
	}
	deno := probe("deno")
	invoke := func() (int, string, string) {
		cmd := exec.Command(binary, source, "--", "two words", "--flag")
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "PATH=") && !strings.HasPrefix(entry, "XDG_CONFIG_HOME=") {
				cmd.Env = append(cmd.Env, entry)
			}
		}
		cmd.Env = append(cmd.Env, "PATH="+bin, "XDG_CONFIG_HOME="+cfgDir)
		return commandStatus(t, cmd)
	}
	check := func(want []string) {
		t.Helper()
		status, out, errOut := invoke()
		var got []string
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("status=%d stdout=%q stderr=%q", status, out, errOut)
		}
		wantJSON, _ := json.Marshal(want)
		gotJSON, _ := json.Marshal(got)
		if status != 7 || errOut != "" || !bytes.Equal(gotJSON, wantJSON) {
			t.Fatalf("status=%d got=%s want=%s stderr=%q", status, gotJSON, wantJSON, errOut)
		}
	}
	check([]string{deno, "run", source, "two words", "--flag"})
	bun := probe("bun")
	node := probe("node")
	check([]string{bun, source, "two words", "--flag"}) // no retry after exit 7
	writeConfig := func(content string) {
		t.Helper()
		if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeConfig(`{"runtimes":{"JavaScript":["node","bun"]}}`)
	check([]string{node, source, "two words", "--flag"})
	custom := probe("custom node")
	data, _ := json.Marshal(config{Runtimes: map[string][]string{"JavaScript": {"node"}}, Tools: map[string]string{"node": custom}})
	writeConfig(string(data))
	check([]string{custom, source, "two words", "--flag"})
	writeConfig(`{"runtimes":{"JavaScript":["node","bun"]},"tools":{"node":"/does-not-exist/bar-node"}}`)
	status, out, errOut := invoke()
	if status != 1 || out != "" || !strings.Contains(errOut, "configured executable") {
		t.Fatalf("broken override: %d %q %q", status, out, errOut)
	}
	// A shebang bypasses both the configured broken runtime and language detection.
	code := "#!" + python + " -u\nimport json,sys\nprint(json.dumps(sys.argv))\nsys.exit(7)\n"
	if err := os.WriteFile(source, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	check([]string{source, "two words", "--flag"})
	writeConfig(`{"typo":true}`)
	status, out, errOut = invoke()
	if status != 1 || out != "" || !strings.Contains(errOut, configPath) {
		t.Fatalf("invalid config: %d %q %q", status, out, errOut)
	}
	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("console.log('source');"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{bun, deno, node, custom} {
		if err := os.Remove(name); err != nil {
			t.Fatal(err)
		}
	}
	status, out, errOut = invoke()
	if status != 1 || out != "" || !strings.Contains(errOut, "install Bun, Deno, or Node.js") {
		t.Fatalf("suggestion: %d %q %q", status, out, errOut)
	}
}

func TestShebangErrorsAndEnv(t *testing.T) {
	for _, line := range []string{"#!\n", "#!/missing/bar-interpreter\n"} {
		var out, errOut bytes.Buffer
		status, err := runShebang([]byte(line), "/source", nil, nil, &out, &errOut)
		if status != 1 || err == nil || out.Len() != 0 {
			t.Fatalf("status=%d err=%v output=%q", status, err, out.String())
		}
	}
	env, err := exec.LookPath("env")
	if err != nil {
		t.Skip("env absent")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 absent")
	}
	source := filepath.Join(t.TempDir(), "unknown.extension")
	code := "#!" + env + " -S python3 -u\nimport sys\nprint(sys.argv[1])\nsys.exit(7)\n"
	if err := os.WriteFile(source, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	status := runCLI([]string{source, "literal arg"}, nil, &out, &errOut)
	if status != 7 || out.String() != "literal arg\n" || errOut.Len() != 0 {
		t.Fatalf("status=%d out=%q err=%q", status, out.String(), errOut.String())
	}
}
