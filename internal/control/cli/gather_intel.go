package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/ja-he/dayplan/internal/config"
	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/intel"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/provider"
	"github.com/ja-he/dayplan/internal/provider/backend"
	"github.com/rs/zerolog/log"
)

type DayplanGatherIntelCommand struct {
}

// Executes the summarize command.
// (This gets called by `go-flags` when `summarize` is provided on the command
// line)
func (command *DayplanGatherIntelCommand) Execute(args []string) error {
	log.Info().Msg("Gathering intel...")
	log.Debug().Msg("Gathering intel...")

	var envData control.EnvData

	// set up dir per option
	dayplanHome := os.Getenv("DAYPLAN_HOME")
	if dayplanHome == "" {
		envData.BaseDirPath = os.Getenv("HOME") + "/.config/dayplan"
	} else {
		envData.BaseDirPath = strings.TrimRight(dayplanHome, "/")
	}

	assumedConfigFilePath := envData.BaseDirPath + "/" + "config.yaml"
	log.Debug().Msgf("Reading config from %v.", assumedConfigFilePath)

	// read config from file (for the category priorities)
	yamlData, err := os.ReadFile(assumedConfigFilePath)
	if err != nil {
		panic(fmt.Sprintf("can't read config file: '%s'", err))
	}
	configData, err := config.ParseConfigAugmentDefaults(config.Light, yamlData)
	if err != nil {
		panic(fmt.Sprintf("can't parse config data: '%s'", err))
	}
	categories := map[model.CategoryName]*model.Category{}
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
		categories[cat.Name] = &cat
	}

	filterCategories := len(DayplanOpts.SummarizeCommand.CategoryFilterString) > 0
	includeCategoriesByName := make(map[model.CategoryName]struct{})
	if filterCategories {
		for _, name := range strings.Split(DayplanOpts.SummarizeCommand.CategoryFilterString, ",") {
			includeCategoriesByName[model.CategoryName(name)] = struct{}{}
		}
	}

	var dataProvider provider.EventProvider
	dataProvider, err = backend.NewFilesDataProvider(
		path.Join(envData.BaseDirPath, "days"),
		&backend.MemoryCategoryProvider{M: categories},
	)
	if err != nil {
		return fmt.Errorf("can't create file data provider (%w)", err)
	}

	log.Debug().Msgf("Will gather from %d intel sources.", len(configData.IntelSources))

	var errs []error
	var events []model.Event
	for _, source := range configData.IntelSources {
		switch source.SourceType {
		case config.IntelSourceTypeHTTP:
			newEvents, err := gatherHTTPIntel(source.Name, *source.HTTPDetails)
			if err != nil {
				errs = append(errs, fmt.Errorf("Unable to gather HTTP intel from source %v", source.Name))
				continue
			}
			log.Debug().Msgf("Found %d events from source %v.", len(newEvents), source.Name)
			events = append(events, newEvents...)

		default:
			errs = append(errs, fmt.Errorf("Unknown source type %v encountered.", source.SourceType))
		}
	}

	for i := range events {
		id, err := dataProvider.AddEvent(events[i])
		if err != nil {
			errs = append(errs, fmt.Errorf("Unable to add event %d (%w).", i, err))
			continue
		}
		log.Debug().Msgf("Added even %d as %v.", i, id)
	}

	if len(events) > 0 {
		log.Debug().Msgf("Committing data provider state.")
		if err := dataProvider.CommitState(); err != nil {
			errs = append(errs, fmt.Errorf("Unable to commit data provider state (%w)", err))
		}
	}

	return errors.Join(errs...)
}

func gatherHTTPIntel(name string, details config.HTTPIntelSourceTypeDetails) ([]model.Event, error) {
	retrieveURL, err := url.JoinPath(details.URL, "/events/retrieve")
	resp, err := http.Post(retrieveURL, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("Unable to GET '%s' (%w)", name, err)
	}
	respBodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to read response body data (%w)", err)
	}

	parsedResponse := intel.RetrieveResponse{}
	if err := json.Unmarshal(respBodyData, &parsedResponse); err != nil {
		return nil, fmt.Errorf("Unable to parse response (may be an error response) (%w)", err)
	}

	for i := range parsedResponse.Events {
		if parsedResponse.Events[i].Category == "" {
			parsedResponse.Events[i].Category = model.CategoryName(name)
		}
	}

	return parsedResponse.Events, nil
}
