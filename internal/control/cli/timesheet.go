package cli

import (
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/config"
	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/provider"
	"github.com/ja-he/dayplan/internal/provider/backend"
	"github.com/ja-he/dayplan/internal/util"
)

// DayplanTimesheetCommand is the command `timesheet`, which produces a timesheet for
// a given category.
//
// A timesheet has entries per day, each of the form
//
//	<start-time>,<break-duration>,<end-time>
//
// e.g.
//
//	08:50,45min,16:20
type DayplanTimesheetCommand struct {
	FromDay string `short:"f" long:"from" description:"the day from which to start summarizing" value-name:"<yyyy-mm-dd>" required:"true"`
	TilDay  string `short:"t" long:"til" description:"the day til which to summarize (inclusive)" value-name:"<yyyy-mm-dd>" required:"true"`

	CategoryIncludeFilter string `long:"category-include-filter" short:"i" description:"the category filter include regex for which to generate the timesheet (empty value is ignored)" value-name:"<regex>"`
	CategoryExcludeFilter string `long:"category-exclude-filter" short:"e" description:"the category filter exclude regex for which to generate the timesheet (empty value is ignored)" value-name:"<regex>"`

	IncludeEmpty   bool   `long:"include-empty"`
	DateFormat     string `long:"date-format" value-name:"<format>" description:"specify the date format (see <https://pkg.go.dev/time#pkg-constants>)" default:"2006-01-02"`
	Enquote        bool   `long:"enquote" description:"add quotes around field values"`
	FieldSeparator string `long:"field-separator" value-name:"<CSV separator (default ',')>" default:","`
	DurationFormat string `long:"duration-format" option:"golang" option:"colon-delimited" default:"golang"`

	LogOutputFile string `long:"log-output-file" description:"specify a log output file (by default they go to stdout)"`
	LogPretty     bool   `long:"log-pretty" description:"prettify log messages"`
	LogLevel      string `long:"log-level" description:"set log level to 'trace', 'debug', 'info', 'warn', 'error'"`
}

// Execute executes the timesheet command.
func (command *DayplanTimesheetCommand) Execute(args []string) error {
	timesheetTimezone := time.Local

	{
		var logFileWriter io.Writer
		if command.LogOutputFile != "" {
			file, err := os.OpenFile(command.LogOutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("could not open file '%s' for logging (%w)", command.LogOutputFile, err)
			}
			if command.LogPretty {
				logFileWriter = zerolog.ConsoleWriter{Out: file}
			} else {
				logFileWriter = file
			}
		}
		logLevel := func() zerolog.Level {
			switch command.LogLevel {
			case "trace":
				return zerolog.TraceLevel
			case "debug":
				return zerolog.DebugLevel
			case "info":
				return zerolog.InfoLevel
			case "warn":
				return zerolog.WarnLevel
			case "error":
				return zerolog.ErrorLevel
			}
			return zerolog.WarnLevel
		}()
		if logFileWriter != nil {
			log.Logger = zerolog.New(logFileWriter)
		}
		log.Logger = log.Logger.Level(logLevel).With().Logger()
	}

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
	categoriesByName, err := backend.GetCategoriesByNameFromConfig(configData)
	if err != nil {
		return fmt.Errorf("can't get categories from config (%w)", err)
	}

	startDate, err := model.DateFromString(command.FromDay)
	if err != nil {
		log.Fatal().Msgf("from date '%s' invalid", command.FromDay)
	}
	finalDate, err := model.DateFromString(command.TilDay)
	if err != nil {
		log.Fatal().Msgf("til date '%s' invalid", command.TilDay)
	}

	categoryProvider := &backend.MemoryCategoryProvider{M: categoriesByName}

	var eventsProvider provider.EventProvider
	eventsProvider, err = backend.NewFilesDataProvider(
		path.Join(envData.BaseDirPath, "days"),
		categoryProvider,
	)

	isMidnight := func(t time.Time) bool {
		return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
	}

	data := make(map[model.Date][]*model.Event)

	events, err := eventsProvider.GetEventsCoveringTimerange(startDate.ToGotime(timesheetTimezone), finalDate.ToGotime(timesheetTimezone).Add(24*time.Hour))
	if err != nil {
		return fmt.Errorf("error while getting events for %s-%s (%w)", startDate.String(), finalDate.String(), err)
	}
	fallbackEnd := time.Now()
	for _, event := range events {
		if event.End == nil {
			event.End = new(time.Time)
			*event.End = fallbackEnd
		}

		if model.DateFromGotime(event.Start, timesheetTimezone) != model.DateFromGotime(*event.End, timesheetTimezone) && isMidnight(*event.End) {
			log.Warn().Msgf("Event '%s' spans more than one day, current timesheet implementation does not consider such events (TODO).", event.ID)
			continue
		}

		eventDate := model.DateFromGotime(event.Start, timesheetTimezone)
		prev, _ := data[eventDate]
		data[eventDate] = append(prev, event) // OK to use prev here without checking the OK-value (_) since if it's nil append can deal with it.
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
	matcher := func(catName model.CategoryName) bool {
		if includeRegex != nil && !includeRegex.MatchString(string(catName)) {
			return false
		}
		if excludeRegex != nil && excludeRegex.MatchString(string(catName)) {
			return false
		}
		return true
	}

	func() {
		fmt.Fprintln(os.Stderr, "PROSPECTIVE MATCHES:")
		for _, cat := range configData.Categories {
			if matcher(model.CategoryName(cat.Name)) {
				fmt.Fprintf(os.Stderr, "  '%s'\n", cat.Name)
			}
		}
	}()

	categoryPriority := map[model.CategoryName]int{}
	for _, cat := range configData.Categories {
		categoryPriority[model.CategoryName(cat.Name)] = cat.Priority
	}

	categoryPriorityProvider := func(catName model.CategoryName) int {
		prio, ok := categoryPriority[catName]
		if !ok {
			return 0
		}
		return prio
	}

	var dates []model.Date
	for date := range data {
		dates = append(dates, date)
	}
	sort.Sort(model.DateSlice(dates))

	for _, date := range dates {
		eventList := model.EventList{Events: data[date]}
		timesheetEntry, err := eventList.GetTimesheetEntry(matcher, categoryPriorityProvider, date, timesheetTimezone, fallbackEnd)
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
					maybeEnquote(date.ToGotime(timesheetTimezone).Format(command.DateFormat)),
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
