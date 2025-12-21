package styling

import (
	"errors"

	"github.com/rs/zerolog/log"

	cfg "github.com/ja-he/dayplan/internal/config"
)

// Stylesheet represents all styles used by the application for rendering.
type Stylesheet struct {
	Theme cfg.ColorschemeType

	Normal           DrawStyling
	NormalEmphasized DrawStyling

	WeatherNormal DrawStyling
	WeatherSunny  DrawStyling
	WeatherRainy  DrawStyling

	TimelineDay   DrawStyling
	TimelineNight DrawStyling
	TimelineNow   DrawStyling

	Status DrawStyling

	CategoryFallback DrawStyling

	LogDefault  DrawStyling
	LogTitleBox DrawStyling

	LogEntryTypeError DrawStyling
	LogEntryTypeWarn  DrawStyling
	LogEntryTypeInfo  DrawStyling
	LogEntryTypeDebug DrawStyling
	LogEntryTypeTrace DrawStyling

	LogEntryLocation DrawStyling
	LogEntryTime     DrawStyling

	Help DrawStyling

	Editor DrawStyling

	SummaryDefault  DrawStyling
	SummaryTitleBox DrawStyling
}

func (s Stylesheet) Verify() {
	if s.Normal == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'Normal'")
	}
	if s.NormalEmphasized == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'NormalEmphasized'")
	}
	if s.WeatherNormal == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'WeatherNormal'")
	}
	if s.WeatherSunny == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'WeatherSunny'")
	}
	if s.WeatherRainy == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'WeatherRainy'")
	}
	if s.TimelineDay == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'TimelineDay'")
	}
	if s.TimelineNight == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'TimelineNight'")
	}
	if s.TimelineNow == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'TimelineNow'")
	}
	if s.Status == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'Status'")
	}
	if s.CategoryFallback == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'CategoryFallback'")
	}
	if s.LogDefault == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'LogDefault'")
	}
	if s.LogTitleBox == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'LogTitleBox'")
	}
	if s.LogEntryTypeError == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'LogEntryTypeError'")
	}
	if s.LogEntryTypeWarn == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'LogEntryTypeWarn'")
	}
	if s.LogEntryTypeInfo == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'LogEntryTypeInfo'")
	}
	if s.LogEntryTypeDebug == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'LogEntryTypeDebug'")
	}
	if s.LogEntryTypeTrace == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'LogEntryTypeTrace'")
	}
	if s.LogEntryLocation == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'LogEntryLocation'")
	}
	if s.LogEntryTime == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'LogEntryTime'")
	}
	if s.Help == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'Help'")
	}
	if s.Editor == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'Editor'")
	}
	if s.SummaryDefault == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'SummaryDefault'")
	}
	if s.SummaryTitleBox == nil {
		log.Fatal().Msgf("bad constructed stylesheet part 'SummaryTitleBox'")
	}
}

// NewStylesheetFromConfig constructs a new stylesheet from a given config
// stylesheet.
func NewStylesheetFromConfig(config cfg.Stylesheet, theme cfg.ColorschemeType) (*Stylesheet, error) {
	stylesheet := Stylesheet{}
	var errs []error
	setStyleFromConfigUnlessError := func(src cfg.Styling) DrawStyling {
		dst, err := StyleFromConfig(src)
		if err != nil {
			errs = append(errs, err)
		}
		return dst
	}
	stylesheet.Theme = theme

	stylesheet.Normal = setStyleFromConfigUnlessError(config.Normal)
	stylesheet.NormalEmphasized = setStyleFromConfigUnlessError(config.NormalEmphasized)
	stylesheet.WeatherNormal = setStyleFromConfigUnlessError(config.WeatherNormal)
	stylesheet.WeatherSunny = setStyleFromConfigUnlessError(config.WeatherSunny)
	stylesheet.WeatherRainy = setStyleFromConfigUnlessError(config.WeatherRainy)
	stylesheet.TimelineDay = setStyleFromConfigUnlessError(config.TimelineDay)
	stylesheet.TimelineNight = setStyleFromConfigUnlessError(config.TimelineNight)
	stylesheet.TimelineNow = setStyleFromConfigUnlessError(config.TimelineNow)
	stylesheet.Status = setStyleFromConfigUnlessError(config.Status)
	stylesheet.LogDefault = setStyleFromConfigUnlessError(config.LogDefault)
	stylesheet.LogTitleBox = setStyleFromConfigUnlessError(config.LogTitleBox)
	stylesheet.LogEntryTypeError = setStyleFromConfigUnlessError(config.LogEntryTypeError)
	stylesheet.LogEntryTypeWarn = setStyleFromConfigUnlessError(config.LogEntryTypeWarn)
	stylesheet.LogEntryTypeInfo = setStyleFromConfigUnlessError(config.LogEntryTypeInfo)
	stylesheet.LogEntryTypeDebug = setStyleFromConfigUnlessError(config.LogEntryTypeDebug)
	stylesheet.LogEntryTypeTrace = setStyleFromConfigUnlessError(config.LogEntryTypeTrace)
	stylesheet.LogEntryLocation = setStyleFromConfigUnlessError(config.LogEntryLocation)
	stylesheet.LogEntryTime = setStyleFromConfigUnlessError(config.LogEntryTime)
	stylesheet.Help = setStyleFromConfigUnlessError(config.Help)
	stylesheet.Editor = setStyleFromConfigUnlessError(config.Editor)
	stylesheet.SummaryDefault = setStyleFromConfigUnlessError(config.SummaryDefault)
	stylesheet.SummaryTitleBox = setStyleFromConfigUnlessError(config.SummaryTitleBox)
	stylesheet.CategoryFallback = setStyleFromConfigUnlessError(config.CategoryFallback)

	if errs != nil {
		return nil, errors.Join(errs...)
	}
	stylesheet.Verify()

	return &stylesheet, nil
}
