package panes

import (
	"github.com/ja-he/dayplan/internal/control/edit"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
)

// StatusPane is a status bar that displays the current date, weekday, and - if
// in a multi-day view - the progress through those days.
type StatusPane struct {
	ui.LeafPane

	currentDate *model.Date

	dayWidth           func() int
	totalDaysInPeriod  func() int
	passedDaysInPeriod func() int
	firstDayXOffset    func() int

	eventEditMode func() edit.EventEditMode
}

// Draw draws this pane.
func (p *StatusPane) Draw() {
	x, y, w, h := p.Dimensions()

	dateWidth := 10 // 2020-02-12 is 10 wide

	bgStyle := p.Stylesheet.Status
	bgStyleEmph := bgStyle.DefaultEmphasized()
	dateStyle := bgStyleEmph
	weekdayStyle := dateStyle.LightenedFG(60)

	// header background
	p.Renderer.DrawBox(0, y, p.firstDayXOffset()+p.totalDaysInPeriod()*p.dayWidth(), h, bgStyle)
	// header bar (filled for days until current)
	p.Renderer.DrawBox(0, y, p.firstDayXOffset()+(p.passedDaysInPeriod())*p.dayWidth(), h, bgStyleEmph)
	// date box background
	p.Renderer.DrawBox(0, y, dateWidth, h, bgStyleEmph)
	// date string
	p.Renderer.DrawText(0, y, dateWidth, 1, dateStyle, p.currentDate.ToString())
	// weekday string
	p.Renderer.DrawText(0, y+1, dateWidth, 1, weekdayStyle, util.TruncateAt(p.currentDate.ToWeekday().String(), dateWidth))

	// mode string
	modeStr := eventEditModeToString(p.eventEditMode())
	p.Renderer.DrawText(x+w-len(modeStr)-2, y+h-1, len(modeStr), 1, bgStyleEmph.DarkenedBG(10).Italicized(), modeStr)
}

func eventEditModeToString(mode edit.EventEditMode) string {
	switch mode {
	case edit.EventEditModeNormal:
		return "-- NORMAL --"
	case edit.EventEditModeMove:
		return "--  MOVE  --"
	case edit.EventEditModeResize:
		return "-- RESIZE --"
	default:
		return "unknown"
	}
}

// GetPositionInfo returns information on a requested position in this pane.
func (p *StatusPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return nil
}

// NewStatusPane constructs and returns a new StatusPane.
func NewStatusPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	currentDate *model.Date,
	dayWidth func() int,
	totalDaysInPeriod func() int,
	passedDaysInPeriod func() int,
	firstDayXOffset func() int,
	eventEditMode func() edit.EventEditMode,
) *StatusPane {
	return &StatusPane{
		LeafPane: ui.LeafPane{
			BasePane: ui.BasePane{
				ID: ui.GeneratePaneID(),
			},
			Renderer:   renderer,
			Dims:       dimensions,
			Stylesheet: stylesheet,
		},
		currentDate:        currentDate,
		dayWidth:           dayWidth,
		totalDaysInPeriod:  totalDaysInPeriod,
		passedDaysInPeriod: passedDaysInPeriod,
		firstDayXOffset:    firstDayXOffset,
		eventEditMode:      eventEditMode,
	}
}
