package cli

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/config"
	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/potatolog"
	"github.com/ja-he/dayplan/internal/provider"
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

	// TODO: probably remove in favor of config before integration
	// Server backend options
	ServerURL      string `long:"server" description:"Use server backend with specified URL (e.g., http://localhost:8080)"`
	ServerUser     string `long:"server-user" description:"Username for server authentication (or set DAYPLAN_SERVER_USER env var)"`
	ServerPassword string `long:"server-password" description:"Password for server authentication (or set DAYPLAN_SERVER_PASSWORD env var)"`
	ServerDBPath   string `long:"server-db" description:"Path to local SQLite cache for server backend (default: $DAYPLAN_HOME/cache.db)"`
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

	// Create the event provider based on configuration
	categoryProvider := &backend.MemoryCategoryProvider{M: categoriesByName}
	eventsProvider, providerCleanup, err := createEventsProvider(
		envData,
		categoryProvider,
		command.ServerURL,
		command.ServerDBPath,
		command.ServerUser,
		command.ServerPassword,
	)
	if err != nil {
		return fmt.Errorf("Unable to create events provider (%w)", err)
	}
	if providerCleanup != nil {
		defer providerCleanup()
	}

	controller, err := NewController(initialDay, envData, categoriesByName, *stylesheet, weatherHandler, suntimesProvider, eventsProvider)
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

// createEventsProvider creates the appropriate EventProvider based on configuration.
// It returns the provider, an optional cleanup function (for graceful shutdown), and any error.
func createEventsProvider(
	envData control.EnvData,
	categoryProvider provider.CategoryProvider,
	serverURLOverride string,
	serverDBPathOverride string,
	serverUserOverride string,
	serverPasswordOverride string,
) (provider.EventProvider, func(), error) {

	// Check if server backend is requested
	serverURL := serverURLOverride
	if serverURL == "" {
		serverURL = os.Getenv("DAYPLAN_SERVER_URL")
	}

	if serverURL != "" {
		return createServerProvider(
			envData,
			categoryProvider,
			serverURL,
			serverDBPathOverride,
			serverUserOverride,
			serverPasswordOverride,
		)
	}

	// Use files backend (default)
	p, err := backend.NewFilesDataProvider(
		path.Join(envData.BaseDirPath, "days"),
		categoryProvider,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot initialize files data provider (%w)", err)
	}
	return p, nil, nil
}

// createServerProvider creates a CachingServerClientDataProvider and handles login.
func createServerProvider(
	envData control.EnvData,
	categoryProvider provider.CategoryProvider,
	serverURL string,
	dbPathOverride string,
	serverUserOverride string,
	serverPasswordOverride string,
) (provider.EventProvider, func(), error) {

	// Determine local cache DB path
	dbPath := dbPathOverride
	if dbPath == "" {
		dbPath = os.Getenv("DAYPLAN_SERVER_DB")
	}
	if dbPath == "" {
		dbPath = "/tmp/dayplan-cache.sqlite"
	}

	// Get credentials
	username := serverUserOverride
	if username == "" {
		username = os.Getenv("DAYPLAN_SERVER_USER")
	}
	password := serverPasswordOverride
	if password == "" {
		password = os.Getenv("DAYPLAN_SERVER_PASSWORD")
	}

	if username == "" || password == "" {
		return nil, nil, fmt.Errorf("server backend requires username and password (use --server-user/--server-password or DAYPLAN_SERVER_USER/DAYPLAN_SERVER_PASSWORD env vars)")
	}

	log.Info().Str("server", serverURL).Str("cache", dbPath).Msg("initializing server backend")

	// Create the provider
	p, err := backend.NewCachingServerClientDataProvider(
		backend.CachingServerClientConfig{
			DBPath:    dbPath,
			ServerURL: serverURL,
		},
		categoryProvider,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot initialize server data provider (%w)", err)
	}

	// Cleanup function for graceful shutdown
	cleanup := func() {
		log.Debug().Msg("closing server provider")
		if err := p.Close(); err != nil {
			log.Error().Err(err).Msg("error closing server provider")
		}
	}

	// Attempt login
	log.Info().Str("user", username).Msg("logging in to server")
	if err := p.Login(username, password); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("server login failed (%w)", err)
	}
	log.Info().Msg("successfully logged in to server")

	// Trigger initial sync
	log.Info().Msg("performing initial sync")
	syncErr := <-p.TriggerSync()
	if syncErr != nil {
		log.Warn().Err(syncErr).Msg("initial sync failed (will continue with local cache)")
	} else {
		log.Info().Msg("initial sync completed")
	}

	return p, cleanup, nil
}
