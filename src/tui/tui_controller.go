package tui

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"dayplan/src/category_style"
	"dayplan/src/model"
	"dayplan/src/program"
	"dayplan/src/weather"

	"github.com/gdamore/tcell/v2"
)

// TODO: this absolutely does not belong here
func (t *TUIController) GetModelFromFileHandler(date model.Date) *model.Model {
	fh, ok := t.FileHandlers[date]
	if ok {
		tmp := fh.Read()
		return tmp
	} else {
		newHandler := NewFileHandler(t.ProgramData.BaseDirPath + "/days/" + date.ToString())
		t.FileHandlers[date] = newHandler
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
	FileHandlers  map[model.Date]*FileHandler
	ProgramData   program.Data
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

func NewTUIController(date model.Date, programData program.Data) *TUIController {
	// read category styles
	var categoryStyling category_style.CategoryStyling
	categoryStyling = *category_style.EmptyCategoryStyling()
	f, err := os.Open(programData.BaseDirPath + "/" + "category-styles")
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			s := scanner.Text()
			categoryStyling.AddStyleFromCfg(s)
		}
		f.Close()
	} else {
		categoryStyling = *category_style.DefaultCategoryStyling()
	}

	tuiModel := NewTUIModel(categoryStyling)
	tuiView := NewTUIView(tuiModel) // <- stuck here!

	coordinatesProvided := (programData.Latitude != "" && programData.Longitude != "")
	owmApiKeyProvided := (programData.OwmApiKey != "")

	// intialize weather handler if geographic location and api key provided
	if coordinatesProvided && owmApiKeyProvided {
		tuiModel.Weather = *weather.NewHandler(programData.Latitude, programData.Longitude, programData.OwmApiKey)
	} else {
		if !owmApiKeyProvided {
			tuiModel.Log.Add("ERROR", "no OWM API key provided -> no weather data")
		}
		if !coordinatesProvided {
			tuiModel.Log.Add("ERROR", "no lat-/longitude provided -> no weather data")
		}
	}

	// set sunrise and sunset if geographic location provided
	// TODO: actually a per-day thing that is being set only once for initial day, fix
	if coordinatesProvided {
		latF, _ := strconv.ParseFloat(programData.Latitude, 64)
		lonF, _ := strconv.ParseFloat(programData.Longitude, 64)
		sunrise, sunset, err := date.GetSunTimes(latF, lonF)
		if err != nil {
			tuiModel.Log.Add("ERROR", fmt.Sprintf("could not get sunrise/-set: '%s'", err))
			tuiModel.SunTimes.Rise = *model.NewTimestamp("00:00")
			tuiModel.SunTimes.Set = *model.NewTimestamp("23:59")
		} else {
			tuiModel.SunTimes.Rise = sunrise
			tuiModel.SunTimes.Set = sunset
		}
	} else {
		// TODO: these settings should obviously be made by one function, however
		//       since the data should also be moved to per-day (or be computed
		//       each day switch?) this is kind of moot
		tuiModel.SunTimes.Rise = *model.NewTimestamp("00:00")
		tuiModel.SunTimes.Set = *model.NewTimestamp("23:59")

		tuiModel.Log.Add("ERROR", "could not fetch lat-&longitude -> no sunrise/-set times known")
	}

	tuiController := TUIController{}
	tuiController.ProgramData = programData

	tuiController.FileHandlers = make(map[model.Date]*FileHandler)
	tuiController.FileHandlers[date] = NewFileHandler(tuiController.ProgramData.BaseDirPath + "/days/" + date.ToString())

	tuiController.model = tuiModel
	tuiController.model.CurrentDate = date
	if tuiController.FileHandlers[date] == nil {
		tuiController.model.AddModel(date, &model.Model{})
	} else {
		tuiController.model.AddModel(date, tuiController.FileHandlers[date].Read())
	}

	tuiController.view = tuiView
	tuiController.model.CurrentCategory.Name = "default"

	return &tuiController
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
		t.model.GetCurrentDateModel().GetEvent(tmp.ID).Name = tmp.Name
	}
	t.model.GetCurrentDateModel().UpdateEventOrder()
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
	newEventID := t.model.GetCurrentDateModel().AddEvent(e)

	// save ID as edited event
	t.EditedEvent = newEventID

	// set mode to resizing
	t.editState = (EditStateMouseEditing | EditStateResizing)
}

func (t *TUIController) goToDay(newDate model.Date) {
	t.model.Log.Add("DEBUG", "going to "+newDate.ToString())

	t.model.Status.Set("day", newDate.ToString())

	if !t.model.HasModel(newDate) {
		// load file
		newModel := t.GetModelFromFileHandler(newDate)
		if newModel == nil {
			panic("newModel nil?!")
		}
		t.model.AddModel(newDate, newModel)
	}

	t.model.CurrentDate = newDate
}

func (t *TUIController) goToPreviousDay() {
	prevDay := t.model.CurrentDate.Prev()
	t.goToDay(prevDay)
}

func (t *TUIController) goToNextDay() {
	nextDay := t.model.CurrentDate.Next()
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
		t.model.AddModel(t.model.CurrentDate, model.NewModel())
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
	go t.FileHandlers[t.model.CurrentDate].Write(t.model.GetCurrentDateModel())
}

func (t *TUIController) updateCursorPos(x, y int) {
	t.model.cursorX, t.model.cursorY = x, y
}

func (t *TUIController) startEdit(id model.EventID) {
	t.model.EventEditor.Active = true
	t.model.EventEditor.TmpEventInfo = *t.model.GetCurrentDateModel().GetEvent(id)
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
				t.model.GetCurrentDateModel().RemoveEvent(t.model.Hovered.EventID)
			case tcell.Button2:
				id := t.model.Hovered.EventID
				if id != 0 && t.model.TimeAtY(y).IsAfter(t.model.GetCurrentDateModel().GetEvent(id).Start) {
					t.model.GetCurrentDateModel().SplitEvent(id, t.model.TimeAtY(y))
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
	event := t.model.GetCurrentDateModel().GetEvent(t.EditedEvent)
	event.End = event.End.Offset(offset).Snap(t.model.Resolution)
}

func (t *TUIController) moveStep(newY int) {
	delta := newY - t.model.cursorY
	offset := t.model.TimeForDistance(delta)
	if t.movePropagate {
		following := t.model.GetCurrentDateModel().GetEventsFrom(t.EditedEvent)
		for _, ptr := range following {
			ptr.Start = ptr.Start.Offset(offset).Snap(t.model.Resolution)
			ptr.End = ptr.End.Offset(offset).Snap(t.model.Resolution)
		}
	} else {
		event := t.model.GetCurrentDateModel().GetEvent(t.EditedEvent)
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
		defer t.view.Screen.Fini()
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
