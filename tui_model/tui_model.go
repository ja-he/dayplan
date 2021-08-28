package tui_model

import (
	"dayplan/model"
	"dayplan/timestamp"
	"dayplan/util"

	"github.com/gdamore/tcell/v2"
)

type TUIModel struct {
	CursorX, CursorY                int
	EventviewOffset, EventviewWidth int
	CategoryStyling                 map[model.Category]tcell.Style
	Positions                       map[model.Event]util.Rect
	Hovered                         eventHoverState
	Model                           *model.Model
	Status                          string
	Resolution                      int
	ScrollOffset                    int
}

type eventHoverState struct {
	Event  *model.Event
	Resize bool
}

func NewTUIModel() *TUIModel {
	var t TUIModel

	t.CategoryStyling = make(map[model.Category]tcell.Style)
	t.Positions = make(map[model.Event]util.Rect)
	t.CategoryStyling[model.Category{Name: "work"}] = tcell.StyleDefault.Background(tcell.NewHexColor(0xccebff)).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "leisure"}] = tcell.StyleDefault.Background(tcell.Color76).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "misc"}] = tcell.StyleDefault.Background(tcell.Color250).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "programming"}] = tcell.StyleDefault.Background(tcell.Color226).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "cooking"}] = tcell.StyleDefault.Background(tcell.Color212).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "fitness"}] = tcell.StyleDefault.Background(tcell.Color208).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "eating"}] = tcell.StyleDefault.Background(tcell.Color224).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "hygiene"}] = tcell.StyleDefault.Background(tcell.Color80).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "cleaning"}] = tcell.StyleDefault.Background(tcell.Color215).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "laundry"}] = tcell.StyleDefault.Background(tcell.Color111).Foreground(tcell.ColorReset)
	t.CategoryStyling[model.Category{Name: "family"}] = tcell.StyleDefault.Background(tcell.Color122).Foreground(tcell.ColorReset)
	t.EventviewOffset = 10
	t.EventviewWidth = 80
	t.Status = "initial status msg"

	t.Resolution = 12
	t.ScrollOffset = 8 * t.Resolution

	return &t
}

func (t *TUIModel) TimeForDistance(dist int) timestamp.TimeOffset {
	add := true
	if dist < 0 {
		dist *= (-1)
		add = false
	}
	minutes := dist * (60 / t.Resolution)
	return timestamp.TimeOffset{T: timestamp.Timestamp{Hour: minutes / 60, Minute: minutes % 60}, Add: add}
}

func (t *TUIModel) SetModel(m *model.Model) {
	t.Model = m
}

func (t *TUIModel) TimeAtY(y int) timestamp.Timestamp {
	minutes := y*(60/t.Resolution) + t.ScrollOffset*(60/t.Resolution)

	ts := timestamp.Timestamp{Hour: minutes / 60, Minute: minutes % 60}

	return ts
}

func (t *TUIModel) ComputeRects() {
	defaultX := t.EventviewOffset
	defaultW := t.EventviewWidth
	active_stack := make([]model.Event, 0)
	for _, e := range t.Model.Events {
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
		y := t.toY(e.Start)
		x := defaultX
		h := t.toY(e.End) - y
		w := defaultW
		for i := 1; i < len(active_stack); i++ {
			x = x + (w / 2)
			w = w / 2
		}
		t.Positions[e] = util.Rect{X: x, Y: y, W: w, H: h}
	}
}

func (t *TUIModel) GetHoveredEvent() eventHoverState {
	if t.CursorX >= t.EventviewOffset &&
		t.CursorX < (t.EventviewOffset+t.EventviewWidth) {
		for i := len(t.Model.Events) - 1; i >= 0; i-- {
			if t.Positions[t.Model.Events[i]].Contains(t.CursorX, t.CursorY) {
				if t.CursorY == (t.Positions[t.Model.Events[i]].Y + t.Positions[t.Model.Events[i]].H - 1) {
					return eventHoverState{&t.Model.Events[i], true}
				} else {
					return eventHoverState{&t.Model.Events[i], false}
				}
			}
		}
	}
	return eventHoverState{nil, false}
}

func (t *TUIModel) toY(ts timestamp.Timestamp) int {
	return ((ts.Hour*t.Resolution - t.ScrollOffset) + (ts.Minute / (60 / t.Resolution)))
}
