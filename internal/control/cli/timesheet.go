package cli

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/ja-he/dayplan/internal/config"
	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/storage"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/util"
)

// TimesheetCommand is the command `timesheet`, which produces a timesheet for
// a given category.
//
// A timesheet has entries per day, each of the form
//
//	<start-time>,<break-duration>,<end-time>
//
// e.g.
//
//	08:50,45min,16:20
type TimesheetCommand struct {
	FromDay string `short:"f" long:"from" description:"the day from which to start summarizing" value-name:"<yyyy-mm-dd>" required:"true"`
	TilDay  string `short:"t" long:"til" description:"the day til which to summarize (inclusive)" value-name:"<yyyy-mm-dd>" required:"true"`

	Category string `long:"category" description:"the category for which to generate the timesheet" value-name:"<category name>" required:"true"`

	IncludeEmpty bool   `long:"include-empty"`
	DateFormat   string `long:"date-format" description:"the date format (see <https://pkg.go.dev/time#pkg-constants>)" default:"2006-01-02"`
	Enquote      bool   `long:"enquote" description:"add quotes around field values"`
	Separator    string `long:"separator" value-name:"<CSV separator (default ',')>" default:","`
}

// Execute executes the timesheet command.
func (command *TimesheetCommand) Execute(args []string) error {
	var envData control.EnvData

	// set up dir per option
	dayplanHome := os.Getenv("DAYPLAN_HOME")
	if dayplanHome == "" {
		envData.BaseDirPath = os.Getenv("HOME") + "/.config/dayplan"
	} else {
		envData.BaseDirPath = strings.TrimRight(dayplanHome, "/")
	}

	// read config from file (for the category priorities)
	yamlData, err := ioutil.ReadFile(envData.BaseDirPath + "/" + "config.yaml")
	if err != nil {
		panic(fmt.Sprintf("can't read config file: '%s'", err))
	}
	configData, err := config.ParseConfigAugmentDefaults(config.Light, yamlData)
	if err != nil {
		panic(fmt.Sprintf("can't parse config data: '%s'", err))
	}
	styledCategories := styling.EmptyCategoryStyling()
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
		style := styling.StyleFromHexSingle(category.Color, false)
		styledCategories.Add(cat, style)
	}

	startDate, err := model.FromString(command.FromDay)
	if err != nil {
		log.Fatalf("from date '%s' invalid", command.FromDay)
	}
	currentDate := startDate
	finalDate, err := model.FromString(command.TilDay)
	if err != nil {
		log.Fatalf("til date '%s' invalid", command.TilDay)
	}

	type dateAndDay struct {
		model.Date
		model.Day
	}

	data := make([]dateAndDay, 0)
	for currentDate != finalDate.Next() {
		fh := storage.NewFileHandler(envData.BaseDirPath + "/days/" + currentDate.ToString())
		categories := make([]model.Category, 0)
		for _, cat := range styledCategories.GetAll() {
			categories = append(categories, cat.Cat)
		}
		data = append(data, dateAndDay{currentDate, *fh.Read(categories)})

		currentDate = currentDate.Next()
	}

	for _, dataEntry := range data {
		timesheetEntry := dataEntry.Day.GetTimesheetEntry(command.Category)

		if !command.IncludeEmpty && timesheetEntry.IsEmpty() {
			continue
		}

		maybeEnquote := func(s string) string {
			if command.Enquote {
				return util.Enquote(s)
			} else {
				return s
			}
		}

		fmt.Println(
			strings.Join(
				[]string{
					maybeEnquote(dataEntry.Date.ToGotime().Format(command.DateFormat)),
					asCSVString(timesheetEntry, maybeEnquote, command.Separator),
				},
				command.Separator,
			),
		)
	}

	return nil
}

// asCSVString returns this TimesheetEntry in CSV format.
func asCSVString(e model.TimesheetEntry, processFieldString func(string) string, separator string) string {
	dur := e.BreakDuration.String()
	if strings.HasSuffix(dur, "m0s") {
		dur = strings.TrimSuffix(dur, "0s")
	}
	return strings.Join(
		[]string{
			processFieldString(e.Start.ToString()),
			processFieldString(dur),
			processFieldString(e.End.ToString()),
		},
		separator,
	)
}
