package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/ja-he/dayplan/src/config"
	"github.com/ja-he/dayplan/src/control"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
)

type TuiCommand struct {
	Day   string `short:"d" long:"day" description:"Specify the day to plan" value-name:"<file>"`
	Theme string `short:"t" long:"theme" choice:"light" choice:"dark" description:"Select a 'dark' or a 'light' default theme (note: only sets defaults, which are individually overridden by settings in config.yaml"`
}

func (command *TuiCommand) Execute(args []string) error {
	var theme config.ColorschemeType
	switch command.Theme {
	case "light":
		theme = config.Light
	case "dark":
		theme = config.Dark
	default:
		theme = config.Dark
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
	var err error
	if command.Day == "" {
		initialDay = model.Date{Year: now.Year(), Month: int(now.Month()), Day: now.Day()}
	} else {
		initialDay, err = model.FromString(command.Day)
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
	configData, err := config.ParseConfigAugmentDefaults(theme, yamlData)
	if err != nil {
		panic(fmt.Sprintf("can't parse config data: '%s'", err))
	}

	// get categories from config
	var categoryStyling styling.CategoryStyling
	categoryStyling = *styling.EmptyCategoryStyling()
	for _, category := range configData.Categories {

		var goal model.Goal
		var err error
		switch {
		case category.Goal.Ranged != nil:
			goal, err = model.NewRangedGoalFromConfig(*category.Goal.Ranged)
		case category.Goal.Workweek != nil:
			goal, err = model.NewWorkweekGoalFromConfig(*category.Goal.Workweek)
		}
		if err != nil {
			return err
		}

		cat := model.Category{
			Name:     category.Name,
			Priority: category.Priority,
			Goal:     goal,
		}
		style := styling.StyleFromHexSingle(category.Color, theme == config.Dark)
		categoryStyling.Add(cat, style)
	}

	stylesheet := styling.NewStylesheetFromConfig(configData.Stylesheet)

	controller := NewController(initialDay, envData, categoryStyling, *stylesheet)

	controller.Run()
	return nil
}
