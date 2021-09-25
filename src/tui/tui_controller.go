package tui

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"time"

	"dayplan/src/model"

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
	model         *TUIModel
	view          *TUIView
	prevX, prevY  int
	editState     EditState
	EditedEvent   model.EventID
	shouldExit    bool
	movePropagate bool
	FileHandler   *FileHandler
}

type EditState int

const (
	EditStateNone          EditState = 0b00000000
	EditStateEditing       EditState = 0b00000001
	EditStateMouseEditing  EditState = 0b00000010
	EditStateSelectEditing EditState = 0b00000100
	EditStateMoving        EditState = 0b00001000
	EditStateResizing      EditState = 0b00010000
	EditStateRenaming      EditState = 0b00100000
)

func (s EditState) toString() string {
	return "TODO"
}

func NewTUIController(view *TUIView, tmodel *TUIModel, filehandler *FileHandler) *TUIController {
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
	t.editState = EditStateNone
	t.EditedEvent = 0
	t.model.EventEditor.Active = false
}

func (t *TUIController) endEdit() {
	t.editState = EditStateNone
	t.EditedEvent = 0
	if t.model.EventEditor.Active {
		t.model.EventEditor.Active = false
		tmp := t.model.EventEditor.TmpEventInfo
		t.model.Model.GetEvent(tmp.ID).Name = tmp.Name
	}
	t.model.Model.UpdateEventOrder()
}

func (t *TUIController) startMouseMove() {
	t.editState = (EditStateMouseEditing | EditStateMoving)
	t.EditedEvent = t.model.Hovered.EventID
}

func (t *TUIController) startMouseResize() {
	t.editState = (EditStateMouseEditing | EditStateResizing)
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
	t.editState = (EditStateMouseEditing | EditStateResizing)
}

func (t *TUIController) handleNoneEditKeyInput(e *tcell.EventKey) {
	switch e.Rune() {
	case 'u':
		go t.model.Weather.Update()
	case 'q':
		t.shouldExit = true
	case 'w':
		t.model.Status = "writing..."
		t.writeModel()
		t.model.Status = "written!"
	case 'c':
		// TODO: all that's needed to clear model (appropriately)?
		t.model.Model = model.NewModel()
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
	t.editState = EditStateEditing
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
		case Status:
		case Weather:
			switch buttons {
			case tcell.WheelUp:
				t.model.ScrollUp()
			case tcell.WheelDown:
				t.model.ScrollDown()
			}
		case Timeline:
			switch buttons {
			case tcell.WheelUp:
				t.model.ScrollUp()
			case tcell.WheelDown:
				t.model.ScrollDown()
			}
		case Events:
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
				case HoverStateNone:
					t.startMouseEventCreation(y)
				case HoverStateResize:
					t.startMouseResize()
				case HoverStateMove:
					t.movePropagate = (e.Modifiers() == tcell.ModCtrl)
					t.startMouseMove()
				}
			case tcell.WheelUp:
				t.model.ScrollUp()
			case tcell.WheelDown:
				t.model.ScrollDown()
			}
		case Tools:
			switch buttons {
			case tcell.Button1:
				cat := t.model.GetCategoryForPos(x, y)
				if cat != nil {
					t.model.CurrentCategory = *cat
				}
			}
		default:
		}
		if pane != Events {
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
	if t.movePropagate {
		following := t.model.Model.GetEventsFrom(t.EditedEvent)
		for _, ptr := range following {
			ptr.Start = ptr.Start.Offset(offset).Snap(t.model.Resolution)
			ptr.End = ptr.End.Offset(offset).Snap(t.model.Resolution)
		}
	} else {
		event := t.model.Model.GetEvent(t.EditedEvent)
		event.Start = event.Start.Offset(offset).Snap(t.model.Resolution)
		event.End = event.End.Offset(offset).Snap(t.model.Resolution)
	}
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

		t.model.Status = model.NewTimestampFromGotime(time.Now()).ToString()

		// TODO: this blocks, meaning if no input is given, the screen doesn't update
		//       what we might want is an input buffer in another goroutine? idk
		ev := t.view.Screen.PollEvent()

		switch t.editState {
		case EditStateNone:
			t.handleNoneEditEvent(ev)
		case (EditStateMouseEditing | EditStateResizing):
			t.handleMouseResizeEditEvent(ev)
		case (EditStateMouseEditing | EditStateMoving):
			t.handleMouseMoveEditEvent(ev)
		case (EditStateEditing):
			t.handleEditEvent(ev)
		}

		switch ev.(type) {
		case *tcell.EventResize:
			t.view.NeedsSync()
			t.model.UIDim.ScreenResize(t.view.Screen.Size())
		}
	}
}
