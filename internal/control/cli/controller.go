package cli

import (
	"fmt"
	"path"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/config"
	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/control/action"
	"github.com/ja-he/dayplan/internal/control/edit"
	"github.com/ja-he/dayplan/internal/control/edit/editors"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/provider"
	"github.com/ja-he/dayplan/internal/provider/backend"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/tui"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/ui/panes"
	"github.com/ja-he/dayplan/internal/weather"
)

// Controller is the struct for the TUI controller.
type Controller struct {
	data *control.ControlData

	// TODO: may want to group these somewhere?
	rootPane          *panes.RootPane
	tasksPane         *panes.BacklogPane
	toolsPane         *panes.ToolsPane
	dayViewMainPane   *panes.Composite
	dayViewEventsPane *panes.EventsPane

	eventsProvider   provider.EventProvider
	suntimesProvider provider.SunTimesProvider
	categoryProvider provider.CategoryProvider
	backlogProvider  provider.TaskProvider

	controllerEvents chan controllerEvent

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

	// TODO: try to get rid of this
	ensureEventsPaneTimestampWithinVisibleScroll func(time.Time)
	createTaskEditorPane                         func() (ui.Pane, error)
	createEventEditorPane                        func() (ui.Pane, error)

	log zerolog.Logger
}

// NewController creates a new Controller.
func NewController(
	date model.Date,
	envData control.EnvData,
	categoriesByName map[model.CategoryName]*model.Category,
	stylesheet styling.Stylesheet,
	weatherHandler *weather.Handler,
	suntimesProvider provider.SunTimesProvider,
) (*Controller, error) {
	controller := Controller{
		log: log.With().Str("component", "controller").Logger(),
	}
	defer controller.goToDay(date)

	controller.data = control.NewControlData()
	controller.data.Weather = weatherHandler
	controller.suntimesProvider = suntimesProvider

	{
		categoryProvider := &backend.MemoryCategoryProvider{M: categoriesByName}
		var p provider.EventProvider
		var err error
		p, err = backend.NewFilesDataProvider(
			path.Join(envData.BaseDirPath, "days"),
			categoryProvider,
		)
		if err != nil {
			return nil, fmt.Errorf("cannot initialize data provider (%w)", err)
		}
		controller.eventsProvider = p
		controller.categoryProvider = categoryProvider
	}

	inputConfig := input.InputConfig{

		Editor: map[input.Keyspec]input.Actionspec{
			"j":       "next-field",
			"k":       "prev-field",
			"i":       "enter-subeditor",
			":w<CR>":  "write",
			"<CR>":    "write-and-quit",
			":wq<CR>": "write-and-quit",
			":q!<CR>": "quit",
			"<ESC>":   "quit",
		},

		StringEditor: input.ModedSpec{
			Normal: map[input.Keyspec]input.Actionspec{
				"h":       "move-cursor-rune-left",
				"l":       "move-cursor-rune-right",
				"<left>":  "move-cursor-rune-left",
				"<right>": "move-cursor-rune-right",
				"0":       "move-cursor-to-beginning",
				"$":       "move-cursor-to-end",
				"w":       "move-cursor-to-next-word-beginning",
				"b":       "move-cursor-to-prev-word-beginning",
				"e":       "move-cursor-to-next-word-end",
				"<ESC>":   "quit",
				"D":       "delete-to-end",
				"d$":      "delete-to-end",
				"d0":      "backspace-to-beginning",
				"C":       "delete-to-end-and-insert",
				"c$":      "delete-to-end-and-insert",
				"c0":      "backspace-to-beginning-and-insert",
				"S":       "delete-everything-and-insert",
				"x":       "delete-rune",
				"s":       "delete-rune-and-insert",
				"i":       "swap-mode-insert",
				"a":       "swap-mode-insert-append",
			},
			Insert: map[input.Keyspec]input.Actionspec{
				"<left>":  "move-cursor-rune-left",
				"<right>": "move-cursor-rune-right",
				"<ESC>":   "swap-mode-normal",
				"<c-bs>":  "backspace",
				"<bs>":    "backspace",
				"<c-u>":   "backspace-to-beginning",
			},
		},
	}

	backlogFilePath := path.Join(envData.BaseDirPath, "days", "backlog.yml") // TODO(ja_he): Migrate 'days' -> 'data', perhaps subdir 'days'
	var err error
	controller.backlogProvider, err = backend.NewBacklogYamlIoProvider(backlogFilePath)
	if err != nil {
		return nil, fmt.Errorf("Unable to create backlog provider for YAML file '%s' (%w)", backlogFilePath, err)
	}
	controller.log.Info().Str("file", backlogFilePath).Msg("successfully created backlog provider")
	go controller.tryLoadBacklog()

	tasksWidth := 40
	toolsWidth := func() int {
		width := 20
		for _, c := range categoriesByName {
			requisiteWidth := len(c.Name) + 4
			if requisiteWidth > width {
				width = requisiteWidth
			}
		}
		return width
	}()
	rightFlexWidth := toolsWidth

	statusHeight := 2
	weatherWidth := 20
	timelineWidth := 10
	editorWidth := 80
	editorHeight := 20

	scrollableZoomableInputMap := map[input.Keyspec]action.Action{
		"<c-u>": action.NewSimple(func() string { return "scroll up" }, func() { controller.ScrollUp(10) }),
		"<c-d>": action.NewSimple(func() string { return "scroll down" }, func() { controller.ScrollDown(10) }),
		"gg":    action.NewSimple(func() string { return "scroll to top" }, controller.ScrollTop),
		"G":     action.NewSimple(func() string { return "scroll to bottom" }, controller.ScrollBottom),
		"+": action.NewSimple(func() string { return "zoom in" }, func() {
			if controller.data.MainTimelineViewParams.NRowsPerHour*2 <= 12 {
				controller.data.MainTimelineViewParams.NRowsPerHour *= 2
				controller.data.MainTimelineViewParams.ScrollOffset *= 2
			}
		}),
		"-": action.NewSimple(func() string { return "zoom out" }, func() {
			if (controller.data.MainTimelineViewParams.NRowsPerHour % 2) == 0 {
				controller.data.MainTimelineViewParams.NRowsPerHour /= 2
				controller.data.MainTimelineViewParams.ScrollOffset /= 2
			} else {
				log.Warn().Msg(fmt.Sprintf("can't decrease resolution below %d", controller.data.MainTimelineViewParams.NRowsPerHour))
			}
		}),
	}
	monthViewMainPaneInputTree, err := input.ConstructInputTree(scrollableZoomableInputMap)
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for month view main pane (%w)", err)
	}
	dayViewScrollablePaneInputTree, err := input.ConstructInputTree(scrollableZoomableInputMap)
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for day view scrollable pane (%w)", err)
	}

	eventsViewBaseInputMap := map[input.Keyspec]action.Action{
		"w": action.NewSimple(func() string { return "write all events data" }, func() {
			err := controller.eventsProvider.CommitState()
			if err != nil {
				log.Error().Err(err).Msg("could not write / commit")
			}
		}),
		"h": action.NewSimple(func() string { return "go to previous day" }, controller.goToPreviousDay),
		"l": action.NewSimple(func() string { return "go to next day" }, controller.goToNextDay),
		"0D": action.NewSimple(func() string { return "clear day's events" }, func() {
			events, err := controller.getCurrentDayEvents()
			if err != nil {
				log.Error().Err(err).Msg("could not get events for current day")
				return
			}
			eventIDs := make([]string, len(events))
			for i, e := range events {
				eventIDs[i] = e.ID
			}
			controller.removeEvents(eventIDs)
		}),
	}
	weekdayPaneInputTree, err := input.ConstructInputTree(eventsViewBaseInputMap)
	if err != nil {
		return nil, fmt.Errorf("could not construct weekday pane input tree (%w)", err)
	}
	monthdayPaneInputTree, err := input.ConstructInputTree(eventsViewBaseInputMap)
	if err != nil {
		return nil, fmt.Errorf("could not construct monthday pane input tree (%w)", err)
	}

	renderer := tui.NewTUIScreenHandler()
	cursorWrangler := ui.NewCursorWrangler(renderer)

	getCategoryStyle := func(n model.CategoryName) (styling.DrawStyling, error) {
		c := categoriesByName[n]
		return styling.StyleFromColorSingle(c.Color, stylesheet.Theme == config.Dark)
	}
	getCategoriesInOrder := func() []*model.Category {
		cats := make([]*model.Category, 0, len(categoriesByName))
		for _, cat := range categoriesByName {
			cats = append(cats, cat)
		}
		sort.Sort(model.ByName(cats))
		return cats
	}

	tasksVisible := false
	tasksVisibleFn := func() bool { return tasksVisible }
	toolsVisible := true
	toolsVisibleFn := func() bool { return toolsVisible }
	helpVisibleFn := func() bool { return controller.data.ShowHelp }

	controller.data.BacklogViewParams = ui.BacklogViewParams{
		NRowsPerHour: &controller.data.MainTimelineViewParams.NRowsPerHour,
		ScrollOffset: 0,
	}
	var scrollBacklogTop func()
	var scrollBacklogBottom func()
	var getBacklogBottomScrollOffset func() int
	tasksInputTree, err := input.ConstructInputTree(
		map[input.Keyspec]action.Action{
			"<c-u>": action.NewSimple(func() string { return "scroll up" }, func() {
				controller.data.BacklogViewParams.SetScrollOffset(controller.data.BacklogViewParams.GetScrollOffset() - 10)
				if controller.data.BacklogViewParams.GetScrollOffset() < 0 {
					scrollBacklogTop()
				}
			}),
			"<c-d>": action.NewSimple(func() string { return "scroll down" }, func() {
				scrollTarget := controller.data.BacklogViewParams.GetScrollOffset() + 10
				if scrollTarget > getBacklogBottomScrollOffset() {
					scrollBacklogBottom()
				} else {
					controller.data.BacklogViewParams.SetScrollOffset(scrollTarget)
				}
			}),
			"j":  action.NewSimple(func() string { return "go down a task" }, controller.goToNextTask),
			"k":  action.NewSimple(func() string { return "go up a task" }, controller.goToPrevTask),
			"gg": action.NewSimple(func() string { return "scroll to top" }, controller.ScrollBacklogToTop),
			"G":  action.NewSimple(func() string { return "scroll to bottom" }, controller.ScrollBacklogToBottom),
			"sn": action.NewSimple(func() string { return "schedule now" }, controller.ScheduleCurrentTaskNow),
			"d":  action.NewSimple(func() string { return "delete task" }, controller.DeleteCurrentTask),
			"l":  action.NewSimple(func() string { return "step into subtasks" }, controller.StepIntoCurrentTaskSubtasks),
			"h":  action.NewSimple(func() string { return "step out to parent task" }, controller.stepOutToParentTask),
			"O":  action.NewSimple(func() string { return "add a new task above the current one" }, controller.insertTaskBeforeCurrent),
			"o":  action.NewSimple(func() string { return "add a new task below the current one" }, controller.insertTaskAfterCurrent),
			"i":  action.NewSimple(func() string { return "add a new subtask of the current task" }, controller.addSubtaskOfCurrent),
			"<cr>": action.NewSimple(func() string { return "begin editing of task" }, func() {
				if controller.data.CurrentTask != nil {
					controller.createAndEnableTaskEditor(*controller.data.CurrentTask)
				}
			}),
			"w": action.NewSimple(func() string { return "store backlog to file" }, controller.writeBacklog),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for tasks pane (%w)", err)
	}
	toolsInputTree, err := input.ConstructInputTree(
		map[input.Keyspec]action.Action{
			"j": action.NewSimple(func() string { return "switch to next category" }, func() {
				controller.log.Debug().Msg("will try to switch to next category")
				cats := getCategoriesInOrder()
				for i, cat := range cats {
					if cat.Name == controller.data.CurrentCategory {
						for ii := i + 1; ii < len(cats); ii++ {
							if !cats[ii].Deprecated {
								prevCategory := controller.data.CurrentCategory
								nextCategory := cats[ii].Name
								controller.data.CurrentCategory = nextCategory
								controller.log.Debug().Msgf("Switched category in tools from '%s' to '%s'", prevCategory, nextCategory)
								return
							}
						}
						controller.log.Warn().Msg("could not find a non-deprecated next category")
					}
				}
				controller.log.Error().Interface("categories", cats).Str("currentCategory", string(controller.data.CurrentCategory)).Msg("Could not find current category in list.")
			}),
			"k": action.NewSimple(func() string { return "switch to previous category" }, func() {
				controller.log.Debug().Msg("will try to switch to prev category")
				cats := getCategoriesInOrder()
				for i, cat := range cats {
					if cat.Name == controller.data.CurrentCategory {
						for ii := i - 1; ii >= 0; ii-- {
							if !cats[ii].Deprecated {
								prevCategory := controller.data.CurrentCategory
								nextCategory := cats[ii].Name
								controller.data.CurrentCategory = nextCategory
								controller.log.Debug().Msgf("Switched category in tools from '%s' to '%s'", prevCategory, nextCategory)
								return
							}
						}
					}
				}
			}),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for tools pane (%w)", err)
	}

	var startMovePushing func()
	// TODO: directly?
	eventsPaneDayInputExtension := map[input.Keyspec]action.Action{
		"j": action.NewSimple(func() string { return "switch to next event" }, controller.switchToNextEventInDay),
		"k": action.NewSimple(func() string { return "switch to previous event" }, controller.switchToPreviousEventInDay),
		"d": action.NewSimple(func() string { return "delete selected event" }, func() {
			event := controller.data.CurrentEventID
			if event != nil {
				controller.removeEvent(*event)
			}
		}),
		"<cr>": action.NewSimple(func() string { return "open the event editor" }, func() {
			eventID := controller.data.CurrentEventID
			if eventID == nil {
				log.Warn().Msgf("ignoring event editing request since no current event selected")
				return
			}
			event, err := controller.eventsProvider.GetEvent(*eventID)
			if err != nil {
				log.Error().Err(err).Msgf("unable to find event for stored ID of current %s", *eventID)
			}

			if controller.data.EventEditor != nil {
				log.Warn().Msgf("was about to construct new event editor but still have old one")
				return
			}
			newEventEditor, err := editors.ConstructEditor("event", event, nil, nil, func(eventToBeWritten any) error {
				event, ok := eventToBeWritten.(*model.Event)
				if !ok {
					return fmt.Errorf("Expected event pointer, got %t.", eventToBeWritten)
				}
				if event.ID != *eventID {
					return fmt.Errorf("Illegally, event ID has changed.")
				}
				err := controller.eventsProvider.SetEventAllData(*eventID, *event)
				if err != nil {
					return fmt.Errorf("Unable to set all event data with data provider: %w", err)
				}
				return nil
			})
			if err != nil {
				log.Warn().Err(err).Msgf("unable to construct event editor")
				return
			}
			var ok bool
			controller.data.EventEditor, ok = newEventEditor.(*editors.Composite)
			if !ok {
				log.Error().Msgf("something went _really_ wrong and the editor constructed for the event is _not_ a composite editor but a %T", newEventEditor)
				controller.data.EventEditor = nil
				return
			}

			eventEditorPane, err := controller.createEventEditorPane()
			if err != nil {
				log.Fatal().Err(err).Msg("could not construct event editor pane (this is likely a serious programming error / omission)")
			}

			controller.rootPane.PushSubpane(eventEditorPane)
			eventEditorDone := make(chan struct{})
			controller.data.EventEditor.AddQuitCallback(func() {
				close(eventEditorDone) // TODO: this can CERTAINLY happen twice; prevent
			})
			go func() {
				<-eventEditorDone
				controller.controllerEvents <- controllerEventEventEditorExit
			}()
		}),
		"o": action.NewSimple(func() string { return "add event after selected" }, func() {
			current, err := func() (*model.Event, error) {
				c := controller.data.CurrentEventID
				if c == nil {
					return nil, nil
				}
				e, err := controller.eventsProvider.GetEvent(*c)
				if err != nil {
					return nil, fmt.Errorf("Could not get event (%w).", err)
				}
				startDate := model.DateFromGotime(e.Start)
				if startDate != controller.data.CurrentDate {
					return nil, fmt.Errorf("Current event somehow not on current date (%s != %s).", startDate, controller.data.CurrentDate)
				}
				return e, nil
			}()
			if err != nil {
				controller.log.Error().Err(err).Msg("could not get current event")
				return
			}
			newEvent := model.Event{
				Name:     "",
				Category: controller.data.CurrentCategory,
			}
			if current == nil {
				now := time.Now()
				if controller.data.CurrentDate.Is(now) {
					newEvent.Start = now
				} else {
					topRowShownTime := controller.data.MainTimelineViewParams.TimeAtY(0)
					newEvent.Start = model.DateAndTimestampToGotime(controller.data.CurrentDate, topRowShownTime)
				}
			} else {
				isMidnight := func(t time.Time) bool {
					return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
				}
				if isMidnight(current.End) {
					controller.log.Warn().Msgf("Unable to add event after current event, which ends at midnight.")
					return
				}
				newEvent.Start = current.End
			}
			eventAfter, err := controller.eventsProvider.GetEventAfter(newEvent.Start)
			if err != nil {
				log.Warn().Err(err).Msg("could not get event after")
			}
			if eventAfter == nil || eventAfter.Start.Sub(newEvent.Start) <= 0 || eventAfter.Start.Sub(newEvent.Start) > (60*time.Minute) {
				if newEvent.Start.Add(60*time.Minute).YearDay() != newEvent.Start.YearDay() {
					midnightBeforeTomorrow := newEvent.Start.Truncate(60 * time.Minute).Add(60 * time.Minute)
					newEvent.End = midnightBeforeTomorrow
				} else {
					newEvent.End = newEvent.Start.Add(60 * time.Minute)
				}
			} else {
				newEvent.End = eventAfter.Start
			}
			eventID, err := controller.eventsProvider.AddEvent(newEvent)
			if err != nil {
				controller.log.Error().Err(err).Msg("Unable to create event.")
				return
			}
			controller.log.Info().Interface("id", eventID).Msgf("Created new event.")
			controller.data.CurrentEventID = &eventID
			controller.ensureCurrentEventVisible()
		}),
		"O": action.NewSimple(func() string { return "add event before selected" }, func() {
			current, err := func() (*model.Event, error) {
				c := controller.data.CurrentEventID
				if c == nil {
					return nil, nil
				}
				e, err := controller.eventsProvider.GetEvent(*c)
				if err != nil {
					return nil, fmt.Errorf("Could not get event (%w).", err)
				}
				startDate := model.DateFromGotime(e.Start)
				if startDate != controller.data.CurrentDate {
					return nil, fmt.Errorf("Current event somehow not on current date (%s != %s).", startDate, controller.data.CurrentDate)
				}
				return e, nil
			}()
			if err != nil {
				controller.log.Error().Err(err).Msg("could not get current event")
				return
			}
			newEvent := model.Event{
				Name:     "",
				Category: controller.data.CurrentCategory,
			}
			if current == nil {
				newEvent.End = time.Now()
			} else {
				newEvent.End = current.Start
			}
			eventBefore, err := controller.eventsProvider.GetEventBefore(newEvent.End)
			if err != nil {
				log.Warn().Err(err).Msgf("could not get event before %s from data provider", newEvent.End.String())
				return
			}
			if eventBefore == nil || newEvent.End.Sub(eventBefore.End) <= 0 || newEvent.End.Sub(eventBefore.End) > (60*time.Minute) {
				newEvent.Start = newEvent.End.Add(-60 * time.Minute)
			} else {
				newEvent.Start = eventBefore.End
			}
			eventID, err := controller.eventsProvider.AddEvent(newEvent)
			if err != nil {
				controller.log.Error().Err(err).Msg("Unable to create event.")
				return
			}
			controller.log.Info().Interface("id", eventID).Msgf("Created new event.")
			controller.data.CurrentEventID = &eventID
			controller.ensureCurrentEventVisible()
		}),
		"<c-o>": action.NewSimple(func() string { return "add event now" }, func() {
			newEvent := &model.Event{
				Name:     "",
				Category: controller.data.CurrentCategory,
			}
			newEvent.Start = time.Now()
			eventAfter, err := controller.eventsProvider.GetEventAfter(newEvent.Start)
			if err != nil {
				log.Warn().Err(err).Msgf("could not get event after %s", newEvent.Start.String())
				return
			}
			if eventAfter != nil && eventAfter.Start.Sub(newEvent.Start).Minutes() < 60 {
				newEvent.End = eventAfter.Start
			} else {
				newEvent.End = newEvent.Start.Add(60 * time.Minute)
			}
			controller.eventsProvider.AddEvent(*newEvent)
			controller.ensureEventsPaneTimestampWithinVisibleScroll(newEvent.Start)
		}),
		"sn": action.NewSimple(func() string { return "split selected event now" }, func() {
			current, err := func() (*model.Event, error) {
				c := controller.data.CurrentEventID
				if c == nil {
					return nil, nil
				}
				return controller.eventsProvider.GetEvent(*c)
			}()
			if err != nil {
				controller.log.Error().Err(err).Msg("could not get current event")
				return
			}
			if current == nil {
				controller.log.Info().Msg("there is no selected event, thus nothing to split")
				return
			}
			now := time.Now()
			if err := controller.eventsProvider.SplitEvent(current.ID, now); err != nil {
				log.Warn().Err(err).Msgf("could not split event at %s", now.String())
				return
			}
		}),
		"sc": action.NewSimple(func() string { return "split selected event at its center" }, func() {
			current, err := func() (*model.Event, error) {
				c := controller.data.CurrentEventID
				if c == nil {
					return nil, nil
				}
				return controller.eventsProvider.GetEvent(*c)
			}()
			if err != nil {
				controller.log.Error().Err(err).Msg("could not get current event")
				return
			}
			if current == nil {
				return
			}
			center := current.Start.Add(current.End.Sub(current.Start) / 2)
			if err := controller.eventsProvider.SplitEvent(current.ID, center); err != nil {
				log.Warn().Err(err).Msgf("could not split event at %s", center.String())
				return
			}
		}),
		"M": action.NewSimple(func() string { return "start move pushing" }, func() { startMovePushing() }),
	}
	eventsPaneDayInputMap := make(map[input.Keyspec]action.Action)
	for input, action := range eventsViewBaseInputMap {
		eventsPaneDayInputMap[input] = action
	}
	for input, action := range eventsPaneDayInputExtension {
		eventsPaneDayInputMap[input] = action
	}
	dayViewEventsPaneInputTree, err := input.ConstructInputTree(eventsPaneDayInputMap)
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for day view pane's events subpane (%w)", err)
	}

	startMovePushing = func() {
		if controller.data.CurrentEventID == nil {
			return
		}

		overlay, err := input.ConstructInputTree(
			map[input.Keyspec]action.Action{
				"n": action.NewSimple(func() string { return "move to now" }, func() { panic("TODO") }),
				"j": action.NewSimple(func() string { return "move down" }, func() {
					if err := controller.moveEventsForwardPushing(); err != nil {
						log.Error().Err(err).Msg("could not move events")
					}
				}),
				"k": action.NewSimple(func() string { return "move up" }, func() {
					if err := controller.moveEventsBackwardPushing(); err != nil {
						log.Error().Err(err).Msg("could not move events")
					}
				}),
				"M": action.NewSimple(func() string { return "exit move mode" }, func() {
					controller.dayViewEventsPane.PopModalOverlay()
					controller.data.EventEditMode = edit.EventEditModeNormal
				}),
				"<esc>": action.NewSimple(func() string { return "exit move mode" }, func() {
					controller.dayViewEventsPane.PopModalOverlay()
					controller.data.EventEditMode = edit.EventEditModeNormal
				}),
				// TODO(ja-he): mode switching
			},
		)
		if err != nil {
			panic(err.Error())
		}
		controller.dayViewEventsPane.ApplyModalOverlay(input.CapturingOverlayWrap(overlay))
		controller.data.EventEditMode = edit.EventEditModeMove
	}
	scrollBacklogTop = func() {
		controller.data.BacklogViewParams.SetScrollOffset(0)
	}
	scrollBacklogBottom = func() {
		controller.data.BacklogViewParams.SetScrollOffset(getBacklogBottomScrollOffset())
	}
	getBacklogBottomScrollOffset = func() int {
		currentScrollOffset := controller.data.BacklogViewParams.GetScrollOffset()
		lastTaskID, err := controller.backlogProvider.GetLastChildTaskID(nil)
		if err != nil {
			controller.log.Error().Err(err).Msg("Unable to get last root task ID.")
			return currentScrollOffset
		}
		if lastTaskID == nil {
			controller.log.Warn().Msg("No tasks, therefore unable to derive bottom scroll offset.")
			return currentScrollOffset
		}
		_, tUB := controller.tasksPane.GetTaskUIYBounds(*lastTaskID)
		_, vUB := controller.tasksPane.GetTaskVisibilityBounds()
		desiredScrollDelta := vUB - tUB - 1
		return currentScrollOffset - desiredScrollDelta
	}

	dayViewEventsPaneInputTree.Root.Children[input.Key{Key: tcell.KeyRune, Ch: 'm'}] = &input.Node{Action: action.NewSimple(func() string { return "enter event move mode" }, func() {
		if controller.data.CurrentEventID == nil {
			return
		}

		eventMoveOverlay, err := input.ConstructInputTree(
			map[input.Keyspec]action.Action{
				"n": action.NewSimple(func() string { return "move to now" }, func() {
					current, err := func() (*model.Event, error) {
						c := controller.data.CurrentEventID
						if c == nil {
							return nil, nil
						}
						return controller.eventsProvider.GetEvent(*c)
					}()
					if err != nil {
						controller.log.Error().Err(err).Msg("could not get current event")
						return
					}
					if current == nil {
						controller.log.Info().Msg("no current event selected, so nothing to move")
						return
					}
					newStart := time.Now()
					newEnd := current.End.Add(newStart.Sub(current.Start))
					controller.eventsProvider.SetEventTimes(current.ID, newStart, newEnd)
					controller.ensureEventsPaneTimestampWithinVisibleScroll(newStart)
					controller.ensureEventsPaneTimestampWithinVisibleScroll(newEnd)
				}),
				"j": action.NewSimple(func() string { return "move down" }, func() {
					currentID := controller.data.CurrentEventID
					if currentID == nil {
						controller.log.Info().Msg("no current event selected, so nothing to move")
						return
					}
					_, _, err := controller.eventsProvider.SnapEventStartPreseveDuration(*currentID, controller.data.MainTimelineViewParams.DurationOfHeight(1))
					if err != nil {
						controller.log.Error().Err(err).Msg("could not snap event")
						return
					}
					_, newEnd, err := controller.eventsProvider.OffsetEventTimes(*currentID, controller.data.MainTimelineViewParams.DurationOfHeight(1))
					if err != nil {
						controller.log.Error().Err(err).Msg("could not move event")
						return
					}
					controller.ensureEventsPaneTimestampWithinVisibleScroll(newEnd)
				}),
				"k": action.NewSimple(func() string { return "move up" }, func() {
					currentID := controller.data.CurrentEventID
					if currentID == nil {
						controller.log.Info().Msg("no current event selected, so nothing to move")
						return
					}
					_, _, err := controller.eventsProvider.SnapEventStartPreseveDuration(*currentID, controller.data.MainTimelineViewParams.DurationOfHeight(1))
					if err != nil {
						controller.log.Error().Err(err).Msg("could not snap event")
						return
					}
					newStart, _, err := controller.eventsProvider.OffsetEventTimes(*currentID, -controller.data.MainTimelineViewParams.DurationOfHeight(1))
					if err != nil {
						controller.log.Error().Err(err).Msg("could not move event")
						return
					}
					controller.ensureEventsPaneTimestampWithinVisibleScroll(newStart)
				}),
				"h": action.NewSimple(func() string { return "move to previous day" }, func() {
					currentEventID := controller.data.CurrentEventID
					if currentEventID == nil {
						controller.log.Warn().Msg("no current event selected, so nothing to move")
						return
					}
					_, _, err := controller.eventsProvider.OffsetEventTimes(*currentEventID, (-24)*time.Hour)
					if err != nil {
						controller.log.Error().Err(err).Msgf("Unable to event %s to previous day.", *currentEventID)
						return
					}
					controller.goToPreviousDay()
					controller.data.CurrentEventID = currentEventID
				}),
				"l": action.NewSimple(func() string { return "move to next day" }, func() {
					currentEventID := controller.data.CurrentEventID
					if currentEventID == nil {
						controller.log.Info().Msg("no current event selected, so nothing to move")
						return
					}
					_, _, err := controller.eventsProvider.OffsetEventTimes(*currentEventID, (+24)*time.Hour)
					if err != nil {
						controller.log.Error().Err(err).Msgf("Unable to event %s to next day.", *currentEventID)
						return
					}
					controller.goToNextDay()
					controller.data.CurrentEventID = currentEventID
				}),
				"m": action.NewSimple(func() string { return "exit move mode" }, func() {
					controller.dayViewEventsPane.PopModalOverlay()
					controller.data.EventEditMode = edit.EventEditModeNormal
				}),
				"<esc>": action.NewSimple(func() string { return "exit move mode" }, func() {
					controller.dayViewEventsPane.PopModalOverlay()
					controller.data.EventEditMode = edit.EventEditModeNormal
				}),
			},
		)
		if err != nil {
			panic(err.Error())
		}
		controller.dayViewEventsPane.ApplyModalOverlay(input.CapturingOverlayWrap(eventMoveOverlay))
		controller.data.EventEditMode = edit.EventEditModeMove
	})}
	dayViewEventsPaneInputTree.Root.Children[input.Key{Key: tcell.KeyRune, Ch: 'r'}] = &input.Node{Action: action.NewSimple(func() string { return "enter event resize mode" }, func() {
		if controller.data.CurrentEventID == nil {
			return
		}

		eventResizeOverlay, err := input.ConstructInputTree(
			map[input.Keyspec]action.Action{
				"n": action.NewSimple(func() string { return "resize to now" }, func() {
					current := controller.data.CurrentEventID
					if current == nil {
						controller.log.Info().Msg("no current event selected, so nothing to resize")
						return
					}
					newEnd := time.Now()
					controller.eventsProvider.SetEventEnd(*current, newEnd)
					controller.ensureEventsPaneTimestampWithinVisibleScroll(newEnd)
				}),
				"j": action.NewSimple(func() string { return "increase size (lengthen)" }, func() {
					var err error
					currentID := controller.data.CurrentEventID
					if currentID == nil {
						controller.log.Info().Msg("no current event selected, so nothing to resize")
						return
					}
					_, err = controller.eventsProvider.OffsetEventEnd(
						*currentID,
						controller.data.MainTimelineViewParams.DurationOfHeight(1),
					)
					if err != nil {
						log.Warn().Err(err).Msg("unable to resize")
						return
					}
					var newEventEnd time.Time
					newEventEnd, err = controller.eventsProvider.SnapEventEnd(
						*currentID,
						controller.data.MainTimelineViewParams.DurationOfHeight(1),
					)
					if err != nil {
						log.Warn().Err(err).Msg("unable to snap")
						return
					}
					controller.ensureEventsPaneTimestampWithinVisibleScroll(newEventEnd)
				}),
				"k": action.NewSimple(func() string { return "decrease size (shorten)" }, func() {
					currentID := controller.data.CurrentEventID
					if currentID == nil {
						controller.log.Info().Msg("no current event selected, so nothing to resize")
						return
					}
					controller.eventsProvider.OffsetEventEnd(
						*currentID,
						-controller.data.MainTimelineViewParams.DurationOfHeight(1),
					)
					newEnd, err := controller.eventsProvider.SnapEventEnd(
						*currentID,
						controller.data.MainTimelineViewParams.DurationOfHeight(1),
					)
					if err != nil {
						controller.log.Error().Err(err).Msg("could not snap event end")
						return
					}
					controller.ensureEventsPaneTimestampWithinVisibleScroll(newEnd)
				}),
				"r": action.NewSimple(func() string { return "exit resize mode" }, func() {
					controller.dayViewEventsPane.PopModalOverlay()
					controller.data.EventEditMode = edit.EventEditModeNormal
				}),
				"<esc>": action.NewSimple(func() string { return "exit resize mode" }, func() {
					controller.dayViewEventsPane.PopModalOverlay()
					controller.data.EventEditMode = edit.EventEditModeNormal
				}),
			},
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to construct input tree for event pane's resize mode (this should really never happen)")
			return
		}
		controller.dayViewEventsPane.ApplyModalOverlay(input.CapturingOverlayWrap(eventResizeOverlay))
		controller.data.EventEditMode = edit.EventEditModeResize
	})}

	var helpContentRegister func()
	rootPaneInputTree, err := input.ConstructInputTree(
		map[input.Keyspec]action.Action{
			"q": action.NewSimple(func() string { return "exit program (unsaved progress is lost)" }, func() { controller.controllerEvents <- controllerEventExit }),
			"P": action.NewSimple(func() string { return "show debug perf pane" }, func() { controller.data.ShowDebug = !controller.data.ShowDebug }),
			"S": action.NewSimple(func() string { return "open summary" }, func() { controller.data.ShowSummary = true }),
			"E": action.NewSimple(func() string { return "toggle log" }, func() { controller.data.ShowLog = !controller.data.ShowLog }),
			"?": action.NewSimple(func() string { return "toggle help" }, func() {
				helpContentRegister()
				controller.data.ShowHelp = true
			}),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for root pane (%w)", err)
	}
	updateMainPaneRightFlexWidth := func() {
		rightFlexWidth = 0
		if controller.tasksPane.IsVisible() {
			rightFlexWidth += tasksWidth
		}
		if controller.toolsPane.IsVisible() {
			rightFlexWidth += toolsWidth
		}
	}
	toggleToolsPane := func() {
		toolsVisible = !toolsVisible
		if !toolsVisible {
			controller.dayViewMainPane.EnsureFocusIsOnVisible()
		}
		updateMainPaneRightFlexWidth()
	}
	toggleTasksPane := func() {
		tasksVisible = !tasksVisible
		if !tasksVisible {
			controller.dayViewMainPane.EnsureFocusIsOnVisible()
		}
		updateMainPaneRightFlexWidth()
	}

	dayViewInputTree, err := input.ConstructInputTree(
		map[input.Keyspec]action.Action{
			"W":      action.NewSimple(func() string { return "update weather" }, controller.updateWeather),
			"t":      action.NewSimple(func() string { return "toggle tools pane" }, toggleToolsPane),
			"T":      action.NewSimple(func() string { return "toggle tasks pane" }, toggleTasksPane),
			"<c-w>h": action.NewSimple(func() string { return "switch to previous pane" }, func() { controller.dayViewMainPane.FocusPrev() }),
			"<c-w>l": action.NewSimple(func() string { return "switch to next pane" }, func() { controller.dayViewMainPane.FocusNext() }),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for day view pane (%w)", err)
	}

	multidayViewEventsWrapperInputMap := scrollableZoomableInputMap
	multidayViewEventsWrapperInputMap["h"] = action.NewSimple(func() string { return "go to previous day" }, controller.goToPreviousDay)
	multidayViewEventsWrapperInputMap["l"] = action.NewSimple(func() string { return "go to next day" }, controller.goToNextDay)
	dayViewMainPaneInputMap := map[input.Keyspec]action.Action{}
	weekViewEventsWrapperInputTree, err := input.ConstructInputTree(multidayViewEventsWrapperInputMap)
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for multi-day wrapper pane (%w)", err)
	}
	monthViewEventsWrapperInputTree, err := input.ConstructInputTree(multidayViewEventsWrapperInputMap)
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for month view wrapper pane (%w)", err)
	}
	weekViewMainPaneInputTree, err := input.ConstructInputTree(dayViewMainPaneInputMap)
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for week view main pane (%w)", err)
	}

	summaryPaneInputTree, err := input.ConstructInputTree(map[input.Keyspec]action.Action{
		"S": action.NewSimple(func() string { return "close summary" }, func() { controller.data.ShowSummary = false }),
		"h": action.NewSimple(func() string { return "switch to previous day/week/month" }, func() {
			switch controller.data.ActiveView() {
			case ui.ViewDay:
				controller.goToPreviousDay()
			case ui.ViewWeek:
				controller.goToDay(controller.data.CurrentDate.GetDayInWeek(0).Backward(7))
			case ui.ViewMonth:
				controller.goToDay(controller.data.CurrentDate.GetDayInMonth(0).Prev().GetDayInMonth(0))
			default:
				panic("unknown view")
			}
		}),
		"l": action.NewSimple(func() string { return "switch to next day/week/month" }, func() {
			switch controller.data.ActiveView() {
			case ui.ViewDay:
				controller.goToNextDay()
			case ui.ViewWeek:
				controller.goToDay(controller.data.CurrentDate.GetDayInWeek(6).Forward(7))
			case ui.ViewMonth:
				controller.goToDay(controller.data.CurrentDate.GetLastOfMonth().Next().GetLastOfMonth())
			default:
				panic("unknown view")
			}
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for summary pane (%w)", err)
	}

	helpPaneInputTree, err := input.ConstructInputTree(
		map[input.Keyspec]action.Action{
			"?": action.NewSimple(func() string { return "close help" }, func() {
				controller.data.ShowHelp = false
			}),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to construct input tree for help pane (%w)", err)
	}

	createWeekViewDayEventsFn := func(dayIndex int) func() (model.Date, *model.EventList, error) {
		return func() (model.Date, *model.EventList, error) {
			startOfDay := controller.data.CurrentDate.GetDayInWeek(dayIndex).ToGotime()
			endOfDay := startOfDay.Add(24 * time.Hour)
			events, err := controller.eventsProvider.GetEventsCoveringTimerange(startOfDay, endOfDay)
			if err != nil {
				log.Warn().Err(err).Time("start-of-day", startOfDay).Time("end-of-day", endOfDay).Msg("could not get events for day")
				return model.Date{}, nil, fmt.Errorf("could not get events for day %d of this week (%w)", dayIndex, err)
			}
			return model.DateFromGotime(startOfDay), &model.EventList{Events: events}, nil
		}
	}
	createMonthViewDayEventsFn := func(dayIndex int) func() (model.Date, *model.EventList, error) {
		return func() (model.Date, *model.EventList, error) {
			startOfDay := controller.data.CurrentDate.GetDayInMonth(dayIndex).ToGotime()
			endOfDay := startOfDay.Add(24 * time.Hour)
			events, err := controller.eventsProvider.GetEventsCoveringTimerange(startOfDay, endOfDay)
			if err != nil {
				log.Warn().Err(err).Time("start-of-day", startOfDay).Time("end-of-day", endOfDay).Msg("could not get events for day")
				return model.Date{}, nil, fmt.Errorf("could not get events for day %d of month (%w)", dayIndex, err)
			}
			return model.DateFromGotime(startOfDay), &model.EventList{Events: events}, nil
		}
	}
	getMouseMode := func() bool { return controller.data.MouseMode }
	getEventEditMode := func() edit.EventEditMode { return controller.data.EventEditMode }
	summaryVisibleFn := func() bool { return controller.data.ShowSummary }
	getSummary := func() (map[model.CategoryName]time.Duration, error) {
		startOfDay := controller.data.CurrentDate.ToGotime()
		endOfDay := startOfDay.Add(24 * time.Hour)
		result, err := controller.eventsProvider.SumUpTimespanByCategory(startOfDay, endOfDay)
		if err != nil {
			return nil, fmt.Errorf("could not sum up timespans by category (%w)", err)
		}
		return result, nil
	}
	logVisibleFn := func() bool { return controller.data.ShowLog }

	getCurrentDateEventsFn := func() (model.Date, *model.EventList, error) {
		d := controller.data.CurrentDate
		startOfDay := d.ToGotime()
		endOfDay := startOfDay.Add(24 * time.Hour)
		l, err := controller.eventsProvider.GetEventsCoveringTimerange(startOfDay, endOfDay)
		if err != nil {
			return model.Date{}, nil, fmt.Errorf("could not get events for day (%w)", err)
		}
		return d, &model.EventList{Events: l}, nil
	}
	getCurrentEventIDFn := func() *model.EventID { return controller.data.CurrentEventID }

	getWeatherDataFn := func() map[model.DateAndTime]weather.Weather {
		return controller.data.Weather.Data
	}
	getCurrentCategoryFn := func() model.CategoryName { return controller.data.CurrentCategory }
	getCurrentTaskFn := func() *model.TaskID { return controller.data.CurrentTask }

	perfPane := panes.NewPerfPane(
		ui.NewConstrainedRenderer(renderer, func() (x, y, w, h int) { return 2, 2, 50, 2 }),
		func() (x, y, w, h int) { return 2, 2, 50, 2 },
		func() bool { return controller.data.ShowDebug },
		&controller.data.RenderTimes,
		&controller.data.EventProcessingTimes,
	)
	uiDimensions, err := computeUIDimensions(
		renderer,

		tasksWidth,
		toolsWidth,
		func() int { return rightFlexWidth },
		statusHeight,
		weatherWidth,
		timelineWidth,
		editorWidth,
		editorHeight,
	)
	rootPane, err := createUI(
		renderer,
		cursorWrangler,
		stylesheet,
		*uiDimensions,

		tasksVisibleFn,
		toolsVisibleFn,
		summaryVisibleFn,
		logVisibleFn,
		helpVisibleFn,
		getMouseMode,
		getEventEditMode,
		func() ui.MouseCursorPos { return controller.data.CursorPos },

		dayViewInputTree,
		dayViewEventsPaneInputTree,
		helpPaneInputTree,
		summaryPaneInputTree,
		tasksInputTree,
		toolsInputTree,
		monthViewMainPaneInputTree,
		dayViewScrollablePaneInputTree,
		weekdayPaneInputTree,
		monthdayPaneInputTree,
		weekViewEventsWrapperInputTree,
		monthViewEventsWrapperInputTree,
		weekViewMainPaneInputTree,
		rootPaneInputTree,

		createWeekViewDayEventsFn,
		createMonthViewDayEventsFn,
		getCategoryStyle,
		getCategoriesInOrder,
		func() model.Date { return controller.data.CurrentDate },
		getSummary,
		getCurrentDateEventsFn,
		&controller.data.MainTimelineViewParams,
		&controller.data.BacklogViewParams,
		getCurrentEventIDFn,
		getWeatherDataFn,
		getCurrentCategoryFn,
		getCurrentTaskFn,

		controller.eventsProvider,
		controller.suntimesProvider,
		controller.categoryProvider,
		controller.backlogProvider,

		perfPane,
	)
	if err != nil {
		renderer.Fini()
		return nil, fmt.Errorf("Unable to construct UI (%w)", err)
	}
	controller.rootPane = rootPane

	controller.data.ActiveView = rootPane.GetView
	rootPaneInputTree.Root.Children[input.Key{Key: tcell.KeyESC}] = &input.Node{
		Action: action.NewSimple(func() string { return "view up" }, func() {
			rootPane.ViewUp()
		}),
	}
	rootPaneInputTree.Root.Children[input.Key{Key: tcell.KeyRune, Ch: 'i'}] = &input.Node{
		Action: action.NewSimple(func() string { return "view down" }, func() {
			rootPane.ViewDown()
		}),
	}

	var helpPane *panes.HelpPane
	if p := rootPane.GetChild("/help"); p == nil {
		renderer.Fini()
		return nil, fmt.Errorf("Unable to get help pane (got nil).")
	} else {
		var ok bool
		helpPane, ok = p.(*panes.HelpPane)
		if !ok {
			renderer.Fini()
			return nil, fmt.Errorf("Got non-nil pane for 'help' pane not of acceptable type (is %T).", p)
		}
	}
	controller.dayViewMainPane = rootPane.GetChild("day-view-main").(*panes.Composite)
	controller.dayViewEventsPane = rootPane.GetChild("day-view-main/day-view-scrollable/events").(*panes.EventsPane)
	controller.tasksPane = rootPane.GetChild("day-view-main/backlog").(*panes.BacklogPane)
	controller.toolsPane = rootPane.GetChild("day-view-main/tools").(*panes.ToolsPane)
	// TODO / XXX: probably initialize those new panes held by controller

	helpContentRegister = func() {
		helpPane.Content = rootPane.GetHelp()
	}

	// TODO(ja-he): move elsewhere
	// TODO(ja-he): There is a bug with this for midnight while moving down; probably need to rethink
	controller.ensureEventsPaneTimestampWithinVisibleScroll = func(t time.Time) {
		ts := *model.NewTimestampFromGotime(t)
		if !controller.data.CurrentDate.Is(t) {
			// If it's not on this date, either we make 00:00 visible or 24:00,
			// depending on whether its before the current date or after it.
			if !t.After(controller.data.CurrentDate.ToGotime()) {
				ts = model.Timestamp{Hour: 0, Minute: 0}
			} else {
				ts = model.Timestamp{Hour: 24, Minute: 0}
			}
		}
		topRowTime := controller.data.MainTimelineViewParams.TimeAtY(0)
		if topRowTime.IsAfter(ts) {
			controller.data.MainTimelineViewParams.ScrollOffset += (controller.data.MainTimelineViewParams.YForTime(ts))
		}
		_, _, _, maxY := uiDimensions.dayViewEventsPaneDimensions()
		bottomRowTime := controller.data.MainTimelineViewParams.TimeAtY(maxY)
		if ts.IsAfter(bottomRowTime) {
			controller.data.MainTimelineViewParams.ScrollOffset += ((controller.data.MainTimelineViewParams.YForTime(ts)) - maxY)
		}
	}

	controller.createEventEditorPane = func() (ui.Pane, error) {
		eventEditorRenderer := ui.NewConstrainedRenderer(renderer, uiDimensions.editorDimensions)
		return panes.NewCompositeEditorPane(
			eventEditorRenderer,
			cursorWrangler,
			func() bool { return true },
			inputConfig,
			stylesheet,
			controller.data.EventEditor,
		)
	}
	controller.createTaskEditorPane = func() (ui.Pane, error) {
		taskEditorRenderer := ui.NewConstrainedRenderer(renderer, uiDimensions.editorDimensions)
		taskEditorPane, err := panes.NewCompositeEditorPane(
			taskEditorRenderer,
			cursorWrangler,
			func() bool { return true },
			inputConfig,
			stylesheet,
			controller.data.TaskEditor,
		)
		if err != nil {
			return nil, fmt.Errorf("could not construct task editor pane (this is likely a serious programming error / omission) (%w)", err)
		}
		return taskEditorPane, nil
	}

	controller.data.EventEditMode = edit.EventEditModeNormal

	controller.tmpStatusYOffsetGetter = func() int { _, y, _, _ := uiDimensions.statusDimensions(); return y }
	controller.data.EnvData = envData
	controller.screenEvents = renderer.GetEventPollable()

	controller.rootPane = rootPane
	controller.data.CurrentCategory = "default"

	controller.timestampGuesser = func(cursorX, cursorY int) model.Timestamp {
		_, yOffset, _, _ := uiDimensions.dayViewEventsPaneDimensions()
		return controller.data.MainTimelineViewParams.TimeAtY(yOffset + cursorY)
	}

	controller.initializedScreen = renderer
	controller.syncer = renderer

	controller.data.MouseEditState = edit.MouseEditStateNone

	return &controller, nil
}

// ScrollUp scrolls the main timeline view up by the given number of rows.
func (c *Controller) ScrollUp(by int) {
	eventviewTopRow := 0
	if c.data.MainTimelineViewParams.ScrollOffset-by >= eventviewTopRow {
		c.data.MainTimelineViewParams.ScrollOffset -= by
	} else {
		c.ScrollTop()
	}
}

func (c *Controller) createAndEnableTaskEditor(id model.TaskID) {
	if c.data.TaskEditor != nil {
		log.Warn().Msg("apparently, task editor was still active when a new one was activated, unexpected / error")
	}
	var taskEditor edit.Editor
	c.backlogProvider.WithTask(id, func(task model.ReadableTask) {
		var err error
		taskEditor, err = editors.ConstructEditor("root", task, nil, nil, nil /* TODO: pass write fn? */)
		if err != nil {
			log.Error().Err(err).Interface("task", task).Msg("was not able to construct editor for task")
		}
	})
	var ok bool
	c.data.TaskEditor, ok = taskEditor.(*editors.Composite)
	if !ok {
		log.Error().Msgf("somehow, the editor is not a task editor but '%t'; this should never happen", taskEditor)
		c.data.TaskEditor = nil
		return
	}

	taskEditorPane, err := c.createTaskEditorPane()
	if err != nil {
		c.log.Error().Err(err).Msg("Unable to construct task editor UI pane.")
		return
	}

	c.rootPane.PushSubpane(taskEditorPane)
	taskEditorDone := make(chan struct{})
	c.data.TaskEditor.AddQuitCallback(func() {
		close(taskEditorDone) // TODO: this can CERTAINLY happen twice; prevent
	})
	go func() {
		<-taskEditorDone
		c.controllerEvents <- controllerEventTaskEditorExit
	}()
}

// ScrollDown scrolls the main timeline view down by the given number of rows.
func (c *Controller) ScrollDown(by int) {
	eventviewBottomRow := c.tmpStatusYOffsetGetter()
	if c.data.MainTimelineViewParams.ScrollOffset+by+eventviewBottomRow <= (24 * c.data.MainTimelineViewParams.NRowsPerHour) {
		c.data.MainTimelineViewParams.ScrollOffset += by
	} else {
		c.ScrollBottom()
	}
}

// ScrollTop scrolls the main timeline view to the top.
func (c *Controller) ScrollTop() {
	c.data.MainTimelineViewParams.ScrollOffset = 0
}

// ScrollBottom scrolls the main timeline view to the bottom.
func (c *Controller) ScrollBottom() {
	eventviewBottomRow := c.tmpStatusYOffsetGetter()
	c.data.MainTimelineViewParams.ScrollOffset = 24*c.data.MainTimelineViewParams.NRowsPerHour - eventviewBottomRow
}

func (c *Controller) endEdit() {
	c.data.MouseEditState = edit.MouseEditStateNone
	c.data.MouseEditedEventID = nil
	if c.data.EventEditor != nil {
		c.data.EventEditor.Write()
		c.data.EventEditor.Quit()
		c.data.EventEditor = nil
	}
	c.rootPane.PopSubpane() // TODO: this will need to be re-done conceptually
}

func (c *Controller) startMouseMove(eventsInfo *ui.EventsPanePositionInfo) {
	if eventsInfo.Event == nil {
		log.Warn().Msg("no event to move, will not start moving")
		return
	}
	c.data.MouseEditState = edit.MouseEditStateMoving
	c.data.MouseEditedEventID = new(model.EventID)
	*c.data.MouseEditedEventID = eventsInfo.Event.ID
	c.data.CurrentMoveStartingOffset = eventsInfo.Time.Sub(eventsInfo.Event.Start)
}

func (c *Controller) startMouseResize(eventsInfo *ui.EventsPanePositionInfo) {
	if eventsInfo.Event == nil {
		log.Warn().Msg("no event to resize, will not start resizing")
		return
	}
	c.data.MouseEditState = edit.MouseEditStateResizing
	c.data.MouseEditedEventID = new(model.EventID)
	*c.data.MouseEditedEventID = eventsInfo.Event.ID
}

func (c *Controller) getDateAtCursor() model.Date {
	var dateAtCursor model.Date
	av := c.data.ActiveView()
	switch av {
	case ui.ViewDay:
		dateAtCursor = c.data.CurrentDate
	case ui.ViewWeek:
		dateAtCursor = c.data.CurrentDate.GetDayInWeek(0)
	case ui.ViewMonth:
		dateAtCursor = c.data.CurrentDate.GetDayInMonth(0)
	default:
		log.Fatal().Int("view", int(av)).Msg("unknown view encountered")
	}
	return dateAtCursor
}

func (c *Controller) startMouseEventCreation(info *ui.EventsPanePositionInfo) {
	timeAtCursor := info.Time

	eventStartTime := timeAtCursor

	log.Debug().Str("position-time", info.Time.String()).Msg("creation called")

	// create event at time with cat etc.
	e := model.Event{}
	e.Category = c.data.CurrentCategory
	e.Name = ""
	e.Start = eventStartTime
	e.End = eventStartTime.Add(c.data.MainTimelineViewParams.DurationOfHeight(1))

	newEventID, err := c.eventsProvider.AddEvent(e)
	if err != nil {
		log.Error().Err(err).Interface("event", e).Msg("error occurred adding event")
		return
	}
	c.data.MouseEditedEventID = new(model.EventID)
	*c.data.MouseEditedEventID = newEventID
	c.data.MouseEditState = edit.MouseEditStateResizing
}

func (c *Controller) goToDay(newDate model.Date) {
	log.Debug().Str("new-date", newDate.String()).Msg("going to new date")
	if c.data.CurrentDate == newDate {
		c.log.Debug().Msgf("Since current date (%s) is new date (%s) nothing to do switching dates.", c.data.CurrentDate, newDate)
		return
	}
	c.data.CurrentDate = newDate
	c.data.CurrentEventID = nil
}

func (c *Controller) goToPreviousDay() {
	prevDay := c.data.CurrentDate.Prev()
	c.goToDay(prevDay)
}

func (c *Controller) goToNextDay() {
	nextDay := c.data.CurrentDate.Next()
	c.goToDay(nextDay)
}

func (c *Controller) updateCursorPos(x, y int) {
	c.data.CursorPos.X, c.data.CursorPos.Y = x, y
}

func (c *Controller) handleMouseNoneEditEvent(e *tcell.EventMouse) {
	c.data.MouseMode = true

	// get new position
	x, y := e.Position()
	c.updateCursorPos(x, y)

	positionInfo := c.rootPane.GetPositionInfo(x, y)
	if positionInfo == nil {
		c.log.Warn().Msgf("Dropping mouse event due to nil position info.")
		return
	}

	buttons := e.Buttons()

	switch positionInfo := positionInfo.(type) {
	case *ui.StatusPanePositionInfo:

	case *ui.WeatherPanePositionInfo:
		switch buttons {
		case tcell.WheelUp:
			c.ScrollUp(1)
		case tcell.WheelDown:
			c.ScrollDown(1)
		}

	case *ui.TimelinePanePositionInfo:
		switch buttons {
		case tcell.WheelUp:
			c.ScrollUp(1)
		case tcell.WheelDown:
			c.ScrollDown(1)
		}

	case *ui.EventsPanePositionInfo:
		eventsInfo := positionInfo

		// if button clicked, handle
		switch buttons {
		case tcell.Button3:
			c.removeEvent(eventsInfo.Event.ID)
		case tcell.Button2:
			event := eventsInfo.Event
			if event != nil && eventsInfo.Time.After(event.Start) {
				c.eventsProvider.SplitEvent(event.ID, eventsInfo.Time)
			}

		case tcell.Button1:
			// we've clicked while not editing
			// now we need to check where the cursor is and either start event
			// creation, resizing or moving
			switch eventsInfo.EventBoxPart {
			case ui.EventBoxNowhere:
				c.startMouseEventCreation(eventsInfo)
			case ui.EventBoxBottomRight:
				c.startMouseResize(eventsInfo)
			case ui.EventBoxInterior:
				c.startMouseMove(eventsInfo)
			case ui.EventBoxTopEdge:
				log.Info().Msgf("would construct editor here, once the programmer has figured out how to do so correctly")
			}

		case tcell.WheelUp:
			c.ScrollUp(1)

		case tcell.WheelDown:
			c.ScrollDown(1)

		}

	case *ui.ToolsPanePositionInfo:
		toolsInfo := positionInfo
		switch buttons {
		case tcell.Button1:
			cat := toolsInfo.Category
			if cat != nil {
				c.data.CurrentCategory = *cat
			}
		}

	}
}

func (c *Controller) handleMouseResizeEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		x, y := e.Position()

		buttons := e.Buttons()

		switch buttons {
		case tcell.Button1:
			eventID := c.data.MouseEditedEventID
			if eventID == nil {
				c.log.Warn().Msg("no event to resize, will not resize")
				return
			}

			cursorDate := c.getDateAtCursor()
			cursorTime := model.DateAndTimestampToGotime(cursorDate, c.timestampGuesser(x, y))
			visualCursorTime := cursorTime.Add(c.data.MainTimelineViewParams.DurationOfHeight(1))

			var err error
			err = c.eventsProvider.SetEventEnd(*eventID, visualCursorTime)
			if err != nil {
				log.Warn().Err(err).Msgf("unable to resize event %s to end at %s", *eventID, visualCursorTime)
			}

		case tcell.ButtonNone:
			c.endEdit()
		}

		c.updateCursorPos(x, y)
	}
}

func (c *Controller) handleMouseMoveEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		x, y := e.Position()

		buttons := e.Buttons()

		switch buttons {
		case tcell.Button1:
			eventID := c.data.MouseEditedEventID
			if eventID == nil {
				c.log.Warn().Msg("no event to move, will not move")
				return
			}
			event, err := c.eventsProvider.GetEvent(*eventID)
			if err != nil {
				log.Error().Err(err).Msg("could not get event")
				return
			}

			cursorDate := c.getDateAtCursor()
			cursorTimestamp := c.timestampGuesser(x, y)
			cursorTime := model.DateAndTimestampToGotime(cursorDate, cursorTimestamp)
			newStartOfEvent := cursorTime.Add(-c.data.CurrentMoveStartingOffset)
			c.eventsProvider.OffsetEventTimes(event.ID, newStartOfEvent.Sub(event.Start))

		case tcell.ButtonNone:
			c.endEdit()
		}

		c.updateCursorPos(x, y)
	}
}

