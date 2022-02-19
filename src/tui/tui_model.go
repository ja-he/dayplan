package tui

import (
	"fmt"
	"sync"

	"github.com/ja-he/dayplan/src/category_style"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/program"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
	"github.com/ja-he/dayplan/src/weather"
)

// For a given active view, returns the 'previous', i.E. 'stepping
// out' from an inner view to an outer one.
// E.g.: Day -> Week -> Month
func PrevView(current ui.ActiveView) ui.ActiveView {
	switch current {
	case ui.ViewDay:
		return ui.ViewWeek
	case ui.ViewWeek:
		return ui.ViewMonth
	case ui.ViewMonth:
		return ui.ViewMonth
	default:
		panic("unknown view!")
	}
}

// For a given active view, returns the 'next', i.E. 'stepping into'
// from an outer view to an inner one.
// E.g.: Month -> Week -> Day
func NextView(current ui.ActiveView) ui.ActiveView {
	switch current {
	case ui.ViewDay:
		return ui.ViewDay
	case ui.ViewWeek:
		return ui.ViewDay
	case ui.ViewMonth:
		return ui.ViewWeek
	default:
		panic("unknown view!")
	}
}

// Returns the active view name as a string.
func toString(av ui.ActiveView) string {
	switch av {
	case ui.ViewDay:
		return "ui.ViewDay"
	case ui.ViewWeek:
		return "ui.ViewWeek"
	case ui.ViewMonth:
		return "ui.ViewMonth"
	default:
		return "unknown"
	}
}

type UIDims struct {
	weatherOffset, timelineOffset, eventsOffset, toolsOffset int
	statusHeight                                             int
	screenWidth, screenHeight                                int
}

