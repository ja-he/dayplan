package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ja-he/dayplan/internal/config"
	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/filehandling"
	"github.com/ja-he/dayplan/internal/model"
)

// AddCommand contains flags for the `summarize` command line command, for
// `go-flags` to parse command line args into.
type AddCommand struct {
	Category string `short:"c" long:"category" description:"the category of the added event(s)" value-name:"<category>" required:"true"`
	Name     string `short:"n" long:"name" description:"the name of the added event(s)" value-name:"<name>" required:"true"`

	Date  string `short:"d" long:"date" description:"the date of the (first) event" value-name:"<yyyy-mm-dd>" required:"true"`
	Start string `short:"s" long:"start" description:"the time at which the event begins" value-name:"<HH:MM>" required:"true"`
	End   string `short:"e" long:"end" description:"the time at which the event ends" value-name:"<HH:MM>" required:"true"`

	RepeatInterval string `short:"r" long:"repeat-interval" description:"the repeat interval; if omitted, no repetition is assumed; requires end (til) date to be specified" choice:"daily" choice:"weekly" choice:"monthly"`
	RepeatTil      string `short:"t" long:"repeat-til" description:"the date until which to repeat the event; requires repeat inteval to be specified" value-name:"<yyyy-mm-dd>"`
}

// Execute executes the add command.
// (This gets called by `go-flags` when `add` is provided on the command line)
func (command *AddCommand) Execute(args []string) error {
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
		panic(fmt.Sprintf("ERROR: can't read config file: '%s'", err))
	}
	configData, err := config.ParseConfigAugmentDefaults(config.Light, yamlData)
	if err != nil {
		panic(fmt.Sprintf("ERROR: can't parse config data: '%s'", err))
	}

	// verify category
	if strings.ContainsRune(command.Category, '|') {
		panic("ERROR: category name cannot contain '|'")
	}
	found := false
	for _, category := range configData.Categories {
		if category.Name == command.Category {
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "WARNING: category '%s' not found in config data\n", command.Category)
	}

	// verify date and time
	date, err := model.FromString(command.Date)
	if err != nil {
		panic(fmt.Sprintf("ERROR: %s", err.Error()))
	}
	start := *model.NewTimestamp(command.Start)
	end := *model.NewTimestamp(command.End)
	if !end.IsAfter(start) {
		panic(fmt.Sprintf("ERROR: end time %s is not after start time %s", end.ToString(), start.ToString()))
	}

	var repeatTilDate model.Date
	if command.RepeatInterval != "" && command.RepeatTil == "" || command.RepeatInterval == "" && command.RepeatTil != "" {
		panic("ERROR: either both repeat interval and 'til' date need to be specified, or neither")
	} else if command.RepeatTil != "" {
		repeatTilDate, err = model.FromString(command.RepeatTil)
		if err != nil {
			panic(fmt.Sprintf("ERROR: %s", err.Error()))
		}
		if !repeatTilDate.IsAfter(date) {
			panic("ERROR: repetition end ('til') date needs to be AFTER start date")
		}
	}

	type fileAndDay struct {
		file *filehandling.FileHandler
		data *model.Day
		date model.Date
	}
	toWrite := []fileAndDay{}

	startDayFile := filehandling.NewFileHandler(envData.BaseDirPath + "/days/" + date.ToString())
	startDay := startDayFile.Read([]model.Category{}) // we don't need the categories for this
	err = startDay.AddEvent(
		&model.Event{
			Start: start,
			End:   end,
			Name:  command.Name,
			Cat:   model.Category{Name: command.Category},
		},
	)
	if err != nil {
		panic(fmt.Sprintf("ERROR: %s", err.Error()))
	}
	toWrite = append(toWrite, fileAndDay{startDayFile, startDay, date})

	if command.RepeatInterval != "" {
		var dateIncrementer func(model.Date) model.Date

		switch command.RepeatInterval {
		case "daily":
			dateIncrementer = func(current model.Date) model.Date { return current.Next() }
		case "weekly":
			dateIncrementer = func(current model.Date) model.Date { return current.Forward(7) }
		case "monthly":
			dateIncrementer = func(current model.Date) model.Date {
				result := current.GetLastOfMonth().Next()
				result.Day = current.Day
				return result
			}
		default:
			panic(fmt.Sprintf("ERROR: unknown repeat interval '%s'", command.RepeatInterval))
		}

		// we already did the first date
		current := dateIncrementer(date)

		for !current.IsAfter(repeatTilDate) {
			currentDayFile := filehandling.NewFileHandler(envData.BaseDirPath + "/days/" + current.ToString())
			currentDay := currentDayFile.Read([]model.Category{}) // we don't need the categories for this
			err = currentDay.AddEvent(
				&model.Event{
					Start: start,
					End:   end,
					Name:  command.Name,
					Cat:   model.Category{Name: command.Category},
				},
			)
			if err != nil {
				panic(fmt.Sprintf("ERROR: %s", err.Error()))
			}
			toWrite = append(toWrite, fileAndDay{currentDayFile, currentDay, current})

			current = dateIncrementer(current)
		}
	}

	// write at the end, so we don't add partial data if we panicked somewhere
	fmt.Println("writing to:")
	for _, writable := range toWrite {
		fmt.Printf(" + %s (%s)\n", writable.date.ToString(), writable.date.ToWeekday().String())
		writable.file.Write(writable.data)
	}

	os.Exit(0)
	return nil
}