func (c *Controller) updateWeather() {
	go func() {
		c.log.Debug().Msgf("updating weather..")
		err := c.data.Weather.Update()
		if err != nil {
			c.log.Error().Err(err).Msg("could not update weather data")
		} else {
			c.log.Debug().Msg("successfully retrieved weather data")
		}
		c.controllerEvents <- controllerEventRender
	}()
}

type controllerEvent int

const (
	controllerEventExit controllerEvent = iota
	controllerEventRender
	controllerEventTaskEditorExit
	controllerEventEventEditorExit
)

// Empties all render events from the channel.
// Returns true, if an exit event was encountered so the caller
// knows to exit.
func emptyRenderEvents(c chan controllerEvent) bool {
	for {
		select {
		case bufferedEvent := <-c:
			switch bufferedEvent {
			case controllerEventRender:
				{
					// dump extra render events
				}
			case controllerEventExit:
				return true
			}
		default:
			return false
		}
	}
}

// Run ...
func (c *Controller) Run() {
	log.Info().Msg("dayplan TUI started")

	c.controllerEvents = make(chan controllerEvent, 32)
	var wg sync.WaitGroup

	// Run the main render loop, that renders or exits when prompted accordingly
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer c.initializedScreen.Fini()
		for controllerEvent := range c.controllerEvents {
			switch controllerEvent {
			case controllerEventRender:
				start := time.Now()

				// empty all further render events before rendering
				exitEventEncounteredOnEmpty := emptyRenderEvents(c.controllerEvents)
				// exit if an exit event was coming up
				if exitEventEncounteredOnEmpty {
					return
				}
				// render
				c.rootPane.Draw()

				end := time.Now()
				c.data.RenderTimes.Add(uint64(end.Sub(start).Microseconds()))

			case controllerEventTaskEditorExit:
				if c.data.TaskEditor == nil {
					log.Warn().Msgf("got task editor exit event, but no task editor active; likely logic error")
				} else {
					c.data.TaskEditor = nil
					c.rootPane.PopSubpane()
					log.Debug().Msgf("removed (presumed) task-editor subpane from root")
					go func() { c.controllerEvents <- controllerEventRender }()
				}

			case controllerEventEventEditorExit:
				if c.data.EventEditor == nil {
					log.Warn().Msgf("got event editor exit event, but no event editor active; likely logic error")
				} else {
					c.data.EventEditor = nil
					c.rootPane.PopSubpane()
					log.Debug().Msgf("removed (presumed) event-editor subpane from root")
					go func() { c.controllerEvents <- controllerEventRender }()
				}

			case controllerEventExit:
				return

			default:
				log.Error().Interface("event", controllerEvent).Msgf("unhandled controller event")
			}
		}
	}()

	// Run the time tracking loop, that updates at the start of every minute
	go func() {
		for {
			now := time.Now()
			next := now.Round(1 * time.Minute).Add(1 * time.Minute)
			time.Sleep(time.Until(next))
			c.controllerEvents <- controllerEventRender
		}
	}()

	// Run the event tracking loop, that waits for and processes events and pings
	// for a redraw (or program exit) after each event.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Interface("panic", r).
					Str("stacktrace", string(debug.Stack())).
					Msg("Caught a panic in event tracking.")
				c.controllerEvents <- controllerEventExit
			}
		}()

		for {
			ev := c.screenEvents.PollEvent()

			start := time.Now()

			{
				switch e := ev.(type) {
				case *tcell.EventKey:
					c.data.MouseMode = false
					c.data.MouseEditState = edit.MouseEditStateNone

					key := input.KeyFromTcellEvent(e)
					inputApplied := c.rootPane.ProcessInput(key)
					if !inputApplied {
						log.Warn().Str("key", key.ToDebugString()).Msg("could not apply key input")
					}

				case *tcell.EventMouse:
					c.data.MouseMode = true

					// get new position
					x, y := e.Position()
					c.updateCursorPos(x, y)

					switch c.data.MouseEditState {
					case edit.MouseEditStateNone:
						c.handleMouseNoneEditEvent(e)
					case edit.MouseEditStateResizing:
						c.handleMouseResizeEditEvent(ev)
					case edit.MouseEditStateMoving:
						c.handleMouseMoveEditEvent(ev)
					}

				case *tcell.EventResize:
					c.syncer.NeedsSync()

				}
			}

			end := time.Now()
			c.data.EventProcessingTimes.Add(uint64(end.Sub(start).Microseconds()))

			c.controllerEvents <- controllerEventRender
		}
	}()

	wg.Wait()
}

