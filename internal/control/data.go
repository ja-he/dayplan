package control

import (
	"time"

	"github.com/ja-he/dayplan/internal/control/edit"
	"github.com/ja-he/dayplan/internal/control/edit/editors"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
	"github.com/ja-he/dayplan/internal/weather"
)

// EnvData represents the environment data.
type EnvData struct {
	BaseDirPath string
	OWMAPIKey   string
	Latitude    string
	Longitude   string
}

// PrevView returns the 'previous' view for a given active view, i.E. 'stepping
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

// Next view, for a given active view, returns the 'next', i.E. 'stepping into'
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

// DayWithInfo represents a day with additional information, such as sunrise /
// sunset times.
type DayWithInfo struct {
	Day      *model.EventList
	SunTimes *model.SunTimes
}

type ControlData struct {
	CursorPos ui.MouseCursorPos

	Categories      []model.Category
	CurrentCategory model.Category

	EnvData EnvData

	CurrentDate  model.Date
	CurrentEvent *model.Event
	Weather      weather.Handler

	EventEditor *editors.Composite
	TaskEditor  *editors.Composite

	ShowLog     bool
	ShowHelp    bool
	ShowSummary bool
	ShowDebug   bool

	MainTimelineViewParams ui.SingleDayViewParams

	ActiveView func() ui.ActiveView

	RenderTimes          util.MetricsHandler
	EventProcessingTimes util.MetricsHandler

	MouseMode     bool
	EventEditMode edit.EventEditMode

	MouseEditState            edit.MouseEditState
	MouseEditedEvent          *model.Event
	CurrentMoveStartingOffset time.Duration
}

func NewControlData(cs styling.CategoryStyling) *ControlData {
	var t ControlData

	// TODO: categories go to the provider as well i guess
	t.Categories = make([]model.Category, 0)
	for _, style := range cs.GetAll() {
		t.Categories = append(t.Categories, style.Cat)
	}

	t.MainTimelineViewParams.NRowsPerHour = 6
	t.MainTimelineViewParams.ScrollOffset = 8 * t.MainTimelineViewParams.NRowsPerHour

	return &t
}
