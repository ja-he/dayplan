package panes

import (
	"fmt"

	"github.com/ja-he/dayplan/internal/control/edit"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/provider"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// StatusPane is a status bar that displays the current date, weekday, and - if
// in a multi-day view - the progress through those days.
type StatusPane struct {
	ui.LeafPane

	getCurrentDate func() model.Date

	dayWidth           func() int
	totalDaysInPeriod  func() int
	passedDaysInPeriod func() int
	firstDayXOffset    func() int

	eventEditMode       func() edit.EventEditMode
	storageProviderInfo provider.DataProviderInfo

	log zerolog.Logger
}

// Draw draws this pane.
func (p *StatusPane) Draw() {
	x, y, w, h := p.Dimensions()

	dateWidth := 10 // 2020-02-12 is 10 wide

	bgStyle := p.Stylesheet.Status
	bgStyleEmph := bgStyle.DefaultEmphasized()
	dateStyle := bgStyleEmph
	weekdayStyle := dateStyle.LightenedFG(60)

	currentDate := p.getCurrentDate()

	// header background
	p.Renderer.DrawBox(0, y, p.firstDayXOffset()+p.totalDaysInPeriod()*p.dayWidth(), h, bgStyle)
	// header bar (filled for days until current)
	p.Renderer.DrawBox(0, y, p.firstDayXOffset()+(p.passedDaysInPeriod())*p.dayWidth(), h, bgStyleEmph)
	// date box background
	p.Renderer.DrawBox(0, y, dateWidth, h, bgStyleEmph)
	// date string
	p.Renderer.DrawText(0, y, dateWidth, 1, dateStyle, currentDate.String())
	// weekday string
	p.Renderer.DrawText(0, y+1, dateWidth, 1, weekdayStyle, util.TruncateAt(currentDate.ToWeekday().String(), dateWidth))

	// mode string
	modeStr := eventEditModeToString(p.eventEditMode())
	p.Renderer.DrawText(x+w-len(modeStr)-2, y+h-1, len(modeStr), 1, bgStyleEmph.DarkenedBG(10).Italicized(), modeStr)

	storageInfoStr, err := p.storageProviderInfo.GetStorageLocationInfo()
	if err != nil {
		p.log.Error().Err(err).Msg("could not get storage location info")
		storageInfoStr = fmt.Sprintf("err:%s", err.Error())
	}
	storageInfoStrWAllowance := w - dateWidth - 20 - 10 // rough guess
	if storageInfoStrWAllowance > 5 {
		storageInfoStr = util.TruncateAt(storageInfoStr, storageInfoStrWAllowance)
		p.Renderer.DrawText(x+dateWidth+5, y, storageInfoStrWAllowance, 1, bgStyleEmph.DarkenedFG(20), storageInfoStr)
	}
	storageFullyCommitted, err := p.storageProviderInfo.FullyCommitted()
	var storageFullyCommittedStr string
	if err != nil {
		p.log.Error().Err(err).Msg("could not get whether-committed info")
		storageFullyCommittedStr = fmt.Sprintf("err:%s", err.Error())
	} else if storageFullyCommitted {
		storageFullyCommittedStr = "fully committed"
	} else {
		storageFullyCommittedStr = "not fully committed"
	}
	p.Renderer.DrawText(x+dateWidth+5, y+h-1, storageInfoStrWAllowance, 1, bgStyle.DarkenedFG(30).Italicized(), storageFullyCommittedStr)
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
	getCurrentDate func() model.Date,
	dayWidth func() int,
	totalDaysInPeriod func() int,
	passedDaysInPeriod func() int,
	firstDayXOffset func() int,
	eventEditMode func() edit.EventEditMode,
	storageProviderInfo provider.DataProviderInfo,
) *StatusPane {
	return &StatusPane{
		LeafPane: ui.LeafPane{
			BasePane: ui.BasePane{
				ID: "status",
			},
			Renderer:   renderer,
			Dims:       dimensions,
			Stylesheet: stylesheet,
		},
		getCurrentDate:      getCurrentDate,
		dayWidth:            dayWidth,
		totalDaysInPeriod:   totalDaysInPeriod,
		passedDaysInPeriod:  passedDaysInPeriod,
		firstDayXOffset:     firstDayXOffset,
		eventEditMode:       eventEditMode,
		storageProviderInfo: storageProviderInfo,
		log:                 log.With().Str("component", "status-pane").Logger(),
	}
}
