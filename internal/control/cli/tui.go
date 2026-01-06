package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/config"
	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/potatolog"
	"github.com/ja-he/dayplan/internal/provider/backend"
	"github.com/ja-he/dayplan/internal/styling"
)

// DayplanTUICommand is the struct for the TUI command.
type DayplanTUICommand struct {
	Day           string `short:"d" long:"day" description:"Specify the day to plan" value-name:"<file>"`
	Theme         string `short:"t" long:"theme" choice:"light" choice:"dark" description:"Select a 'dark' or a 'light' default theme (note: only sets defaults, which are individually overridden by settings in config.yaml"`
	LogOutputFile string `short:"l" long:"log-output-file" description:"specify a log output file (otherwise logs dropped)"`
	LogPretty     bool   `short:"p" long:"log-pretty" description:"prettify logs to file"`
	LogLevel      string `long:"log-level" description:"set log level to 'trace', 'debug', 'info', 'warn', 'error'"`
}

// Execute runs the TUI command.
func (command *DayplanTUICommand) Execute(_ []string) error {
	// create TUI logger
	var logWriter io.Writer
	if command.LogOutputFile != "" {
		var fileLogger io.Writer
		file, err := os.OpenFile(command.LogOutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("could not open file '%s' for logging (%w)", command.LogOutputFile, err)
		}
		if command.LogPretty {
			fileLogger = zerolog.ConsoleWriter{Out: file}
		} else {
			fileLogger = file
		}
		logWriter = zerolog.MultiLevelWriter(fileLogger, &potatolog.GlobalMemoryLogReaderWriter)
	} else {
		logWriter = &potatolog.GlobalMemoryLogReaderWriter
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
	tuiLogger := zerolog.New(logWriter).Level(logLevel).With().Timestamp().Caller().Logger()

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
		initialDay, err = model.DateFromString(command.Day)
		if err != nil {
			return fmt.Errorf("could not parse given date (%w)", err)
		}
	}

	envData.OWMAPIKey = os.Getenv("OWM_API_KEY")

	envData.Latitude = os.Getenv("LATITUDE")
	envData.Longitude = os.Getenv("LONGITUDE")

	// read config from file
	yamlData, err := os.ReadFile(envData.BaseDirPath + "/" + "config.yaml")
	if err != nil {
		log.Warn().Err(err).Msg("can't read config file: '%s', using defaults")
		yamlData = make([]byte, 0)
	}
	configData, err := config.ParseConfigAugmentDefaults(theme, yamlData)
	if err != nil {
		return fmt.Errorf("can't parse config data (%w)", err)
	}

	// get categories from config
	categoriesByName, err := backend.GetCategoriesByNameFromConfig(configData)
	if err != nil {
		return fmt.Errorf("can't get categories from config (%w)", err)
	}

	stylesheet, err := styling.NewStylesheetFromConfig(configData.Stylesheet, theme)
	if err != nil {
		return fmt.Errorf("could not create stylsheet from config: %w", err)
	}

	// now that the screen is initialized, we'll always want the TUI logger, so
	// we're making it the global logger
	previouslySetLogger := log.Logger
	log.Logger = tuiLogger
	log.Debug().Msg("set up logging to only TUI")

	weatherHandler, suntimesProvider, err := createWeatherAndSuntimes(envData)
	if err != nil {
		return fmt.Errorf("Unable to initialize weather or suntimes handling (%w)", err)
	}

	controller, err := NewController(initialDay, envData, categoriesByName, *stylesheet, weatherHandler, suntimesProvider)
	if err != nil {
		log.Logger = previouslySetLogger
		log.Error().Err(err).Msgf("something went wrong setting up the TUI, will check unpublished logs and return error")

		// The TUI was perhaps not set up and we have to assume that the logs have not been written anywhere.
		// To inform the user, we'll print the logs to stderr.
		unpublishedLog := potatolog.GlobalMemoryLogReaderWriter.Get()
		log.Warn().Msgf("have %d unpublished log entries which will be published now", len(unpublishedLog))
		for _, entry := range unpublishedLog {
			catchupLogger := log.Logger.With().Str("source", "catchup").Logger()

			e := func() *zerolog.Event {
				switch entry["level"] {
				case "trace":
					return catchupLogger.Trace()
				case "debug":
					return catchupLogger.Debug()
				case "info":
					return catchupLogger.Info()
				case "warn":
					return catchupLogger.Warn()
				case "error":
					return catchupLogger.Error()
				}
				return catchupLogger.Error()
			}()

			getEntryAsString := func(id string) string {
				untyped, ok := entry[id]
				if !ok {
					return "<noentry>"
				}
				if str, ok := untyped.(string); ok {
					return str
				}
				return fmt.Sprintf("<nonstring>: %v", untyped)
			}
			msg := getEntryAsString("message")
			caller := getEntryAsString("caller")
			timestamp := getEntryAsString("time")
			e = e.Str("true-caller", caller).Str("true-timestamp", timestamp)
			for k, v := range entry {
				if k == "message" || k == "caller" || k == "timestamp" {
					continue
				}
				e = e.Interface(k, v)
			}
			e.Msg(msg)
		}
		return fmt.Errorf("could not set up TUI (%w)", err)
	}

	controller.Run()
	return nil
}
