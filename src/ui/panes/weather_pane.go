package panes

import (
	"fmt"

	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/weather"
)

// WeatherPane shows a timeline of hourly weather information blocks at a
// timescale that can be in line with an similarly positioned TimelinePane.
type WeatherPane struct {
	renderer   ui.ConstrainedRenderer
	dimensions func() (x, y, w, h int)
	stylesheet styling.Stylesheet

	weather     *weather.Handler
	currentDate *model.Date
	viewParams  *ui.ViewParams
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *WeatherPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draw draws this pane.
func (p *WeatherPane) Draw() {
	x, y, w, h := p.Dimensions()

	p.renderer.DrawBox(x, y, w, h, p.stylesheet.Normal)

	for timestamp := *model.NewTimestamp("00:00"); timestamp.Legal(); timestamp.Hour++ {
		row := p.toY(timestamp)
		if row >= y+h {
			break
		}

		index := model.DayAndTime{
			Date:      *p.currentDate,
			Timestamp: timestamp,
		}

		weather, ok := p.weather.Data[index]
		if ok {
			weatherStyling := p.stylesheet.WeatherRegular
			switch {
			case weather.PrecipitationProbability > .25:
				weatherStyling = p.stylesheet.WeatherRainy
			case weather.Clouds < 25:
				weatherStyling = p.stylesheet.WeatherSunny
			}

			p.renderer.DrawBox(x, row, w, p.viewParams.NRowsPerHour, weatherStyling)

			p.renderer.DrawText(x, row, w, 1, weatherStyling, weather.Info)
			p.renderer.DrawText(x, row+1, w, 1, weatherStyling, fmt.Sprintf("%2.0f°C", weather.TempC))
			p.renderer.DrawText(x, row+2, w, 1, weatherStyling, fmt.Sprintf("%d%% clouds", weather.Clouds))
			p.renderer.DrawText(x, row+3, w, 1, weatherStyling, fmt.Sprintf("%d%% humidity", weather.Humidity))
			p.renderer.DrawText(x, row+4, w, 1, weatherStyling, fmt.Sprintf("%2.0f%% chance of rain", 100.0*weather.PrecipitationProbability))
		}
	}
}

// TODO: remove
func (p *WeatherPane) toY(ts model.Timestamp) int {
	return ((ts.Hour*p.viewParams.NRowsPerHour - p.viewParams.ScrollOffset) + (ts.Minute / (60 / p.viewParams.NRowsPerHour)))
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *WeatherPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}

// NewWeatherPane constructs and returns a new WeatherPane.
func NewWeatherPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	currentDate *model.Date,
	weather *weather.Handler,
	viewParams *ui.ViewParams,
) *WeatherPane {
	return &WeatherPane{
		renderer:    renderer,
		dimensions:  dimensions,
		stylesheet:  stylesheet,
		currentDate: currentDate,
		weather:     weather,
		viewParams:  viewParams,
	}
}
