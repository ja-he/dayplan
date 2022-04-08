package control

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ja-he/dayplan/src/filehandling"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/styling"
	"github.com/ja-he/dayplan/src/tui"
	"github.com/ja-he/dayplan/src/ui"
	"github.com/ja-he/dayplan/src/ui/panes"
	"github.com/ja-he/dayplan/src/util"
	"github.com/ja-he/dayplan/src/weather"

	"github.com/gdamore/tcell/v2"
)

// TODO: this absolutely does not belong here
func (t *Controller) GetDayFromFileHandler(date model.Date) *model.Day {
	t.fhMutex.RLock()
	fh, ok := t.FileHandlers[date]
	t.fhMutex.RUnlock()
	if ok {
		tmp := fh.Read(t.data.CategoryStyling.GetKnownCategoriesByName())
		return tmp
	} else {
		newHandler := filehandling.NewFileHandler(t.data.EnvData.BaseDirPath + "/days/" + date.ToString())
		t.fhMutex.Lock()
		t.FileHandlers[date] = newHandler
		t.fhMutex.Unlock()
		tmp := newHandler.Read(t.data.CategoryStyling.GetKnownCategoriesByName())
		return tmp
	}
}

type EditedEvent struct {
	ID                    model.EventID
	prevEditStepTimestamp model.Timestamp
}