func (c *Controller) getCurrentViewEvents() ([]*model.Event, error) {
	av := c.data.ActiveView()
	switch av {
	case ui.ViewDay:
		return c.getCurrentDayEvents()
	case ui.ViewWeek:
		return c.getCurrentWeekEvents()
	case ui.ViewMonth:
		return c.getCurrentMonthEvents()
	default:
		return nil, fmt.Errorf("unknown view (%d) in summary data gathering", av)
	}
}

func (c *Controller) getCurrentDayEvents() ([]*model.Event, error) {
	startTime := c.data.CurrentDate.ToGotime()
	endTime := startTime.Add(24 * time.Hour)
	events, err := c.eventsProvider.GetEventsCoveringTimerange(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("could not get events for current day (%w)", err)
	}
	return events, nil
}

func (c *Controller) getCurrentWeekEvents() ([]*model.Event, error) {
	startTime := c.data.CurrentDate.ToGotime()
	endTime := startTime.Add(7 * 24 * time.Hour)
	events, err := c.eventsProvider.GetEventsCoveringTimerange(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("could not get events for current week (%w)", err)
	}
	return events, nil
}

func (c *Controller) getCurrentMonthEvents() ([]*model.Event, error) {
	startTime := c.data.CurrentDate.ToGotime()
	endTime := startTime.AddDate(0, 1, 0)
	events, err := c.eventsProvider.GetEventsCoveringTimerange(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("could not get events for current month (%w)", err)
	}
	return events, nil
}

