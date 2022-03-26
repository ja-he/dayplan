package panes

import (
	"fmt"
	"math"

	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/potatolog"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
)

// An EventsPane displays a single days events.
// It can be configured to display events with more decorations and padding
// (e.g., when displaying a single day in the UI), or to be space efficient and
// hide some details (e.g., for showing events as part of multiple EventPanes in
// in the month view.
type EventsPane struct {
	renderer ui.ConstrainedRenderer

	dimensions func() (x, y, w, h int)
	stylesheet styling.Stylesheet

	day func() *model.Day

	categories *styling.CategoryStyling
	viewParams *ui.ViewParams
	cursor     *ui.MouseCursorPos

	logReader potatolog.LogReader
	logWriter potatolog.LogWriter

	padRight       int
	drawTimestamps bool
	drawNames      bool
	isCurrent      func() bool

	// TODO: get rid of this
	positions map[model.EventID]util.Rect
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
// GetPositionInfo returns information on a requested position in this pane.
func (p *EventsPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

// GetPositionInfo returns information on a requested position in this pane.
// Importantly, when there is an event at the position, it will inform of that
// in detail.
func (p *EventsPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return ui.NewPositionInfo(
		ui.EventsPaneType,
		nil,
		nil,
		nil,
		nil,
		p.getEventForPos(x, y),
	)
}

// Draw draws this pane.
func (p *EventsPane) Draw() {
	x, y, w, h := p.Dimensions()
	p.renderer.DrawBox(x, y, w, h, p.stylesheet.Normal)

	day := p.day()

	if day == nil {
		p.logWriter.Add("DEBUG", "current day nil on render; skipping")
		// TODO: just draw this, man
		return
	}
	p.positions = p.computeRects(day, x, y, w-p.padRight, h)
	for _, e := range day.Events {
		style, err := p.categories.GetStyle(e.Cat)
		styling := style
		if err != nil {
			p.logWriter.Add("ERROR", err.Error())
			styling = p.stylesheet.CategoryFallback
		}
		if !p.isCurrent() {
			styling = styling.DefaultDimmed()
		}

		// based on event state, draw a box or maybe a smaller one, or ...
		pos := p.positions[e.ID]
		var timestampWidth int
		if p.drawTimestamps {
			timestampWidth = 5
		} else {
			timestampWidth = 0
		}
		namePadding := 1
		nameWidth := pos.W - (2 * namePadding) - timestampWidth
		hovered := p.getEventForPos(p.cursor.X, p.cursor.Y)
		if hovered == nil || hovered.Event() != e.ID || hovered.EventBoxPart() == ui.EventBoxNowhere {
			p.renderer.DrawBox(pos.X, pos.Y, pos.W, pos.H, styling)
			if p.drawNames {
				p.renderer.DrawText(pos.X+namePadding, pos.Y, nameWidth, pos.H, styling, util.TruncateAt(e.Name, nameWidth))
			}
			if p.drawTimestamps {
				p.renderer.DrawText(pos.X+pos.W-5, pos.Y, 5, 1, styling, e.Start.ToString())
				p.renderer.DrawText(pos.X+pos.W-5, pos.Y+pos.H-1, 5, 1, styling, e.End.ToString())
			}
		} else {
			selectionStyling := styling.DefaultEmphasized()
			switch hovered.EventBoxPart() {
			case ui.EventBoxBottomRight:
				p.renderer.DrawBox(pos.X, pos.Y, pos.W, pos.H-1, styling)
				p.renderer.DrawBox(pos.X, pos.Y+pos.H-1, pos.W, 1, selectionStyling)
				if p.drawNames {
					p.renderer.DrawText(pos.X+namePadding, pos.Y, nameWidth, pos.H, styling, util.TruncateAt(e.Name, nameWidth))
				}
				if p.drawTimestamps {
					p.renderer.DrawText(pos.X+pos.W-5, pos.Y, 5, 1, styling, e.Start.ToString())
					p.renderer.DrawText(pos.X+pos.W-5, pos.Y+pos.H-1, 5, 1, selectionStyling, e.End.ToString())
				}
			case ui.EventBoxInterior:
				p.renderer.DrawBox(pos.X, pos.Y, pos.W, pos.H, selectionStyling)
				if p.drawNames {
					p.renderer.DrawText(pos.X+namePadding, pos.Y, nameWidth, pos.H, selectionStyling, util.TruncateAt(e.Name, nameWidth))
				}
				if p.drawTimestamps {
					p.renderer.DrawText(pos.X+pos.W-5, pos.Y, 5, 1, selectionStyling, e.Start.ToString())
					p.renderer.DrawText(pos.X+pos.W-5, pos.Y+pos.H-1, 5, 1, selectionStyling, e.End.ToString())
				}
			case ui.EventBoxTopEdge:
				p.renderer.DrawBox(pos.X, pos.Y, pos.W, pos.H, styling)
				if p.drawNames {
					p.renderer.DrawText(pos.X+namePadding, pos.Y, nameWidth, pos.H, selectionStyling, util.TruncateAt(e.Name, nameWidth))
				}
				if p.drawTimestamps {
					p.renderer.DrawText(pos.X+pos.W-5, pos.Y, 5, 1, styling, e.Start.ToString())
					p.renderer.DrawText(pos.X+pos.W-5, pos.Y+pos.H-1, 5, 1, styling, e.End.ToString())
				}
			default:
				panic(fmt.Sprint("don't know this hover state:", hovered.EventBoxPart().ToString()))
			}
		}
	}
}

func (p *EventsPane) getEventForPos(x, y int) ui.EventsPanePositionInfo {
	dimX, _, dimW, _ := p.Dimensions()

	if x >= dimX &&
		x < (dimX+dimW) {
		currentDay := p.day()
		for i := len(currentDay.Events) - 1; i >= 0; i-- {
			eventPos := p.positions[currentDay.Events[i].ID]
			if eventPos.Contains(x, y) {
				var hover ui.EventBoxPart
				switch {
				case y == (eventPos.Y+eventPos.H-1) && x > eventPos.X+eventPos.W-5:
					hover = ui.EventBoxBottomRight
				case y == (eventPos.Y):
					hover = ui.EventBoxTopEdge
				default:
					hover = ui.EventBoxInterior
				}
				return &EventsPanePositionInfo{
					eventID:      currentDay.Events[i].ID,
					eventBoxPart: hover,
					time:         p.viewParams.TimeAtY(y),
				}
			}
		}
	}
	return &EventsPanePositionInfo{
		eventID:      0,
		eventBoxPart: ui.EventBoxNowhere,
		time:         p.viewParams.TimeAtY(y),
	}
}

// EventsPanePositionInfo provides information on a position in an EventsPane,
// implementing the ui.EventsPanePositionInfo interface.
type EventsPanePositionInfo struct {
	eventID      model.EventID
	eventBoxPart ui.EventBoxPart
	time         model.Timestamp
}

// Event returns the ID of the event at the position, 0 if no event at
// position.
func (i *EventsPanePositionInfo) Event() model.EventID { return i.eventID }

// EventBoxPart returns the part of the event box that corresponds to the
// position (which can be EventBoxNowhere, if no event at position).
func (i *EventsPanePositionInfo) EventBoxPart() ui.EventBoxPart { return i.eventBoxPart }

// Time returns the time that corresponds to the position (specifically the
// y-value of the position).
func (i *EventsPanePositionInfo) Time() model.Timestamp { return i.time }

func (p *EventsPane) computeRects(day *model.Day, offsetX, offsetY, width, height int) map[model.EventID]util.Rect {
	activeStack := make([]model.Event, 0)
	positions := make(map[model.EventID]util.Rect)
	for _, e := range day.Events {
		// remove all stacked elements that have finished
		for i := len(activeStack) - 1; i >= 0; i-- {
			if e.Start.IsAfter(activeStack[i].End) || e.Start == activeStack[i].End {
				activeStack = activeStack[:i]
			} else {
				break
			}
		}
		activeStack = append(activeStack, e)
		// based on event state, draw a box or maybe a smaller one, or ...
		x := offsetX
		y := p.toY(e.Start) + offsetY
		w := width
		h := p.toY(e.End) + offsetY - y

		// scale the width by 3/4 for every extra item on the stack, so for one
		// item stacked underneath the current items width will be (3/4) ** 1 = 75%
		// of the original width, for four it would be (3/4) ** 4 = (3**4)/(4**4)
		// or 31.5 % of the width, etc.
		widthFactor := 0.75
		w = int(float64(w) * math.Pow(widthFactor, float64(len(activeStack)-1)))
		x += (width - w)

		positions[e.ID] = util.Rect{X: x, Y: y, W: w, H: h}
	}
	return positions
}

func (p *EventsPane) toY(ts model.Timestamp) int {
	return ((ts.Hour*p.viewParams.NRowsPerHour - p.viewParams.ScrollOffset) + (ts.Minute / (60 / p.viewParams.NRowsPerHour)))
}

// MaybeEventsPane wraps an EventsPane with a condition and exposes its
// functionality, iff the condition holds. Otherwise its operations do or
// return nothing.
//
// This type exists primarily to allow us to represent the variable length
// months in panes easily.
type MaybeEventsPane struct {
	eventsPane ui.Pane
	condition  func() bool
}

// Draw draws this pane.
func (p *MaybeEventsPane) Draw() {
	if p.condition() {
		p.eventsPane.Draw()
	}
}

// Dimensions gives the dimensions (x-axis offset, y-axis offset, width,
// height) for this pane.
func (p *MaybeEventsPane) Dimensions() (x, y, w, h int) { return p.eventsPane.Dimensions() }

// GetPositionInfo returns information on a requested position in the
// underlying EventsPane pane, but only if this pane's condition is true.
func (p *MaybeEventsPane) GetPositionInfo(x, y int) ui.PositionInfo {
	someInfo := p.eventsPane.GetPositionInfo(x, y)
	if p.condition() {
		return someInfo
	}
	return ui.NewPositionInfo(
		ui.NoPane,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
}

// NewEventsPane constructs and returns a new EventsPane.
func NewEventsPane(
	renderer ui.ConstrainedRenderer,
	dimensions func() (x, y, w, h int),
	stylesheet styling.Stylesheet,
	day func() *model.Day,
	categories *styling.CategoryStyling,
	viewParams *ui.ViewParams,
	cursor *ui.MouseCursorPos,
	padRight int,
	drawTimestamps bool,
	drawNames bool,
	isCurrent func() bool,
	logReader potatolog.LogReader,
	logWriter potatolog.LogWriter,
	positions map[model.EventID]util.Rect,
) *EventsPane {
	return &EventsPane{
		renderer:       renderer,
		dimensions:     dimensions,
		stylesheet:     stylesheet,
		day:            day,
		categories:     categories,
		viewParams:     viewParams,
		cursor:         cursor,
		padRight:       padRight,
		drawTimestamps: drawTimestamps,
		drawNames:      drawNames,
		isCurrent:      isCurrent,
		logReader:      logReader,
		logWriter:      logWriter,
		positions:      positions,
	}
}

// NewMaybeEventsPane constructs and returns a new MaybeEventsPane.
func NewMaybeEventsPane(
	condition func() bool,
	eventsPane *EventsPane,
) *MaybeEventsPane {
	return &MaybeEventsPane{
		condition:  condition,
		eventsPane: eventsPane,
	}
}
