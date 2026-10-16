// Command iceflow is a local-first personal productivity CLI for tasks,
// reminders and a self-built command dictionary.
package main

import (
	"os"

	"github.com/aice109/ice-flow-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
