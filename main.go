package main

import (
	"os"
	"strings"
	"time"

	"github.com/ja-he/dayplan/src/cli"
	"github.com/ja-he/dayplan/src/control"
	"github.com/ja-he/dayplan/src/model"

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

	if cli.Opts.Version {
		cmd := cli.VersionCommand{}
		cmd.Execute([]string{})
	}

	var envData control.EnvData

	// set up dir per option
	dayplanHome := os.Getenv("DAYPLAN_HOME")
	if dayplanHome == "" {
		envData.BaseDirPath = os.Getenv("HOME") + "/.config/dayplan"
	} else {
		envData.BaseDirPath = strings.TrimRight(dayplanHome, "/")
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

	envData.OwmApiKey = os.Getenv("OWM_API_KEY")

	envData.Latitude = os.Getenv("LATITUDE")
	envData.Longitude = os.Getenv("LONGITUDE")

	controller := control.NewController(initialDay, envData)

	controller.Run()
}
