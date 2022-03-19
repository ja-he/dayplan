package main

import (
	"os"
	"strings"
	"time"

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

	// read category styles
	var categoryStyling styling.CategoryStyling
	categoryStyling = *styling.EmptyCategoryStyling()
	styleFilePath := envData.BaseDirPath + "/" + "category-styles.yaml"
	styledInputs, err := styling.ReadCategoryStylingFile(styleFilePath)
	if err != nil {
		panic(err)
	}
	for _, styledInput := range styledInputs {
		categoryStyling.AddStyleFromInput(styledInput)
	}

	stylesheet := styling.Stylesheet{
		Normal: styling.StyleFromHex("#000000", "#ffffff"),

		WeatherRegular: styling.StyleFromHex("#ccebff", "#ffffff"),
		WeatherRainy:   styling.StyleFromHex("#000000", "#ccebff"),
		WeatherSunny:   styling.StyleFromHex("#000000", "#fff0cc"),

		TimelineDay:   styling.StyleFromHex("#f0f0f0", "#ffffff"),
		TimelineNight: styling.StyleFromHex("#f0f0f0", "#000000"),
		TimelineNow:   styling.StyleFromHex("#ffffff", "#ff0000").Bolded(),

		Status: styling.StyleFromHex(("#000000"), "#f0f0f0"),

		CategoryFallback: styling.StyleFromHex("#000000", "#CD5C5C"),

		LogDefault:       styling.StyleFromHex("#000000", "#ffffff"),
		LogTitleBox:      styling.StyleFromHex("#000000", "#f0f0f0").Bolded(),
		LogEntryType:     styling.StyleFromHex("#cccccc", "#ffffff").Italicized(),
		LogEntryLocation: styling.StyleFromHex("#cccccc", "#ffffff"),
		LogEntryTime:     styling.StyleFromHex("#f0f0f0", "#ffffff"),

		Help: styling.StyleFromHex("#000000", "#f0f0f0"),

		Editor: styling.StyleFromHex("#000000", "#f0f0f0"),

		SummaryDefault:  styling.StyleFromHex("#000000", "#ffffff"),
		SummaryTitleBox: styling.StyleFromHex("#000000", "#f0f0f0").Bolded(),
	}

	controller := control.NewController(initialDay, envData, categoryStyling, stylesheet)

	controller.Run()
}
