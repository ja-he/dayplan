package tui_controller

import (
	"fmt"
	"sort"

	"dayplan/model"
	"dayplan/timestamp"
	"dayplan/tui_model"
	"dayplan/tui_view"

	"github.com/gdamore/tcell/v2"
)

type TUIController struct {
	model       *tui_model.TUIModel
	view        *tui_view.TUIView
	editState   EditState
	EditedEvent *model.Event
}

type EditState int

const (
	None EditState = iota
	Moving
	Resizing
)

func (s EditState) toString() string {
	switch s {
	case None:
		return "None"
	case Moving:
		return "Moving"
	case Resizing:
		return "Resizing"
	default:
		return "UNKNOWN / ERR"
	}
}

func NewTUIController(view *tui_view.TUIView, model *tui_model.TUIModel) *TUIController {
	t := TUIController{}

	t.model = model
	t.view = view

	return &t
}

// TODO: this is still a big monolith and needs to be broken up / abolished
func (t *TUIController) Run() {
	for i := 0; i >= 0; i++ {
		t.view.Model.Status = fmt.Sprintf("i = %d", i)
		t.view.Render()

		// TODO: this blocks, meaning if no input is given, the screen doesn't update
		//       what we might want is an input buffer in another goroutine? idk
		ev := t.view.Screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventResize:
			t.view.Screen.Sync()
		case *tcell.EventKey:
			// TODO: handle keys
			return
		case *tcell.EventMouse:
			// TODO: handle mouse input
			oldY := t.model.CursorY
			t.model.CursorX, t.model.CursorY = ev.Position()
			button := ev.Buttons()

			if button == tcell.Button1 {
				switch t.editState {
				case None:
					hovered := t.model.GetHoveredEvent()
					if hovered.Event != nil {
						t.EditedEvent = hovered.Event
						if hovered.Resize {
							t.editState = Resizing
						} else {
							t.editState = Moving
						}
					} else {
						e := model.Event{Name: "New Event"}
						e.Start = t.model.TimeAtY(t.model.CursorY)
						e.End = e.Start.Offset(timestamp.TimeOffset{T: timestamp.Timestamp{Hour: 0, Minute: 5}, Add: true})
						t.model.Model.AddEvent(e)
						sort.Sort(model.ByStart(t.model.Model.Events))
						t.editState = Resizing
						t.EditedEvent = &e
					}
				case Moving:
					t.EditedEvent.Snap(t.model.Resolution)
					t.EventMove(t.model.CursorY - oldY)
				case Resizing:
					t.EventResize(t.model.CursorY - oldY)
				}
			} else if button == tcell.WheelUp {
				newHourOffset := ((t.model.ScrollOffset / t.model.Resolution) - 1)
				if newHourOffset >= 0 {
					t.model.ScrollOffset = newHourOffset * t.model.Resolution
				}
			} else if button == tcell.WheelDown {
				newHourOffset := ((t.model.ScrollOffset / t.model.Resolution) + 1)
				if newHourOffset <= 24 {
					t.model.ScrollOffset = newHourOffset * t.model.Resolution
				}
			} else {
				t.editState = None
				sort.Sort(model.ByStart(t.model.Model.Events))
				t.model.Hovered = t.model.GetHoveredEvent()
			}
		}
	}
}

func (t *TUIController) EventMove(dist int) {
	e := t.EditedEvent
	timeOffset := t.model.TimeForDistance(dist)
	e.Start = e.Start.Offset(timeOffset).Snap(t.model.Resolution)
	e.End = e.End.Offset(timeOffset).Snap(t.model.Resolution)
}
func (t *TUIController) EventResize(dist int) {
	e := t.EditedEvent
	timeOffset := t.model.TimeForDistance(dist)
	newEnd := e.End.Offset(timeOffset).Snap(t.model.Resolution)
	if newEnd.IsAfter(e.Start) {
		e.End = newEnd
	}
}
