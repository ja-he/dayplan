package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/control/cli"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	var opts cli.DayplanIntelServerCommandLineOpts
	parser := flags.NewParser(&opts, flags.Default)
	parser.SubcommandsOptional = false

	_, err := parser.Parse()
	if flags.WroteHelp(err) {
		os.Exit(0)
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "fatal error (e.g. flag parsing):\n > %s\n", err.Error())
		os.Exit(1)
	}
}
