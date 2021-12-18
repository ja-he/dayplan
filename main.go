package main

import (
	"os"
	"strings"
	"time"

	"dayplan/src/model"
	"dayplan/src/program"
	"dayplan/src/tui"

	"github.com/jessevdk/go-flags"
)

var commandLineOpts struct {
	Day string `short:"d" long:"day" description:"Specify the day to plan" value-name:"<file>"`
}

// MAIN
func main() {
	// parse the flags
	_, err := flags.Parse(&commandLineOpts)
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
	if commandLineOpts.Day == "" {
		initialDay = model.Date{Year: now.Year(), Month: int(now.Month()), Day: now.Day()}
	} else {
		initialDay, err = model.FromString(commandLineOpts.Day)
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
