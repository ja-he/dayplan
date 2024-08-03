package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ja-he/dayplan/internal/config"
	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/storage"
)

// AddCommand contains flags for the `summarize` command line command, for
// `go-flags` to parse command line args into.
type AddCommand struct {
	Category string `short:"c" long:"category" description:"the category of the added event(s)" value-name:"<category>" required:"true"`
	Name     string `short:"n" long:"name" description:"the name of the added event(s)" value-name:"<name>" required:"true"`

	Start string `short:"s" long:"start" description:"the time at which the event begins" value-name:"<TIME>" required:"true"`
	End   string `short:"e" long:"end" description:"the time at which the event ends" value-name:"<TIME>" required:"true"`

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
	categoryName := model.CategoryName(command.Category)
	found := false
	for _, categoryFromConfig := range configData.Categories {
		if categoryFromConfig.Name == string(categoryName) {
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "WARNING: category '%s' not found in config data\n", categoryName)
	}

	// verify times
	layout := "2006-01-02 15:04:05"
	start, err := time.Parse(layout, command.Start)
	if err != nil {
		return fmt.Errorf("could not parse start time '%s' (%w)", command.Start, err)
	}
	end, err := time.Parse(layout, command.End)
	if err != nil {
		return fmt.Errorf("could not parse end time '%s' (%w)", command.End, err)
	}
	if !end.After(start) {
		panic(fmt.Sprintf("ERROR: end time %s is not after start time %s", end.String(), start.String()))
	}

	var repeatTilTime time.Time
	if command.RepeatInterval != "" && command.RepeatTil == "" || command.RepeatInterval == "" && command.RepeatTil != "" {
		panic("ERROR: either both repeat interval and 'til' date need to be specified, or neither")
	} else if command.RepeatTil != "" {
		repeatTilTime, err = time.Parse(layout, command.RepeatTil)
		if err != nil {
			return fmt.Errorf("could not parse repeat-til time '%s' (%w)", command.RepeatTil, err)
		}
		if !repeatTilTime.After(start) || !repeatTilTime.After(end) {
			return fmt.Errorf("repeat-til time '%s' is not after start time '%s' and end time '%s'", repeatTilTime.String(), start.String(), end.String())
		}
	}

	panic("TODO: need to implement provider setup i suppose")
	var provider storage.DataProvider

	var events []model.Event
	events = append(events, model.Event{
		Start: start,
		End:   end,
		Name:  command.Name,
		Cat:   model.Category{Name: categoryName},
	})

	if command.RepeatInterval != "" {
		var increment func(time.Time) time.Time

		switch command.RepeatInterval {
		case "daily":
			increment = func(t time.Time) time.Time { return t.AddDate(0, 0, 1) }
		case "weekly":
			increment = func(t time.Time) time.Time { return t.AddDate(0, 0, 7) }
		case "monthly":
			increment = func(t time.Time) time.Time { return t.AddDate(0, 1, 0) }
		default:
			panic(fmt.Sprintf("ERROR: unknown repeat interval '%s'", command.RepeatInterval))
		}

		// we already did the first date
		currentStart := increment(start)
		currentEnd := increment(end)

		for !currentStart.After(repeatTilTime) {
			event := model.Event{
				Start: currentStart,
				End:   currentEnd,
				Name:  command.Name,
				Cat:   model.Category{Name: categoryName},
			}
			events = append(events, event)

			currentStart = increment(currentStart)
			currentEnd = increment(currentEnd)
		}
	}

	// write at the end, so we don't add partial data if we panicked somewhere
	fmt.Println("writing to:")
	for _, event := range events {
		fmt.Printf(" + %s\n", event.String())
		_, err := provider.AddEvent(event)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: could not add event to provider (%s)\n", err.Error())
		}
	}

	os.Exit(0)
	return nil
}
