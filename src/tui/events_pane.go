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
)

type EventsPane struct {
	renderer ConstrainedRenderer

	dimensions func() (x, y, w, h int)

	day func() *model.Day

	categories *category_style.CategoryStyling
	viewParams *ViewParams
	cursor     *CursorPos

	logReader potatolog.LogReader
	logWriter potatolog.LogWriter

	padRight       int
	drawTimestamps bool
	drawNames      bool
	isCurrent      func() bool

	// TODO: get rid of this
	positions map[model.EventID]util.Rect
}

func (p *EventsPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *EventsPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return &TUIPositionInfo{
		paneType: ui.EventsUIPanelType,
		weather:  nil,
		timeline: nil,
		tools:    nil,
		status:   nil,
		events:   p.getEventForPos(x, y),
	}
}

func (t *EventsPane) Draw() {
	x, y, w, h := t.Dimensions()

	day := t.day()

	if day == nil {
		t.logWriter.Add("DEBUG", "current day nil on render; skipping")
		// TODO: just draw this, man
		return
	}
	t.positions = t.ComputeRects(day, x, y, w-t.padRight, h)
	for _, e := range day.Events {
		style, err := t.categories.GetStyle(e.Cat)
		if err != nil {
			t.logWriter.Add("ERROR", err.Error())
		}
		if !t.isCurrent() {
			style = colors.DefaultDim(style)
		}

		// based on event state, draw a box or maybe a smaller one, or ...
		p := t.positions[e.ID]
		var timestampWidth int
		if t.drawTimestamps {
			timestampWidth = 5
		} else {
			timestampWidth = 0
		}
		namePadding := 1
		nameWidth := p.W - (2 * namePadding) - timestampWidth
		hovered := t.getEventForPos(t.cursor.X, t.cursor.Y)
		if hovered == nil || hovered.Event() != e.ID || hovered.EventBoxPart() == ui.EventBoxNowhere {
			t.renderer.DrawBox(p.X, p.Y, p.W, p.H, style)
			if t.drawNames {
				t.renderer.DrawText(p.X+namePadding, p.Y, nameWidth, p.H, style, util.TruncateAt(e.Name, nameWidth))
			}
			if t.drawTimestamps {
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
			}
		} else {
			selStyle := colors.DefaultEmphasize(style)
			switch hovered.EventBoxPart() {
			case ui.EventBoxBottomRight:
				t.renderer.DrawBox(p.X, p.Y, p.W, p.H-1, style)
				t.renderer.DrawBox(p.X, p.Y+p.H-1, p.W, 1, selStyle)
				if t.drawNames {
					t.renderer.DrawText(p.X+namePadding, p.Y, nameWidth, p.H, style, util.TruncateAt(e.Name, nameWidth))
				}
				if t.drawTimestamps {
					t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
					t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
				}
			case ui.EventBoxInterior:
				t.renderer.DrawBox(p.X, p.Y, p.W, p.H, selStyle)
				if t.drawNames {
					t.renderer.DrawText(p.X+namePadding, p.Y, nameWidth, p.H, selStyle, util.TruncateAt(e.Name, nameWidth))
				}
				if t.drawTimestamps {
					t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, selStyle, e.Start.ToString())
					t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
				}
			case ui.EventBoxTopEdge:
				t.renderer.DrawBox(p.X, p.Y, p.W, p.H, style)
				if t.drawNames {
					t.renderer.DrawText(p.X+namePadding, p.Y, nameWidth, p.H, selStyle, util.TruncateAt(e.Name, nameWidth))
				}
				if t.drawTimestamps {
					t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
					t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
				}
			default:
				panic(fmt.Sprint("don't know this hover state:", hovered.EventBoxPart().ToString()))
			}
		}
	}
}

func (t *EventsPane) getEventForPos(x, y int) ui.EventsPanelPositionInfo {
	dimX, _, dimW, _ := t.Dimensions()

	if x >= dimX &&
		x < (dimX+dimW) {
		currentDay := t.day()
		for i := len(currentDay.Events) - 1; i >= 0; i-- {
			eventPos := t.positions[currentDay.Events[i].ID]
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
				return &EventsPanelPositionInfo{
					eventID:      currentDay.Events[i].ID,
					eventBoxPart: hover,
					time:         t.viewParams.TimeAtY(y),
				}
			}
		}
	}
	return &EventsPanelPositionInfo{
		eventID:      0,
		eventBoxPart: ui.EventBoxNowhere,
		time:         t.viewParams.TimeAtY(y),
	}
}

type EventsPanelPositionInfo struct {
	eventID      model.EventID
	eventBoxPart ui.EventBoxPart
	time         model.Timestamp
}

func (i *EventsPanelPositionInfo) Event() model.EventID          { return i.eventID }
func (i *EventsPanelPositionInfo) EventBoxPart() ui.EventBoxPart { return i.eventBoxPart }
func (i *EventsPanelPositionInfo) Time() model.Timestamp         { return i.time }

func (t *EventsPane) ComputeRects(day *model.Day, offsetX, offsetY, width, height int) map[model.EventID]util.Rect {
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
		x := offsetX
		y := t.toY(e.Start) + offsetY
		w := width
		h := t.toY(e.End) + offsetY - y

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

func (t *EventsPane) toY(ts model.Timestamp) int {
	return ((ts.Hour*t.viewParams.NRowsPerHour - t.viewParams.ScrollOffset) + (ts.Minute / (60 / t.viewParams.NRowsPerHour)))
}

// this type exists to allow us to represent the variable length months in
// panes easily.
type MaybeEventsPane struct {
	eventsPane ui.UIPane
	condition  func() bool
}

func (p *MaybeEventsPane) Draw() {
	if p.condition() {
		p.eventsPane.Draw()
	}
}
func (p *MaybeEventsPane) Dimensions() (x, y, w, h int) { return p.eventsPane.Dimensions() }
func (p *MaybeEventsPane) GetPositionInfo(x, y int) ui.PositionInfo {
	someInfo := p.eventsPane.GetPositionInfo(x, y)
	if p.condition() {
		return someInfo
	} else {
		return &TUIPositionInfo{
			paneType: ui.None,
			weather:  nil,
			timeline: nil,
			tools:    nil,
			status:   nil,
			events:   nil,
		}
	}
}
