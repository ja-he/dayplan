package cli

import (
	"fmt"
	"math"
	"time"

	"github.com/ja-he/dayplan/internal/control/edit"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/input/processors"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/potatolog"
	"github.com/ja-he/dayplan/internal/storage"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/ui/panes"
	"github.com/ja-he/dayplan/internal/weather"
)

type UIDimensions struct {
	screenDimensions                func() (x, y, w, h int)
	helpDimensions                  func() (x, y, w, h int)
	tasksDimensions                 func() (x, y, w, h int)
	toolsDimensions                 func() (x, y, w, h int)
	statusDimensions                func() (x, y, w, h int)
	editorDimensions                func() (x, y, w, h int)
	weekViewMainPaneDimensions      func() (x, y, w, h int)
	monthViewMainPaneDimensions     func() (x, y, w, h int)
	dayViewMainPaneDimensions       func() (x, y, w, h int)
	dayViewScrollablePaneDimensions func() (x, y, w, h int)
	dayViewEventsPaneDimensions     func() (x, y, w, h int)
	weatherDimensions               func() (x, y, w, h int)
	dayViewTimelineDimensions       func() (x, y, w, h int)
	weekViewTimelineDimensions      func() (x, y, w, h int)
	monthViewTimelineDimensions     func() (x, y, w, h int)
	weekdayDimensions               func(dayIndex int) func() (x, y, w, h int)
	monthdayDimensions              func(dayIndex int) func() (x, y, w, h int)

	timelineWidth int
}

func computeUIDimensions(
	renderer ui.ConstrainedRenderer,

	tasksWidth int,
	toolsWidth int,
	rightFlexWidthFn func() int,
	statusHeight int,
	weatherWidth int,
	timelineWidth int,
	editorWidth int,
	editorHeight int,
) (*UIDimensions, error) {
	var result UIDimensions

	screenSize := func() (w, h int) { _, _, w, h = renderer.Dimensions(); return }
	result.screenDimensions = func() (x, y, w, h int) {
		screenWidth, screenHeight := screenSize()
		return 0, 0, screenWidth, screenHeight
	}
	result.helpDimensions = result.screenDimensions
	result.tasksDimensions = func() (x, y, w, h int) {
		screenWidth, screenHeight := screenSize()
		return screenWidth - rightFlexWidthFn(), 0, tasksWidth, screenHeight - statusHeight
	}
	result.toolsDimensions = func() (x, y, w, h int) {
		screenWidth, screenHeight := screenSize()
		return screenWidth - toolsWidth, 0, toolsWidth, screenHeight - statusHeight
	}
	result.statusDimensions = func() (x, y, w, h int) {
		screenWidth, screenHeight := screenSize()
		return 0, screenHeight - statusHeight, screenWidth, statusHeight
	}
	result.editorDimensions = func() (x, y, w, h int) {
		screenWidth, screenHeight := screenSize()
		taskEditorBoxWidth := int(math.Min(float64(editorWidth), float64(screenWidth)))
		taskEditorBoxHeight := int(math.Min(float64(editorHeight), float64(screenHeight)))
		return (screenWidth / 2) - (taskEditorBoxWidth / 2), (screenHeight / 2) - (taskEditorBoxHeight / 2), taskEditorBoxWidth, taskEditorBoxHeight
	}
	result.weekViewMainPaneDimensions = result.screenDimensions
	result.monthViewMainPaneDimensions = result.screenDimensions
	result.dayViewMainPaneDimensions = result.screenDimensions
	result.dayViewScrollablePaneDimensions = func() (x, y, w, h int) {
		parentX, parentY, parentW, parentH := result.dayViewMainPaneDimensions()
		return parentX, parentY, parentW - rightFlexWidthFn(), parentH - statusHeight
	}
	result.dayViewEventsPaneDimensions = func() (x, y, w, h int) {
		ox, oy, ow, oh := result.dayViewScrollablePaneDimensions()
		x = ox + weatherWidth + timelineWidth
		y = oy
		w = ow - x
		h = oh
		return x, y, w, h
	}
	result.weatherDimensions = func() (x, y, w, h int) {
		parentX, parentY, _, parentH := result.dayViewScrollablePaneDimensions()
		return parentX, parentY, weatherWidth, parentH
	}
	result.dayViewTimelineDimensions = func() (x, y, w, h int) {
		_, _, _, parentH := result.dayViewScrollablePaneDimensions()
		return 0 + weatherWidth, 0, timelineWidth, parentH
	}
	result.weekViewTimelineDimensions = func() (x, y, w, h int) {
		_, screenHeight := screenSize()
		return 0, 0, timelineWidth, screenHeight - statusHeight
	}
	result.monthViewTimelineDimensions = result.weekViewTimelineDimensions
	result.weekdayDimensions = func(dayIndex int) func() (x, y, w, h int) {
		return func() (x, y, w, h int) {
			baseX, baseY, baseW, baseH := result.weekViewMainPaneDimensions()
			eventsWidth := baseW - timelineWidth
			dayWidth := eventsWidth / 7
			return baseX + timelineWidth + (dayIndex * dayWidth), baseY, dayWidth, baseH - statusHeight
		}
	}
	result.monthdayDimensions = func(dayIndex int) func() (x, y, w, h int) {
		return func() (x, y, w, h int) {
			baseX, baseY, baseW, baseH := result.monthViewMainPaneDimensions()
			eventsWidth := baseW - timelineWidth
			dayWidth := eventsWidth / 31
			return baseX + timelineWidth + (dayIndex * dayWidth), baseY, dayWidth, baseH - statusHeight
		}
	}
	result.timelineWidth = timelineWidth

	return &result, nil
}

