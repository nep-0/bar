package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Match kernel-style shebang parsing: interpreter plus one optional argument.
// In particular, env -S receives its whole argument and performs its own splitting.
func runShebang(content []byte, source string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	line, _, _ := bytes.Cut(content, []byte{'\n'})
	command := strings.TrimSpace(string(line[2:]))
	split := strings.IndexAny(command, " \t")
	interpreter := command
	var commandArgs []string
	if split >= 0 {
		interpreter = command[:split]
		if argument := strings.TrimSpace(command[split:]); argument != "" {
			commandArgs = append(commandArgs, argument)
		}
	}
	if interpreter == "" {
		return 1, fmt.Errorf("empty shebang in %q", source)
	}
	path, err := exec.LookPath(interpreter)
	if err != nil {
		return 1, fmt.Errorf("shebang interpreter %q unavailable: %w; install the requested interpreter or correct the shebang", interpreter, err)
	}
	commandArgs = append(commandArgs, source)
	commandArgs = append(commandArgs, args...)
	return runCommand(exec.Command(path, commandArgs...), stdin, stdout, stderr)
}
