package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/ja-he/dayplan/src/config"
	"github.com/ja-he/dayplan/src/control"
	"github.com/ja-he/dayplan/src/control/cli"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"

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

	// read config from file
	yamlData, err := ioutil.ReadFile(envData.BaseDirPath + "/" + "config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: can't read config file: '%s'.\n", err)
		fmt.Fprintf(os.Stderr, "         using defaults.\n")
		yamlData = make([]byte, 0)
	}
	configData, err := config.ParseConfigAugmentDefaults(yamlData)
	if err != nil {
		panic(fmt.Sprintf("can't parse config data: '%s'", err))
	}

	// get categories from config
	var categoryStyling styling.CategoryStyling
	categoryStyling = *styling.EmptyCategoryStyling()
	for _, category := range configData.Categories {
		categoryStyling.AddStyleFromInput(category)
	}

	stylesheet := styling.NewStylesheetFromConfig(configData.Stylesheet)

	controller := control.NewController(initialDay, envData, categoryStyling, *stylesheet)

	controller.Run()
}
