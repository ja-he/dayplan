package tui

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ja-he/dayplan/src/category_style"
	"github.com/ja-he/dayplan/src/filehandling"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/program"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/util"
	"github.com/ja-he/dayplan/src/weather"

	"github.com/gdamore/tcell/v2"
)

// TODO: this absolutely does not belong here
func (t *TUIController) GetDayFromFileHandler(date model.Date) *model.Day {
	t.fhMutex.RLock()
	fh, ok := t.FileHandlers[date]
	t.fhMutex.RUnlock()
	if ok {
		tmp := fh.Read(t.model.CategoryStyling.GetKnownCategoriesByName())
		return tmp
	} else {
		newHandler := filehandling.NewFileHandler(t.model.ProgramData.BaseDirPath + "/days/" + date.ToString())
		t.fhMutex.Lock()
		t.FileHandlers[date] = newHandler
		t.fhMutex.Unlock()
		tmp := newHandler.Read(t.model.CategoryStyling.GetKnownCategoriesByName())
		return tmp
	}
}

type EditedEvent struct {
	ID                    model.EventID
	prevEditStepTimestamp model.Timestamp
}

type TUIController struct {
	model         *TUIModel
	tui           ui.MainUIPanel
	editState     EditState
	EditedEvent   EditedEvent
	movePropagate bool
	fhMutex       sync.RWMutex
	FileHandlers  map[model.Date]*filehandling.FileHandler
	bump          chan ControllerEvent
	// NOTE(ja-he):
	//   a `tcell.Screen` unites rendering and event polling functionalities.
	//   holding this interface pointer allows us to leave the rendering to the
	//   panels which use the renderer which uses the screen, while still being
	//   able to use the screen here to get events. The name should indicate that
	//   usage of this field should be restricted to event polling.
	screenEvents *tcell.Screen
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
	styleFilePath := programData.BaseDirPath + "/" + "category-styles.yaml"
	styledInputs, err := category_style.ReadCategoryStylingFile(styleFilePath)
	if err != nil {
		panic(err)
	}
	for _, styledInput := range styledInputs {
		categoryStyling.AddStyleFromInput(styledInput)
	}

	renderer := NewTUIRenderer()
	tuiModel := NewTUIModel(categoryStyling)
	w, h := renderer.GetScreenDimensions()
	tui := TUI{
		renderer: renderer,

		tools: &ToolsPanel{
			renderer:        renderer,
			currentCategory: &tuiModel.CurrentCategory,
			categories:      &tuiModel.CategoryStyling,
			horizPadding:    1,
			vertPadding:     1,
			gap:             0,
		},
		status: &StatusPanel{
			renderer: renderer,

			currentDate: &tuiModel.CurrentDate,
			activeView:  &tuiModel.activeView,
		},

		days:            &tuiModel.Days,
		currentDate:     &tuiModel.CurrentDate,
		currentCategory: &tuiModel.CurrentCategory,
		categories:      &tuiModel.CategoryStyling,
		eventEditor:     &tuiModel.EventEditor,
		showHelp:        &tuiModel.showHelp,
		showSummary:     &tuiModel.showSummary,
		showLog:         &tuiModel.showLog,
		activeView:      &tuiModel.activeView,
		logReader:       &tuiModel.Log,
		logWriter:       &tuiModel.Log,
		weather:         &tuiModel.Weather,
		viewParams:      &tuiModel.ViewParams,
		cursor:          &tuiModel.cursorPos,

		positions: make(map[model.EventID]util.Rect),
	}
	tui.uiDimensions.Initialize(20, 10, 20, w, h)
	tuiModel.UIDim = &tui.uiDimensions

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

	// process latitude longitude
	// TODO
	var suntimes model.SunTimes
	if !coordinatesProvided {
		tuiModel.Log.Add("ERROR", "could not fetch lat-&longitude -> no sunrise/-set times known")
	} else {
		latF, parseErr := strconv.ParseFloat(programData.Latitude, 64)
		lonF, parseErr := strconv.ParseFloat(programData.Longitude, 64)
		if parseErr != nil {
			tuiModel.Log.Add("ERROR", fmt.Sprint("parse error:", parseErr))
		} else {
			suntimes = date.GetSunTimes(latF, lonF)
		}
	}

	tuiController := TUIController{}
	tuiModel.ProgramData = programData
	tuiController.screenEvents = renderer.GetEventPollable()

	tuiController.fhMutex.Lock()
	defer tuiController.fhMutex.Unlock()
	tuiController.FileHandlers = make(map[model.Date]*filehandling.FileHandler)
	tuiController.FileHandlers[date] = filehandling.NewFileHandler(tuiModel.ProgramData.BaseDirPath + "/days/" + date.ToString())

	tuiController.model = tuiModel
	tuiController.model.CurrentDate = date
	if tuiController.FileHandlers[date] == nil {
		tuiController.model.Days.AddDay(date, &model.Day{}, &suntimes)
	} else {
		tuiController.model.Days.AddDay(date, tuiController.FileHandlers[date].Read(tuiController.model.CategoryStyling.GetKnownCategoriesByName()), &suntimes)
	}

	tuiController.tui = &tui
	tuiController.model.CurrentCategory.Name = "default"

	tuiController.loadDaysForView(tuiController.model.activeView)

	return &tuiController
}

