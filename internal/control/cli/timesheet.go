package cli

import (
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/ja-he/dayplan/internal/config"
	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/storage/providers"
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

	CategoryIncludeFilter string `long:"category-include-filter" short:"i" description:"the category filter include regex for which to generate the timesheet (empty value is ignored)" value-name:"<regex>"`
	CategoryExcludeFilter string `long:"category-exclude-filter" short:"e" description:"the category filter exclude regex for which to generate the timesheet (empty value is ignored)" value-name:"<regex>"`

	IncludeEmpty   bool   `long:"include-empty"`
	DateFormat     string `long:"date-format" value-name:"<format>" description:"specify the date format (see <https://pkg.go.dev/time#pkg-constants>)" default:"2006-01-02"`
	Enquote        bool   `long:"enquote" description:"add quotes around field values"`
	FieldSeparator string `long:"field-separator" value-name:"<CSV separator (default ',')>" default:","`
	DurationFormat string `long:"duration-format" option:"golang" option:"colon-delimited" default:"golang"`
}

// Execute executes the timesheet command.
func (command *TimesheetCommand) Execute(args []string) error {
	if command.CategoryIncludeFilter == "" && command.CategoryExcludeFilter == "" {
		return fmt.Errorf("at least one of '--category-include-filter'/'-i' and '--category-exclude-filter'/'-e' is required")
	}

	var envData control.EnvData

	// set up dir per option
	dayplanHome := os.Getenv("DAYPLAN_HOME")
	if dayplanHome == "" {
		envData.BaseDirPath = os.Getenv("HOME") + "/.config/dayplan"
	} else {
		envData.BaseDirPath = strings.TrimRight(dayplanHome, "/")
	}

	// read config from file (for the category priorities)
	yamlData, err := os.ReadFile(envData.BaseDirPath + "/" + "config.yaml")
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
		model.EventList
	}

	data := make([]dateAndDay, 0)
	for currentDate != finalDate.Next() {
		fh := providers.NewFileHandler(path.Join(envData.BaseDirPath+"days"), currentDate)
		categories := make([]model.Category, 0)
		for _, cat := range styledCategories.GetAll() {
			categories = append(categories, cat.Cat)
		}
		data = append(data, dateAndDay{currentDate, *fh.Read(categories)})

		currentDate = currentDate.Next()
	}

	var includeRegex, excludeRegex *regexp.Regexp
	if command.CategoryIncludeFilter != "" {
		includeRegex, err = regexp.Compile(command.CategoryIncludeFilter)
		if err != nil {
			return fmt.Errorf("category include filter regex is invalid (%s)", err.Error())
		}
	}
	if command.CategoryExcludeFilter != "" {
		excludeRegex, err = regexp.Compile(command.CategoryExcludeFilter)
		if err != nil {
			return fmt.Errorf("category exclude filter regex is invalid (%s)", err.Error())
		}
	}
	matcher := func(catName string) bool {
		if includeRegex != nil && !includeRegex.MatchString(catName) {
			return false
		}
		if excludeRegex != nil && excludeRegex.MatchString(catName) {
			return false
		}
		return true
	}

	func() {
		fmt.Fprintln(os.Stderr, "PROSPECTIVE MATCHES:")
		for _, cat := range configData.Categories {
			if matcher(cat.Name) {
				fmt.Fprintf(os.Stderr, "  '%s'\n", cat.Name)
			}
		}
	}()

	for _, dataEntry := range data {
		timesheetEntry, err := dataEntry.EventList.GetTimesheetEntry(matcher)
		if err != nil {
			return fmt.Errorf("error while getting timesheet entry: %s", err)
		}

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

		stringifyTimestamp := func(ts model.Timestamp) string {
			return ts.ToString()
		}

		stringifyDuration := func(dur time.Duration) string {
			switch command.DurationFormat {
			case "golang":
				str := dur.String()
				if strings.HasSuffix(str, "m0s") {
					str = strings.TrimSuffix(str, "0s")
				}
				return str
			case "colon-delimited":
				durHours := dur.Truncate(time.Hour)
				durMins := (dur - durHours)
				return fmt.Sprintf("%d:%02d", int(durHours.Hours()), int(durMins.Minutes()))
			default:
				panic("unhandled case '" + command.DurationFormat + "'")
			}
		}

		fmt.Println(
			strings.Join(
				[]string{
					maybeEnquote(dataEntry.Date.ToGotime().Format(command.DateFormat)),
					asCSVString(*timesheetEntry, maybeEnquote, stringifyTimestamp, stringifyDuration, command.FieldSeparator),
				},
				command.FieldSeparator,
			),
		)
	}

	return nil
}

// asCSVString returns this TimesheetEntry in CSV format.
func asCSVString(e model.TimesheetEntry, processFieldString func(string) string, stringifyTimestamp func(model.Timestamp) string, stringifyDuration func(time.Duration) string, separator string) string {
	return strings.Join(
		[]string{
			processFieldString(stringifyTimestamp(e.Start)),
			processFieldString(stringifyDuration(e.BreakDuration)),
			processFieldString(stringifyTimestamp(e.End)),
		},
		separator,
	)
}
