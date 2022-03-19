package cli

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/ja-he/dayplan/src/config"
	"github.com/ja-he/dayplan/src/control"
	"github.com/ja-he/dayplan/src/filehandling"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/util"
)

// Flags for the `summarize` command line command, for `go-flags` to parse
// command line args into.
type SummarizeCommand struct {
	FromDay string `short:"f" long:"from" description:"the day from which to start summarizing" value-name:"<yyyy-mm-dd>" required:"true"`
	TilDay  string `short:"t" long:"til" description:"the day til which to summarize (inclusive)" value-name:"<yyyy-mm-dd>" required:"true"`

	HumanReadable        bool   `long:"human-readable" description:"format times as hours and minutes"`
	CategoryFilterString string `long:"category-filter" description:"a filter for categories; any named categories included; all included if omitted" value-name:"<cat1>,<cat2>,..."`

	Verbose bool `short:"v" long:"verbose" description:"provide verbose output"`
}

// Executes the summarize command.
// (This gets called by `go-flags` when `summarize` is provided on the command
// line)
func (command *SummarizeCommand) Execute(args []string) error {
	summarize()
	return nil
}

func summarize() {
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
	configData, err := config.ParseConfigAugmentDefaults(yamlData)
	if err != nil {
		panic(fmt.Sprintf("can't parse config data: '%s'", err))
	}
	categories := styling.EmptyCategoryStyling()
	for _, category := range configData.Categories {
		categories.AddStyleFromInput(category)
	}

	currentDate, err := model.FromString(Opts.SummarizeCommand.FromDay)
	if err != nil {
		log.Fatalf("from date '%s' invalid", Opts.SummarizeCommand.FromDay)
	}
	finalDate, err := model.FromString(Opts.SummarizeCommand.TilDay)
	if err != nil {
		log.Fatalf("til date '%s' invalid", Opts.SummarizeCommand.TilDay)
	}

	filterCategories := len(Opts.SummarizeCommand.CategoryFilterString) > 0
	includeCategoriesByName := make(map[string]struct{})
	if filterCategories {
		for _, name := range strings.Split(Opts.SummarizeCommand.CategoryFilterString, ",") {
			includeCategoriesByName[name] = struct{}{}
		}
	}

	// TODO: can probably make this mostly async?
	days := make([]model.Day, 0)
	for currentDate != finalDate.Next() {
		fh := filehandling.NewFileHandler(envData.BaseDirPath + "/days/" + currentDate.ToString())
		days = append(days, *fh.Read(categories.GetKnownCategoriesByName()))

		currentDate = currentDate.Next()
	}

	totalSummary := make(map[model.Category]int)
	for _, day := range days {
		daySummary := day.SumUpByCategory()
		for category, duration := range daySummary {
			totalSummary[category] += duration
		}
	}

	if Opts.SummarizeCommand.Verbose {
		fmt.Println("dayplan time summary:")

		fmt.Println("from:            ", Opts.SummarizeCommand.FromDay)
		fmt.Println("til:             ", Opts.SummarizeCommand.TilDay)
		fmt.Println("category filter: ", Opts.SummarizeCommand.CategoryFilterString)

		fmt.Println("read", len(days), "days")
		fmt.Println("total summary:")
	}

	for category, duration := range totalSummary {
		_, categoryIncluded := includeCategoriesByName[category.Name]
		if filterCategories && !categoryIncluded {
			continue
		}

		var durationStr string
		if Opts.SummarizeCommand.HumanReadable {
			durationStr = fmt.Sprint(util.DurationToString(duration))
		} else {
			durationStr = fmt.Sprint(duration, " min")
		}
		fmt.Print("  ")
		fmt.Printf("% 20s (prio:% 3d): % 10s\n", category.Name, category.Priority, durationStr)
	}

	os.Exit(0)
}
