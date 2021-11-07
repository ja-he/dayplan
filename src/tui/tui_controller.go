package tui

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"dayplan/src/model"

	"github.com/gdamore/tcell/v2"
)

// TODO: this absolutely does not belong here
type Program struct {
	BaseDir      string
	FileHandlers map[model.Day]*FileHandler
}

func (p *Program) GetModelFromFileHandler(d model.Day) *model.Model {
	fh, ok := p.FileHandlers[d]
	if ok {
		tmp := fh.Read()
		return tmp
	} else {
		newHandler := NewFileHandler(p.BaseDir + "/days/" + d.ToString())
		p.FileHandlers[d] = newHandler
		tmp := newHandler.Read()
		return tmp
	}
}

type FileHandler struct {
	mutex    sync.Mutex
	filename string
}

func NewFileHandler(filename string) *FileHandler {
	f := FileHandler{filename: filename}
	return &f
}

func (h *FileHandler) Write(m *model.Model) {
	h.mutex.Lock()
	f, err := os.OpenFile(h.filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("error opening file '%s'", h.filename)
	}

	writer := bufio.NewWriter(f)
	for _, line := range m.ToSlice() {
		_, _ = writer.WriteString(line + "\n")
	}
	writer.Flush()
	f.Close()
	h.mutex.Unlock()
}

func (h *FileHandler) Read() *model.Model {
	m := model.NewModel()

	h.mutex.Lock()
	f, err := os.Open(h.filename)
	fileExists := (err == nil)
	if fileExists {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			s := scanner.Text()
			m.AddEvent(*model.NewEvent(s))
		}
		f.Close()
	}
	h.mutex.Unlock()

	return m
}

type TUIController struct {
	model         *TUIModel
	view          *TUIView
	editState     EditState
	EditedEvent   model.EventID
	movePropagate bool
	Program       Program
	bump          chan ControllerEvent
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

func NewTUIController(view *TUIView, tmodel *TUIModel, day model.Day, prog Program) *TUIController {
	t := TUIController{}
	t.Program = prog

	t.Program.FileHandlers = make(map[model.Day]*FileHandler)
	t.Program.FileHandlers[day] = NewFileHandler(prog.BaseDir + "/days/" + day.ToString())

	t.model = tmodel
	t.model.CurrentDay = day
	if t.Program.FileHandlers[day] == nil {
		t.model.AddModel(day, &model.Model{})
	} else {
		t.model.AddModel(day, t.Program.FileHandlers[day].Read())
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
		t.model.GetCurrentDayModel().GetEvent(tmp.ID).Name = tmp.Name
	}
	t.model.GetCurrentDayModel().UpdateEventOrder()
	t.model.Hovered.EventID = 0
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
	newEventID := t.model.GetCurrentDayModel().AddEvent(e)

	// save ID as edited event
	t.EditedEvent = newEventID

	// set mode to resizing
	t.editState = (EditStateMouseEditing | EditStateResizing)
}

func (t *TUIController) goToDay(newDay model.Day) {
	t.model.Log.Add("DEBUG", "going to "+newDay.ToString())

	if !t.model.HasModel(newDay) {
		// load file
		newModel := t.Program.GetModelFromFileHandler(newDay)
		if newModel == nil {
			panic("newModel nil?!")
		}
		t.model.AddModel(newDay, newModel)
	}

	t.model.CurrentDay = newDay
}

func (t *TUIController) goToPreviousDay() {
	prevDay := t.model.CurrentDay.Prev()
	t.goToDay(prevDay)
}

func (t *TUIController) goToNextDay() {
	nextDay := t.model.CurrentDay.Next()
	t.goToDay(nextDay)
}

func (t *TUIController) handleNoneEditKeyInput(e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyCtrlU:
		t.model.ScrollUp(10)
	case tcell.KeyCtrlD:
		t.model.ScrollDown(10)
	}
	switch e.Rune() {
	case 'u':
		go func() {
			err := t.model.Weather.Update()
			if err != nil {
				t.model.Log.Add("ERROR", err.Error())
			} else {
				t.model.Log.Add("DEBUG", "successfully retrieved weather data")
			}
			t.bump <- ControllerEventRender
			t.model.Status.Set("owm-qcount", fmt.Sprint(t.model.Weather.GetQueryCount()))
		}()
	case 'q':
		t.bump <- ControllerEventExit
	case 'g':
		t.model.ScrollTop()
	case 'G':
		t.model.ScrollBottom()
	case 'w':
		t.writeModel()
	case 'h':
		t.goToPreviousDay()
	case 'l':
		t.goToNextDay()
	case 'E':
		t.model.showLog = !t.model.showLog
	case 'c':
		// TODO: all that's needed to clear model (appropriately)?
		t.model.AddModel(t.model.CurrentDay, model.NewModel())
	case '+':
		if t.model.Resolution*2 <= 12 {
			t.model.Resolution *= 2
			t.model.ScrollOffset *= 2
		}
	case '-':
		if (t.model.Resolution % 2) == 0 {
			t.model.Resolution /= 2
			t.model.ScrollOffset /= 2
		} else {
			t.model.Log.Add("WARNING", fmt.Sprintf("can't decrease resolution below %d", t.model.Resolution))
		}
	}
}