func (t *TUIController) abortEdit() {
	t.editState = EditStateNone
	t.EditedEvent = EditedEvent{0, model.Timestamp{Hour: 0, Minute: 0}}
	t.model.EventEditor.Active = false
}

func (t *TUIController) endEdit() {
	t.editState = EditStateNone
	t.EditedEvent = EditedEvent{0, model.Timestamp{Hour: 0, Minute: 0}}
	if t.model.EventEditor.Active {
		t.model.EventEditor.Active = false
		tmp := t.model.EventEditor.TmpEventInfo
		t.model.GetCurrentDay().GetEvent(tmp.ID).Name = tmp.Name
	}
	t.model.GetCurrentDay().UpdateEventOrder()
}

func (t *TUIController) startMouseMove(eventsInfo ui.EventsPanelPositionInfo) {
	t.editState = (EditStateMouseEditing | EditStateMoving)
	t.EditedEvent.ID = eventsInfo.Event
	t.EditedEvent.prevEditStepTimestamp = eventsInfo.TimeUnderCursor
}

func (t *TUIController) startMouseResize(eventsInfo ui.EventsPanelPositionInfo) {
	t.editState = (EditStateMouseEditing | EditStateResizing)
	t.EditedEvent.ID = eventsInfo.Event
	t.EditedEvent.prevEditStepTimestamp = eventsInfo.TimeUnderCursor
}

func (t *TUIController) startMouseEventCreation(info ui.EventsPanelPositionInfo) {
	// find out cursor time
	start := info.TimeUnderCursor

	t.model.Log.Add("DEBUG", fmt.Sprintf("creation called for '%s'", info.TimeUnderCursor.ToString()))

	// create event at time with cat etc.
	e := model.Event{}
	e.Cat = t.model.CurrentCategory
	e.Name = ""
	e.Start = start
	e.End = start.OffsetMinutes(+10)

	// give to model, get ID
	newEventID := t.model.GetCurrentDay().AddEvent(e)

	// save ID as edited event
	t.EditedEvent.ID = newEventID
	t.EditedEvent.prevEditStepTimestamp = e.Start

	// set mode to resizing
	t.editState = (EditStateMouseEditing | EditStateResizing)
}

func (t *TUIController) goToDay(newDate model.Date) {
	t.model.Log.Add("DEBUG", "going to "+newDate.ToString())

	t.model.CurrentDate = newDate
	t.loadDaysForView(t.model.activeView)
}

func (t *TUIController) goToPreviousDay() {
	prevDay := t.model.CurrentDate.Prev()
	t.goToDay(prevDay)
}

func (t *TUIController) goToNextDay() {
	nextDay := t.model.CurrentDate.Next()
	t.goToDay(nextDay)
}

// Loads the requested date's day from its file handler, if it has
// not already been loaded.
func (t *TUIController) loadDay(date model.Date) {
	if !t.model.Days.HasDay(date) {
		// load file
		newDay := t.GetDayFromFileHandler(date)
		if newDay == nil {
			panic("newDay nil?!")
		}

		var suntimes model.SunTimes
		coordinatesProvided := (t.model.ProgramData.Latitude != "" && t.model.ProgramData.Longitude != "")
		if coordinatesProvided {
			latF, parseErr := strconv.ParseFloat(t.model.ProgramData.Latitude, 64)
			lonF, parseErr := strconv.ParseFloat(t.model.ProgramData.Longitude, 64)
			if parseErr != nil {
				t.model.Log.Add("ERROR", fmt.Sprint("parse error:", parseErr))
			} else {
				suntimes = date.GetSunTimes(latF, lonF)
			}
		}

		t.model.Days.AddDay(date, newDay, &suntimes)
	}
}

