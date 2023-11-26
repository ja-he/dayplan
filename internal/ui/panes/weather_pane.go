package panes

import (
	"fmt"
	"time"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/weather"
)

// WeatherPane shows a timeline of hourly weather information blocks at a
// timescale that can be in line with an similarly positioned TimelinePane.
type WeatherPane struct {
	ui.LeafPane

	weather     *weather.Handler
	currentDate *model.Date
	viewParams  ui.TimespanViewParams
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *WeatherPane) Dimensions() (x, y, w, h int) {
	return p.Dims()
}

// Draw draws this pane.
func (p *WeatherPane) Draw() {
	x, y, w, h := p.Dimensions()

	p.Renderer.DrawBox(x, y, w, h, p.Stylesheet.Normal)

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
			weatherStyling := p.Stylesheet.WeatherNormal
			switch {
			case weather.PrecipitationProbability > .25:
				weatherStyling = p.Stylesheet.WeatherRainy
			case weather.Clouds < 25:
				weatherStyling = p.Stylesheet.WeatherSunny
			}

			p.Renderer.DrawBox(x, row, w, int(p.viewParams.HeightOfDuration(time.Hour)), weatherStyling)

			p.Renderer.DrawText(x, row, w, 1, weatherStyling, weather.Info)
			p.Renderer.DrawText(x, row+1, w, 1, weatherStyling, fmt.Sprintf("%2.0f°C", weather.TempC))
			p.Renderer.DrawText(x, row+2, w, 1, weatherStyling, fmt.Sprintf("%d%% clouds", weather.Clouds))
			p.Renderer.DrawText(x, row+3, w, 1, weatherStyling, fmt.Sprintf("%d%% humidity", weather.Humidity))
			p.Renderer.DrawText(x, row+4, w, 1, weatherStyling, fmt.Sprintf("%2.0f%% chance of rain", 100.0*weather.PrecipitationProbability))
		}
	}
}

// TODO: remove
func (p *WeatherPane) toY(ts model.Timestamp) int {
	return ((ts.Hour*int(p.viewParams.HeightOfDuration(time.Hour)) - p.viewParams.GetScrollOffset()) + (ts.Minute / (60 / int(p.viewParams.HeightOfDuration(time.Hour)))))
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
	viewParams ui.TimespanViewParams,
) *WeatherPane {
	return &WeatherPane{
		LeafPane: ui.LeafPane{
			BasePane: ui.BasePane{
				ID: ui.GeneratePaneID(),
			},
			Renderer:   renderer,
			Dims:       dimensions,
			Stylesheet: stylesheet,
		},
		currentDate: currentDate,
		weather:     weather,
		viewParams:  viewParams,
	}
}
