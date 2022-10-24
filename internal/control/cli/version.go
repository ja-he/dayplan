package cli

import (
	"fmt"
	"os"
)

// For proper builds, these variables should be set via ldflags.
var version = "development"
var hash = "unknown"

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
	fmt.Printf("%s (%s)\n", version, hash)
	os.Exit(0)
}