// Starts Loads for all days visible in the view.
// E.g. for ui.ViewMonth it would start the load for all days from
// first to last day of the month.
// Warning: does not guarantee days will be loaded (non-nil) after
// this returns.
func (t *TUIController) loadDaysForView(view ui.ActiveView) {
	switch view {
	case ui.ViewDay:
		t.loadDay(t.model.CurrentDate)
	case ui.ViewWeek:
		{
			monday, sunday := t.model.CurrentDate.Week()
			for current := monday; current != sunday.Next(); current = current.Next() {
				go func(d model.Date) {
					t.loadDay(d)
					t.bump <- ControllerEventRender
				}(current)
			}
		}
	case ui.ViewMonth:
		{
			first, last := t.model.CurrentDate.MonthBounds()
			for current := first; current != last.Next(); current = current.Next() {
				go func(d model.Date) {
					t.loadDay(d)
					t.bump <- ControllerEventRender
				}(current)
			}
		}
	default:
		panic("unknown ActiveView")
	}
}

func (t *TUIController) handleNoneEditKeyInput(e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyESC:
		prevView := PrevView(t.model.activeView)
		t.loadDaysForView(prevView)
		t.model.activeView = prevView
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
		}()
	case 'i':
		nextView := NextView(t.model.activeView)
		t.loadDaysForView(nextView)
		t.model.activeView = nextView
	case 'q':
		t.bump <- ControllerEventExit
	case 'g':
		t.model.ScrollTop()
	case 'G':
		t.model.ScrollBottom()
	case 'w':
		t.writeModel()
	case 'j':
		t.model.ScrollDown(1)
	case 'k':
		t.model.ScrollUp(1)
	case 'h':
		t.goToPreviousDay()
	case 'l':
		t.goToNextDay()
	case 'S':
		t.model.showSummary = !t.model.showSummary
	case 'E':
		t.model.showLog = !t.model.showLog
	case '?':
		t.model.showHelp = !t.model.showHelp
	case 'c':
		// TODO: all that's needed to clear model (appropriately)?
		t.model.Days.AddDay(t.model.CurrentDate, model.NewDay(), t.model.GetCurrentSuntimes())
	case '+':
		if t.model.ViewParams.NRowsPerHour*2 <= 12 {
			t.model.ViewParams.NRowsPerHour *= 2
			t.model.ViewParams.ScrollOffset *= 2
		}
	case '-':
		if (t.model.ViewParams.NRowsPerHour % 2) == 0 {
			t.model.ViewParams.NRowsPerHour /= 2
			t.model.ViewParams.ScrollOffset /= 2
		} else {
			t.model.Log.Add("WARNING", fmt.Sprintf("can't decrease resolution below %d", t.model.ViewParams.NRowsPerHour))
		}
	}
}

func (t *TUIController) writeModel() {
	go func() {
		t.fhMutex.RLock()
		t.FileHandlers[t.model.CurrentDate].Write(t.model.GetCurrentDay())
		t.fhMutex.RUnlock()
	}()
}

func (t *TUIController) updateCursorPos(x, y int) {
	t.model.cursorPos.X, t.model.cursorPos.Y = x, y
}

func (t *TUIController) startEdit(id model.EventID) {
	t.model.EventEditor.Active = true
	t.model.EventEditor.TmpEventInfo = *t.model.GetCurrentDay().GetEvent(id)
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

		positionInfo := t.tui.GetPositionInfo(x, y)
		if positionInfo == nil {
			return
		}

		buttons := e.Buttons()

		paneType := positionInfo.PaneType()
		switch paneType {
		case ui.StatusUIPanelType:
		case ui.WeatherUIPanelType:
			switch buttons {
			case tcell.WheelUp:
				t.model.ScrollUp(1)
			case tcell.WheelDown:
				t.model.ScrollDown(1)
			}
		case ui.TimelineUIPanelType:
			switch buttons {
			case tcell.WheelUp:
				t.model.ScrollUp(1)
			case tcell.WheelDown:
				t.model.ScrollDown(1)
			}
		case ui.EventsUIPanelType:
			eventsInfo := *positionInfo.GetExtraEventsInfo()
			t.model.Log.Add("DEBUG", fmt.Sprint(eventsInfo))

			// if button clicked, handle
			switch buttons {
			case tcell.Button3:
				t.model.GetCurrentDay().RemoveEvent(eventsInfo.Event)
			case tcell.Button2:
				id := eventsInfo.Event
				if id != 0 && eventsInfo.TimeUnderCursor.IsAfter(t.model.GetCurrentDay().GetEvent(id).Start) {
					t.model.GetCurrentDay().SplitEvent(id, eventsInfo.TimeUnderCursor)
				}
			case tcell.Button1:
				// we've clicked while not editing
				// now we need to check where the cursor is and either start event
				// creation, resizing or moving
				switch eventsInfo.HoverState {
				case ui.EventHoverStateNone:
					t.startMouseEventCreation(eventsInfo)
				case ui.EventHoverStateResize:
					t.startMouseResize(eventsInfo)
				case ui.EventHoverStateMove:
					t.movePropagate = (e.Modifiers() == tcell.ModCtrl)
					t.startMouseMove(eventsInfo)
				case ui.EventHoverStateEdit:
					t.startEdit(eventsInfo.Event)
				}
			case tcell.WheelUp:
				t.model.ScrollUp(1)
			case tcell.WheelDown:
				t.model.ScrollDown(1)
			}
		case ui.ToolsUIPanelType:
			toolsInfo := *positionInfo.GetExtraToolsInfo()
			t.model.Log.Add("DEBUG", fmt.Sprint("tools info:", toolsInfo))
			switch buttons {
			case tcell.Button1:
				cat := toolsInfo.Category
				if cat != nil {
					t.model.CurrentCategory = *cat
				}
			}
		default:
		}
	}
}

