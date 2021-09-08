package tui_controller

import (
	"bufio"
	"log"
	"os"
	"strconv"

	"dayplan/hover_state"
	"dayplan/model"
	"dayplan/tui_model"
	"dayplan/tui_view"

	"github.com/gdamore/tcell/v2"
)

type FileHandler struct {
	filename string
}

func NewFileHandler(filename string) *FileHandler {
	f := FileHandler{filename: filename}
	return &f
}

func (h *FileHandler) Write(m *model.Model) {
	f, err := os.OpenFile(h.filename, os.O_TRUNC|os.O_WRONLY, 0644)
	defer f.Close()
	if err != nil {
		log.Fatalf("cannot read file '%s'", h.filename)
	}

	writer := bufio.NewWriter(f)
	defer writer.Flush()
	for _, line := range m.ToSlice() {
		_, _ = writer.WriteString(line + "\n")
	}
}

func (h *FileHandler) Read() *model.Model {
	m := model.NewModel()

	f, err := os.Open(h.filename)
	if err != nil {
		log.Fatalf("cannot read file '%s'", h.filename)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		m.AddEvent(*model.NewEvent(s))
	}

	return m
}

type TUIController struct {
	model        *tui_model.TUIModel
	view         *tui_view.TUIView
	prevX, prevY int
	editState    EditState
	EditedEvent  model.EventID
	shouldExit   bool
	FileHandler  *FileHandler
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

func NewTUIController(view *tui_view.TUIView, tmodel *tui_model.TUIModel, filehandler *FileHandler) *TUIController {
	t := TUIController{}

	t.FileHandler = filehandler

	t.model = tmodel
	if t.FileHandler == nil {
		t.model.Model = &model.Model{}
	} else {
		t.model.Model = t.FileHandler.Read()
	}

	t.view = view
	t.model.CurrentCategory.Name = "default"

	return &t
}

func (t *TUIController) abortEdit() {
	t.editState = None
	t.EditedEvent = 0
	t.model.EventEditor.Active = false
}

func (t *TUIController) endEdit() {
	t.editState = None
	t.EditedEvent = 0
	if t.model.EventEditor.Active {
		t.model.EventEditor.Active = false
		tmp := t.model.EventEditor.TmpEventInfo
		t.model.Model.GetEvent(tmp.ID).Name = tmp.Name
	}
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
	e.Cat = t.model.CurrentCategory
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

func (t *TUIController) handleNoneEditKeyInput(e *tcell.EventKey) {
	switch e.Rune() {
	case 'q':
		t.shouldExit = true
	case 'w':
		t.model.Status = "writing..."
		t.writeModel()
		t.model.Status = "written!"
	}
}

func (t *TUIController) writeModel() {
	t.FileHandler.Write(t.model.Model)
}

func (t *TUIController) updateCursorPos(x, y int) {
	t.prevX, t.prevY = x, y
}

func (t *TUIController) startEdit(id model.EventID) {
	t.model.EventEditor.Active = true
	t.model.EventEditor.TmpEventInfo = *t.model.Model.GetEvent(id)
	t.editState = Editing
}

func (t *TUIController) handleNoneEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventKey:
		t.handleNoneEditKeyInput(e)
	case *tcell.EventMouse:
		// get new position
		x, y := e.Position()
		t.updateCursorPos(x, y)

		buttons := e.Buttons()

		pane := t.model.UIDim.WhichUIPane(x, y)
		switch pane {
		case tui_model.Status:
		case tui_model.Weather:
			switch buttons {
			case tcell.WheelUp:
				t.model.ScrollUp()
			case tcell.WheelDown:
				t.model.ScrollDown()
			}
		case tui_model.Timeline:
			switch buttons {
			case tcell.WheelUp:
				t.model.ScrollUp()
			case tcell.WheelDown:
				t.model.ScrollDown()
			}
		case tui_model.Events:
			// if mouse over event, update hover info in tui model
			t.model.Hovered = t.model.GetEventForPos(x, y)

			// if button clicked, handle
			switch buttons {
			case tcell.Button3:
				t.model.Model.RemoveEvent(t.model.Hovered.EventID)
			case tcell.Button2:
				id := t.model.Hovered.EventID
				if id != 0 {
					t.startEdit(id)
				}
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
			case tcell.WheelUp:
				t.model.ScrollUp()
			case tcell.WheelDown:
				t.model.ScrollDown()
			}
		case tui_model.Tools:
			switch buttons {
			case tcell.Button1:
				cat := t.model.GetCategoryForPos(x, y)
				if cat != nil {
					t.model.CurrentCategory = *cat
				}
			}
		default:
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

func (t *TUIController) handleEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventKey:
		switch e.Key() {
		case tcell.KeyEsc:
			t.abortEdit()
		case tcell.KeyEnter:
			t.endEdit()
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			textlen := len(t.model.EventEditor.TmpEventInfo.Name)
			if textlen > 0 {
				t.model.EventEditor.TmpEventInfo.Name = t.model.EventEditor.TmpEventInfo.Name[:textlen-1]
			}
		case tcell.KeyCtrlU:
			t.model.EventEditor.TmpEventInfo.Name = ""
		default:
			rune := e.Rune()
			if strconv.IsPrint(rune) {
				t.model.EventEditor.TmpEventInfo.Name += string(rune)
			}
		}
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
		case (Editing):
			t.handleEditEvent(ev)
		}

		switch ev.(type) {
		case *tcell.EventResize:
			t.view.NeedsSync()
			t.model.UIDim.ScreenResize(t.view.Screen.Size())
		}
	}
}
