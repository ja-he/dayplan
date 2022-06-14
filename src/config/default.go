package config

// Default returns the default colorscheme for the given type (light or dark).
func Default(colorschemeType ColorschemeType) Config {
	return Config{
		Stylesheet: defaultStylesheet(colorschemeType),
		Categories: []Category{
			{Bg: "#cccccc", Name: "default"},
			{Bg: "#ffdccc", Name: "mountainbiking"},
			{Bg: "#c2edab", Name: "a-wandering beneath the clear blue sky"},
			{Bg: "#ccebff", Name: "visiting a china-town section in a major city"},
			{Bg: "#ccffe6", Name: "writing initials in wet cement"},
		},
	}
}

func defaultStylesheet(colorschemeType ColorschemeType) Stylesheet {
	if colorschemeType == Dark {
		return Stylesheet{
			Normal:           Styling{Fg: "#ffffff", Bg: "#000000", Style: &FontStyle{}},
			NormalEmphasized: Styling{Fg: "#ffffff", Bg: "#202020", Style: &FontStyle{}},
			WeatherNormal:    Styling{Fg: "#f0f0f0", Bg: "#404040", Style: &FontStyle{}},
			WeatherSunny:     Styling{Fg: "#fff0cc", Bg: "#734700", Style: &FontStyle{}},
			WeatherRainy:     Styling{Fg: "#ccebff", Bg: "#0067ab", Style: &FontStyle{}},
			TimelineDay:      Styling{Fg: "#f0f0f0", Bg: "#000000", Style: &FontStyle{}},
			TimelineNight:    Styling{Fg: "#f0f0f0", Bg: "#222255", Style: &FontStyle{}},
			TimelineNow:      Styling{Fg: "#ffffff", Bg: "#cc0000", Style: &FontStyle{Bold: true}},
			Status:           Styling{Fg: "#f0f0f0", Bg: "#000000", Style: &FontStyle{}},
			LogDefault:       Styling{Fg: "#ffffff", Bg: "#000000", Style: &FontStyle{}},
			LogTitleBox:      Styling{Fg: "#f0f0f0", Bg: "#000000", Style: &FontStyle{Bold: true}},
			LogEntryType:     Styling{Fg: "#c0c0c0", Bg: "#000000", Style: &FontStyle{Italic: true}},
			LogEntryLocation: Styling{Fg: "#c0c0c0", Bg: "#000000", Style: &FontStyle{}},
			LogEntryTime:     Styling{Fg: "#808080", Bg: "#000000", Style: &FontStyle{}},
			Help:             Styling{Fg: "#ffffff", Bg: "#404040", Style: &FontStyle{}},
			Editor:           Styling{Fg: "#ffffff", Bg: "#606060", Style: &FontStyle{}},
			SummaryDefault:   Styling{Fg: "#ffffff", Bg: "#000000", Style: &FontStyle{}},
			SummaryTitleBox:  Styling{Fg: "#f0f0f0", Bg: "#000000", Style: &FontStyle{Bold: true}},
			CategoryFallback: Styling{Fg: "#882222", Bg: "#ffaaaa", Style: &FontStyle{}},
		}
	} else {
		return Stylesheet{
			Normal:           Styling{Fg: "#000000", Bg: "#ffffff", Style: &FontStyle{}},
			NormalEmphasized: Styling{Fg: "#000000", Bg: "#f0f0f0", Style: &FontStyle{}},
			WeatherNormal:    Styling{Fg: "#404040", Bg: "#f0f0f0", Style: &FontStyle{}},
			WeatherSunny:     Styling{Fg: "#000000", Bg: "#fff0cc", Style: &FontStyle{}},
			WeatherRainy:     Styling{Fg: "#000000", Bg: "#ccebff", Style: &FontStyle{}},
			TimelineDay:      Styling{Fg: "#c0c0c0", Bg: "#ffffff", Style: &FontStyle{}},
			TimelineNight:    Styling{Fg: "#f0f0f0", Bg: "#000000", Style: &FontStyle{}},
			TimelineNow:      Styling{Fg: "#ffffff", Bg: "#ff0000", Style: &FontStyle{Bold: true}},
			Status:           Styling{Fg: "#000000", Bg: "#f0f0f0", Style: &FontStyle{}},
			LogDefault:       Styling{Fg: "#000000", Bg: "#ffffff", Style: &FontStyle{}},
			LogTitleBox:      Styling{Fg: "#000000", Bg: "#f0f0f0", Style: &FontStyle{Bold: true}},
			LogEntryType:     Styling{Fg: "#cccccc", Bg: "#ffffff", Style: &FontStyle{Italic: true}},
			LogEntryLocation: Styling{Fg: "#cccccc", Bg: "#ffffff", Style: &FontStyle{}},
			LogEntryTime:     Styling{Fg: "#f0f0f0", Bg: "#ffffff", Style: &FontStyle{}},
			Help:             Styling{Fg: "#000000", Bg: "#f0f0f0", Style: &FontStyle{}},
			Editor:           Styling{Fg: "#000000", Bg: "#cccccc", Style: &FontStyle{}},
			SummaryDefault:   Styling{Fg: "#000000", Bg: "#ffffff", Style: &FontStyle{}},
			SummaryTitleBox:  Styling{Fg: "#000000", Bg: "#f0f0f0", Style: &FontStyle{Bold: true}},
			CategoryFallback: Styling{Fg: "#ffaaaa", Bg: "#882222", Style: &FontStyle{}},
		}
	}
}
