package tui

import (
	"fmt"
	"math"

	"github.com/ja-he/dayplan/src/category_style"
	"github.com/ja-he/dayplan/src/colors"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
	"github.com/ja-he/dayplan/src/weather"

	"github.com/gdamore/tcell/v2"
)

type DayViewMainPane struct {
	renderer *TUIRenderer

	dimensions func() (x, y, w, h int)

	tools    ui.UIPane
	status   ui.UIPane
	timeline ui.UIPane

	days        *DaysData
	currentDate *model.Date
	categories  *category_style.CategoryStyling
	logReader   potatolog.LogReader
	logWriter   potatolog.LogWriter
	weather     *weather.Handler
	viewParams  *ViewParams
	cursor      *CursorPos

	// TODO: get rid of this
	positions map[model.EventID]util.Rect
}

func (p *DayViewMainPane) Draw() {
	// TODO
	p.drawWeather()
	// TODO timeline
	p.drawEvents()

	p.tools.Draw()
	p.timeline.Draw()
	p.status.Draw()
}

func (p *DayViewMainPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *DayViewMainPane) GetPositionInfo(x, y int) ui.PositionInfo {
	paneType := p.whichPane(x, y)

	switch paneType {
	case ui.ToolsUIPanelType:
		temp := p.tools.GetPositionInfo(x, y)
		return &TUIPositionTimestampGuessingWrapper{
			baseInfo:       temp,
			timestampGuess: p.TimeAtY(y),
		}
	default:
		return &TUIPositionInfo{
			paneType:       paneType,
			weather:        ui.WeatherPanelPositionInfo{},
			timeline:       ui.TimelinePanelPositionInfo{},
			tools:          ui.ToolsPanelPositionInfo{},
			status:         ui.StatusPanelPositionInfo{},
			events:         p.getEventForPos(x, y),
			timestampGuess: p.TimeAtY(y),
		}
	}
}

func (t *DayViewMainPane) whichPane(x, y int) ui.UIPaneType {
	// TODO: right now, this hardcodes some stuff it shouldn't.
	//       that's a painpoint of trying to keep compilation alive across the
	//       refactoring process; once done, it should be as easy as asking every
	//       pane for dimensions and then delegating to the pane that contains the
	//       position.
	_, _, w, h := t.Dimensions()
	fakeStatusHeight := 2
	mainPaneHeight := h - fakeStatusHeight
	fakeStatusOffset := h - fakeStatusHeight
	fakeWeatherOffset := 0
	fakeWeatherWidth := 20
	fakeTimelineOffset := fakeWeatherOffset + fakeWeatherWidth
	fakeTimelineWidth := 10
	fakeToolsWidth := 20
	fakeToolsOffset := w - fakeToolsWidth
	fakeEventsOffset := fakeTimelineOffset + fakeTimelineWidth
	fakeEventsWidth := w - fakeToolsWidth - fakeEventsOffset

	if x < 0 || y < 0 {
		panic("negative x or y")
	}
	if x > w || y > h {
		panic(fmt.Sprintf("x or y too large (x,y = %d,%d | screen = %d,%d)", x, y, w, h))
	}

	statusPane := util.Rect{X: 0, Y: fakeStatusOffset, W: w, H: fakeStatusHeight}
	weatherPane := util.Rect{X: fakeWeatherOffset, Y: 0, W: fakeWeatherWidth, H: mainPaneHeight}
	timelinePane := util.Rect{X: fakeTimelineOffset, Y: 0, W: fakeTimelineWidth, H: mainPaneHeight}
	eventsPane := util.Rect{X: fakeEventsOffset, Y: 0, W: fakeEventsWidth, H: mainPaneHeight}
	toolsPane := util.Rect{X: fakeToolsOffset, Y: 0, W: fakeToolsWidth, H: mainPaneHeight}

	if statusPane.Contains(x, y) {
		return ui.StatusUIPanelType
	}
	if weatherPane.Contains(x, y) {
		return ui.WeatherUIPanelType
	}
	if timelinePane.Contains(x, y) {
		return ui.TimelineUIPanelType
	}
	if eventsPane.Contains(x, y) {
		return ui.EventsUIPanelType
	}
	if toolsPane.Contains(x, y) {
		return ui.ToolsUIPanelType
	}

	panic(fmt.Sprintf("Unknown UI pos (%d,%d)", x, y))
}

