package tui

import (
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

	days        *DaysData
	currentDate *model.Date
	categories  *category_style.CategoryStyling
	viewParams  *ViewParams
	cursor      *CursorPos

	logReader potatolog.LogReader
	logWriter potatolog.LogWriter

	// TODO: get rid of this
	positions map[model.EventID]util.Rect
}

func (p *EventsPane) Dimensions() (x, y, w, h int) {
	return p.dimensions()
}

func (p *EventsPane) GetPositionInfo(x, y int) ui.PositionInfo {
	return &TUIPositionInfo{
		paneType:       ui.EventsUIPanelType,
		weather:        ui.WeatherPanelPositionInfo{},
		timeline:       ui.TimelinePanelPositionInfo{},
		tools:          ui.ToolsPanelPositionInfo{},
		status:         ui.StatusPanelPositionInfo{},
		events:         p.getEventForPos(x, y),
		timestampGuess: p.TimeAtY(y),
	}
}

func (t *EventsPane) Draw() {
	x, y, w, h := t.Dimensions()

	day := t.days.GetDay(*t.currentDate)
	if day == nil {
		t.logWriter.Add("DEBUG", "current day nil on render; skipping")
		return
	}
	t.positions = t.ComputeRects(day, x, y, w-2, h)
	for _, e := range day.Events {
		style, err := t.categories.GetStyle(e.Cat)
		if err != nil {
			t.logWriter.Add("ERROR", err.Error())
		}
		// based on event state, draw a box or maybe a smaller one, or ...
		p := t.positions[e.ID]
		hovered := t.getEventForPos(t.cursor.X, t.cursor.Y)
		if hovered.Event != e.ID {
			t.renderer.DrawBox(p.X, p.Y, p.W, p.H, style)
			t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, style, util.TruncateAt(e.Name, p.W-7))
			t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
			t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
		} else {
			selStyle := colors.DefaultEmphasize(style)
			switch hovered.HoverState {
			case ui.EventHoverStateResize:
				t.renderer.DrawBox(p.X, p.Y, p.W, p.H-1, style)
				t.renderer.DrawBox(p.X, p.Y+p.H-1, p.W, 1, selStyle)
				t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, style, util.TruncateAt(e.Name, p.W-7))
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			case ui.EventHoverStateMove:
				t.renderer.DrawBox(p.X, p.Y, p.W, p.H, selStyle)
				t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, selStyle, util.TruncateAt(e.Name, p.W-7))
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, selStyle, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, selStyle, e.End.ToString())
			case ui.EventHoverStateEdit:
				t.renderer.DrawBox(p.X, p.Y, p.W, p.H, style)
				t.renderer.DrawText(p.X+1, p.Y, p.W-7, p.H, selStyle, util.TruncateAt(e.Name, p.W-7))
				t.renderer.DrawText(p.X+p.W-5, p.Y, 5, 1, style, e.Start.ToString())
				t.renderer.DrawText(p.X+p.W-5, p.Y+p.H-1, 5, 1, style, e.End.ToString())
			default:
				panic("don't know this hover state!")
			}
		}
	}
}

func (t *EventsPane) getEventForPos(x, y int) ui.EventsPanelPositionInfo {
	dimX, _, dimW, _ := t.Dimensions()

	if x >= dimX &&
		x < (dimX+dimW) {
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

// TODO: remove, this will be part of info returned to controller on query
func (t *EventsPane) TimeAtY(y int) model.Timestamp {
	minutes := y*(60/t.viewParams.NRowsPerHour) + t.viewParams.ScrollOffset*(60/t.viewParams.NRowsPerHour)

	ts := model.Timestamp{Hour: minutes / 60, Minute: minutes % 60}

	return ts
}

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
