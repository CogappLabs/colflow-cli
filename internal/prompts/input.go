package prompts

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Ask reads a line from stdin with a prompt. Returns trimmed input or default if empty.
func Ask(label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return def
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// Confirm asks a yes/no question. Returns true for y/yes, false for n/no, default otherwise.
func Confirm(label string, def bool) bool {
	suffix := "[y/N]"
	if def {
		suffix = "[Y/n]"
	}
	fmt.Printf("%s %s: ", label, suffix)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return def
	}
	line = strings.ToLower(strings.TrimSpace(line))
	if line == "" {
		return def
	}
	return line == "y" || line == "yes"
}
