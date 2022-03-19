package styling

import (
	"github.com/ja-he/dayplan/src/config"
)

// Stylesheet represents all styles used by the application for rendering.
type Stylesheet struct {
	Normal DrawStyling

	WeatherNormal DrawStyling
	WeatherSunny  DrawStyling
	WeatherRainy  DrawStyling

	TimelineDay   DrawStyling
	TimelineNight DrawStyling
	TimelineNow   DrawStyling

	Status DrawStyling

	CategoryFallback DrawStyling

	LogDefault       DrawStyling
	LogTitleBox      DrawStyling
	LogEntryType     DrawStyling
	LogEntryLocation DrawStyling
	LogEntryTime     DrawStyling

	Help DrawStyling

	Editor DrawStyling

	SummaryDefault  DrawStyling
	SummaryTitleBox DrawStyling
}

func NewStylesheetFromConfig(config config.Stylesheet) *Stylesheet {
	stylesheet := Stylesheet{}

	stylesheet.Normal = StyleFromConfig(config.Normal)
	stylesheet.WeatherNormal = StyleFromConfig(config.WeatherNormal)
	stylesheet.WeatherSunny = StyleFromConfig(config.WeatherSunny)
	stylesheet.WeatherRainy = StyleFromConfig(config.WeatherRainy)
	stylesheet.TimelineDay = StyleFromConfig(config.TimelineDay)
	stylesheet.TimelineNight = StyleFromConfig(config.TimelineNight)
	stylesheet.TimelineNow = StyleFromConfig(config.TimelineNow)
	stylesheet.Status = StyleFromConfig(config.Status)
	stylesheet.LogDefault = StyleFromConfig(config.LogDefault)
	stylesheet.LogTitleBox = StyleFromConfig(config.LogTitleBox)
	stylesheet.LogEntryType = StyleFromConfig(config.LogEntryType)
	stylesheet.LogEntryLocation = StyleFromConfig(config.LogEntryLocation)
	stylesheet.LogEntryTime = StyleFromConfig(config.LogEntryTime)
	stylesheet.Help = StyleFromConfig(config.Help)
	stylesheet.Editor = StyleFromConfig(config.Editor)
	stylesheet.SummaryDefault = StyleFromConfig(config.SummaryDefault)
	stylesheet.SummaryTitleBox = StyleFromConfig(config.SummaryTitleBox)
	stylesheet.CategoryFallback = StyleFromConfig(config.CategoryFallback)

	return &stylesheet
}
