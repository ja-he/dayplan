package config

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Stylesheet Stylesheet `yaml:"stylesheet"`
	Categories []Category `yaml:"categories"`
}

type Stylesheet struct {
	Normal           Styling `yaml:"normal"`
	WeatherNormal    Styling `yaml:"weather-normal"`
	WeatherSunny     Styling `yaml:"weather-sunny"`
	WeatherRainy     Styling `yaml:"weather-rainy"`
	TimelineDay      Styling `yaml:"timeline-day"`
	TimelineNight    Styling `yaml:"timeline-night"`
	TimelineNow      Styling `yaml:"timeline-now"`
	Status           Styling `yaml:"status"`
	LogDefault       Styling `yaml:"log-default"`
	LogTitleBox      Styling `yaml:"log-title-box"`
	LogEntryType     Styling `yaml:"log-entry-type"`
	LogEntryLocation Styling `yaml:"log-entry-location"`
	LogEntryTime     Styling `yaml:"log-entry-time"`
	Help             Styling `yaml:"help"`
	Editor           Styling `yaml:"editor"`
	SummaryDefault   Styling `yaml:"summary-default"`
	SummaryTitleBox  Styling `yaml:"summary-title-box"`
	CategoryFallback Styling `yaml:"category-fallback"`
}

type Styling struct {
	Fg    string     `yaml:"fg"`
	Bg    string     `yaml:"bg"`
	Style *FontStyle `yaml:"style"`
}

type FontStyle struct {
	Bold       bool `yaml:"bold,omitempty"`
	Italic     bool `yaml:"italic,omitempty"`
	Underlined bool `yaml:"underlined,omitempty"`
}

type Category struct {
	Name     string `yaml:"name,omitempty"`
	Fg       string `yaml:"fg,omitempty"`
	Bg       string `yaml:"bg,omitempty"`
	Priority int    `yaml:"priority,omitempty"`
}

func ParseConfigAugmentDefaults(defaultTheme ColorschemeType, yamlData []byte) (Config, error) {
	var defaultConfig Config
	switch defaultTheme {
	case Dark:
		defaultConfig = Default(Dark)
	case Light:
		defaultConfig = Default(Light)
	}

	parsedConfig := Config{}
	err := yaml.Unmarshal(yamlData, &parsedConfig)
	if err != nil {
		return defaultConfig, fmt.Errorf("error unmarshaling yaml (%s)", err)
	}

	result := defaultConfig.augmentWith(parsedConfig)

	return result, nil
}

func (base Config) augmentWith(augment Config) Config {
	result := base

	result.Stylesheet = base.Stylesheet.augmentWith(augment.Stylesheet)

	if len(augment.Categories) > 0 {
		result.Categories = augment.Categories
	}

	return result
}

func (base Stylesheet) augmentWith(augment Stylesheet) Stylesheet {
	result := base

	result.Normal.overwriteIfDefined(augment.Normal)
	result.WeatherNormal.overwriteIfDefined(augment.WeatherNormal)
	result.WeatherSunny.overwriteIfDefined(augment.WeatherSunny)
	result.WeatherRainy.overwriteIfDefined(augment.WeatherRainy)
	result.TimelineDay.overwriteIfDefined(augment.TimelineDay)
	result.TimelineNight.overwriteIfDefined(augment.TimelineNight)
	result.TimelineNow.overwriteIfDefined(augment.TimelineNow)
	result.Status.overwriteIfDefined(augment.Status)
	result.LogDefault.overwriteIfDefined(augment.LogDefault)
	result.LogTitleBox.overwriteIfDefined(augment.LogTitleBox)
	result.LogEntryType.overwriteIfDefined(augment.LogEntryType)
	result.LogEntryLocation.overwriteIfDefined(augment.LogEntryLocation)
	result.LogEntryTime.overwriteIfDefined(augment.LogEntryTime)
	result.Help.overwriteIfDefined(augment.Help)
	result.Editor.overwriteIfDefined(augment.Editor)
	result.SummaryDefault.overwriteIfDefined(augment.SummaryDefault)
	result.SummaryTitleBox.overwriteIfDefined(augment.SummaryTitleBox)

	return result
}

func (s *Styling) overwriteIfDefined(augment Styling) {
	if augment.Fg != "" && augment.Bg != "" {
		s.Fg = augment.Fg
		s.Bg = augment.Bg
	}
	if augment.Style != nil {
		s.Style.Bold = augment.Style.Bold
		s.Style.Italic = augment.Style.Italic
		s.Style.Underlined = augment.Style.Underlined
	}
}

type ColorschemeType = int

const (
	_ ColorschemeType = iota
	Dark
	Light
)
