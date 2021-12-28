package cli

import (
	"fmt"
	"os"
)

const version = "0.1.1"

// Flags for the `version` command line command, for `go-flags` to parse
// command line args into.
type VersionCommand struct {
}

// Executes the version command.
// (This gets called by `go-flags` when `version` is provided on the command
// line)
func (command *VersionCommand) Execute(args []string) error {
	showVersion()
	return nil
}

func showVersion() {
	fmt.Println(version)
	os.Exit(0)
}
