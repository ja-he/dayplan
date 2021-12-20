package main

import (
	"os"
	"strings"
	"time"

	"github.com/ja-he/dayplan/src/cli"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/program"
	"github.com/ja-he/dayplan/src/tui"

	"github.com/jessevdk/go-flags"
)

// MAIN
func main() {
	// parse the flags
	parser := flags.NewParser(&cli.Opts, flags.Default)
	parser.SubcommandsOptional = true

	_, err := parser.Parse()
	if flags.WroteHelp(err) {
		os.Exit(0)
	} else if err != nil {
		panic("some flag parsing error occurred")
	}

	var programData program.Data

	// set up dir per option
	dayplanHome := os.Getenv("DAYPLAN_HOME")
	if dayplanHome == "" {
		programData.BaseDirPath = os.Getenv("HOME") + "/.config/dayplan"
	} else {
		programData.BaseDirPath = strings.TrimRight(dayplanHome, "/")
	}

	// infer initial day either from input file or current date
	now := time.Now()
	var initialDay model.Date
	if cli.Opts.Day == "" {
		initialDay = model.Date{Year: now.Year(), Month: int(now.Month()), Day: now.Day()}
	} else {
		initialDay, err = model.FromString(cli.Opts.Day)
		if err != nil {
			panic(err) // TODO
		}
	}

	programData.OwmApiKey = os.Getenv("OWM_API_KEY")

	programData.Latitude = os.Getenv("LATITUDE")
	programData.Longitude = os.Getenv("LONGITUDE")

	controller := tui.NewTUIController(initialDay, programData)

	controller.Run()
}