func (c *Controller) ensureCurrentEventVisible() {
	id := c.data.CurrentEventID
	if id == nil {
		c.log.Info().Msg("no current event selected, so nothing to ensure visible")
		return
	}
	e, err := c.eventsProvider.GetEvent(*id)
	if err != nil {
		c.log.Error().Err(err).Msg("could not get current event while ensuring visibility")
		return
	}
	c.ensureEventsPaneTimestampWithinVisibleScroll(e.Start)
	c.ensureEventsPaneTimestampWithinVisibleScroll(e.End)
}

func (c *Controller) switchToNextEventInDay() {
	defer c.ensureCurrentEventVisible()

	if c.data.CurrentEventID == nil {
		candidate, err := c.eventsProvider.GetEventAfter(c.data.CurrentDate.ToGotime())
		if err != nil {
			c.log.Error().Err(err).Stringer("date", c.data.CurrentDate).Msg("could not get next for current date")
			return
		}
		if candidate == nil {
			c.log.Warn().Msgf("No model after exists.")
			return
		}
		if model.DateFromGotime(candidate.Start) == c.data.CurrentDate {
			c.data.CurrentEventID = new(model.EventID)
			*c.data.CurrentEventID = candidate.ID
			c.log.Debug().Stringer("event", candidate).Msg("switched to next event")
			return
		}
		c.log.Debug().Msg("no event on current day")
		return
	}

	currentEventID := *c.data.CurrentEventID
	{
		e, err := c.eventsProvider.GetEvent(currentEventID)
		if err != nil {
			c.log.Error().
				Str("ID", currentEventID).
				Msg("Could not find current event to check validity, will set to nil.")
			c.data.CurrentEventID = nil
			return
		}
		currentDate := c.data.CurrentDate
		eStartDate := model.DateFromGotime(e.Start)
		if currentDate != eStartDate {
			c.log.Error().
				Stringer("currentDate", currentDate).
				Stringer("eStartDate", eStartDate).
				Str("ID", currentEventID).
				Msg("Current event is not on current date.")
		}
	}

	next, err := c.eventsProvider.GetFollowingEvent(currentEventID)
	if err != nil {
		c.log.Error().Err(err).Str("id", string(currentEventID)).Msg("could not get following event of current event")
		return
	}
	if next == nil {
		c.log.Info().Msg("there is no next event")
		return
	}
	if model.DateFromGotime(next.Start) != c.data.CurrentDate {
		c.log.Info().Msg("next event is on a different day")
		return
	}

	c.data.CurrentEventID = new(model.EventID)
	*c.data.CurrentEventID = next.ID
	c.log.Debug().Stringer("event", next).Msg("switched to next event")
}