func (t *TUIController) writeModel() {
	go t.Program.FileHandlers[t.model.CurrentDay].Write(t.model.GetCurrentDayModel())
}

func (t *TUIController) updateCursorPos(x, y int) {
	t.model.cursorX, t.model.cursorY = x, y
}

func (t *TUIController) startEdit(id model.EventID) {
	t.model.EventEditor.Active = true
	t.model.EventEditor.TmpEventInfo = *t.model.GetCurrentDayModel().GetEvent(id)
	t.model.EventEditor.CursorPos = len([]rune(t.model.EventEditor.TmpEventInfo.Name))
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
		case UIStatus:
		case UIWeather:
			switch buttons {
			case tcell.WheelUp:
				t.model.ScrollUp(1)
			case tcell.WheelDown:
				t.model.ScrollDown(1)
			}
		case UITimeline:
			switch buttons {
			case tcell.WheelUp:
				t.model.ScrollUp(1)
			case tcell.WheelDown:
				t.model.ScrollDown(1)
			}
		case UIEvents:
			// if mouse over event, update hover info in tui model
			t.model.Hovered = t.model.GetEventForPos(x, y)

			// if button clicked, handle
			switch buttons {
			case tcell.Button3:
				t.model.GetCurrentDayModel().RemoveEvent(t.model.Hovered.EventID)
			case tcell.Button2:
				id := t.model.Hovered.EventID
				if id != 0 && t.model.TimeAtY(y).IsAfter(t.model.GetCurrentDayModel().GetEvent(id).Start) {
					t.model.GetCurrentDayModel().SplitEvent(id, t.model.TimeAtY(y))
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
				case HoverStateEdit:
					t.startEdit(t.model.Hovered.EventID)
				}
			case tcell.WheelUp:
				t.model.ScrollUp(1)
			case tcell.WheelDown:
				t.model.ScrollDown(1)
			}
		case UITools:
			switch buttons {
			case tcell.Button1:
				cat := t.model.GetCategoryForPos(x, y)
				if cat != nil {
					t.model.CurrentCategory = *cat
				}
			}
		default:
		}
		if pane != UIEvents {
			t.model.ClearHover()
		}
	}
}

func (t *TUIController) resizeStep(newY int) {
	delta := newY - t.model.cursorY
	offset := t.model.TimeForDistance(delta)
	event := t.model.GetCurrentDayModel().GetEvent(t.EditedEvent)
	event.End = event.End.Offset(offset).Snap(t.model.Resolution)
}

func (t *TUIController) moveStep(newY int) {
	delta := newY - t.model.cursorY
	offset := t.model.TimeForDistance(delta)
	if t.movePropagate {
		following := t.model.GetCurrentDayModel().GetEventsFrom(t.EditedEvent)
		for _, ptr := range following {
			ptr.Start = ptr.Start.Offset(offset).Snap(t.model.Resolution)
			ptr.End = ptr.End.Offset(offset).Snap(t.model.Resolution)
		}
	} else {
		event := t.model.GetCurrentDayModel().GetEvent(t.EditedEvent)
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
		editor := &t.model.EventEditor

		switch e.Key() {
		case tcell.KeyEsc:
			t.abortEdit()

		case tcell.KeyEnter:
			t.endEdit()

		case tcell.KeyDelete, tcell.KeyCtrlD:
			editor.deleteRune()

		case tcell.KeyBackspace, tcell.KeyBackspace2:
			editor.backspaceRune()

		case tcell.KeyCtrlE:
			editor.moveCursorToEnd()

		case tcell.KeyCtrlA:
			editor.moveCursorToBeginning()

		case tcell.KeyCtrlU:
			editor.backspaceToBeginning()

		case tcell.KeyLeft:
			editor.moveCursorLeft()

		case tcell.KeyRight:
			editor.moveCursorRight()

		default:
			editor.addRune(e.Rune())
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

type ControllerEvent int

const (
	ControllerEventExit ControllerEvent = iota
	ControllerEventRender
)

func (t *TUIController) Run() {

	t.bump = make(chan ControllerEvent)
	var wg sync.WaitGroup

	// Run the main render loop, that renders or exits when prompted accordingly
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			controllerEvent := <-t.bump
			switch controllerEvent {
			case ControllerEventRender:
				t.view.Render()
			case ControllerEventExit:
				return
			}
		}
	}()

	// Run the time tracking loop, that updates at the start of every minute
	go func() {
		for {
			now := time.Now()
			next := now.Round(1 * time.Minute).Add(1 * time.Minute)
			time.Sleep(time.Until(next))
			t.bump <- ControllerEventRender
		}
	}()

	// Run the event tracking loop, that waits for and processes events and pings
	// for a redraw (or program exit) after each event.
	go func() {
		for {
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

			t.bump <- ControllerEventRender
		}
	}()

	wg.Wait()
}
