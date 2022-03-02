package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/weather"
)

type WeatherPane struct {
	renderer   ui.ConstrainedRenderer
	dimensions func() (x, y, w, h int)

	weather     *weather.Handler
	currentDate *model.Date
	viewParams  *ViewParams
}

func (p *WeatherPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// TODO: pretty sure this doesn't respect dimensions currently
func (p *WeatherPane) Draw() {
	x, y, w, h := p.Dimensions()

	// TODO: these are temporarily still hardcoded, will be moved with
	//       customizable styling being implemented
	backgroundStyling := NewStyling(tcell.ColorBlack, tcell.ColorWhite)
	regularStyling := NewStyling(tcell.ColorLightBlue, tcell.ColorWhite)
	rainyStyling := NewStyling(tcell.ColorBlack, tcell.NewHexColor(0xccebff))
	sunnyStyling := NewStyling(tcell.ColorBlack, tcell.NewHexColor(0xfff0cc))

	p.renderer.DrawBox(x, y, w, h, backgroundStyling)

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
			weatherStyling := regularStyling
			switch {
			case weather.PrecipitationProbability > .25:
				weatherStyling = rainyStyling
			case weather.Clouds < 25:
				weatherStyling = sunnyStyling
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
func (t *WeatherPane) toY(ts model.Timestamp) int {
	return ((ts.Hour*t.viewParams.NRowsPerHour - t.viewParams.ScrollOffset) + (ts.Minute / (60 / t.viewParams.NRowsPerHour)))
}

func (p *WeatherPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}
