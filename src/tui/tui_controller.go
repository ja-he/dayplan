package tui

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ja-he/dayplan/src/filehandling"
	"github.com/ja-he/dayplan/src/model"
	"github.com/ja-he/dayplan/src/program"
	"github.com/ja-he/dayplan/src/styling"
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
	tui           ui.MainUIPane
	editState     EditState
	EditedEvent   EditedEvent
	movePropagate bool
	fhMutex       sync.RWMutex
	FileHandlers  map[model.Date]*filehandling.FileHandler
	bump          chan ControllerEvent

	// TODO: remove, obviously
	tmpStatusYOffsetGetter func() int

	// When creating or editing events with the mouse, we probably don't want to
	// end the edit if the mouse leaves the events pane. Instead the more
	// intuitive behavior for users is that it simply continue as long as the
	// mouse button is held, regardless of the actual pane under the cursor.
	// This helps guess at timestamps for those edits without having the panes
	// awkwardly accessing information that they shouldn't need to.
	timestampGuesser func(int, int) model.Timestamp

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

type TUIStylesheet struct {
	normal           styling.DrawStyling
	weatherRegular   styling.DrawStyling
	weatherSunny     styling.DrawStyling
	weatherRainy     styling.DrawStyling
	timelineDay      styling.DrawStyling
	timelineNight    styling.DrawStyling
	timelineNow      styling.DrawStyling
	status           styling.DrawStyling
	categoryFallback styling.DrawStyling
	logDefault       styling.DrawStyling
	logTitleBox      styling.DrawStyling
	logEntryType     styling.DrawStyling
	logEntryLocation styling.DrawStyling
	logEntryTime     styling.DrawStyling
	help             styling.DrawStyling
	editor           styling.DrawStyling
	summaryDefault   styling.DrawStyling
	summaryTitleBox  styling.DrawStyling
}

func (s *TUIStylesheet) Normal() styling.DrawStyling           { return s.normal }
func (s *TUIStylesheet) WeatherRegular() styling.DrawStyling   { return s.weatherRegular }
func (s *TUIStylesheet) WeatherSunny() styling.DrawStyling     { return s.weatherSunny }
func (s *TUIStylesheet) WeatherRainy() styling.DrawStyling     { return s.weatherRainy }
func (s *TUIStylesheet) TimelineDay() styling.DrawStyling      { return s.timelineDay }
func (s *TUIStylesheet) TimelineNight() styling.DrawStyling    { return s.timelineNight }
func (s *TUIStylesheet) TimelineNow() styling.DrawStyling      { return s.timelineNow }
func (s *TUIStylesheet) Status() styling.DrawStyling           { return s.status }
func (s *TUIStylesheet) CategoryFallback() styling.DrawStyling { return s.categoryFallback }
func (s *TUIStylesheet) LogDefault() styling.DrawStyling       { return s.logDefault }
func (s *TUIStylesheet) LogTitleBox() styling.DrawStyling      { return s.logTitleBox }
func (s *TUIStylesheet) LogEntryType() styling.DrawStyling     { return s.logEntryType }
func (s *TUIStylesheet) LogEntryLocation() styling.DrawStyling { return s.logEntryLocation }
func (s *TUIStylesheet) LogEntryTime() styling.DrawStyling     { return s.logEntryTime }
func (s *TUIStylesheet) Help() styling.DrawStyling             { return s.help }
func (s *TUIStylesheet) Editor() styling.DrawStyling           { return s.editor }
func (s *TUIStylesheet) SummaryDefault() styling.DrawStyling   { return s.summaryDefault }
func (s *TUIStylesheet) SummaryTitleBox() styling.DrawStyling  { return s.summaryTitleBox }