type Controller struct {
	data             *ControlData
	rootPane         ui.Pane
	editState        EditState
	EditedEvent      EditedEvent
	movePropagate    bool
	fhMutex          sync.RWMutex
	FileHandlers     map[model.Date]*filehandling.FileHandler
	controllerEvents chan ControllerEvent

	// TODO: remove, obviously
	tmpStatusYOffsetGetter func() int

	// When creating or editing events with the mouse, we probably don't want to
	// end the edit if the mouse leaves the events pane. Instead the more
	// intuitive behavior for users is that it simply continue as long as the
	// mouse button is held, regardless of the actual pane under the cursor.
	// This helps guess at timestamps for those edits without having the panes
	// awkwardly accessing information that they shouldn't need to.
	timestampGuesser func(int, int) model.Timestamp

	screenEvents      tui.EventPollable
	initializedScreen tui.InitializedScreen
	syncer            tui.ScreenSynchronizer
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

func NewController(date model.Date, envData EnvData, categoryStyling styling.CategoryStyling, stylesheet styling.Stylesheet) *Controller {

	data := NewControlData(categoryStyling)

	toolsWidth := 20
	statusHeight := 2
	weatherWidth := 20
	timelineWidth := 10
	helpWidth := 80
	helpHeight := 30
	editorWidth := 80
	editorHeight := 20

	renderer := tui.NewTUIScreenHandler()
	screenSize := renderer.GetScreenDimensions
	screenDimensions := func() (x, y, w, h int) {
		screenWidth, screenHeight := screenSize()
		return 0, 0, screenWidth, screenHeight
	}
	centeredFloat := func(floatWidth, floatHeight int) func() (x, y, w, h int) {
		return func() (x, y, w, h int) {
			screenWidth, screenHeight := screenSize()
			return (screenWidth / 2) - (floatWidth / 2), (screenHeight / 2) - (floatHeight / 2), floatWidth, floatHeight
		}
	}
	helpDimensions := centeredFloat(helpWidth, helpHeight)
	editorDimensions := centeredFloat(editorWidth, editorHeight)
	toolsDimensions := func() (x, y, w, h int) {
		screenWidth, screeenHeight := screenSize()
		return screenWidth - toolsWidth, 0, toolsWidth, screeenHeight - statusHeight
	}
	statusDimensions := func() (x, y, w, h int) {
		screenWidth, screeenHeight := screenSize()
		return 0, screeenHeight - statusHeight, screenWidth, statusHeight
	}
	dayViewMainPaneDimensions := screenDimensions
	weekViewMainPaneDimensions := screenDimensions
	monthViewMainPaneDimensions := screenDimensions
	weatherDimensions := func() (x, y, w, h int) {
		mainPaneXOffset, mainPaneYOffset, _, mainPaneHeight := dayViewMainPaneDimensions()
		return mainPaneXOffset, mainPaneYOffset, weatherWidth, mainPaneHeight - statusHeight
	}
	dayViewEventsPaneDimensions := func() (x, y, w, h int) {
		ox, oy, ow, oh := dayViewMainPaneDimensions()
		x = ox + weatherWidth + timelineWidth
		y = oy
		w = ow - x - toolsWidth
		h = oh - statusHeight
		return x, y, w, h
	}
	dayViewTimelineDimensions := func() (x, y, w, h int) {
		_, _, _, mainPaneHeight := dayViewMainPaneDimensions()
		return 0 + weatherWidth, 0, timelineWidth, mainPaneHeight - statusHeight
	}
	weekViewTimelineDimensions := func() (x, y, w, h int) {
		_, screenHeight := screenSize()
		return 0, 0, timelineWidth, screenHeight - statusHeight
	}
	monthViewTimelineDimensions := weekViewTimelineDimensions
	weekdayDimensions := func(dayIndex int) func() (x, y, w, h int) {
		return func() (x, y, w, h int) {
			baseX, baseY, baseW, baseH := weekViewMainPaneDimensions()
			eventsWidth := baseW - timelineWidth
			dayWidth := eventsWidth / 7
			return baseX + timelineWidth + (dayIndex * dayWidth), baseY, dayWidth, baseH - statusHeight
		}
	}
	weekdayPane := func(dayIndex int) ui.Pane {
		return panes.NewEventsPane(
			tui.NewConstrainedRenderer(renderer, weekdayDimensions(dayIndex)),
			weekdayDimensions(dayIndex),
			stylesheet,
			func() *model.Day { return data.Days.GetDay(data.CurrentDate.GetDayInWeek(dayIndex)) },
			&data.CategoryStyling,
			&data.ViewParams,
			&data.cursorPos,
			0,
			false,
			true,
			func() bool { return data.CurrentDate.GetDayInWeek(dayIndex) == data.CurrentDate },
			&data.Log,
			&data.Log,
			make(map[model.EventID]util.Rect),
		)
	}
	monthdayDimensions := func(dayIndex int) func() (x, y, w, h int) {
		return func() (x, y, w, h int) {
			baseX, baseY, baseW, baseH := monthViewMainPaneDimensions()
			eventsWidth := baseW - timelineWidth
			dayWidth := eventsWidth / 31
			return baseX + timelineWidth + (dayIndex * dayWidth), baseY, dayWidth, baseH - statusHeight
		}
	}
	monthdayPane := func(dayIndex int) ui.Pane {
		return panes.NewMaybeEventsPane(
			func() bool { return data.CurrentDate.GetDayInMonth(dayIndex).Month == data.CurrentDate.Month },
			panes.NewEventsPane(
				tui.NewConstrainedRenderer(renderer, monthdayDimensions(dayIndex)),
				monthdayDimensions(dayIndex),
				stylesheet,
				func() *model.Day { return data.Days.GetDay(data.CurrentDate.GetDayInMonth(dayIndex)) },
				&data.CategoryStyling,
				&data.ViewParams,
				&data.cursorPos,
				0,
				false,
				false,
				func() bool { return data.CurrentDate.GetDayInMonth(dayIndex) == data.CurrentDate },
				&data.Log,
				&data.Log,
				make(map[model.EventID]util.Rect),
			),
		)
	}

	weekViewEventsPanes := make([]ui.Pane, 7)
	for i := range weekViewEventsPanes {
		weekViewEventsPanes[i] = weekdayPane(i)
	}

	monthViewEventsPanes := make([]ui.Pane, 31)
	for i := range monthViewEventsPanes {
		monthViewEventsPanes[i] = monthdayPane(i)
	}

	statusPane := panes.NewStatusPane(
		tui.NewConstrainedRenderer(renderer, statusDimensions),
		statusDimensions,
		stylesheet,
		&data.CurrentDate,
		func() int {
			_, _, w, _ := statusDimensions()
			switch data.activeView {
			case ui.ViewDay:
				return w - timelineWidth
			case ui.ViewWeek:
				return (w - timelineWidth) / 7
			case ui.ViewMonth:
				return (w - timelineWidth) / 31
			default:
				panic("unknown view for status rendering")
			}
		},
		func() int {
			switch data.activeView {
			case ui.ViewDay:
				return 1
			case ui.ViewWeek:
				return 7
			case ui.ViewMonth:
				return data.CurrentDate.GetLastOfMonth().Day
			default:
				panic("unknown view for status rendering")
			}
		},
		func() int {
			switch data.activeView {
			case ui.ViewDay:
				return 1
			case ui.ViewWeek:
				switch data.CurrentDate.ToWeekday() {
				case time.Monday:
					return 1
				case time.Tuesday:
					return 2
				case time.Wednesday:
					return 3
				case time.Thursday:
					return 4
				case time.Friday:
					return 5
				case time.Saturday:
					return 6
				case time.Sunday:
					return 7
				default:
					panic("unknown weekday for status rendering")
				}
			case ui.ViewMonth:
				return data.CurrentDate.Day
			default:
				panic("unknown view for status rendering")
			}
		},
		func() int { return timelineWidth },
	)

	rootPane := panes.NewRootPane(
		renderer,
		screenDimensions,

		panes.NewDayViewMainPane(
			dayViewMainPaneDimensions,
			panes.NewEventsPane(
				tui.NewConstrainedRenderer(renderer, dayViewEventsPaneDimensions),
				dayViewEventsPaneDimensions,
				stylesheet,
				data.GetCurrentDay,
				&data.CategoryStyling,
				&data.ViewParams,
				&data.cursorPos,
				2,
				true,
				true,
				func() bool { return true },
				&data.Log,
				&data.Log,
				make(map[model.EventID]util.Rect),
			),
			panes.NewToolsPane(
				tui.NewConstrainedRenderer(renderer, toolsDimensions),
				toolsDimensions,
				stylesheet,
				&data.CurrentCategory,
				&data.CategoryStyling,
				1,
				1,
				0,
			),
			statusPane,
			panes.NewTimelinePane(
				tui.NewConstrainedRenderer(renderer, dayViewTimelineDimensions),
				dayViewTimelineDimensions,
				stylesheet,
				data.GetCurrentSuntimes,
				func() *model.Timestamp {
					if data.CurrentDate.Is(time.Now()) {
						return model.NewTimestampFromGotime(time.Now())
					} else {
						return nil
					}
				},
				&data.ViewParams,
			),
			panes.NewWeatherPane(
				tui.NewConstrainedRenderer(renderer, weatherDimensions),
				weatherDimensions,
				stylesheet,
				&data.CurrentDate,
				&data.Weather,
				&data.ViewParams,
			),
		),
		panes.NewWeekViewMainPane(
			weekViewMainPaneDimensions,
			statusPane,
			panes.NewTimelinePane(
				tui.NewConstrainedRenderer(renderer, weekViewTimelineDimensions),
				weekViewTimelineDimensions,
				stylesheet,
				func() *model.SunTimes { return nil },
				func() *model.Timestamp { return nil },
				&data.ViewParams,
			),
			weekViewEventsPanes,
			&data.CategoryStyling,
			&data.Log,
			&data.Log,
			&data.ViewParams,
		),
		panes.NewMonthViewMainPane(
			monthViewMainPaneDimensions,
			statusPane,
			panes.NewTimelinePane(
				tui.NewConstrainedRenderer(renderer, monthViewTimelineDimensions),
				monthViewTimelineDimensions,
				stylesheet,
				func() *model.SunTimes { return nil },
				func() *model.Timestamp { return nil },
				&data.ViewParams,
			),
			monthViewEventsPanes,
			&data.CategoryStyling,
			&data.Log,
			&data.Log,
			&data.ViewParams,
		),

		panes.NewSummaryPane(
			tui.NewConstrainedRenderer(renderer, screenDimensions),
			screenDimensions,
			stylesheet,
			func() bool { return data.showSummary },
			func() string {
				dateString := ""
				switch data.activeView {
				case ui.ViewDay:
					dateString = data.CurrentDate.ToString()
				case ui.ViewWeek:
					start, end := data.CurrentDate.Week()
					dateString = fmt.Sprintf("week %s..%s", start.ToString(), end.ToString())
				case ui.ViewMonth:
					dateString = fmt.Sprintf("%s %d", data.CurrentDate.ToGotime().Month().String(), data.CurrentDate.Year)
				}
				return fmt.Sprintf("SUMMARY (%s)", dateString)
			},
			func() []*model.Day {
				switch data.activeView {
				case ui.ViewDay:
					result := make([]*model.Day, 1)
					result[0] = data.Days.GetDay(data.CurrentDate)
					return result
				case ui.ViewWeek:
					result := make([]*model.Day, 7)
					start, end := data.CurrentDate.Week()
					for current, i := start, 0; current != end.Next(); current = current.Next() {
						result[i] = data.Days.GetDay(current)
						i++
					}
					return result
				case ui.ViewMonth:
					start, end := data.CurrentDate.MonthBounds()
					result := make([]*model.Day, end.Day)
					for current, i := start, 0; current != end.Next(); current = current.Next() {
						result[i] = data.Days.GetDay(current)
						i++
					}
					return result
				default:
					panic("unknown view in summary data gathering")
				}
			},
			&data.CategoryStyling,
		),
		panes.NewLogPane(
			tui.NewConstrainedRenderer(renderer, screenDimensions),
			screenDimensions,
			stylesheet,
			func() bool { return data.showLog },
			func() string { return "LOG" },
			&data.Log,
		),
		panes.NewHelpPane(
			tui.NewConstrainedRenderer(renderer, helpDimensions),
			helpDimensions,
			stylesheet,
			func() bool { return data.showHelp },
		),
		panes.NewEditorPane(
			tui.NewConstrainedRenderer(renderer, editorDimensions),
			renderer,
			editorDimensions,
			stylesheet,
			func() bool { return data.EventEditor.Active },
			func() string { return data.EventEditor.TmpEventInfo.Name },
			func() int { return data.EventEditor.CursorPos },
		),

		panes.NewPerfPane(
			tui.NewConstrainedRenderer(renderer, func() (x, y, w, h int) { return 2, 2, 50, 2 }),
			func() (x, y, w, h int) { return 2, 2, 50, 2 },
			func() bool { return data.showDebug },
			&data.renderTimes,
			&data.eventProcessingTimes,
		),

		func() ui.ActiveView { return data.activeView },
	)

	coordinatesProvided := (envData.Latitude != "" && envData.Longitude != "")
	owmApiKeyProvided := (envData.OwmApiKey != "")

	// intialize weather handler if geographic location and api key provided
	if coordinatesProvided && owmApiKeyProvided {
		data.Weather = *weather.NewHandler(envData.Latitude, envData.Longitude, envData.OwmApiKey)
	} else {
		if !owmApiKeyProvided {
			data.Log.Add("ERROR", "no OWM API key provided -> no weather data")
		}
		if !coordinatesProvided {
			data.Log.Add("ERROR", "no lat-/longitude provided -> no weather data")
		}
	}

	// process latitude longitude
	// TODO
	var suntimes model.SunTimes
	if !coordinatesProvided {
		data.Log.Add("ERROR", "could not fetch lat-&longitude -> no sunrise/-set times known")
	} else {
		latF, parseErr := strconv.ParseFloat(envData.Latitude, 64)
		lonF, parseErr := strconv.ParseFloat(envData.Longitude, 64)
		if parseErr != nil {
			data.Log.Add("ERROR", fmt.Sprint("parse error:", parseErr))
		} else {
			suntimes = date.GetSunTimes(latF, lonF)
		}
	}

	controller := Controller{}
	controller.tmpStatusYOffsetGetter = func() int { _, y, _, _ := statusDimensions(); return y }
	data.EnvData = envData
	controller.screenEvents = renderer.GetEventPollable()

	controller.fhMutex.Lock()
	defer controller.fhMutex.Unlock()
	controller.FileHandlers = make(map[model.Date]*filehandling.FileHandler)
	controller.FileHandlers[date] = filehandling.NewFileHandler(data.EnvData.BaseDirPath + "/days/" + date.ToString())

	controller.data = data
	controller.data.CurrentDate = date
	if controller.FileHandlers[date] == nil {
		controller.data.Days.AddDay(date, &model.Day{}, &suntimes)
	} else {
		controller.data.Days.AddDay(date, controller.FileHandlers[date].Read(controller.data.CategoryStyling.GetKnownCategoriesByName()), &suntimes)
	}

	controller.rootPane = rootPane
	controller.data.CurrentCategory.Name = "default"

	controller.loadDaysForView(controller.data.activeView)

	controller.timestampGuesser = func(cursorX, cursorY int) model.Timestamp {
		_, yOffset, _, _ := dayViewEventsPaneDimensions()
		return data.ViewParams.TimeAtY(yOffset + cursorY)
	}

	controller.initializedScreen = renderer
	controller.syncer = renderer

	return &controller
}

func (t *Controller) ScrollUp(by int) {
	eventviewTopRow := 0
	if t.data.ViewParams.ScrollOffset-by >= eventviewTopRow {
		t.data.ViewParams.ScrollOffset -= by
	} else {
		t.ScrollTop()
	}
}

func (t *Controller) ScrollDown(by int) {
	eventviewBottomRow := t.tmpStatusYOffsetGetter()
	if t.data.ViewParams.ScrollOffset+by+eventviewBottomRow <= (24 * t.data.ViewParams.NRowsPerHour) {
		t.data.ViewParams.ScrollOffset += by
	} else {
		t.ScrollBottom()
	}
}

func (t *Controller) ScrollTop() {
	t.data.ViewParams.ScrollOffset = 0
}

func (t *Controller) ScrollBottom() {
	eventviewBottomRow := t.tmpStatusYOffsetGetter()
	t.data.ViewParams.ScrollOffset = 24*t.data.ViewParams.NRowsPerHour - eventviewBottomRow
}

func (t *Controller) abortEdit() {
	t.editState = EditStateNone
	t.EditedEvent = EditedEvent{0, model.Timestamp{Hour: 0, Minute: 0}}
	t.data.EventEditor.Active = false
}

func (t *Controller) endEdit() {
	t.editState = EditStateNone
	t.EditedEvent = EditedEvent{0, model.Timestamp{Hour: 0, Minute: 0}}
	if t.data.EventEditor.Active {
		t.data.EventEditor.Active = false
		tmp := t.data.EventEditor.TmpEventInfo
		t.data.GetCurrentDay().GetEvent(tmp.ID).Name = tmp.Name
	}
	t.data.GetCurrentDay().UpdateEventOrder()
}

func (t *Controller) startMouseMove(eventsInfo ui.EventsPanePositionInfo) {
	t.editState = (EditStateMouseEditing | EditStateMoving)
	t.EditedEvent.ID = eventsInfo.Event()
	t.EditedEvent.prevEditStepTimestamp = eventsInfo.Time()
}

func (t *Controller) startMouseResize(eventsInfo ui.EventsPanePositionInfo) {
	t.editState = (EditStateMouseEditing | EditStateResizing)
	t.EditedEvent.ID = eventsInfo.Event()
	t.EditedEvent.prevEditStepTimestamp = eventsInfo.Time()
}

func (t *Controller) startMouseEventCreation(info ui.EventsPanePositionInfo) {
	// find out cursor time
	start := info.Time()

	t.data.Log.Add("DEBUG", fmt.Sprintf("creation called for '%s'", info.Time().ToString()))

	// create event at time with cat etc.
	e := model.Event{}
	e.Cat = t.data.CurrentCategory
	e.Name = ""
	e.Start = start
	e.End = start.OffsetMinutes(+10)

	// give to model, get ID
	newEventID := t.data.GetCurrentDay().AddEvent(e)

	// save ID as edited event
	t.EditedEvent.ID = newEventID
	t.EditedEvent.prevEditStepTimestamp = e.Start

	// set mode to resizing
	t.editState = (EditStateMouseEditing | EditStateResizing)
}

func (t *Controller) goToDay(newDate model.Date) {
	t.data.Log.Add("DEBUG", "going to "+newDate.ToString())

	t.data.CurrentDate = newDate
	t.loadDaysForView(t.data.activeView)
}

func (t *Controller) goToPreviousDay() {
	prevDay := t.data.CurrentDate.Prev()
	t.goToDay(prevDay)
}

func (t *Controller) goToNextDay() {
	nextDay := t.data.CurrentDate.Next()
	t.goToDay(nextDay)
}

// Loads the requested date's day from its file handler, if it has
// not already been loaded.
func (t *Controller) loadDay(date model.Date) {
	if !t.data.Days.HasDay(date) {
		// load file
		newDay := t.GetDayFromFileHandler(date)
		if newDay == nil {
			panic("newDay nil?!")
		}

		var suntimes model.SunTimes
		coordinatesProvided := (t.data.EnvData.Latitude != "" && t.data.EnvData.Longitude != "")
		if coordinatesProvided {
			latF, parseErr := strconv.ParseFloat(t.data.EnvData.Latitude, 64)
			lonF, parseErr := strconv.ParseFloat(t.data.EnvData.Longitude, 64)
			if parseErr != nil {
				t.data.Log.Add("ERROR", fmt.Sprint("parse error:", parseErr))
			} else {
				suntimes = date.GetSunTimes(latF, lonF)
			}
		}

		t.data.Days.AddDay(date, newDay, &suntimes)
	}
}

// Starts Loads for all days visible in the view.
// E.g. for ui.ViewMonth it would start the load for all days from
// first to last day of the month.
// Warning: does not guarantee days will be loaded (non-nil) after
// this returns.
func (t *Controller) loadDaysForView(view ui.ActiveView) {
	switch view {
	case ui.ViewDay:
		t.loadDay(t.data.CurrentDate)
	case ui.ViewWeek:
		{
			monday, sunday := t.data.CurrentDate.Week()
			for current := monday; current != sunday.Next(); current = current.Next() {
				go func(d model.Date) {
					t.loadDay(d)
					t.controllerEvents <- ControllerEventRender
				}(current)
			}
		}
	case ui.ViewMonth:
		{
			first, last := t.data.CurrentDate.MonthBounds()
			for current := first; current != last.Next(); current = current.Next() {
				go func(d model.Date) {
					t.loadDay(d)
					t.controllerEvents <- ControllerEventRender
				}(current)
			}
		}
	default:
		panic("unknown ActiveView")
	}
}

func (t *Controller) handleNoneEditKeyInput(e *tcell.EventKey) {
	switch e.Key() {
	case tcell.KeyESC:
		prevView := PrevView(t.data.activeView)
		t.loadDaysForView(prevView)
		t.data.activeView = prevView
	case tcell.KeyCtrlU:
		t.ScrollUp(10)
	case tcell.KeyCtrlD:
		t.ScrollDown(10)
	}
	switch e.Rune() {
	case 'u':
		go func() {
			err := t.data.Weather.Update()
			if err != nil {
				t.data.Log.Add("ERROR", err.Error())
			} else {
				t.data.Log.Add("DEBUG", "successfully retrieved weather data")
			}
			t.controllerEvents <- ControllerEventRender
		}()
	case 'i':
		nextView := NextView(t.data.activeView)
		t.loadDaysForView(nextView)
		t.data.activeView = nextView
	case 'q':
		t.controllerEvents <- ControllerEventExit
	case 'P':
		t.data.showDebug = !t.data.showDebug
	case 'd':
		eventsInfo := t.rootPane.GetPositionInfo(t.data.cursorPos.X, t.data.cursorPos.Y).GetExtraEventsInfo()
		if eventsInfo != nil {
			t.data.GetCurrentDay().RemoveEvent(eventsInfo.Event())
		}
	case 'g':
		t.ScrollTop()
	case 'G':
		t.ScrollBottom()
	case 'w':
		t.writeModel()
	case 'j':
		t.ScrollDown(1)
	case 'k':
		t.ScrollUp(1)
	case 'h':
		t.goToPreviousDay()
	case 'l':
		t.goToNextDay()
	case 'S':
		t.data.showSummary = !t.data.showSummary
	case 'E':
		t.data.showLog = !t.data.showLog
	case '?':
		t.data.showHelp = !t.data.showHelp
	case 'c':
		// TODO: all that's needed to clear model (appropriately)?
		t.data.Days.AddDay(t.data.CurrentDate, model.NewDay(), t.data.GetCurrentSuntimes())
	case '+':
		if t.data.ViewParams.NRowsPerHour*2 <= 12 {
			t.data.ViewParams.NRowsPerHour *= 2
			t.data.ViewParams.ScrollOffset *= 2
		}
	case '-':
		if (t.data.ViewParams.NRowsPerHour % 2) == 0 {
			t.data.ViewParams.NRowsPerHour /= 2
			t.data.ViewParams.ScrollOffset /= 2
		} else {
			t.data.Log.Add("WARNING", fmt.Sprintf("can't decrease resolution below %d", t.data.ViewParams.NRowsPerHour))
		}
	}
}

func (t *Controller) writeModel() {
	go func() {
		t.fhMutex.RLock()
		t.FileHandlers[t.data.CurrentDate].Write(t.data.GetCurrentDay())
		t.fhMutex.RUnlock()
	}()
}

func (t *Controller) updateCursorPos(x, y int) {
	t.data.cursorPos.X, t.data.cursorPos.Y = x, y
}

func (t *Controller) startEdit(id model.EventID) {
	t.data.EventEditor.Active = true
	t.data.EventEditor.TmpEventInfo = *t.data.GetCurrentDay().GetEvent(id)
	t.data.EventEditor.CursorPos = len([]rune(t.data.EventEditor.TmpEventInfo.Name))
	t.editState = EditStateEditing
}

func (t *Controller) handleNoneEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventKey:
		t.handleNoneEditKeyInput(e)
	case *tcell.EventMouse:
		// get new position
		x, y := e.Position()
		t.updateCursorPos(x, y)

		positionInfo := t.rootPane.GetPositionInfo(x, y)
		if positionInfo == nil {
			return
		}

		buttons := e.Buttons()

		paneType := positionInfo.PaneType()
		switch paneType {
		case ui.StatusPaneType:
		case ui.WeatherPaneType:
			switch buttons {
			case tcell.WheelUp:
				t.ScrollUp(1)
			case tcell.WheelDown:
				t.ScrollDown(1)
			}
		case ui.TimelinePaneType:
			switch buttons {
			case tcell.WheelUp:
				t.ScrollUp(1)
			case tcell.WheelDown:
				t.ScrollDown(1)
			}
		case ui.EventsPaneType:
			eventsInfo := positionInfo.GetExtraEventsInfo()

			// if button clicked, handle
			switch buttons {
			case tcell.Button3:
				t.data.GetCurrentDay().RemoveEvent(eventsInfo.Event())
			case tcell.Button2:
				id := eventsInfo.Event()
				if id != 0 && eventsInfo.Time().IsAfter(t.data.GetCurrentDay().GetEvent(id).Start) {
					t.data.GetCurrentDay().SplitEvent(id, eventsInfo.Time())
				}
			case tcell.Button1:
				// we've clicked while not editing
				// now we need to check where the cursor is and either start event
				// creation, resizing or moving
				switch eventsInfo.EventBoxPart() {
				case ui.EventBoxNowhere:
					t.startMouseEventCreation(eventsInfo)
				case ui.EventBoxBottomRight:
					t.startMouseResize(eventsInfo)
				case ui.EventBoxInterior:
					t.movePropagate = (e.Modifiers() == tcell.ModCtrl)
					t.startMouseMove(eventsInfo)
				case ui.EventBoxTopEdge:
					t.startEdit(eventsInfo.Event())
				}
			case tcell.WheelUp:
				t.ScrollUp(1)
			case tcell.WheelDown:
				t.ScrollDown(1)
			}
		case ui.ToolsPaneType:
			toolsInfo := positionInfo.GetExtraToolsInfo()
			switch buttons {
			case tcell.Button1:
				cat := toolsInfo.Category()
				if cat != nil {
					t.data.CurrentCategory = *cat
				}
			}
		default:
		}
	}
}

