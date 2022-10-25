package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/ja-he/dayplan/internal/control/cli"
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
		fmt.Fprintf(os.Stderr, "flag parsing error:\n > %s\n", err.Error())
		os.Exit(1)
	}

	if cli.Opts.Version {
		cmd := cli.VersionCommand{}
		err := cmd.Execute([]string{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "exited with error:\n > %s\n", err.Error())
			os.Exit(1)
		}
	}
}