func (d *UIDims) WhichUIPane(x, y int) ui.UIPaneType {
	if x < 0 || y < 0 {
		panic("negative x or y")
	}
	if x > d.screenWidth || y > d.screenHeight {
		panic(fmt.Sprintf("x or y too large (x,y = %d,%d | screen = %d,%d)", x, y, d.screenWidth, d.screenHeight))
	}

	mainPaneHeight := d.screenHeight - d.StatusHeight()

	statusPane := util.Rect{X: 0, Y: d.StatusOffset(), W: d.screenWidth, H: d.StatusHeight()}
	weatherPane := util.Rect{X: d.WeatherOffset(), Y: 0, W: d.WeatherWidth(), H: mainPaneHeight}
	timelinePane := util.Rect{X: d.TimelineOffset(), Y: 0, W: d.TimelineWidth(), H: mainPaneHeight}
	eventsPane := util.Rect{X: d.EventsOffset(), Y: 0, W: d.EventsWidth(), H: mainPaneHeight}
	toolsPane := util.Rect{X: d.ToolsOffset(), Y: 0, W: d.ToolsWidth(), H: mainPaneHeight}

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

func (d *UIDims) ScreenSize() (int, int) {
	return d.screenWidth, d.screenHeight
}

func (d *UIDims) ScreenResize(width, height int) {
	if height <= d.statusHeight {
		panic(fmt.Sprintf("screensize of %d too little with statusline height of %d", height, d.statusHeight))
	}

	toolsWidth := d.ToolsWidth()
	d.toolsOffset = width - toolsWidth
	if d.toolsOffset >= width {
		panic("offset > width")
	}

	d.screenWidth = width
	d.screenHeight = height
}

func (d *UIDims) Initialize(weatherWidth, timelineWidth, toolsWidth int,
	screenWidth, screenHeight int) {
	d.weatherOffset = 0
	d.timelineOffset = d.weatherOffset + weatherWidth
	d.eventsOffset = d.timelineOffset + timelineWidth
	d.toolsOffset = screenWidth - toolsWidth
	d.statusHeight = 2
	d.screenWidth = screenWidth
	d.screenHeight = screenHeight
}

func (ui *UIDims) WeatherOffset() int  { return ui.weatherOffset }
func (ui *UIDims) WeatherWidth() int   { return ui.timelineOffset - ui.weatherOffset }
func (ui *UIDims) TimelineOffset() int { return ui.timelineOffset }
func (ui *UIDims) TimelineWidth() int  { return ui.eventsOffset - ui.timelineOffset }
func (ui *UIDims) EventsOffset() int   { return ui.eventsOffset }
func (ui *UIDims) EventsWidth() int    { return (ui.toolsOffset - ui.eventsOffset) }
func (ui *UIDims) ToolsOffset() int    { return ui.toolsOffset }
func (ui *UIDims) ToolsWidth() int     { return ui.screenWidth - ui.ToolsOffset() }
func (ui *UIDims) StatusHeight() int   { return ui.statusHeight }
func (ui *UIDims) StatusOffset() int   { return ui.screenHeight - ui.statusHeight }

type DayWithInfo struct {
	Day      *model.Day
	SunTimes *model.SunTimes
}

type ViewParams struct {
	NRowsPerHour int
	ScrollOffset int
}

type CursorPos struct {
	X, Y int
}

type TUIModel struct {
	// TODO: remove from here
	UIDim *UIDims

	cursorPos CursorPos

	CategoryStyling category_style.CategoryStyling

	ProgramData program.Data

	Days        DaysData
	CurrentDate model.Date
	Weather     weather.Handler

	EventEditor EventEditor
	showLog     bool
	showHelp    bool
	showSummary bool

	ViewParams ViewParams

	CurrentCategory model.Category
	activeView      ui.ActiveView

	Log potatolog.Log
}

type DaysData struct {
	daysMutex sync.RWMutex
	days      map[model.Date]DayWithInfo
}

func (t *TUIModel) ScrollUp(by int) {
	eventviewTopRow := 0
	if t.ViewParams.ScrollOffset-by >= eventviewTopRow {
		t.ViewParams.ScrollOffset -= by
	} else {
		t.ScrollTop()
	}
}

func (t *TUIModel) ScrollDown(by int) {
	eventviewBottomRow := t.UIDim.screenHeight - t.UIDim.statusHeight
	if t.ViewParams.ScrollOffset+by+eventviewBottomRow <= (24 * t.ViewParams.NRowsPerHour) {
		t.ViewParams.ScrollOffset += by
	} else {
		t.ScrollBottom()
	}
}

func (t *TUIModel) ScrollTop() {
	t.ViewParams.ScrollOffset = 0
}

func (t *TUIModel) ScrollBottom() {
	eventviewBottomRow := t.UIDim.screenHeight - t.UIDim.statusHeight
	t.ViewParams.ScrollOffset = 24*t.ViewParams.NRowsPerHour - eventviewBottomRow
}

func NewTUIModel(cs category_style.CategoryStyling) *TUIModel {
	var t TUIModel

	t.Days = DaysData{
		days: make(map[model.Date]DayWithInfo),
	}

	t.CategoryStyling = cs

	t.ViewParams.NRowsPerHour = 6
	t.ViewParams.ScrollOffset = 8 * t.ViewParams.NRowsPerHour

	t.activeView = ui.ViewDay

	return &t
}

func (d *DaysData) HasDay(date model.Date) bool {
	d.daysMutex.RLock()
	defer d.daysMutex.RUnlock()
	_, ok := d.days[date]
	return ok
}

func (t *TUIModel) GetCurrentDay() *model.Day {
	return t.Days.GetDay(t.CurrentDate)
}

// Get the suntimes of the current date of the model.
func (t *TUIModel) GetCurrentSuntimes() *model.SunTimes {
	return t.Days.GetSuntimes(t.CurrentDate)
}

// Get the suntimes of the provided date of the model.
func (d *DaysData) GetSuntimes(date model.Date) *model.SunTimes {
	d.daysMutex.RLock()
	defer d.daysMutex.RUnlock()
	return d.days[date].SunTimes
}

// Get the day of the provided date of the model.
func (d *DaysData) GetDay(date model.Date) *model.Day {
	d.daysMutex.RLock()
	defer d.daysMutex.RUnlock()
	return d.days[date].Day
}

func (d *DaysData) AddDay(date model.Date, day *model.Day, suntimes *model.SunTimes) {
	if day == nil {
		panic("will not add a nil model")
	}

	d.daysMutex.Lock()
	defer d.daysMutex.Unlock()
	d.days[date] = DayWithInfo{day, suntimes}
}