func (t *Controller) resizeStep(nextCursortime model.Timestamp) {
	prevCursortime := t.EditedEvent.prevEditStepTimestamp
	offset := prevCursortime.DurationInMinutesUntil(nextCursortime)
	event := t.data.GetCurrentDay().GetEvent(t.EditedEvent.ID)
	event.End = event.End.OffsetMinutes(offset).Snap(t.data.ViewParams.NRowsPerHour)
	t.EditedEvent.prevEditStepTimestamp = nextCursortime
}

func (t *Controller) moveStep(nextCursortime model.Timestamp) {
	prevCursortime := t.EditedEvent.prevEditStepTimestamp
	offset := prevCursortime.DurationInMinutesUntil(nextCursortime)
	if t.movePropagate {
		following := t.data.GetCurrentDay().GetEventsFrom(t.EditedEvent.ID)
		for _, ptr := range following {
			ptr.Start = ptr.Start.OffsetMinutes(offset).Snap(t.data.ViewParams.NRowsPerHour)
			ptr.End = ptr.End.OffsetMinutes(offset).Snap(t.data.ViewParams.NRowsPerHour)
		}
	} else {
		event := t.data.GetCurrentDay().GetEvent(t.EditedEvent.ID)
		event.Start = event.Start.OffsetMinutes(offset).Snap(t.data.ViewParams.NRowsPerHour)
		event.End = event.End.OffsetMinutes(offset).Snap(t.data.ViewParams.NRowsPerHour)
	}
	t.EditedEvent.prevEditStepTimestamp = nextCursortime
}