// TODO: remove, this will be part of info returned to controller on query
func (t *DayViewMainPane) TimeAtY(y int) model.Timestamp {
	minutes := y*(60/t.viewParams.NRowsPerHour) + t.viewParams.ScrollOffset*(60/t.viewParams.NRowsPerHour)

	ts := model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}

	return ts
}

func (t *DayViewMainPane) getEventForPos(x, y int) ui.EventsPanelPositionInfo {
	// TODO: this function is here temporarily, excluse the hacks
	_, _, w, _ := t.Dimensions()
	fakeWeatherOffset := 0
	fakeWeatherWidth := 20
	fakeTimelineOffset := fakeWeatherOffset + fakeWeatherWidth
	fakeTimelineWidth := 10
	fakeToolsWidth := 20
	fakeEventsOffset := fakeTimelineOffset + fakeTimelineWidth
	fakeEventsWidth := w - fakeToolsWidth - fakeEventsOffset

	if x >= fakeEventsOffset &&
		x < (fakeEventsOffset+fakeEventsWidth) {
		currentDay := t.days.GetDay(*t.currentDate)
		for i := len(currentDay.Events) - 1; i >= 0; i-- {
			eventPos := t.positions[currentDay.Events[i].ID]
			if eventPos.Contains(x, y) {
				var hover ui.EventHoverState
				switch {
				case y == (eventPos.Y+eventPos.H-1) && x > eventPos.X+eventPos.W-5:
					hover = ui.EventHoverStateResize
				case y == (eventPos.Y):
					hover = ui.EventHoverStateEdit
				default:
					hover = ui.EventHoverStateMove
				}
				return ui.EventsPanelPositionInfo{
					Event:           currentDay.Events[i].ID,
					HoverState:      hover,
					TimeUnderCursor: t.TimeAtY(y),
				}
			}
		}
	}
	return ui.EventsPanelPositionInfo{
		Event:           0,
		HoverState:      ui.EventHoverStateNone,
		TimeUnderCursor: t.TimeAtY(y),
	}
}

func (t *DayViewMainPane) toY(ts model.Timestamp) int {
	return ((ts.Hour*t.viewParams.NRowsPerHour - t.viewParams.ScrollOffset) + (ts.Minute / (60 / t.viewParams.NRowsPerHour)))
}

func (t *DayViewMainPane) drawEvents() {
	// TODO
	_, _, w, h := t.Dimensions()
	fakeWeatherOffset := 0
	fakeWeatherWidth := 20
	fakeTimelineOffset := fakeWeatherOffset + fakeWeatherWidth
	fakeTimelineWidth := 10
	fakeToolsWidth := 20
	fakeEventsOffset := fakeTimelineOffset + fakeTimelineWidth
	fakeEventsWidth := w - fakeToolsWidth - fakeEventsOffset

	day := t.days.GetDay(*t.currentDate)
	if day == nil {
		t.logWriter.Add("DEBUG", "current day nil on render; skipping")
		return
	}
	t.positions = t.ComputeRects(day, fakeEventsOffset, 0, fakeEventsWidth-2, h)
	for _, e := range day.Events {
		style, err := t.categories.GetStyle(e.Cat)
		if err != nil {
			t.logWriter.Add("ERROR", err.Error())
		}
		// based on event state, draw a box or maybe a smaller one, or ...
		p := t.positions[e.ID]
		hovered := t.getEventForPos(t.cursor.X, t.cursor.Y)
		if hovered.Event != e.ID {
			t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H)
			t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, style, util.TruncateAt(e.Name, p.W-7))
			t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
			t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
		} else {
			selStyle := colors.DefaultEmphasize(style)
			switch hovered.HoverState {
			case ui.EventHoverStateResize:
				t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H-1)
				t.renderer.DrawBox(selStyle, p.X, p.Y+p.H-1, p.W, 1)
				t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, style, util.TruncateAt(e.Name, p.W-7))
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			case ui.EventHoverStateMove:
				t.renderer.DrawBox(selStyle, p.X, p.Y, p.W, p.H)
				t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, selStyle, util.TruncateAt(e.Name, p.W-7))
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, selStyle, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			case ui.EventHoverStateEdit:
				t.renderer.DrawBox(style, p.X, p.Y, p.W, p.H)
				t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, selStyle, util.TruncateAt(e.Name, p.W-7))
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
			default:
				panic("don't know this hover state!")
			}
		}
	}
}

