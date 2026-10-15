package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// confirm asks a yes/no question when stdin is a terminal, defaulting to no.
// It returns false without prompting when stdin is not interactive.
func confirm(prompt string) bool {
	if !isTerminal(os.Stdin) {
		return false
	}
	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}
