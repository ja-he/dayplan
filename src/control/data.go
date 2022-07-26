package control

import (
	"sync"

	"github.com/ja-he/dayplan/src/control/editor"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
	"github.com/ja-he/dayplan/src/weather"
)

type EnvData struct {
	BaseDirPath string
	OwmApiKey   string
	Latitude    string
	Longitude   string
}

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

type DayWithInfo struct {
	Day      *model.Day
	SunTimes *model.SunTimes
}

type ControlData struct {
	CursorPos ui.MouseCursorPos

	Categories      []model.Category
	CurrentCategory model.Category

	EnvData EnvData

	Days        DaysData
	CurrentDate model.Date
	Weather     weather.Handler

	EventEditor editor.EventEditor
	ShowLog     bool
	ShowHelp    bool
	ShowSummary bool
	ShowDebug   bool

	ViewParams ui.ViewParams

	ActiveView func() ui.ActiveView

	RenderTimes          util.MetricsHandler
	EventProcessingTimes util.MetricsHandler

	MouseMode     bool
	EventEditMode EventEditMode

	MouseEditState                   MouseEditState
	MouseEditedEvent                 *model.Event
	CurrentMoveStartingOffsetMinutes int
}

type MouseEditState int

const (
	_ MouseEditState = iota
	MouseEditStateNone
	MouseEditStateMoving
	MouseEditStateResizing
)

func (s MouseEditState) toString() string {
	return "TODO"
}

type EventEditMode = int

const (
	_ EventEditMode = iota
	EventEditModeNormal
	EventEditModeMove
	EventEditModeResize
)

type DaysData struct {
	daysMutex sync.RWMutex
	days      map[model.Date]DayWithInfo
}

func NewControlData(cs styling.CategoryStyling) *ControlData {
	var t ControlData

	t.Days = DaysData{
		days: make(map[model.Date]DayWithInfo),
	}

	t.Categories = make([]model.Category, 0)
	for _, style := range cs.GetAll() {
		t.Categories = append(t.Categories, style.Cat)
	}

	t.ViewParams.NRowsPerHour = 6
	t.ViewParams.ScrollOffset = 8 * t.ViewParams.NRowsPerHour

	return &t
}

func (d *DaysData) HasDay(date model.Date) bool {
	d.daysMutex.RLock()
	defer d.daysMutex.RUnlock()
	_, ok := d.days[date]
	return ok
}

func (t *ControlData) GetCurrentDay() *model.Day {
	return t.Days.GetDay(t.CurrentDate)
}

// Get the suntimes of the current date of the model.
func (t *ControlData) GetCurrentSuntimes() *model.SunTimes {
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
