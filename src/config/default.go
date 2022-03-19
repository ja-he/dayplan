package config

func Default(colorschemeType ColorschemeType) Config {
	return Config{
		Stylesheet: defaultStylesheet(colorschemeType),
		Categories: []Category{
			{Fg: "#000000", Bg: "#cccccc", Name: "default"},
			{Fg: "#000000", Bg: "#ffdccc", Name: "mountainbiking"},
			{Fg: "#000000", Bg: "#c2edab", Name: "a-wandering beneath the clear blue sky"},
			{Fg: "#000000", Bg: "#ccebff", Name: "visiting a china-town section in a major city"},
			{Fg: "#000000", Bg: "#ccffe6", Name: "writing initials in wet cement"},
		},
	}
}

func defaultStylesheet(colorschemeType ColorschemeType) Stylesheet {
	if colorschemeType == Dark {
		return Stylesheet{
			Normal:           Styling{Fg: "#ffffff", Bg: "#000000"},
			WeatherNormal:    Styling{Fg: "#f0f0f0", Bg: "#404040"},
			WeatherSunny:     Styling{Fg: "#fff0cc", Bg: "#734700"},
			WeatherRainy:     Styling{Fg: "#ccebff", Bg: "#0067ab"},
			TimelineDay:      Styling{Fg: "#f0f0f0", Bg: "#000000"},
			TimelineNight:    Styling{Fg: "#f0f0f0", Bg: "#222255"},
			TimelineNow:      Styling{Fg: "#ffffff", Bg: "#cc0000", Bold: true},
			Status:           Styling{Fg: "#f0f0f0", Bg: "#000000"},
			LogDefault:       Styling{Fg: "#ffffff", Bg: "#000000"},
			LogTitleBox:      Styling{Fg: "#f0f0f0", Bg: "#000000", Bold: true},
			LogEntryType:     Styling{Fg: "#c0c0c0", Bg: "#000000", Italic: true},
			LogEntryLocation: Styling{Fg: "#c0c0c0", Bg: "#000000"},
			LogEntryTime:     Styling{Fg: "#808080", Bg: "#000000"},
			Help:             Styling{Fg: "#ffffff", Bg: "#404040"},
			Editor:           Styling{Fg: "#ffffff", Bg: "#404040"},
			SummaryDefault:   Styling{Fg: "#ffffff", Bg: "#000000"},
			SummaryTitleBox:  Styling{Fg: "#f0f0f0", Bg: "#000000", Bold: true},
			CategoryFallback: Styling{Fg: "#882222", Bg: "#ffaaaa"},
		}
	} else {
		return Stylesheet{
			Normal:           Styling{Fg: "#000000", Bg: "#ffffff"},
			WeatherNormal:    Styling{Fg: "#404040", Bg: "#f0f0f0"},
			WeatherSunny:     Styling{Fg: "#000000", Bg: "#fff0cc"},
			WeatherRainy:     Styling{Fg: "#000000", Bg: "#ccebff"},
			TimelineDay:      Styling{Fg: "#c0c0c0", Bg: "#ffffff"},
			TimelineNight:    Styling{Fg: "#f0f0f0", Bg: "#000000"},
			TimelineNow:      Styling{Fg: "#ffffff", Bg: "#ff0000", Bold: true},
			Status:           Styling{Fg: "#000000", Bg: "#f0f0f0"},
			LogDefault:       Styling{Fg: "#000000", Bg: "#ffffff"},
			LogTitleBox:      Styling{Fg: "#000000", Bg: "#f0f0f0", Bold: true},
			LogEntryType:     Styling{Fg: "#cccccc", Bg: "#ffffff", Italic: true},
			LogEntryLocation: Styling{Fg: "#cccccc", Bg: "#ffffff"},
			LogEntryTime:     Styling{Fg: "#f0f0f0", Bg: "#ffffff"},
			Help:             Styling{Fg: "#000000", Bg: "#f0f0f0"},
			Editor:           Styling{Fg: "#000000", Bg: "#f0f0f0"},
			SummaryDefault:   Styling{Fg: "#000000", Bg: "#ffffff"},
			SummaryTitleBox:  Styling{Fg: "#000000", Bg: "#f0f0f0", Bold: true},
			CategoryFallback: Styling{Fg: "#ffaaaa", Bg: "#882222"},
		}
	}
}
