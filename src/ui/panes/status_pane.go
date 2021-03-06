package panes

import (
	"github.com/ja-he/dayplan/src/control"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

// StatusPane is a status bar that displays the current date, weekday, and - if
// in a multi-day view - the progress through those days.
type StatusPane struct {
	Leaf

	currentDate *model.Date

	dayWidth           func() int
	totalDaysInPeriod  func() int
	passedDaysInPeriod func() int
	firstDayXOffset    func() int

	eventEditMode func() control.EventEditMode
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *StatusPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// Draw draws this pane.
func (p *StatusPane) Draw() {
	x, y, w, h := p.dimensions()

	dateWidth := 10 // 2020-02-12 is 10 wide

	bgStyle := p.stylesheet.Status
	bgStyleEmph := bgStyle.DefaultEmphasized()
	dateStyle := bgStyleEmph
	weekdayStyle := dateStyle.LightenedFG(60)

	// header background
	p.renderer.DrawBox(0, y, p.firstDayXOffset()+p.totalDaysInPeriod()*p.dayWidth(), h, bgStyle)
	// header bar (filled for days until current)
	p.renderer.DrawBox(0, y, p.firstDayXOffset()+(p.passedDaysInPeriod())*p.dayWidth(), h, bgStyleEmph)
	// date box background
	p.renderer.DrawBox(0, y, dateWidth, h, bgStyleEmph)
	// date string
	p.renderer.DrawText(0, y, dateWidth, 1, dateStyle, p.currentDate.ToString())
	// weekday string
	p.renderer.DrawText(0, y+1, dateWidth, 1, weekdayStyle, util.TruncateAt(p.currentDate.ToWeekday().String(), dateWidth))

	// mode string
	modeStr := eventEditModeToString(p.eventEditMode())
	p.renderer.DrawText(x+w-len(modeStr)-2, y+h-1, len(modeStr), 1, bgStyleEmph.DarkenedBG(10).Italicized(), modeStr)
}

func eventEditModeToString(mode control.EventEditMode) string {
	switch mode {
	case control.EventEditModeNormal:
		return "-- NORMAL --"
	case control.EventEditModeMove:
		return "--  MOVE  --"
	case control.EventEditModeResize:
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
	eventEditMode func() control.EventEditMode,
) *StatusPane {
	return &StatusPane{
		Leaf: Leaf{
			Base: Base{
				ID: ui.GeneratePaneID(),
			},
			renderer:   renderer,
			dimensions: dimensions,
			stylesheet: stylesheet,
		},
		currentDate:        currentDate,
		dayWidth:           dayWidth,
		totalDaysInPeriod:  totalDaysInPeriod,
		passedDaysInPeriod: passedDaysInPeriod,
		firstDayXOffset:    firstDayXOffset,
		eventEditMode:      eventEditMode,
	}
}