func (c *Controller) switchToPreviousEventInDay() {
	defer c.ensureCurrentEventVisible()

	if c.data.CurrentEventID == nil {
		candidate, err := c.eventsProvider.GetEventBefore(c.data.CurrentDate.ToGotime().Add(24 * time.Hour))
		if err != nil {
			c.log.Error().Err(err).Stringer("date", c.data.CurrentDate).Msg("could not get prev for current date")
			return
		}
		if candidate == nil {
			c.log.Warn().Msgf("No model before exists.")
			return
		}
		if model.DateFromGotime(candidate.Start) == c.data.CurrentDate {
			c.data.CurrentEventID = new(model.EventID)
			*c.data.CurrentEventID = candidate.ID
			c.log.Debug().Stringer("event", candidate).Msg("switched to prev event")
			return
		}
		c.log.Debug().Msg("no event on current day")
		return
	}

	prev, err := c.eventsProvider.GetPrecedingEvent(*c.data.CurrentEventID)
	if err != nil {
		c.log.Error().Err(err).Stringer("date", c.data.CurrentDate).Msg("could not get prev for current date")
		return
	}
	if prev == nil {
		c.log.Info().Msg("there is no prev event")
		return
	}
	if model.DateFromGotime(prev.Start) != c.data.CurrentDate {
		c.log.Info().Msg("prev event is on a different day")
		return
	}

	// current event ID is not nil, so we can just set it to the previous event's
	*c.data.CurrentEventID = prev.ID
	c.log.Debug().Stringer("event", prev).Msg("switched to prev event")
}