func (t *Controller) handleMouseResizeEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		x, y := e.Position()

		buttons := e.Buttons()

		switch buttons {
		case tcell.Button1:
			timestampGuess := t.timestampGuesser(x, y)
			t.resizeStep(timestampGuess)
		case tcell.ButtonNone:
			t.endEdit()
		}

		t.updateCursorPos(x, y)
	}
}

func (t *Controller) handleEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventKey:
		editor := &t.data.EventEditor

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

func (t *Controller) handleMouseMoveEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		x, y := e.Position()

		buttons := e.Buttons()

		switch buttons {
		case tcell.Button1:
			timestampGuess := t.timestampGuesser(x, y)
			t.moveStep(timestampGuess)
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

func (t *Controller) Run() {
	t.controllerEvents = make(chan ControllerEvent, 32)
	var wg sync.WaitGroup

	// Run the main render loop, that renders or exits when prompted accordingly
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer t.initializedScreen.Fini()
		for {

			select {
			case controllerEvent := <-t.controllerEvents:
				switch controllerEvent {
				case ControllerEventRender:
					start := time.Now()

					// empty all further render events before rendering
					exitEventEncounteredOnEmpty := emptyRenderEvents(t.controllerEvents)
					// exit if an exit event was coming up
					if exitEventEncounteredOnEmpty {
						return
					}
					// render
					t.rootPane.Draw()

					end := time.Now()
					t.data.renderTimes.Add(uint64(end.Sub(start).Microseconds()))
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
			t.controllerEvents <- ControllerEventRender
		}
	}()

	// Run the event tracking loop, that waits for and processes events and pings
	// for a redraw (or program exit) after each event.
	go func() {
		for {
			ev := t.screenEvents.PollEvent()

			start := time.Now()

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
				t.syncer.NeedsSync()
			}

			end := time.Now()
			t.data.eventProcessingTimes.Add(uint64(end.Sub(start).Microseconds()))

			t.controllerEvents <- ControllerEventRender
		}
	}()

	wg.Wait()
}
