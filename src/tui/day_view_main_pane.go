package tui

import (
	"fmt"

	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

type DayViewMainPane struct {
	renderer *TUIScreenHandler

	dimensions func() (x, y, w, h int)

	events   ui.UIPane
	tools    ui.UIPane
	status   ui.UIPane
	timeline ui.UIPane
	weather  ui.UIPane
}

func (p *DayViewMainPane) Draw() {
	p.events.Draw()

	p.weather.Draw()
	p.timeline.Draw()
	p.tools.Draw()
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
	case ui.EventsUIPanelType:
		return p.events.GetPositionInfo(x, y)
	default:
		return &TUIPositionInfo{
			paneType:       paneType,
			weather:        ui.WeatherPanelPositionInfo{},
			timeline:       ui.TimelinePanelPositionInfo{},
			tools:          ui.ToolsPanelPositionInfo{},
			status:         ui.StatusPanelPositionInfo{},
			events:         ui.EventsPanelPositionInfo{},
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
	x, _, _, _ := t.events.Dimensions()
	timestamp, err := t.events.GetPositionInfo(x, y).GetCursorTimestampGuess()
	if err != nil {
		panic(fmt.Sprint("error guessing timestamp by asking events:", err))
	}
	return *timestamp
}