func (c *Controller) moveEventsForwardPushing() error {
	pushDuration := c.data.MainTimelineViewParams.DurationOfHeight(1)
	pushResolution := c.data.MainTimelineViewParams.DurationOfHeight(1)
	return c.moveEventsPushingBy(pushDuration, pushResolution)
}

func (c *Controller) moveEventsBackwardPushing() error {
	pushDuration := -c.data.MainTimelineViewParams.DurationOfHeight(1)
	pushResolution := c.data.MainTimelineViewParams.DurationOfHeight(1)
	return c.moveEventsPushingBy(pushDuration, pushResolution)
}

// moves events pushing other events
//
// d this is how "far" everything gets pushed.
//
// m is basically the "modulus", i.e. if something needs to get snapped to the
// grid of visible resolution, this is that, not sure yet if needed really.
func (c *Controller) moveEventsPushingBy(d, m time.Duration) error {
	return fmt.Errorf("unimplemented (this should push for %s (with res %s))", d, m)
}

func (c *Controller) removeEvent(id model.EventID) {
	isCurrentEvent := c.data.CurrentEventID != nil && *c.data.CurrentEventID == id
	var newCurrentEventID *model.EventID
	if isCurrentEvent {
		nextEvent, err := c.eventsProvider.GetFollowingEvent(id)
		if err != nil {
			c.log.Error().Err(err).Msg("could not get following event")
		} else if nextEvent == nil || !c.data.CurrentDate.Is(nextEvent.Start) {
			prevEvent, err := c.eventsProvider.GetPrecedingEvent(id)
			if err != nil {
				c.log.Error().Err(err).Msg("could not get preceding event")
			} else if nextEvent != nil && c.data.CurrentDate.Is(prevEvent.End) {
				newCurrentEventID = new(model.EventID)
				*newCurrentEventID = prevEvent.ID
				log.Debug().Msgf("will switch to previous event: %s", *newCurrentEventID)
			}
		} else {
			newCurrentEventID = new(model.EventID)
			*newCurrentEventID = nextEvent.ID
			log.Debug().Msgf("will switch to next event: %s", *newCurrentEventID)
		}
		if newCurrentEventID == nil {
			c.log.Debug().Msg("no next/prev event to switch to")
		}
	}
	err := c.eventsProvider.RemoveEvent(id)
	if err != nil {
		c.log.Error().Err(err).Msg("could not remove event")
		return
	}
	if isCurrentEvent {
		if newCurrentEventID != nil {
			c.log.Debug().Msgf("updating current event to %s", *newCurrentEventID)
			c.data.CurrentEventID = new(model.EventID)
			*c.data.CurrentEventID = *newCurrentEventID
			c.ensureCurrentEventVisible()
		} else {
			c.log.Debug().Msg("nilling current event")
			c.data.CurrentEventID = nil
		}
	}
}

func (c *Controller) removeEvents(ids []model.EventID) {
	err := c.eventsProvider.RemoveEvents(ids)
	if err != nil {
		c.log.Error().Err(err).Msg("could not remove events")
		return
	}

	c.log.Warn().Msgf("missing implementation of current-event updating after removing multiple events (just nilling)")
	c.data.CurrentEventID = nil
}
