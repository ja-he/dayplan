package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/control/cli"
	"github.com/ja-he/dayplan/internal/potatolog"
)

// MAIN
func main() {
	// Set up the global logger with a switchable writer, initially targeting stderr.
	// Subcommands (such as tui) can switch the target without breaking loggers
	// that were created before the switch.
	potatolog.SwitchToTTY(os.Stderr)
	log.Logger = log.Output(potatolog.GlobalLogWriter)

	// parse the flags
	parser := flags.NewParser(&cli.DayplanOpts, flags.Default)
	parser.SubcommandsOptional = false

	_, err := parser.Parse()
	if flags.WroteHelp(err) {
		os.Exit(0)
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "fatal error (e.g. flag parsing):\n > %s\n", err.Error())
		os.Exit(1)
	}

	if cli.DayplanOpts.Version {
		cmd := cli.VersionCommand{}
		err := cmd.Execute([]string{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "exited with error:\n > %s\n", err.Error())
			os.Exit(1)
		}
	}
}
