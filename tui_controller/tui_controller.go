package tui_controller

import (
	"dayplan/hover_state"
	"dayplan/model"
	"dayplan/tui_model"
	"dayplan/tui_view"

	"github.com/gdamore/tcell/v2"
)

type TUIController struct {
	model           *tui_model.TUIModel
	view            *tui_view.TUIView
	prevX, prevY    int
	editState       EditState
	EditedEvent     model.EventID
	currentCategory model.Category
	shouldExit      bool
}

type EditState int

const (
	None          EditState = 0b00000000
	Editing       EditState = 0b00000001
	MouseEditing  EditState = 0b00000010
	SelectEditing EditState = 0b00000100
	Moving        EditState = 0b00001000
	Resizing      EditState = 0b00010000
	Renaming      EditState = 0b00100000
)

func (s EditState) toString() string {
	return "TODO"
}

func NewTUIController(view *tui_view.TUIView, model *tui_model.TUIModel) *TUIController {
	t := TUIController{}

	t.model = model
	t.view = view
	t.currentCategory.Name = "default"

	return &t
}

func (t *TUIController) endEdit() {
	t.editState = None
	t.EditedEvent = 0
	t.model.Model.UpdateEventOrder()
}

func (t *TUIController) startMouseMove() {
	t.editState = (MouseEditing | Moving)
	t.EditedEvent = t.model.Hovered.EventID
}

func (t *TUIController) startMouseResize() {
	t.editState = (MouseEditing | Resizing)
	t.EditedEvent = t.model.Hovered.EventID
}

func (t *TUIController) startMouseEventCreation(cursorPosY int) {
	// find out cursor time
	start := t.model.TimeAtY(cursorPosY)

	// create event at time with cat etc.
	e := model.Event{}
	e.Cat = t.currentCategory
	e.Name = "?"
	e.Start = start
	e.End = start.OffsetMinutes(+10)

	// give to model, get ID
	newEventID := t.model.Model.AddEvent(e)
	t.model.Model.UpdateEventOrder()

	// save ID as edited event
	t.EditedEvent = newEventID

	// set mode to resizing
	t.editState = (MouseEditing | Resizing)
}

func (t *TUIController) handleKeyInput(e *tcell.EventKey) {
	switch e.Rune() {
	case 'q':
		t.shouldExit = true
	}
}

func (t *TUIController) updateCursorPos(x, y int) {
	t.prevX, t.prevY = x, y
}

func (t *TUIController) handleNoneEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventKey:
		t.handleKeyInput(e)
	case *tcell.EventMouse:
		// get new position
		x, y := e.Position()
		t.updateCursorPos(x, y)

    pane := t.model.UIDim.WhichUIPane(x, y)
		switch pane {
		case tui_model.Status:
			t.model.Status = "Status"
		case tui_model.Weather:
			t.model.Status = "Weather"
		case tui_model.Timeline:
			t.model.Status = "Timeline"
		case tui_model.Events:
			t.model.Status = "Events"

			// if mouse over event, update hover info in tui model
			t.model.Hovered = t.model.GetEventForPos(x, y)

			// if button clicked, handle
			buttons := e.Buttons()
			switch buttons {
			case tcell.Button1:
				// we've clicked while not editing
				// now we need to check where the cursor is and either start event
				// creation, resizing or moving
				switch t.model.Hovered.HoverState {
				case hover_state.None:
					t.startMouseEventCreation(y)
				case hover_state.Resize:
					t.startMouseResize()
				case hover_state.Move:
					t.startMouseMove()
				}
			}
		case tui_model.Tools:
			t.model.Status = "Tools"
		default:
			t.model.Status = "WTF?!"
		}
    if pane != tui_model.Events {
      t.model.ClearHover()
    }

	}
}

func (t *TUIController) resizeStep(newY int) {
	delta := newY - t.prevY
	offset := t.model.TimeForDistance(delta)
	event := t.model.Model.GetEvent(t.EditedEvent)
	event.End = event.End.Offset(offset).Snap(t.model.Resolution)
}

func (t *TUIController) moveStep(newY int) {
	delta := newY - t.prevY
	offset := t.model.TimeForDistance(delta)
	event := t.model.Model.GetEvent(t.EditedEvent)
	event.Start = event.Start.Offset(offset).Snap(t.model.Resolution)
	event.End = event.End.Offset(offset).Snap(t.model.Resolution)
}

func (t *TUIController) handleMouseResizeEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		x, y := e.Position()

		buttons := e.Buttons()

		switch buttons {
		case tcell.Button1:
			t.resizeStep(y)
		case tcell.ButtonNone:
			t.endEdit()
		}

		t.updateCursorPos(x, y)
	}
}

func (t *TUIController) handleMouseMoveEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		x, y := e.Position()

		buttons := e.Buttons()

		switch buttons {
		case tcell.Button1:
			t.moveStep(y)
		case tcell.ButtonNone:
			t.endEdit()
		}

		t.updateCursorPos(x, y)
	}
}

func (t *TUIController) Run() {
	for i := 0; i >= 0; i++ {
		if t.shouldExit {
			return
		}

		t.view.Render()
		// t.view.Model.Status = fmt.Sprintf("i = %d", i)

		// TODO: this blocks, meaning if no input is given, the screen doesn't update
		//       what we might want is an input buffer in another goroutine? idk
		ev := t.view.Screen.PollEvent()

		switch t.editState {
		case None:
			t.handleNoneEditEvent(ev)
		case (MouseEditing | Resizing):
			t.handleMouseResizeEditEvent(ev)
		case (MouseEditing | Moving):
			t.handleMouseMoveEditEvent(ev)
		}

		switch ev.(type) {
		case *tcell.EventResize:
			t.view.NeedsSync()
			t.model.UIDim.ScreenResize(t.view.Screen.Size())
		}
	}
}