func (t *TUIController) resizeStep(nextCursortime model.Timestamp) {
	prevCursortime := t.EditedEvent.prevEditStepTimestamp
	offset := prevCursortime.DurationInMinutesUntil(nextCursortime)
	event := t.model.GetCurrentDay().GetEvent(t.EditedEvent.ID)
	event.End = event.End.OffsetMinutes(offset).Snap(t.model.ViewParams.NRowsPerHour)
	t.EditedEvent.prevEditStepTimestamp = nextCursortime
}

func (t *TUIController) moveStep(nextCursortime model.Timestamp) {
	prevCursortime := t.EditedEvent.prevEditStepTimestamp
	offset := prevCursortime.DurationInMinutesUntil(nextCursortime)
	if t.movePropagate {
		following := t.model.GetCurrentDay().GetEventsFrom(t.EditedEvent.ID)
		for _, ptr := range following {
			ptr.Start = ptr.Start.OffsetMinutes(offset).Snap(t.model.ViewParams.NRowsPerHour)
			ptr.End = ptr.End.OffsetMinutes(offset).Snap(t.model.ViewParams.NRowsPerHour)
		}
	} else {
		event := t.model.GetCurrentDay().GetEvent(t.EditedEvent.ID)
		event.Start = event.Start.OffsetMinutes(offset).Snap(t.model.ViewParams.NRowsPerHour)
		event.End = event.End.OffsetMinutes(offset).Snap(t.model.ViewParams.NRowsPerHour)
	}
	t.EditedEvent.prevEditStepTimestamp = nextCursortime
}

func (t *TUIController) handleMouseResizeEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		x, y := e.Position()

		positionInfo := t.tui.GetPositionInfo(x, y)

		buttons := e.Buttons()

		switch buttons {
		case tcell.Button1:
			timeUnderCursor, _ := positionInfo.GetCursorTimestampGuess()
			t.resizeStep(*timeUnderCursor)
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
		positionInfo := t.tui.GetPositionInfo(x, y)

		buttons := e.Buttons()

		switch buttons {
		case tcell.Button1:
			timestampGuess, _ := positionInfo.GetCursorTimestampGuess()
			t.moveStep(*timestampGuess)
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

// Empties all render events from the channel.
// Returns true, if an exit event was encountered so the caller
// knows to exit.
func emptyRenderEvents(c chan ControllerEvent) bool {
	for {
		select {
		case bufferedEvent := <-c:
			switch bufferedEvent {
			case ControllerEventRender:
				{
					// dump extra render events
				}
			case ControllerEventExit:
				return true
			}
		default:
			return false
		}
	}
}

func (t *TUIController) Run() {

	t.bump = make(chan ControllerEvent, 32)
	var wg sync.WaitGroup

	// Run the main render loop, that renders or exits when prompted accordingly
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer t.tui.Close()
		for {

			select {
			case controllerEvent := <-t.bump:
				switch controllerEvent {
				case ControllerEventRender:
					// empty all further render events before rendering
					exitEventEncounteredOnEmpty := emptyRenderEvents(t.bump)
					// exit if an exit event was coming up
					if exitEventEncounteredOnEmpty {
						return
					}
					// render
					t.tui.Draw(0, 0, t.model.UIDim.screenWidth, t.model.UIDim.screenHeight)
				case ControllerEventExit:
					return
				}
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
			ev := (*t.screenEvents).PollEvent()

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
				t.tui.NeedsSync()
				t.model.UIDim.ScreenResize((*t.screenEvents).Size())
			}

			t.bump <- ControllerEventRender
		}
	}()

	wg.Wait()
}
