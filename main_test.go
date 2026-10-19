package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseInput(t *testing.T) {
	for _, tc := range []struct {
		name   string
		args   []string
		file   string
		child  []string
		status int
		help   bool
	}{
		{name: "missing", status: 2},
		{name: "unknown option", args: []string{"--unknown"}, status: 2},
		{name: "help", args: []string{"--help"}, help: true},
		{name: "short help", args: []string{"-h"}, help: true},
		{name: "arguments", args: []string{"a.py", "--", "two words", "--flag", "--"}, file: "a.py", child: []string{"two words", "--flag", "--"}},
		{name: "leading separator", args: []string{"--", "-a.py", "-h"}, file: "-a.py", child: []string{"-h"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			file, child, status := parseInput(tc.args, &out, &errOut)
			if file != tc.file || !reflect.DeepEqual(child, tc.child) || status != tc.status {
				t.Fatalf("got %q %q %d", file, child, status)
			}
			if tc.help && (!strings.Contains(out.String(), usage) || !strings.Contains(out.String(), supportedLanguages) || errOut.Len() != 0) {
				t.Fatalf("help streams: %q %q", out.String(), errOut.String())
			}
			if status == 2 && !strings.Contains(errOut.String(), usage) {
				t.Fatalf("missing usage: %q", errOut.String())
			}
		})
	}
}

func TestReadSource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "empty.py")
	if err := os.WriteFile(source, nil, 0600); err != nil {
		t.Fatal(err)
	}
	path, content, err := readSource(source)
	if err != nil || path != source || len(content) != 0 {
		t.Fatalf("read: %q %q %v", path, content, err)
	}
	link := filepath.Join(dir, "link.py")
	if err := os.Symlink(source, link); err == nil {
		if _, _, err := readSource(link); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{dir, filepath.Join(dir, "missing.py")} {
		if _, _, err := readSource(path); err == nil || !strings.Contains(err.Error(), path) {
			t.Fatalf("expected named error for %q: %v", path, err)
		}
	}
}