func NewTUIController(date model.Date, programData program.Data) *TUIController {
	// read category styles
	var categoryStyling CategoryStyling
	categoryStyling = *EmptyCategoryStyling()
	styleFilePath := programData.BaseDirPath + "/" + "category-styles.yaml"
	styledInputs, err := ReadCategoryStylingFile(styleFilePath)
	if err != nil {
		panic(err)
	}
	for _, styledInput := range styledInputs {
		categoryStyling.AddStyleFromInput(styledInput)
	}

	stylesheet := TUIStylesheet{
		normal: NewStyling(tcell.ColorBlack, tcell.ColorWhite),

		weatherRegular: NewStyling(tcell.ColorLightBlue, tcell.ColorWhite),
		weatherRainy:   NewStyling(tcell.ColorBlack, tcell.NewHexColor(0xccebff)),
		weatherSunny:   NewStyling(tcell.ColorBlack, tcell.NewHexColor(0xfff0cc)),

		timelineDay:   NewStyling(tcell.ColorLightGray, tcell.ColorWhite),
		timelineNight: NewStyling(tcell.ColorLightGray, tcell.ColorBlack),
		timelineNow:   NewStyling(tcell.ColorWhite, tcell.ColorRed).Bolded(),

		status: NewStyling((tcell.ColorBlack), styling.ColorFromHexString("#f0f0f0")),

		categoryFallback: NewStyling(tcell.ColorBlack, tcell.ColorIndianRed),

		logDefault:       NewStyling(tcell.ColorBlack, tcell.ColorWhite),
		logTitleBox:      NewStyling(tcell.ColorBlack, tcell.ColorLightGrey).Bolded(),
		logEntryType:     NewStyling(tcell.ColorDarkGrey, tcell.ColorWhite).Italicized(),
		logEntryLocation: NewStyling(tcell.ColorDarkGrey, tcell.ColorWhite),
		logEntryTime:     NewStyling(tcell.ColorLightGrey, tcell.ColorWhite),

		help: NewStyling(tcell.ColorBlack, tcell.ColorLightGrey),

		editor: NewStyling(tcell.ColorBlack, tcell.ColorLightGrey),

		summaryDefault:  NewStyling(tcell.ColorBlack, tcell.ColorWhite),
		summaryTitleBox: NewStyling(tcell.ColorBlack, tcell.ColorLightGrey).Bolded(),
	}

	tuiModel := NewTUIModel(categoryStyling)

	toolsWidth := 20
	statusHeight := 2
	weatherWidth := 20
	timelineWidth := 10
	helpWidth := 80
	helpHeight := 30
	editorWidth := 80
	editorHeight := 20

	renderer := NewTUIRenderer()
	screenSize := func() (w, h int) { return renderer.GetScreenDimensions() }
	screenDimensions := func() (x, y, w, h int) { w, h = screenSize(); return 0, 0, w, h }
	centeredFloat := func(floatWidth, floatHeight int) func() (x, y, w, h int) {
		return func() (x, y, w, h int) {
			screenWidth, screenHeight := screenSize()
			return (screenWidth / 2) - (floatWidth / 2), (screenHeight / 2) - (floatHeight / 2), floatWidth, floatHeight
		}
	}
	helpDimensions := centeredFloat(helpWidth, helpHeight)
	editorDimensions := centeredFloat(editorWidth, editorHeight)
	toolsDimensions := func() (x, y, w, h int) { w, h = screenSize(); return w - toolsWidth, 0, toolsWidth, h - statusHeight }
	statusDimensions := func() (x, y, w, h int) { w, h = screenSize(); return 0, h - statusHeight, w, statusHeight }
	dayViewMainPaneDimensions := screenDimensions
	weekViewMainPaneDimensions := screenDimensions
	monthViewMainPaneDimensions := screenDimensions
	weatherDimensions := func() (x, y, w, h int) {
		x, y, _, h = dayViewMainPaneDimensions()
		return x, y, weatherWidth, h - statusHeight
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
		_, _, w, h = dayViewMainPaneDimensions()
		return 0 + weatherWidth, 0, timelineWidth, h - statusHeight
	}
	weekViewTimelineDimensions := func() (x, y, w, h int) {
		w, h = screenSize()
		return 0, 0, timelineWidth, h - statusHeight
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
	weekdayPane := func(dayIndex int) ui.UIPane {
		return &EventsPane{
			renderer:       &TUIConstrainedRenderer{screenHandler: renderer, constraint: weekdayDimensions(dayIndex)},
			dimensions:     weekdayDimensions(dayIndex),
			stylesheet:     &stylesheet,
			day:            func() *model.Day { return tuiModel.Days.GetDay(tuiModel.CurrentDate.GetDayInWeek(dayIndex)) },
			categories:     &tuiModel.CategoryStyling,
			viewParams:     &tuiModel.ViewParams,
			cursor:         &tuiModel.cursorPos,
			padRight:       0,
			drawTimestamps: false,
			drawNames:      true,
			isCurrent:      func() bool { return tuiModel.CurrentDate.GetDayInWeek(dayIndex) == tuiModel.CurrentDate },
			logReader:      &tuiModel.Log,
			logWriter:      &tuiModel.Log,
			positions:      make(map[model.EventID]util.Rect),
		}
	}
	monthdayDimensions := func(dayIndex int) func() (x, y, w, h int) {
		return func() (x, y, w, h int) {
			baseX, baseY, baseW, baseH := monthViewMainPaneDimensions()
			eventsWidth := baseW - timelineWidth
			dayWidth := eventsWidth / 31
			return baseX + timelineWidth + (dayIndex * dayWidth), baseY, dayWidth, baseH - statusHeight
		}
	}
	monthdayPane := func(dayIndex int) ui.UIPane {
		return &MaybeEventsPane{
			condition: func() bool { return tuiModel.CurrentDate.GetDayInMonth(dayIndex).Month == tuiModel.CurrentDate.Month },
			eventsPane: &EventsPane{
				renderer:       &TUIConstrainedRenderer{screenHandler: renderer, constraint: monthdayDimensions(dayIndex)},
				dimensions:     monthdayDimensions(dayIndex),
				stylesheet:     &stylesheet,
				day:            func() *model.Day { return tuiModel.Days.GetDay(tuiModel.CurrentDate.GetDayInMonth(dayIndex)) },
				categories:     &tuiModel.CategoryStyling,
				viewParams:     &tuiModel.ViewParams,
				cursor:         &tuiModel.cursorPos,
				padRight:       0,
				drawTimestamps: false,
				drawNames:      false,
				isCurrent:      func() bool { return tuiModel.CurrentDate.GetDayInMonth(dayIndex) == tuiModel.CurrentDate },
				logReader:      &tuiModel.Log,
				logWriter:      &tuiModel.Log,
				positions:      make(map[model.EventID]util.Rect),
			},
		}
	}

	weekViewEventsPanes := make([]ui.UIPane, 7)
	for i := range weekViewEventsPanes {
		weekViewEventsPanes[i] = weekdayPane(i)
	}

	monthViewEventsPanes := make([]ui.UIPane, 31)
	for i := range monthViewEventsPanes {
		monthViewEventsPanes[i] = monthdayPane(i)
	}

	statusPane := &StatusPane{
		renderer:    &TUIConstrainedRenderer{screenHandler: renderer, constraint: statusDimensions},
		dimensions:  statusDimensions,
		stylesheet:  &stylesheet,
		currentDate: &tuiModel.CurrentDate,
		dayWidth: func() int {
			_, _, w, _ := statusDimensions()
			switch tuiModel.activeView {
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
		totalDaysInPeriod: func() int {
			switch tuiModel.activeView {
			case ui.ViewDay:
				return 1
			case ui.ViewWeek:
				return 7
			case ui.ViewMonth:
				return tuiModel.CurrentDate.GetLastOfMonth().Day
			default:
				panic("unknown view for status rendering")
			}
		},
		passedDaysInPeriod: func() int {
			switch tuiModel.activeView {
			case ui.ViewDay:
				return 1
			case ui.ViewWeek:
				switch tuiModel.CurrentDate.ToWeekday() {
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
				return tuiModel.CurrentDate.Day
			default:
				panic("unknown view for status rendering")
			}
		},
		firstDayXOffset: func() int { return timelineWidth },
	}

	tui := TUI{
		renderer:   renderer,
		dimensions: screenDimensions,

		dayViewMainPane: &DayViewMainPane{
			dimensions: dayViewMainPaneDimensions,
			events: &EventsPane{
				renderer:       &TUIConstrainedRenderer{screenHandler: renderer, constraint: dayViewEventsPaneDimensions},
				dimensions:     dayViewEventsPaneDimensions,
				stylesheet:     &stylesheet,
				day:            tuiModel.GetCurrentDay,
				categories:     &tuiModel.CategoryStyling,
				viewParams:     &tuiModel.ViewParams,
				cursor:         &tuiModel.cursorPos,
				padRight:       2,
				drawTimestamps: true,
				drawNames:      true,
				isCurrent:      func() bool { return true },
				logReader:      &tuiModel.Log,
				logWriter:      &tuiModel.Log,
				positions:      make(map[model.EventID]util.Rect),
			},
			tools: &ToolsPane{
				renderer:        &TUIConstrainedRenderer{screenHandler: renderer, constraint: toolsDimensions},
				dimensions:      toolsDimensions,
				stylesheet:      &stylesheet,
				currentCategory: &tuiModel.CurrentCategory,
				categories:      &tuiModel.CategoryStyling,
				horizPadding:    1,
				vertPadding:     1,
				gap:             0,
			},
			status: statusPane,
			timeline: &TimelinePane{
				renderer:   &TUIConstrainedRenderer{screenHandler: renderer, constraint: dayViewTimelineDimensions},
				dimensions: dayViewTimelineDimensions,
				stylesheet: &stylesheet,
				suntimes:   tuiModel.GetCurrentSuntimes,
				currentTime: func() *model.Timestamp {
					if tuiModel.CurrentDate.Is(time.Now()) {
						return model.NewTimestampFromGotime(time.Now())
					} else {
						return nil
					}
				},
				viewParams: &tuiModel.ViewParams,
			},
			weather: &WeatherPane{
				renderer:    &TUIConstrainedRenderer{screenHandler: renderer, constraint: weatherDimensions},
				dimensions:  weatherDimensions,
				stylesheet:  &stylesheet,
				currentDate: &tuiModel.CurrentDate,
				weather:     &tuiModel.Weather,
				viewParams:  &tuiModel.ViewParams,
			},
		},
		weekViewMainPane: &WeekViewMainPane{
			dimensions: weekViewMainPaneDimensions,
			status:     statusPane,
			timeline: &TimelinePane{
				renderer:    &TUIConstrainedRenderer{screenHandler: renderer, constraint: weekViewTimelineDimensions},
				dimensions:  weekViewTimelineDimensions,
				stylesheet:  &stylesheet,
				suntimes:    func() *model.SunTimes { return nil },
				currentTime: func() *model.Timestamp { return nil },
				viewParams:  &tuiModel.ViewParams,
			},
			days:       weekViewEventsPanes,
			categories: &tuiModel.CategoryStyling,
			logReader:  &tuiModel.Log,
			logWriter:  &tuiModel.Log,
			viewParams: &tuiModel.ViewParams,

			positions: make(map[model.EventID]util.Rect),
		},
		monthViewMainPane: &MonthViewMainPane{
			dimensions: monthViewMainPaneDimensions,
			status:     statusPane,
			timeline: &TimelinePane{
				renderer:    &TUIConstrainedRenderer{screenHandler: renderer, constraint: monthViewTimelineDimensions},
				dimensions:  monthViewTimelineDimensions,
				stylesheet:  &stylesheet,
				suntimes:    func() *model.SunTimes { return nil },
				currentTime: func() *model.Timestamp { return nil },
				viewParams:  &tuiModel.ViewParams,
			},
			days:       monthViewEventsPanes,
			categories: &tuiModel.CategoryStyling,
			logReader:  &tuiModel.Log,
			logWriter:  &tuiModel.Log,
			viewParams: &tuiModel.ViewParams,
		},

		summary: &SummaryPane{
			renderer:   &TUIConstrainedRenderer{screenHandler: renderer, constraint: screenDimensions},
			dimensions: screenDimensions,
			stylesheet: &stylesheet,
			condition:  func() bool { return tuiModel.showSummary },
			titleString: func() string {
				dateString := ""
				switch tuiModel.activeView {
				case ui.ViewDay:
					dateString = tuiModel.CurrentDate.ToString()
				case ui.ViewWeek:
					start, end := tuiModel.CurrentDate.Week()
					dateString = fmt.Sprintf("week %s..%s", start.ToString(), end.ToString())
				case ui.ViewMonth:
					dateString = fmt.Sprintf("%s %d", tuiModel.CurrentDate.ToGotime().Month().String(), tuiModel.CurrentDate.Year)
				}
				return fmt.Sprintf("SUMMARY (%s)", dateString)
			},
			days: func() []*model.Day {
				switch tuiModel.activeView {
				case ui.ViewDay:
					result := make([]*model.Day, 1)
					result[0] = tuiModel.Days.GetDay(tuiModel.CurrentDate)
					return result
				case ui.ViewWeek:
					result := make([]*model.Day, 7)
					start, end := tuiModel.CurrentDate.Week()
					for current, i := start, 0; current != end.Next(); current = current.Next() {
						result[i] = tuiModel.Days.GetDay(current)
						i++
					}
					return result
				case ui.ViewMonth:
					start, end := tuiModel.CurrentDate.MonthBounds()
					result := make([]*model.Day, end.Day)
					for current, i := start, 0; current != end.Next(); current = current.Next() {
						result[i] = tuiModel.Days.GetDay(current)
						i++
					}
					return result
				default:
					panic("unknown view in summary data gathering")
				}
			},
			categories: &tuiModel.CategoryStyling,
		},
		log: &LogPane{
			renderer:    &TUIConstrainedRenderer{screenHandler: renderer, constraint: screenDimensions},
			dimensions:  screenDimensions,
			stylesheet:  &stylesheet,
			condition:   func() bool { return tuiModel.showLog },
			titleString: func() string { return "LOG" },
			logReader:   &tuiModel.Log,
		},
		help: &HelpPane{
			renderer:   &TUIConstrainedRenderer{screenHandler: renderer, constraint: helpDimensions},
			dimensions: helpDimensions,
			stylesheet: &stylesheet,
			condition:  func() bool { return tuiModel.showHelp },
		},
		editor: &EditorPane{
			renderer:         &TUIConstrainedRenderer{screenHandler: renderer, constraint: editorDimensions},
			cursorController: renderer,
			dimensions:       editorDimensions,
			stylesheet:       &stylesheet,
			condition:        func() bool { return tuiModel.EventEditor.Active },
			name:             func() string { return tuiModel.EventEditor.TmpEventInfo.Name },
			cursorPos:        func() int { return tuiModel.EventEditor.CursorPos },
		},

		activeView: &tuiModel.activeView,
	}

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
	tuiController.tmpStatusYOffsetGetter = func() int { _, y, _, _ := statusDimensions(); return y }
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

	tuiController.timestampGuesser = func(cursorX, cursorY int) model.Timestamp {
		_, yOffset, _, _ := dayViewEventsPaneDimensions()
		return tuiModel.ViewParams.TimeAtY(yOffset + cursorY)
	}

	return &tuiController
}

func (t *TUIController) ScrollUp(by int) {
	eventviewTopRow := 0
	if t.model.ViewParams.ScrollOffset-by >= eventviewTopRow {
		t.model.ViewParams.ScrollOffset -= by
	} else {
		t.ScrollTop()
	}
}

func (t *TUIController) ScrollDown(by int) {
	eventviewBottomRow := t.tmpStatusYOffsetGetter()
	if t.model.ViewParams.ScrollOffset+by+eventviewBottomRow <= (24 * t.model.ViewParams.NRowsPerHour) {
		t.model.ViewParams.ScrollOffset += by
	} else {
		t.ScrollBottom()
	}
}

func (t *TUIController) ScrollTop() {
	t.model.ViewParams.ScrollOffset = 0
}

func (t *TUIController) ScrollBottom() {
	eventviewBottomRow := t.tmpStatusYOffsetGetter()
	t.model.ViewParams.ScrollOffset = 24*t.model.ViewParams.NRowsPerHour - eventviewBottomRow
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

func (t *TUIController) startMouseMove(eventsInfo ui.EventsPanePositionInfo) {
	t.editState = (EditStateMouseEditing | EditStateMoving)
	t.EditedEvent.ID = eventsInfo.Event()
	t.EditedEvent.prevEditStepTimestamp = eventsInfo.Time()
}

func (t *TUIController) startMouseResize(eventsInfo ui.EventsPanePositionInfo) {
	t.editState = (EditStateMouseEditing | EditStateResizing)
	t.EditedEvent.ID = eventsInfo.Event()
	t.EditedEvent.prevEditStepTimestamp = eventsInfo.Time()
}

func (t *TUIController) startMouseEventCreation(info ui.EventsPanePositionInfo) {
	// find out cursor time
	start := info.Time()

	t.model.Log.Add("DEBUG", fmt.Sprintf("creation called for '%s'", info.Time().ToString()))

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
		t.ScrollUp(10)
	case tcell.KeyCtrlD:
		t.ScrollDown(10)
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
	case 'd':
		eventsInfo := t.tui.GetPositionInfo(t.model.cursorPos.X, t.model.cursorPos.Y).GetExtraEventsInfo()
		if eventsInfo != nil {
			t.model.GetCurrentDay().RemoveEvent(eventsInfo.Event())
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
		case ui.StatusUIPaneType:
		case ui.WeatherUIPaneType:
			switch buttons {
			case tcell.WheelUp:
				t.ScrollUp(1)
			case tcell.WheelDown:
				t.ScrollDown(1)
			}
		case ui.TimelineUIPaneType:
			switch buttons {
			case tcell.WheelUp:
				t.ScrollUp(1)
			case tcell.WheelDown:
				t.ScrollDown(1)
			}
		case ui.EventsUIPaneType:
			eventsInfo := positionInfo.GetExtraEventsInfo()
			t.model.Log.Add("DEBUG", fmt.Sprint(eventsInfo))

			// if button clicked, handle
			switch buttons {
			case tcell.Button3:
				t.model.GetCurrentDay().RemoveEvent(eventsInfo.Event())
			case tcell.Button2:
				id := eventsInfo.Event()
				if id != 0 && eventsInfo.Time().IsAfter(t.model.GetCurrentDay().GetEvent(id).Start) {
					t.model.GetCurrentDay().SplitEvent(id, eventsInfo.Time())
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
		case ui.ToolsUIPaneType:
			toolsInfo := positionInfo.GetExtraToolsInfo()
			t.model.Log.Add("DEBUG", fmt.Sprint("tools info:", toolsInfo))
			switch buttons {
			case tcell.Button1:
				cat := toolsInfo.Category()
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
					t.tui.Draw()
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
			}

			t.bump <- ControllerEventRender
		}
	}()

	wg.Wait()
}
