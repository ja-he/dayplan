package styling

// Stylesheet represents all styles used by the application for rendering.
type Stylesheet interface {
	Normal() DrawStyling

	WeatherRegular() DrawStyling
	WeatherSunny() DrawStyling
	WeatherRainy() DrawStyling

	TimelineDay() DrawStyling
	TimelineNight() DrawStyling
	TimelineNow() DrawStyling

	Status() DrawStyling

	CategoryFallback() DrawStyling

	LogDefault() DrawStyling
	LogTitleBox() DrawStyling
	LogEntryType() DrawStyling
	LogEntryLocation() DrawStyling
	LogEntryTime() DrawStyling

	Help() DrawStyling

	Editor() DrawStyling

	SummaryDefault() DrawStyling
	SummaryTitleBox() DrawStyling
}
