package cli

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ja-he/dayplan/internal/config"
	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/storage"
	"github.com/ja-he/dayplan/internal/storage/providers"
	"github.com/ja-he/dayplan/internal/styling"
)

// Flags for the `summarize` command line command, for `go-flags` to parse
// command line args into.
type SummarizeCommand struct {
	From time.Time `short:"f" long:"from" description:"the timestamp from which to start summarizing" value-name:"<TIME>" required:"true"`
	Til  time.Time `short:"t" long:"til" description:"the timestamp until which to summarize" value-name:"<TIME>" required:"true"`

	HumanReadable        bool   `long:"human-readable" description:"format times as hours and minutes"`
	CategoryFilterString string `long:"category-filter" description:"a filter for categories; any named categories included; all included if omitted" value-name:"<cat1>,<cat2>,..."`

	Verbose bool `short:"v" long:"verbose" description:"provide verbose output"`
}

// Executes the summarize command.
// (This gets called by `go-flags` when `summarize` is provided on the command
// line)
func (command *SummarizeCommand) Execute(args []string) error {
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
			Name:     model.CategoryName(category.Name),
			Priority: category.Priority,
			Goal:     goal,
		}
		style := styling.StyleFromHexSingle(category.Color, false)
		styledCategories.Add(cat, style)
	}

	if command.Til.Before(command.From) {
		return fmt.Errorf("from-time must be before til-time")
	}

	filterCategories := len(Opts.SummarizeCommand.CategoryFilterString) > 0
	includeCategoriesByName := make(map[model.CategoryName]struct{})
	if filterCategories {
		for _, name := range strings.Split(Opts.SummarizeCommand.CategoryFilterString, ",") {
			includeCategoriesByName[model.CategoryName(name)] = struct{}{}
		}
	}

	// TODO: can probably make this mostly async?
	var dataProvider storage.DataProvider
	dataProvider, err = providers.NewFilesDataProvider(path.Join(envData.BaseDirPath, "days"))
	if err != nil {
		return fmt.Errorf("can't create file data provider (%w)", err)
	}

	categoryIncluded := func(categoryName model.CategoryName) bool {
		_, ok := includeCategoriesByName[categoryName]
		return ok
	}

	totalSummary := dataProvider.SumUpTimespanByCategory(command.From, command.Til)

	if Opts.SummarizeCommand.Verbose {
		fmt.Println("dayplan time summary:")

		fmt.Println("from:            ", command.From)
		fmt.Println("til:             ", command.Til)
		fmt.Println("category filter: ", command.CategoryFilterString)

		fmt.Println("total summary:")
	}

	categoriesByName := styledCategories.GetKnownCategoriesByName()
	for categoryName, duration := range totalSummary {
		category, ok := categoriesByName[categoryName]
		if !ok {
			fmt.Fprint(os.Stderr, "warning: category '", categoryName, "' not found in config\n")
			category = &model.Category{
				Name:       categoryName,
				Priority:   0,
				Goal:       nil,
				Deprecated: false,
			}
		}

		if filterCategories && !categoryIncluded(categoryName) {
			continue
		}

		var durationStr string
		if Opts.SummarizeCommand.HumanReadable {
			durationStr = fmt.Sprint(duration.String())
		} else {
			durationStr = fmt.Sprint(duration, " min")
		}

		var goalStr string = ""
		if category.Goal != nil {
			goal := model.GoalForRange(category.Goal, command.From, command.Til)
			actual := time.Duration(duration) * time.Minute
			deficit := goal - actual
			deficitStr := fmt.Sprint(deficit - (deficit % time.Minute))
			goalStr = fmt.Sprintf("(%.2f%% of goal, %s deficit)", (float64(actual)/float64(goal))*100.0, deficitStr)
		}

		fmt.Print("  ")
		fmt.Printf("% 20s (prio:% 3d): % 10s %s\n", category.Name, category.Priority, durationStr, goalStr)
	}

	return nil
}