func createUI(
	renderer ui.RendererWithOrchestratorControl,
	cursorWrangler *ui.CursorWrangler,
	stylesheet styling.Stylesheet,
	uiDimensions UIDimensions,

	tasksVisibleFn func() bool,
	toolsVisibleFn func() bool,
	summaryVisibleFn func() bool,
	logVisibleFn func() bool,
	helpVisibleFn func() bool,
	getMouseMode func() bool,
	getEventEditMode func() edit.EventEditMode,
	getCursorPos func() ui.MouseCursorPos,

	dayViewInputTree input.SimpleInputProcessor,
	dayViewEventsPaneInputTree input.SimpleInputProcessor,
	helpPaneInputTree input.SimpleInputProcessor,
	summaryPaneInputTree input.SimpleInputProcessor,
	tasksInputTree input.SimpleInputProcessor,
	toolsInputTree input.SimpleInputProcessor,
	monthViewMainPaneInputTree input.SimpleInputProcessor,
	dayViewScrollablePaneInputTree input.SimpleInputProcessor,
	weekdayPaneInputTree input.SimpleInputProcessor,
	monthdayPaneInputTree input.SimpleInputProcessor,
	weekViewEventsWrapperInputTree input.SimpleInputProcessor,
	monthViewEventsWrapperInputTree input.SimpleInputProcessor,
	weekViewMainPaneInputTree input.SimpleInputProcessor,
	rootPaneInputTree input.SimpleInputProcessor,

	createWeekViewDayEventsFn func(dayIndex int) func() (model.Date, *model.EventList, error),
	createMonthViewDayEventsFn func(dayIndex int) func() (model.Date, *model.EventList, error),
	getCategoryStyle func(n model.CategoryName) (styling.DrawStyling, error),
	getCategoriesInOrder func() []*model.Category,
	getCurrentDate func() model.Date,
	getSummary func() (map[model.CategoryName]time.Duration, error),
	getCurrentDateEventsFn func() (model.Date, *model.EventList, error),
	viewParams ui.TimespanViewParams,
	backlogViewParams ui.TimeViewParams,
	getCurrentEventIDFn func() *model.EventID,
	getWeatherDataFn func() map[model.DateAndTime]weather.Weather,
	getCurrentCategoryFn func() model.CategoryName,
	getCurrentTaskFn func() *model.TaskID,

	storageProviderInfo storage.DataProviderInfo,
	suntimesProvider storage.SunTimesProvider,
	categoryProvider storage.CategoryProvider,
	backlogProvider storage.BacklogProvider,

	perfPane ui.Pane,

) (*panes.RootPane, error) {
	var getActiveView func() ui.ActiveView

	weekdayPane := func(dayIndex int) *panes.EventsPane {
		return panes.NewEventsPane(
			ui.NewConstrainedRenderer(renderer, uiDimensions.weekdayDimensions(dayIndex)),
			uiDimensions.weekdayDimensions(dayIndex),
			stylesheet,
			processors.NewModalInputProcessor(weekdayPaneInputTree),
			createWeekViewDayEventsFn(dayIndex),
			getCategoryStyle,
			viewParams,
			getCursorPos,
			0,
			false,
			true,
			false,
			func() bool { c := getCurrentDate(); return c.GetDayInWeek(dayIndex) == c },
			func() *model.EventID { return nil /* TODO */ },
			getMouseMode,
			fmt.Sprintf("events-%d", dayIndex),
		)
	}
	monthdayPane := func(dayIndex int) ui.Pane {
		return panes.NewMaybePane(
			func() bool {
				c := getCurrentDate()
				return c.GetDayInMonth(dayIndex).Month == c.Month
			},
			panes.NewEventsPane(
				ui.NewConstrainedRenderer(renderer, uiDimensions.monthdayDimensions(dayIndex)),
				uiDimensions.monthdayDimensions(dayIndex),
				stylesheet,
				processors.NewModalInputProcessor(monthdayPaneInputTree),
				createMonthViewDayEventsFn(dayIndex),
				getCategoryStyle,
				viewParams,
				getCursorPos,
				0,
				false,
				false,
				false,
				func() bool { c := getCurrentDate(); return c.GetDayInMonth(dayIndex) == c },
				func() *model.EventID { return nil /* TODO */ },
				getMouseMode,
				fmt.Sprintf("events-%d", dayIndex),
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
		ui.NewConstrainedRenderer(renderer, uiDimensions.statusDimensions),
		uiDimensions.statusDimensions,
		stylesheet,
		getCurrentDate,
		func() int {
			_, _, w, _ := uiDimensions.statusDimensions()
			switch getActiveView() {
			case ui.ViewDay:
				return w - uiDimensions.timelineWidth
			case ui.ViewWeek:
				return (w - uiDimensions.timelineWidth) / 7
			case ui.ViewMonth:
				return (w - uiDimensions.timelineWidth) / 31
			default:
				panic("unknown view for status rendering")
			}
		},
		func() int {
			switch getActiveView() {
			case ui.ViewDay:
				return 1
			case ui.ViewWeek:
				return 7
			case ui.ViewMonth:
				return getCurrentDate().GetLastOfMonth().Day
			default:
				panic("unknown view for status rendering")
			}
		},
		func() int {
			switch getActiveView() {
			case ui.ViewDay:
				return 1
			case ui.ViewWeek:
				switch getCurrentDate().ToWeekday() {
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
				return getCurrentDate().Day
			default:
				panic("unknown view for status rendering")
			}
		},
		func() int { return uiDimensions.timelineWidth },
		getEventEditMode,
		storageProviderInfo,
	)

	weekViewEventsWrapper := panes.NewWrapperPane(
		weekViewEventsPanes,
		[]ui.Pane{},
		processors.NewModalInputProcessor(weekViewEventsWrapperInputTree),
		"week-view-events-wrapper",
	)
	monthViewEventsWrapper := panes.NewWrapperPane(
		monthViewEventsPanes,
		[]ui.Pane{},
		processors.NewModalInputProcessor(monthViewEventsWrapperInputTree),
		"month-view-events-wrapper",
	)

	weekViewMainPane := panes.NewWrapperPane(
		[]ui.Pane{
			statusPane,
			panes.NewTimelinePane(
				ui.NewConstrainedRenderer(renderer, uiDimensions.weekViewTimelineDimensions),
				uiDimensions.weekViewTimelineDimensions,
				stylesheet,
				func() model.SunTimes { return suntimesProvider.Get(getCurrentDate().GetDayInWeek(0)) },
				func() *model.Timestamp { return nil },
				viewParams,
			),
			weekViewEventsWrapper,
		},
		[]ui.Pane{
			weekViewEventsWrapper,
		},
		processors.NewModalInputProcessor(weekViewMainPaneInputTree),
		"week-view-main",
	)
	monthViewMainPane := panes.NewWrapperPane(
		[]ui.Pane{
			statusPane,
			panes.NewTimelinePane(
				ui.NewConstrainedRenderer(renderer, uiDimensions.monthViewTimelineDimensions),
				uiDimensions.monthViewTimelineDimensions,
				stylesheet,
				func() model.SunTimes { return suntimesProvider.Get(getCurrentDate().GetDayInMonth(0)) },
				func() *model.Timestamp { return nil },
				viewParams,
			),
			monthViewEventsWrapper,
		},
		[]ui.Pane{
			monthViewEventsWrapper,
		},
		processors.NewModalInputProcessor(monthViewMainPaneInputTree),
		"month-view-main",
	)

	dayViewMainPane, err := createDayViewMainPane(
		renderer,
		stylesheet,

		uiDimensions,

		tasksVisibleFn,
		toolsVisibleFn,
		getMouseMode,
		getCursorPos,

		getCategoryStyle,
		getCategoriesInOrder,
		getCurrentDateEventsFn,
		viewParams,
		backlogViewParams,
		getCurrentEventIDFn,
		getCurrentDate,
		getWeatherDataFn,
		getCurrentCategoryFn,
		getCurrentTaskFn,

		dayViewInputTree,
		dayViewEventsPaneInputTree,
		tasksInputTree,
		toolsInputTree,
		dayViewScrollablePaneInputTree,

		suntimesProvider,
		backlogProvider,

		statusPane,
	)
	if err != nil {
		return nil, fmt.Errorf("Unable to construct day view main pane (%w)", err)
	}

	helpPane := panes.NewHelpPane(
		ui.NewConstrainedRenderer(renderer, uiDimensions.helpDimensions),
		uiDimensions.helpDimensions,
		stylesheet,
		helpVisibleFn,
		processors.NewModalInputProcessor(helpPaneInputTree),
	)

	rootPane := panes.NewRootPane(
		renderer,
		cursorWrangler,
		uiDimensions.screenDimensions,

		dayViewMainPane,
		weekViewMainPane,
		monthViewMainPane,

		panes.NewSummaryPane(
			ui.NewConstrainedRenderer(renderer, uiDimensions.screenDimensions),
			uiDimensions.screenDimensions,
			stylesheet,
			summaryVisibleFn,
			func() string {
				dateString := ""
				c := getCurrentDate()
				switch getActiveView() {
				case ui.ViewDay:
					dateString = c.String()
				case ui.ViewWeek:
					start, end := c.WeekBounds()
					dateString = fmt.Sprintf("week %s..%s", start.String(), end.String())
				case ui.ViewMonth:
					dateString = fmt.Sprintf("%s %d", c.ToGotime().Month().String(), c.Year)
				}
				return fmt.Sprintf("SUMMARY (%s)", dateString)
			},
			getSummary,
			categoryProvider,
			getCategoryStyle,
			processors.NewModalInputProcessor(summaryPaneInputTree),
		),
		panes.NewLogPane(
			ui.NewConstrainedRenderer(renderer, uiDimensions.screenDimensions),
			uiDimensions.screenDimensions,
			stylesheet,
			logVisibleFn,
			func() string { return "LOG" },
			&potatolog.GlobalMemoryLogReaderWriter,
		),
		helpPane,

		perfPane,
		processors.NewModalInputProcessor(rootPaneInputTree),
		dayViewMainPane,
	)
	getActiveView = rootPane.GetView

	return rootPane, nil
}

func createDayViewMainPane(
	renderer ui.ConstrainedRenderer,
	stylesheet styling.Stylesheet,

	uiDimensions UIDimensions,

	tasksVisibleFn func() bool,
	toolsVisibleFn func() bool,
	getMouseMode func() bool,
	getCursorPos func() ui.MouseCursorPos,

	getCategoryStyle func(n model.CategoryName) (styling.DrawStyling, error),
	getCategoriesInOrder func() []*model.Category,
	getCurrentDateEventsFn func() (model.Date, *model.EventList, error),
	viewParams ui.TimespanViewParams,
	backlogViewParams ui.TimeViewParams,
	getCurrentEventIDFn func() *model.EventID,
	getCurrentDate func() model.Date,
	getWeatherDataFn func() map[model.DateAndTime]weather.Weather,
	getCurrentCategoryFn func() model.CategoryName,
	getCurrentTaskFn func() *model.TaskID,

	dayViewInputTree input.SimpleInputProcessor,
	dayViewEventsPaneInputTree input.SimpleInputProcessor,
	tasksInputTree input.SimpleInputProcessor,
	toolsInputTree input.SimpleInputProcessor,
	dayViewScrollablePaneInputTree input.SimpleInputProcessor,

	suntimesProvider storage.SunTimesProvider,
	backlogProvider storage.BacklogProvider,

	statusPane ui.Pane,
) (*panes.Composite, error) {
	dayEventsPane := panes.NewEventsPane(
		ui.NewConstrainedRenderer(renderer, uiDimensions.dayViewEventsPaneDimensions),
		uiDimensions.dayViewEventsPaneDimensions,
		stylesheet,
		processors.NewModalInputProcessor(dayViewEventsPaneInputTree),
		getCurrentDateEventsFn,
		getCategoryStyle,
		viewParams,
		getCursorPos,
		2,
		true,
		true,
		true,
		func() bool { return true },
		getCurrentEventIDFn,
		getMouseMode,
		"events",
	)

	dayViewScrollablePane := panes.NewWrapperPane(
		[]ui.Pane{
			dayEventsPane,
			panes.NewTimelinePane(
				ui.NewConstrainedRenderer(renderer, uiDimensions.dayViewTimelineDimensions),
				uiDimensions.dayViewTimelineDimensions,
				stylesheet,
				func() model.SunTimes { return suntimesProvider.Get(getCurrentDate()) },
				func() *model.Timestamp {
					now := time.Now()
					if getCurrentDate().Is(now) {
						return model.NewTimestampFromGotime(now)
					}
					return nil
				},
				viewParams,
			),
			panes.NewWeatherPane(
				ui.NewConstrainedRenderer(renderer, uiDimensions.weatherDimensions),
				uiDimensions.weatherDimensions,
				stylesheet,
				getCurrentDate,
				getWeatherDataFn,
				viewParams,
			),
		},
		[]ui.Pane{
			dayEventsPane,
		},
		processors.NewModalInputProcessor(dayViewScrollablePaneInputTree),
		"day-view-scrollable",
	)

	tasksPane := panes.NewBacklogPane(
		ui.NewConstrainedRenderer(renderer, uiDimensions.tasksDimensions),
		uiDimensions.tasksDimensions,
		stylesheet,
		processors.NewModalInputProcessor(tasksInputTree),
		backlogViewParams,
		getCurrentTaskFn,
		backlogProvider,
		getCategoryStyle,
		tasksVisibleFn,
	)
	toolsPane := panes.NewToolsPane(
		ui.NewConstrainedRenderer(renderer, uiDimensions.toolsDimensions),
		uiDimensions.toolsDimensions,
		stylesheet,
		processors.NewModalInputProcessor(toolsInputTree),
		getCurrentCategoryFn,
		getCategoryStyle,
		getCategoriesInOrder,
		2,
		1,
		0,
		toolsVisibleFn,
	)

	dayViewMainPane := panes.NewWrapperPane(
		[]ui.Pane{
			dayViewScrollablePane,
			tasksPane,
			toolsPane,
			statusPane,
		},
		[]ui.Pane{
			dayViewScrollablePane,
			tasksPane,
			toolsPane,
		},
		processors.NewModalInputProcessor(dayViewInputTree),
		"day-view-main",
	)
	return dayViewMainPane, nil
}