func (t *DayViewMainPane) ComputeRects(day *model.Day, offsetX, offsetY, width, height int) map[model.EventID]util.Rect {
	active_stack := make([]model.Event, 0)
	positions := make(map[model.EventID]util.Rect)
	for _, e := range day.Events {
		// remove all stacked elements that have finished
		for i := len(active_stack) - 1; i >= 0; i-- {
			if e.Start.IsAfter(active_stack[i].End) || e.Start == active_stack[i].End {
				active_stack = active_stack[:i]
			} else {
				break
			}
		}
		active_stack = append(active_stack, e)
		// based on event state, draw a box or maybe a smaller one, or ...
		y := t.toY(e.Start) + offsetY
		x := offsetX
		h := t.toY(e.End) + offsetY - y
		w := width

		// scale the width by 3/4 for every extra item on the stack, so for one
		// item stacked underneath the current items width will be (3/4) ** 1 = 75%
		// of the original width, for four it would be (3/4) ** 4 = (3**4)/(4**4)
		// or 31.5 % of the width, etc.
		widthFactor := 0.75
		w = int(float64(w) * math.Pow(widthFactor, float64(len(active_stack)-1)))
		x += (width - w)

		positions[e.ID] = util.Rect{X: x, Y: y, W: w, H: h}
	}
	return positions
}

func (t *DayViewMainPane) drawWeather() {
	// TODO: right now, this hardcodes some stuff it shouldn't.
	fakeWeatherOffset := 0
	fakeWeatherWidth := 20

	for timestamp := *model.NewTimestamp("00:00"); timestamp.Legal(); timestamp.Hour++ {
		y := t.toY(timestamp)

		index := model.DayAndTime{
			Date:      *t.currentDate,
			Timestamp: timestamp,
		}

		weather, ok := t.weather.Data[index]
		if ok {
			weatherStyle := tcell.StyleDefault.Foreground(tcell.ColorLightBlue)
			switch {
			case weather.PrecipitationProbability > .25:
				weatherStyle = weatherStyle.Background(tcell.NewHexColor(0xccebff)).Foreground(tcell.ColorBlack)
			case weather.Clouds < 25:
				weatherStyle = weatherStyle.Background(tcell.NewHexColor(0xfff0cc)).Foreground(tcell.ColorBlack)
			}

			t.renderer.DrawBox(weatherStyle, fakeWeatherOffset, y, fakeWeatherWidth, t.viewParams.NRowsPerHour)

			t.renderer.DrawText(fakeWeatherOffset, y, fakeWeatherWidth, 0, weatherStyle, weather.Info)
			t.renderer.DrawText(fakeWeatherOffset, y+1, fakeWeatherWidth, 0, weatherStyle, fmt.Sprintf("%2.0f°C", weather.TempC))
			t.renderer.DrawText(fakeWeatherOffset, y+2, fakeWeatherWidth, 0, weatherStyle, fmt.Sprintf("%d%% clouds", weather.Clouds))
			t.renderer.DrawText(fakeWeatherOffset, y+3, fakeWeatherWidth, 0, weatherStyle, fmt.Sprintf("%d%% humidity", weather.Humidity))
			t.renderer.DrawText(fakeWeatherOffset, y+4, fakeWeatherWidth, 0, weatherStyle, fmt.Sprintf("%2.0f%% chance of rain", 100.0*weather.PrecipitationProbability))
		}
	}
}
