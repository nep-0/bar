package main

import "testing"

func TestDetectLanguage(t *testing.T) {
	for _, tc := range []struct{ name, filename, content, want, err string }{
		{name: "shebang", filename: "script", content: "#!/usr/bin/env python3\nprint('hello')\n", want: "Python"},
		{name: "unknown", filename: "unrecognized", content: "", err: "could not detect a programming language"},
		{name: "binary", filename: "code.py", content: "\x00\x01\x02", err: "binary files cannot be run"},
		{name: "unsupported", filename: "data.json", content: "{\"answer\":42}", want: "JSON"},
		{name: "empty recognized", filename: "empty.py", want: "Python"},
		{name: "ambiguous", filename: "main.pl", content: "print \"recipe-ok:$ARGV[0]\\n\"; exit 7;", err: "ambiguous language: Perl, Prolog, Raku"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := detectLanguage(tc.filename, []byte(tc.content))
			if tc.err != "" {
				if err == nil || err.Error() != tc.err {
					t.Fatalf("got %q, %v", got, err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
