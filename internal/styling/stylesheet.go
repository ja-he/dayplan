package styling

import (
	"github.com/ja-he/dayplan/internal/config"
)

// Stylesheet represents all styles used by the application for rendering.
type Stylesheet struct {
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

// NewStylesheetFromConfig constructs a new stylesheet from a given config
// stylesheet.
func NewStylesheetFromConfig(config config.Stylesheet) *Stylesheet {
	stylesheet := Stylesheet{}

	stylesheet.Normal = StyleFromConfig(config.Normal)
	stylesheet.NormalEmphasized = StyleFromConfig(config.NormalEmphasized)
	stylesheet.WeatherNormal = StyleFromConfig(config.WeatherNormal)
	stylesheet.WeatherSunny = StyleFromConfig(config.WeatherSunny)
	stylesheet.WeatherRainy = StyleFromConfig(config.WeatherRainy)
	stylesheet.TimelineDay = StyleFromConfig(config.TimelineDay)
	stylesheet.TimelineNight = StyleFromConfig(config.TimelineNight)
	stylesheet.TimelineNow = StyleFromConfig(config.TimelineNow)
	stylesheet.Status = StyleFromConfig(config.Status)
	stylesheet.LogDefault = StyleFromConfig(config.LogDefault)
	stylesheet.LogTitleBox = StyleFromConfig(config.LogTitleBox)
	stylesheet.LogEntryTypeError = StyleFromConfig(config.LogEntryTypeError)
	stylesheet.LogEntryTypeWarn = StyleFromConfig(config.LogEntryTypeWarn)
	stylesheet.LogEntryTypeInfo = StyleFromConfig(config.LogEntryTypeInfo)
	stylesheet.LogEntryTypeDebug = StyleFromConfig(config.LogEntryTypeDebug)
	stylesheet.LogEntryTypeTrace = StyleFromConfig(config.LogEntryTypeTrace)
	stylesheet.LogEntryLocation = StyleFromConfig(config.LogEntryLocation)
	stylesheet.LogEntryTime = StyleFromConfig(config.LogEntryTime)
	stylesheet.Help = StyleFromConfig(config.Help)
	stylesheet.Editor = StyleFromConfig(config.Editor)
	stylesheet.SummaryDefault = StyleFromConfig(config.SummaryDefault)
	stylesheet.SummaryTitleBox = StyleFromConfig(config.SummaryTitleBox)
	stylesheet.CategoryFallback = StyleFromConfig(config.CategoryFallback)

	return &stylesheet
}
