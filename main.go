package main

import (
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/ja-he/dayplan/src/control/cli"
)

// MAIN
func main() {
	// parse the flags
	parser := flags.NewParser(&cli.Opts, flags.Default)
	parser.SubcommandsOptional = false

	_, err := parser.Parse()
	if flags.WroteHelp(err) {
		os.Exit(0)
	} else if err != nil {
		panic("some flag parsing error occurred")
	}

	if cli.Opts.Version {
		cmd := cli.VersionCommand{}
		cmd.Execute([]string{})
	}
}
