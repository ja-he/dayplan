package cli

import (
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/control"
	"github.com/ja-he/dayplan/internal/control/action"
	"github.com/ja-he/dayplan/internal/control/edit"
	"github.com/ja-he/dayplan/internal/control/edit/editors"
	"github.com/ja-he/dayplan/internal/input"
	"github.com/ja-he/dayplan/internal/input/processors"
	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/potatolog"
	"github.com/ja-he/dayplan/internal/storage"
	"github.com/ja-he/dayplan/internal/styling"
	"github.com/ja-he/dayplan/internal/tui"
	"github.com/ja-he/dayplan/internal/ui"
	"github.com/ja-he/dayplan/internal/ui/panes"
	"github.com/ja-he/dayplan/internal/weather"

	"github.com/gdamore/tcell/v2"
)

// TODO: this absolutely does not belong here
func (t *Controller) GetDayFromFileHandler(date model.Date) *model.Day {
	t.fhMutex.RLock()
	fh, ok := t.FileHandlers[date]
	t.fhMutex.RUnlock()
	if ok {
		tmp := fh.Read(t.data.Categories)
		return tmp
	} else {
		newHandler := storage.NewFileHandler(t.data.EnvData.BaseDirPath + "/days/" + date.ToString())
		t.fhMutex.Lock()
		t.FileHandlers[date] = newHandler
		t.fhMutex.Unlock()
		tmp := newHandler.Read(t.data.Categories)
		return tmp
	}
}

// Controller
type Controller struct {
	data     *control.ControlData
	rootPane *panes.RootPane

	fhMutex          sync.RWMutex
	FileHandlers     map[model.Date]*storage.FileHandler
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

// NewController
func NewController(
	date model.Date,
	envData control.EnvData,
	categoryStyling styling.CategoryStyling,
	stylesheet styling.Stylesheet,
	stderrLogger zerolog.Logger,
	tuiLogger zerolog.Logger,
) *Controller {
	controller := Controller{}

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
				"<ESC>":   "quit",
				"D":       "delete-to-end",
				"d$":      "delete-to-end",
				"C":       "delete-to-end-and-insert",
				"c$":      "delete-to-end-and-insert",
				"x":       "delete-rune",
				"s":       "delete-rune-and-insert",
				"i":       "swap-mode-insert",
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

	categoryGetter := func(name string) model.Category {
		cat, ok := categoryStyling.GetKnownCategoriesByName()[name]
		if ok {
			return *cat
		}
		return model.Category{
			Name: name,
		}
	}

	controller.data = control.NewControlData(categoryStyling)
	backlogFilePath := path.Join(envData.BaseDirPath, "days", "backlog.yml") // TODO(ja_he): Migrate 'days' -> 'data', perhaps subdir 'days'
	backlog, err := func() (*model.Backlog, error) {
		backlogReader, err := os.Open(backlogFilePath)
		if err != nil {
			return &model.Backlog{}, err
		}
		defer backlogReader.Close()
		return model.BacklogFromReader(backlogReader, categoryGetter)
	}()
	if err != nil {
		tuiLogger.Error().Err(err).Str("file", backlogFilePath).Msg("could not read backlog")
	} else {
		tuiLogger.Info().Str("file", backlogFilePath).Msg("successfully read backlog")
	}
	tuiLogger.Debug().Interface("backlog", backlog).Msg("backlog")

	tasksWidth := 40
	toolsWidth := func() int {
		width := 20
		for _, c := range categoryStyling.GetAll() {
			requisiteWidth := len(c.Cat.Name) + 4
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
		"<c-u>": action.NewSimple(func() string { return "scoll up" }, func() { controller.ScrollUp(10) }),
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

	eventsViewBaseInputMap := map[input.Keyspec]action.Action{
		"w": action.NewSimple(func() string { return "write day to file" }, controller.writeModel),
		"h": action.NewSimple(func() string { return "go to previous day" }, controller.goToPreviousDay),
		"l": action.NewSimple(func() string { return "go to next day" }, controller.goToNextDay),
		"c": action.NewSimple(func() string { return "clear day's events" }, func() {
			controller.data.Days.AddDay(controller.data.CurrentDate, model.NewDay(), controller.data.GetCurrentSuntimes())
		}),
	}

	renderer := tui.NewTUIScreenHandler()
	screenSize := func() (w, h int) { _, _, w, h = renderer.Dimensions(); return }
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
	helpDimensions := screenDimensions
	editorDimensions := centeredFloat(editorWidth, editorHeight)
	tasksDimensions := func() (x, y, w, h int) {
		screenWidth, screenHeight := screenSize()
		return screenWidth - rightFlexWidth, 0, tasksWidth, screenHeight - statusHeight
	}
	toolsDimensions := func() (x, y, w, h int) {
		screenWidth, screenHeight := screenSize()
		return screenWidth - toolsWidth, 0, toolsWidth, screenHeight - statusHeight
	}
	statusDimensions := func() (x, y, w, h int) {
		screenWidth, screenHeight := screenSize()
		return 0, screenHeight - statusHeight, screenWidth, statusHeight
	}
	dayViewMainPaneDimensions := screenDimensions
	dayViewScrollablePaneDimensions := func() (x, y, w, h int) {
		parentX, parentY, parentW, parentH := dayViewMainPaneDimensions()
		return parentX, parentY, parentW - rightFlexWidth, parentH - statusHeight
	}
	weekViewMainPaneDimensions := screenDimensions
	monthViewMainPaneDimensions := screenDimensions
	weatherDimensions := func() (x, y, w, h int) {
		parentX, parentY, _, parentH := dayViewScrollablePaneDimensions()
		return parentX, parentY, weatherWidth, parentH
	}
	dayViewEventsPaneDimensions := func() (x, y, w, h int) {
		ox, oy, ow, oh := dayViewScrollablePaneDimensions()
		x = ox + weatherWidth + timelineWidth
		y = oy
		w = ow - x
		h = oh
		return x, y, w, h
	}
	dayViewTimelineDimensions := func() (x, y, w, h int) {
		_, _, _, parentH := dayViewScrollablePaneDimensions()
		return 0 + weatherWidth, 0, timelineWidth, parentH
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
	weekdayPaneInputTree, err := input.ConstructInputTree(eventsViewBaseInputMap)
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("could not construct weekday pane input tree")
	}
	weekdayPane := func(dayIndex int) *panes.EventsPane {
		return panes.NewEventsPane(
			ui.NewConstrainedRenderer(renderer, weekdayDimensions(dayIndex)),
			weekdayDimensions(dayIndex),
			stylesheet,
			processors.NewModalInputProcessor(weekdayPaneInputTree),
			func() *model.Day {
				return controller.data.Days.GetDay(controller.data.CurrentDate.GetDayInWeek(dayIndex))
			},
			categoryStyling.GetStyle,
			&controller.data.MainTimelineViewParams,
			&controller.data.CursorPos,
			0,
			false,
			true,
			false,
			func() bool { return controller.data.CurrentDate.GetDayInWeek(dayIndex) == controller.data.CurrentDate },
			func() *model.Event { return nil /* TODO */ },
			func() bool { return controller.data.MouseMode },
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
	monthdayPaneInputTree, err := input.ConstructInputTree(eventsViewBaseInputMap)
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("could not construct monthday pane input tree")
	}
	monthdayPane := func(dayIndex int) ui.Pane {
		return panes.NewMaybePane(
			func() bool {
				return controller.data.CurrentDate.GetDayInMonth(dayIndex).Month == controller.data.CurrentDate.Month
			},
			panes.NewEventsPane(
				ui.NewConstrainedRenderer(renderer, monthdayDimensions(dayIndex)),
				monthdayDimensions(dayIndex),
				stylesheet,
				processors.NewModalInputProcessor(monthdayPaneInputTree),
				func() *model.Day {
					return controller.data.Days.GetDay(controller.data.CurrentDate.GetDayInMonth(dayIndex))
				},
				categoryStyling.GetStyle,
				&controller.data.MainTimelineViewParams,
				&controller.data.CursorPos,
				0,
				false,
				false,
				false,
				func() bool { return controller.data.CurrentDate.GetDayInMonth(dayIndex) == controller.data.CurrentDate },
				func() *model.Event { return nil /* TODO */ },
				func() bool { return controller.data.MouseMode },
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
		ui.NewConstrainedRenderer(renderer, statusDimensions),
		statusDimensions,
		stylesheet,
		&controller.data.CurrentDate,
		func() int {
			_, _, w, _ := statusDimensions()
			switch controller.data.ActiveView() {
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
			switch controller.data.ActiveView() {
			case ui.ViewDay:
				return 1
			case ui.ViewWeek:
				return 7
			case ui.ViewMonth:
				return controller.data.CurrentDate.GetLastOfMonth().Day
			default:
				panic("unknown view for status rendering")
			}
		},
		func() int {
			switch controller.data.ActiveView() {
			case ui.ViewDay:
				return 1
			case ui.ViewWeek:
				switch controller.data.CurrentDate.ToWeekday() {
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
				return controller.data.CurrentDate.Day
			default:
				panic("unknown view for status rendering")
			}
		},
		func() int { return timelineWidth },
		func() edit.EventEditMode { return controller.data.EventEditMode },
	)

	cursorWrangler := ui.NewCursorWrangler(renderer)

	var currentTask *model.Task
	setCurrentTask := func(t *model.Task) { currentTask = t }
	backlogViewParams := ui.BacklogViewParams{
		NRowsPerHour: &controller.data.MainTimelineViewParams.NRowsPerHour,
		ScrollOffset: 0,
	}
	var ensureBacklogTaskVisible func(t *model.Task)
	var scrollBacklogTop func()
	var scrollBacklogBottom func()
	var backlogSetCurrentToTopmost func()
	var backlogSetCurrentToBottommost func()
	var getBacklogBottomScrollOffset func() int
	var offsetCurrentTask func(tl []*model.Task, setToNext bool) bool
	popAndScheduleCurrentTask := func(when *time.Time) {
		// pass nil time to not schedule
		if currentTask == nil {
			return
		}
		scheduledTask := currentTask
		prev, next, parentage, err := backlog.Pop(scheduledTask)
		if err != nil {
			log.Error().
				Err(err).
				Interface("task", currentTask).
				Interface("backlog", backlog).
				Msg("could not find task")
		} else {
			// update current task
			currentTask = func() *model.Task {
				switch {
				case next != nil:
					return next
				case prev != nil:
					return prev
				case len(parentage) > 0:
					return parentage[0]
				default:
					return nil
				}
			}()
			// schedule task, if time for that was given
			if when != nil {
				namePrefix := ""
				for _, parent := range parentage {
					namePrefix = parent.Name + ": " + namePrefix
				}
				newEvents := scheduledTask.ToEvent(*when, namePrefix)
				for _, newEvent := range newEvents {
					controller.data.GetCurrentDay().AddEvent(newEvent)
				}
			}
		}
	}
	createAndEnableTaskEditor := func(task *model.Task) {
		if controller.data.TaskEditor != nil {
			log.Warn().Msg("apparently, task editor was still active when a new one was activated, unexpected / error")
		}
		var err error
		taskEditor, err := editors.ConstructEditor("root", task, nil, func() (bool, bool) { return true, true })
		if err != nil {
			log.Error().Err(err).Interface("task", task).Msg("was not able to construct editor for task")
			return
		}
		var ok bool
		controller.data.TaskEditor, ok = taskEditor.(*editors.Composite)
		if !ok {
			log.Error().Msgf("somehow, the editor is not a task editor but '%t'; this should never happen", taskEditor)
			controller.data.TaskEditor = nil
			return
		}
		taskEditorPane, err := controller.data.TaskEditor.GetPane(
			ui.NewConstrainedRenderer(renderer, func() (x, y, w, h int) {
				screenWidth, screenHeight := screenSize()
				taskEditorBoxWidth := int(math.Min(80, float64(screenWidth)))
				taskEditorBoxHeight := int(math.Min(20, float64(screenHeight)))
				return (screenWidth / 2) - (taskEditorBoxWidth / 2), (screenHeight / 2) - (taskEditorBoxHeight / 2), taskEditorBoxWidth, taskEditorBoxHeight
			}),
			func() bool { return true },
			inputConfig,
			stylesheet,
			cursorWrangler,
		)
		if err != nil {
			log.Error().Err(err).Msgf("could not construct task editor pane")
			controller.data.TaskEditor = nil
			return
		}
		log.Info().Str("info", taskEditorPane.(*panes.CompositeEditorPane).GetDebugInfo()).Msg("here is the debug info for the task editor pane")
		controller.rootPane.PushSubpane(taskEditorPane)
		taskEditorDone := make(chan struct{})
		controller.data.TaskEditor.AddQuitCallback(func() {
			close(taskEditorDone) // TODO: this can CERTAINLY happen twice; prevent
		})
		go func() {
			<-taskEditorDone
			controller.controllerEvents <- ControllerEventTaskEditorExit
		}()
	}
	tasksInputTree, err := input.ConstructInputTree(
		map[input.Keyspec]action.Action{
			"<c-u>": action.NewSimple(func() string { return "scroll up" }, func() {
				backlogViewParams.SetScrollOffset(backlogViewParams.GetScrollOffset() - 10)
				if backlogViewParams.GetScrollOffset() < 0 {
					scrollBacklogTop()
				}
			}),
			"<c-d>": action.NewSimple(func() string { return "scroll down" }, func() {
				scrollTarget := backlogViewParams.GetScrollOffset() + 10
				if scrollTarget > getBacklogBottomScrollOffset() {
					scrollBacklogBottom()
				} else {
					backlogViewParams.SetScrollOffset(scrollTarget)
				}
			}),
			"j": action.NewSimple(func() string { return "go down a task" }, func() {
				if currentTask == nil {
					if len(backlog.Tasks) > 0 {
						currentTask = backlog.Tasks[0]
					}
					return
				}

				found := offsetCurrentTask(backlog.Tasks, true)
				if !found {
					setCurrentTask(nil)
				}
				ensureBacklogTaskVisible(currentTask)
			}),
			"k": action.NewSimple(func() string { return "go up a task" }, func() {
				if currentTask == nil {
					if len(backlog.Tasks) > 0 {
						currentTask = backlog.Tasks[0]
					}
					return
				}

				found := offsetCurrentTask(backlog.Tasks, false)
				if !found {
					setCurrentTask(nil)
				}
				ensureBacklogTaskVisible(currentTask)
			}),
			"gg": action.NewSimple(func() string { return "scroll to top" }, func() {
				backlogSetCurrentToTopmost()
			}),
			"G": action.NewSimple(func() string { return "scroll to bottom" }, func() {
				backlogSetCurrentToBottommost()
			}),
			"sn": action.NewSimple(func() string { return "schedule now" }, func() {
				when := time.Now()
				popAndScheduleCurrentTask(&when)
			}),
			"d": action.NewSimple(func() string { return "delete task" }, func() {
				popAndScheduleCurrentTask(nil)
			}),
			"l": action.NewSimple(func() string { return "step into subtasks" }, func() {
				if currentTask == nil {
					return
				}
				if len(currentTask.Subtasks) > 0 {
					currentTask = currentTask.Subtasks[0]
					ensureBacklogTaskVisible(currentTask)
				} else {
					log.Debug().Msg("current task has no subtasks, so remaining at it")
				}
			}),
			"h": action.NewSimple(func() string { return "step out to parent task" }, func() {
				var findParent func(searchedTask *model.Task, parent *model.Task, tasks []*model.Task) *model.Task
				findParent = func(searchedTask *model.Task, parent *model.Task, parentsTasks []*model.Task) *model.Task {
					for _, t := range parentsTasks {
						if t == searchedTask {
							return parent
						}
						maybeParent := findParent(searchedTask, t, t.Subtasks)
						if maybeParent != nil {
							return maybeParent
						}
					}
					return nil
				}
				maybeParent := findParent(currentTask, nil, backlog.Tasks)
				if maybeParent != nil {
					setCurrentTask(maybeParent)
					ensureBacklogTaskVisible(currentTask)
				} else {
					log.Debug().Msg("could not find parent, so not changing current task")
				}
			}),
			"O": action.NewSimple(func() string { return "add a new task above the current one" }, func() {
				if currentTask == nil {
					log.Debug().Msgf("asked to add a task after to nil current task, adding as first")
					newTask := backlog.AddLast()
					newTask.Name = "" // user should be hinted to change this quite quickly, i.e. via immediate editor activation
					newTask.Category = controller.data.CurrentCategory
					currentTask = newTask
					createAndEnableTaskEditor(currentTask)
					return
				}
				newTask, parent, err := backlog.AddBefore(currentTask)
				if err != nil {
					log.Error().Err(err).Msgf("was unable to add a task after '%s'", currentTask.Name)
					return
				}
				newTask.Name = "" // user should be hinted to change this quite quickly, i.e. via immediate editor activation
				if parent != nil {
					newTask.Category = parent.Category
				} else {
					newTask.Category = controller.data.CurrentCategory
				}
				currentTask = newTask
				createAndEnableTaskEditor(currentTask)
			}),
			"o": action.NewSimple(func() string { return "add a new task below the current one" }, func() {
				if currentTask == nil {
					log.Debug().Msgf("asked to add a task after to nil current task, adding as first")
					newTask := backlog.AddLast()
					newTask.Name = "" // user should be hinted to change this quite quickly, i.e. via immediate editor activation
					newTask.Category = controller.data.CurrentCategory
					currentTask = newTask
					createAndEnableTaskEditor(currentTask)
					return
				}
				newTask, parent, err := backlog.AddAfter(currentTask)
				if err != nil {
					log.Error().Err(err).Msgf("was unable to add a task after '%s'", currentTask.Name)
					return
				}
				newTask.Name = "" // user should be hinted to change this quite quickly, i.e. via immediate editor activation
				if parent != nil {
					newTask.Category = parent.Category
				} else {
					newTask.Category = controller.data.CurrentCategory
				}
				currentTask = newTask
				createAndEnableTaskEditor(currentTask)
			}),
			"i": action.NewSimple(func() string { return "add a new subtask of the current task" }, func() {
				if currentTask == nil {
					log.Warn().Msgf("asked to add a subtask to nil current task")
					return
				}
				newTask := &model.Task{
					Name:     "", // user should be hinted to change this quite quickly, i.e. via immediate editor activation
					Category: currentTask.Category,
				}
				currentTask.Subtasks = append(currentTask.Subtasks, newTask)
				currentTask = newTask
				createAndEnableTaskEditor(currentTask)
			}),
			"<cr>": action.NewSimple(func() string { return "begin editing of task" }, func() { createAndEnableTaskEditor(currentTask) }),
			"w": action.NewSimple(func() string { return "store backlog to file" }, func() {
				writer, err := os.OpenFile(backlogFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
				if err != nil {
					log.Error().Err(err).Msgf("unable to write open backlog file '%s' for writing", backlogFilePath)
					return
				}
				defer writer.Close()
				err = backlog.Write(writer)
				if err != nil {
					log.Error().Err(err).Msg("unable to write backlog to writer")
					return
				}
				log.Info().Msgf("wrote backlog to '%s' sucessfully", backlogFilePath)
			}),
		},
	)
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for tasks pane")
	}
	toolsInputTree, err := input.ConstructInputTree(
		map[input.Keyspec]action.Action{
			"j": action.NewSimple(func() string { return "switch to next category" }, func() {
				for i, cat := range controller.data.Categories {
					if cat == controller.data.CurrentCategory {
						for ii := i + 1; ii < len(controller.data.Categories); ii++ {
							if !controller.data.Categories[ii].Deprecated {
								controller.data.CurrentCategory = controller.data.Categories[ii]
								return
							}
						}
					}
				}
			}),
			"k": action.NewSimple(func() string { return "switch to previous category" }, func() {
				for i, cat := range controller.data.Categories {
					if cat == controller.data.CurrentCategory {
						for ii := i - 1; ii >= 0; ii-- {
							if !controller.data.Categories[ii].Deprecated {
								controller.data.CurrentCategory = controller.data.Categories[ii]
								return
							}
						}
					}
				}
			}),
		},
	)
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for tools pane")
	}

	// TODO(ja-he): move elsewhere
	ensureEventsPaneTimestampVisible := func(time model.Timestamp) {
		topRowTime := controller.data.MainTimelineViewParams.TimeAtY(0)
		if topRowTime.IsAfter(time) {
			controller.data.MainTimelineViewParams.ScrollOffset += (controller.data.MainTimelineViewParams.YForTime(time))
		}
		_, _, _, maxY := dayViewEventsPaneDimensions()
		bottomRowTime := controller.data.MainTimelineViewParams.TimeAtY(maxY)
		if time.IsAfter(bottomRowTime) {
			controller.data.MainTimelineViewParams.ScrollOffset += ((controller.data.MainTimelineViewParams.YForTime(time)) - maxY)
		}
	}
	var startMovePushing func()
	var pushEditorAsRootSubpane func()
	// TODO: directly?
	eventsPaneDayInputExtension := map[input.Keyspec]action.Action{
		"j": action.NewSimple(func() string { return "switch to next event" }, func() {
			controller.data.GetCurrentDay().CurrentNext()
			if controller.data.GetCurrentDay().Current != nil {
				ensureEventsPaneTimestampVisible(controller.data.GetCurrentDay().Current.Start)
				ensureEventsPaneTimestampVisible(controller.data.GetCurrentDay().Current.End)
			}
		}),
		"k": action.NewSimple(func() string { return "switch to previous event" }, func() {
			controller.data.GetCurrentDay().CurrentPrev()
			if controller.data.GetCurrentDay().Current != nil {
				ensureEventsPaneTimestampVisible(controller.data.GetCurrentDay().Current.End)
				ensureEventsPaneTimestampVisible(controller.data.GetCurrentDay().Current.Start)
			}
		}),
		"d": action.NewSimple(func() string { return "delete selected event" }, func() {
			event := controller.data.GetCurrentDay().Current
			if event != nil {
				controller.data.GetCurrentDay().RemoveEvent(event)
			}
		}),
		"<cr>": action.NewSimple(func() string { return "open the event editor" }, func() {
			event := controller.data.GetCurrentDay().Current
			if event != nil {
				controller.data.EventEditor.Activate(event)
			}
			pushEditorAsRootSubpane()
		}),
		"o": action.NewSimple(func() string { return "add event after selected" }, func() {
			current := controller.data.GetCurrentDay().Current
			newEvent := &model.Event{
				Name: "",
				Cat:  controller.data.CurrentCategory,
			}
			if current == nil {
				newEvent.Start = model.NewTimestampFromGotime(time.Now()).
					Snap(int(controller.data.MainTimelineViewParams.DurationOfHeight(1) / time.Minute))
			} else {
				newEvent.Start = current.End
			}
			eventAfter := controller.data.GetCurrentDay().GetNextEventAfter(newEvent.Start)
			if eventAfter != nil && newEvent.Start.DurationInMinutesUntil(eventAfter.Start) < 60 {
				newEvent.End = eventAfter.Start
			} else {
				newEvent.End = newEvent.Start.OffsetMinutes(60)
			}
			controller.data.GetCurrentDay().AddEvent(newEvent)
			ensureEventsPaneTimestampVisible(newEvent.End)
		}),
		"O": action.NewSimple(func() string { return "add event before selected" }, func() {
			current := controller.data.GetCurrentDay().Current
			newEvent := &model.Event{
				Name: "",
				Cat:  controller.data.CurrentCategory,
			}
			if current == nil {
				newEvent.End = model.NewTimestampFromGotime(time.Now()).
					Snap(int(controller.data.MainTimelineViewParams.DurationOfHeight(1) / time.Minute))
			} else {
				newEvent.End = current.Start
			}
			eventBefore := controller.data.GetCurrentDay().GetPrevEventBefore(newEvent.End)
			if eventBefore != nil && eventBefore.End.DurationInMinutesUntil(newEvent.End) < 60 {
				newEvent.Start = eventBefore.End
			} else {
				newEvent.Start = newEvent.End.OffsetMinutes(-60)
			}
			controller.data.GetCurrentDay().AddEvent(newEvent)
			ensureEventsPaneTimestampVisible(newEvent.Start)
		}),
		"<c-o>": action.NewSimple(func() string { return "add event now" }, func() {
			newEvent := &model.Event{
				Name: "",
				Cat:  controller.data.CurrentCategory,
			}
			newEvent.Start = *model.NewTimestampFromGotime(time.Now())
			eventAfter := controller.data.GetCurrentDay().GetNextEventAfter(newEvent.Start)
			if eventAfter != nil && newEvent.Start.DurationInMinutesUntil(eventAfter.Start) < 60 {
				newEvent.End = eventAfter.Start
			} else {
				newEvent.End = newEvent.Start.OffsetMinutes(60)
			}
			controller.data.GetCurrentDay().AddEvent(newEvent)
			ensureEventsPaneTimestampVisible(newEvent.Start)
		}),
		"sn": action.NewSimple(func() string { return "split selected event now" }, func() {
			current := controller.data.GetCurrentDay().Current
			if current == nil {
				return
			}
			now := model.NewTimestampFromGotime(time.Now())
			controller.data.GetCurrentDay().SplitEvent(current, *now)
		}),
		"sc": action.NewSimple(func() string { return "split selected event at its center" }, func() {
			current := controller.data.GetCurrentDay().Current
			if current == nil {
				return
			}
			center := current.Start.OffsetMinutes(current.Start.DurationInMinutesUntil(current.End) / 2)
			controller.data.GetCurrentDay().SplitEvent(current, center)
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
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for day view pane's events subpane")
	}

	tasksVisible := false
	toolsVisible := true
	tasksPane := panes.NewBacklogPane(
		ui.NewConstrainedRenderer(renderer, tasksDimensions),
		tasksDimensions,
		stylesheet,
		processors.NewModalInputProcessor(tasksInputTree),
		&backlogViewParams,
		func() *model.Task { return currentTask },
		backlog,
		categoryStyling.GetStyle,
		func() bool { return tasksVisible },
	)
	toolsPane := panes.NewToolsPane(
		ui.NewConstrainedRenderer(renderer, toolsDimensions),
		toolsDimensions,
		stylesheet,
		processors.NewModalInputProcessor(toolsInputTree),
		&controller.data.CurrentCategory,
		&categoryStyling,
		2,
		1,
		0,
		func() bool { return toolsVisible },
	)
	dayEventsPane := panes.NewEventsPane(
		ui.NewConstrainedRenderer(renderer, dayViewEventsPaneDimensions),
		dayViewEventsPaneDimensions,
		stylesheet,
		processors.NewModalInputProcessor(dayViewEventsPaneInputTree),
		controller.data.GetCurrentDay,
		categoryStyling.GetStyle,
		&controller.data.MainTimelineViewParams,
		&controller.data.CursorPos,
		2,
		true,
		true,
		true,
		func() bool { return true },
		func() *model.Event { return controller.data.GetCurrentDay().Current },
		func() bool { return controller.data.MouseMode },
	)
	startMovePushing = func() {
		if controller.data.GetCurrentDay().Current == nil {
			return
		}

		overlay, err := input.ConstructInputTree(
			map[input.Keyspec]action.Action{
				"n": action.NewSimple(func() string { return "move to now" }, func() { panic("TODO") }),
				"j": action.NewSimple(func() string { return "move down" }, func() {
					err := controller.data.GetCurrentDay().MoveEventsPushingBy(
						controller.data.GetCurrentDay().Current,
						int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
						int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
					)
					if err != nil {
						panic(err)
					} else {
						ensureEventsPaneTimestampVisible(controller.data.GetCurrentDay().Current.End)
					}
				}),
				"k": action.NewSimple(func() string { return "move up" }, func() {
					err := controller.data.GetCurrentDay().MoveEventsPushingBy(
						controller.data.GetCurrentDay().Current,
						-int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
						int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
					)
					if err != nil {
						panic(err)
					} else {
						ensureEventsPaneTimestampVisible(controller.data.GetCurrentDay().Current.Start)
					}
				}),
				"M":     action.NewSimple(func() string { return "exit move mode" }, func() { dayEventsPane.PopModalOverlay(); controller.data.EventEditMode = edit.EventEditModeNormal }),
				"<esc>": action.NewSimple(func() string { return "exit move mode" }, func() { dayEventsPane.PopModalOverlay(); controller.data.EventEditMode = edit.EventEditModeNormal }),
				// TODO(ja-he): mode switching
			},
		)
		if err != nil {
			panic(err.Error())
		}
		dayEventsPane.ApplyModalOverlay(input.CapturingOverlayWrap(overlay))
		controller.data.EventEditMode = edit.EventEditModeMove
	}
	ensureBacklogTaskVisible = func(t *model.Task) {
		viewportLB, viewportUB := tasksPane.GetTaskVisibilityBounds()
		taskLB, taskUB := tasksPane.GetTaskUIYBounds(t)
		if taskLB < viewportLB {
			backlogViewParams.SetScrollOffset(backlogViewParams.GetScrollOffset() - (viewportLB - taskLB))
		} else if taskUB > viewportUB {
			backlogViewParams.SetScrollOffset(backlogViewParams.GetScrollOffset() - (viewportUB - taskUB))
		}
	}
	scrollBacklogTop = func() {
		backlogViewParams.SetScrollOffset(0)
	}
	scrollBacklogBottom = func() {
		backlogViewParams.SetScrollOffset(getBacklogBottomScrollOffset())
	}
	getBacklogBottomScrollOffset = func() int {
		if len(backlog.Tasks) == 0 {
			return 0
		}
		lastTask := backlog.Tasks[len(backlog.Tasks)-1]
		currentScrollOffset := backlogViewParams.GetScrollOffset()
		_, tUB := tasksPane.GetTaskUIYBounds(lastTask)
		_, vUB := tasksPane.GetTaskVisibilityBounds()
		desiredScrollDelta := vUB - tUB - 1
		return currentScrollOffset - desiredScrollDelta
	}
	backlogSetCurrentToTopmost = func() {
		if len(backlog.Tasks) == 0 {
			return
		}
		currentTask = backlog.Tasks[0]
		scrollBacklogTop()
	}
	backlogSetCurrentToBottommost = func() {
		if len(backlog.Tasks) == 0 {
			return
		}
		currentTask = backlog.Tasks[len(backlog.Tasks)-1]
		scrollBacklogBottom()
	}
	offsetCurrentTask = func(tl []*model.Task, setToNext bool) bool {
		if len(tl) == 0 {
			return false
		}

		for i, t := range tl {
			if currentTask == t {
				if setToNext {
					if i < len(tl)-1 {
						setCurrentTask(tl[i+1])
					} else {
						log.Debug().Msg("not allowing selecting next task, as at last task in scope")
					}
				} else {
					if i > 0 {
						setCurrentTask(tl[i-1])
					} else {
						log.Debug().Msg("not allowing selecting previous task, as at first task in scope")
					}
				}
				return true
			}
			if offsetCurrentTask(t.Subtasks, setToNext) {
				return true
			}
		}

		return false
	}

	dayViewEventsPaneInputTree.Root.Children[input.Key{Key: tcell.KeyRune, Ch: 'm'}] = &input.Node{Action: action.NewSimple(func() string { return "enter event move mode" }, func() {
		if controller.data.GetCurrentDay().Current == nil {
			return
		}

		eventMoveOverlay, err := input.ConstructInputTree(
			map[input.Keyspec]action.Action{
				"n": action.NewSimple(func() string { return "move to now" }, func() {
					current := controller.data.GetCurrentDay().Current
					newStart := *model.NewTimestampFromGotime(time.Now())
					controller.data.GetCurrentDay().MoveSingleEventTo(current, newStart)
					ensureEventsPaneTimestampVisible(current.Start)
					ensureEventsPaneTimestampVisible(current.End)
				}),
				"j": action.NewSimple(func() string { return "move down" }, func() {
					current := controller.data.GetCurrentDay().Current
					controller.data.GetCurrentDay().MoveSingleEventBy(
						current,
						int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
						int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
					)
					ensureEventsPaneTimestampVisible(current.End)
				}),
				"k": action.NewSimple(func() string { return "move up" }, func() {
					current := controller.data.GetCurrentDay().Current
					controller.data.GetCurrentDay().MoveSingleEventBy(
						current,
						-int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
						int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
					)
					ensureEventsPaneTimestampVisible(current.Start)
				}),
				"h": action.NewSimple(func() string { return "move to previous day" }, func() {
					event := controller.data.GetCurrentDay().Current
					controller.data.GetCurrentDay().RemoveEvent(event)
					controller.goToPreviousDay()
					controller.data.GetCurrentDay().AddEvent(event)
				}),
				"l": action.NewSimple(func() string { return "move to next day" }, func() {
					event := controller.data.GetCurrentDay().Current
					controller.data.GetCurrentDay().RemoveEvent(event)
					controller.goToNextDay()
					controller.data.GetCurrentDay().AddEvent(event)
				}),
				"m":     action.NewSimple(func() string { return "exit move mode" }, func() { dayEventsPane.PopModalOverlay(); controller.data.EventEditMode = edit.EventEditModeNormal }),
				"<esc>": action.NewSimple(func() string { return "exit move mode" }, func() { dayEventsPane.PopModalOverlay(); controller.data.EventEditMode = edit.EventEditModeNormal }),
			},
		)
		if err != nil {
			panic(err.Error())
		}
		dayEventsPane.ApplyModalOverlay(input.CapturingOverlayWrap(eventMoveOverlay))
		controller.data.EventEditMode = edit.EventEditModeMove
	})}
	dayViewEventsPaneInputTree.Root.Children[input.Key{Key: tcell.KeyRune, Ch: 'r'}] = &input.Node{Action: action.NewSimple(func() string { return "enter event resize mode" }, func() {
		if controller.data.GetCurrentDay().Current == nil {
			return
		}

		eventResizeOverlay, err := input.ConstructInputTree(
			map[input.Keyspec]action.Action{
				"n": action.NewSimple(func() string { return "resize to now" }, func() {
					current := controller.data.GetCurrentDay().Current
					newEnd := *model.NewTimestampFromGotime(time.Now())
					controller.data.GetCurrentDay().ResizeTo(current, newEnd)
					ensureEventsPaneTimestampVisible(newEnd)
				}),
				"j": action.NewSimple(func() string { return "increase size (lengthen)" }, func() {
					var err error
					current := controller.data.GetCurrentDay().Current
					err = controller.data.GetCurrentDay().ResizeBy(
						current,
						int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
					)
					if err != nil {
						log.Warn().Err(err).Msg("unable to resize")
					}
					err = controller.data.GetCurrentDay().SnapEnd(
						current,
						int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
					)
					if err != nil {
						log.Warn().Err(err).Msg("unable to snap")
					}
					ensureEventsPaneTimestampVisible(current.End)
				}),
				"k": action.NewSimple(func() string { return "decrease size (shorten)" }, func() {
					current := controller.data.GetCurrentDay().Current
					controller.data.GetCurrentDay().ResizeBy(
						current,
						-int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
					)
					controller.data.GetCurrentDay().SnapEnd(
						current,
						int(controller.data.MainTimelineViewParams.DurationOfHeight(1)/time.Minute),
					)
					ensureEventsPaneTimestampVisible(current.End)
				}),
				"r":     action.NewSimple(func() string { return "exit resize mode" }, func() { dayEventsPane.PopModalOverlay(); controller.data.EventEditMode = edit.EventEditModeNormal }),
				"<esc>": action.NewSimple(func() string { return "exit resize mode" }, func() { dayEventsPane.PopModalOverlay(); controller.data.EventEditMode = edit.EventEditModeNormal }),
			},
		)
		if err != nil {
			stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for event pane's resize mode")
		}
		dayEventsPane.ApplyModalOverlay(input.CapturingOverlayWrap(eventResizeOverlay))
		controller.data.EventEditMode = edit.EventEditModeResize
	})}

	var helpContentRegister func()
	rootPaneInputTree, err := input.ConstructInputTree(
		map[input.Keyspec]action.Action{
			"q": action.NewSimple(func() string { return "exit program (unsaved progress is lost)" }, func() { controller.controllerEvents <- ControllerEventExit }),
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
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for root pane")
	}
	var ensureDayViewMainPaneFocusIsOnVisible func()
	updateMainPaneRightFlexWidth := func() {
		rightFlexWidth = 0
		if tasksPane.IsVisible() {
			rightFlexWidth += tasksWidth
		}
		if toolsPane.IsVisible() {
			rightFlexWidth += toolsWidth
		}
	}
	toggleToolsPane := func() {
		toolsVisible = !toolsVisible
		if !toolsVisible {
			ensureDayViewMainPaneFocusIsOnVisible()
		}
		updateMainPaneRightFlexWidth()
	}
	toggleTasksPane := func() {
		tasksVisible = !tasksVisible
		if !tasksVisible {
			ensureDayViewMainPaneFocusIsOnVisible()
		}
		updateMainPaneRightFlexWidth()
	}

	var dayViewFocusNext, dayViewFocusPrev func()
	dayViewInputTree, err := input.ConstructInputTree(
		map[input.Keyspec]action.Action{
			"W":      action.NewSimple(func() string { return "update weather" }, controller.updateWeather),
			"t":      action.NewSimple(func() string { return "toggle tools pane" }, toggleToolsPane),
			"T":      action.NewSimple(func() string { return "toggle tasks pane" }, toggleTasksPane),
			"<c-w>h": action.NewSimple(func() string { return "switch to previous pane" }, func() { dayViewFocusPrev() }),
			"<c-w>l": action.NewSimple(func() string { return "switch to next pane" }, func() { dayViewFocusNext() }),
		},
	)
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for day view pane")
	}

	dayViewScrollablePaneInputTree, err := input.ConstructInputTree(scrollableZoomableInputMap)
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for day view scrollable pane")
	}
	dayViewScrollablePane := panes.NewWrapperPane(
		[]ui.Pane{
			dayEventsPane,
			panes.NewTimelinePane(
				ui.NewConstrainedRenderer(renderer, dayViewTimelineDimensions),
				dayViewTimelineDimensions,
				stylesheet,
				controller.data.GetCurrentSuntimes,
				func() *model.Timestamp {
					if controller.data.CurrentDate.Is(time.Now()) {
						return model.NewTimestampFromGotime(time.Now())
					} else {
						return nil
					}
				},
				&controller.data.MainTimelineViewParams,
			),
			panes.NewWeatherPane(
				ui.NewConstrainedRenderer(renderer, weatherDimensions),
				weatherDimensions,
				stylesheet,
				&controller.data.CurrentDate,
				&controller.data.Weather,
				&controller.data.MainTimelineViewParams,
			),
		},
		[]ui.Pane{
			dayEventsPane,
		},
		processors.NewModalInputProcessor(dayViewScrollablePaneInputTree),
	)
	multidayViewEventsWrapperInputMap := scrollableZoomableInputMap
	multidayViewEventsWrapperInputMap["h"] = action.NewSimple(func() string { return "go to previous day" }, controller.goToPreviousDay)
	multidayViewEventsWrapperInputMap["l"] = action.NewSimple(func() string { return "go to next day" }, controller.goToNextDay)
	weekViewEventsWrapperInputTree, err := input.ConstructInputTree(multidayViewEventsWrapperInputMap)
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for multi-day wrapper pane")
	}
	weekViewEventsWrapper := panes.NewWrapperPane(
		weekViewEventsPanes,
		[]ui.Pane{},
		processors.NewModalInputProcessor(weekViewEventsWrapperInputTree),
	)
	monthViewEventsWrapperInputTree, err := input.ConstructInputTree(multidayViewEventsWrapperInputMap)
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for month view wrapper pane")
	}
	monthViewEventsWrapper := panes.NewWrapperPane(
		monthViewEventsPanes,
		[]ui.Pane{},
		processors.NewModalInputProcessor(monthViewEventsWrapperInputTree),
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
	)
	ensureDayViewMainPaneFocusIsOnVisible = dayViewMainPane.EnsureFocusIsOnVisible
	weekViewMainPaneInputTree, err := input.ConstructInputTree(map[input.Keyspec]action.Action{})
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for week view main pane")
	}
	weekViewMainPane := panes.NewWrapperPane(
		[]ui.Pane{
			statusPane,
			panes.NewTimelinePane(
				ui.NewConstrainedRenderer(renderer, weekViewTimelineDimensions),
				weekViewTimelineDimensions,
				stylesheet,
				func() *model.SunTimes { return nil },
				func() *model.Timestamp { return nil },
				&controller.data.MainTimelineViewParams,
			),
			weekViewEventsWrapper,
		},
		[]ui.Pane{
			weekViewEventsWrapper,
		},
		processors.NewModalInputProcessor(weekViewMainPaneInputTree),
	)
	monthViewMainPaneInputTree, err := input.ConstructInputTree(scrollableZoomableInputMap)
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for month view main pane")
	}
	monthViewMainPane := panes.NewWrapperPane(
		[]ui.Pane{
			statusPane,
			panes.NewTimelinePane(
				ui.NewConstrainedRenderer(renderer, monthViewTimelineDimensions),
				monthViewTimelineDimensions,
				stylesheet,
				func() *model.SunTimes { return nil },
				func() *model.Timestamp { return nil },
				&controller.data.MainTimelineViewParams,
			),
			monthViewEventsWrapper,
		},
		[]ui.Pane{
			monthViewEventsWrapper,
		},
		processors.NewModalInputProcessor(monthViewMainPaneInputTree),
	)
	dayViewFocusNext = dayViewMainPane.FocusNext
	dayViewFocusPrev = dayViewMainPane.FocusPrev

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
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for summary pane")
	}

	var editorStartInsertMode func()
	var editorLeaveInsertMode func()
	editorInsertMode, err := processors.NewTextInputProcessor( // TODO rename
		map[input.Keyspec]action.Action{
			"<ESC>":   action.NewSimple(func() string { return "exit insert mode" }, func() { editorLeaveInsertMode() }),
			"<c-a>":   action.NewSimple(func() string { return "move cursor to beginning" }, controller.data.EventEditor.MoveCursorToBeginning),
			"<del>":   action.NewSimple(func() string { return "delete character" }, controller.data.EventEditor.DeleteRune),
			"<c-d>":   action.NewSimple(func() string { return "delete character" }, controller.data.EventEditor.DeleteRune),
			"<c-bs>":  action.NewSimple(func() string { return "backspace" }, controller.data.EventEditor.BackspaceRune),
			"<bs>":    action.NewSimple(func() string { return "backspace" }, controller.data.EventEditor.BackspaceRune),
			"<c-e>":   action.NewSimple(func() string { return "move cursor to end" }, controller.data.EventEditor.MoveCursorToEnd),
			"<c-u>":   action.NewSimple(func() string { return "backspace to beginning" }, controller.data.EventEditor.BackspaceToBeginning),
			"<left>":  action.NewSimple(func() string { return "move cursor left" }, controller.data.EventEditor.MoveCursorLeft),
			"<right>": action.NewSimple(func() string { return "move cursor right" }, controller.data.EventEditor.MoveCursorRight),
		},
		controller.data.EventEditor.AddRune,
	)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not construct editor insert mode processor")
	}
	editorNormalModeTree, err := input.ConstructInputTree(
		map[input.Keyspec]action.Action{
			"<esc>": action.NewSimple(func() string { return "abord edit, discard changes" }, controller.abortEdit),
			"<cr>":  action.NewSimple(func() string { return "finish edit, commit changes" }, controller.endEdit),
			"i":     action.NewSimple(func() string { return "enter insert mode" }, func() { editorStartInsertMode() }),
			"a": action.NewSimple(func() string { return "enter insert mode (after character)" }, func() {
				controller.data.EventEditor.MoveCursorRightA()
				editorStartInsertMode()
			}),
			"A": action.NewSimple(func() string { return "enter insert mode (at end)" }, func() {
				controller.data.EventEditor.MoveCursorPastEnd()
				editorStartInsertMode()
			}),
			"0": action.NewSimple(func() string { return "move cursor to beginning" }, controller.data.EventEditor.MoveCursorToBeginning),
			"$": action.NewSimple(func() string { return "move cursor to end" }, controller.data.EventEditor.MoveCursorToEnd),
			"h": action.NewSimple(func() string { return "move cursor left" }, controller.data.EventEditor.MoveCursorLeft),
			"l": action.NewSimple(func() string { return "move cursor right" }, controller.data.EventEditor.MoveCursorRight),
			"w": action.NewSimple(func() string { return "move cursor to next word beginning" }, controller.data.EventEditor.MoveCursorNextWordBeginning),
			"b": action.NewSimple(func() string { return "move cursor to previous word beginning" }, controller.data.EventEditor.MoveCursorPrevWordBeginning),
			"e": action.NewSimple(func() string { return "move cursor to next word end" }, controller.data.EventEditor.MoveCursorNextWordEnd),
			"x": action.NewSimple(func() string { return "delete character" }, controller.data.EventEditor.DeleteRune),
			"C": action.NewSimple(func() string { return "delete to end" }, func() {
				controller.data.EventEditor.DeleteToEnd()
				editorStartInsertMode()
			}),
			"dd": action.NewSimple(func() string { return "clear text content" }, func() { controller.data.EventEditor.Clear() }),
			"cc": action.NewSimple(func() string { return "clear text content, enter insert" }, func() {
				controller.data.EventEditor.Clear()
				editorStartInsertMode()
			}),
		},
	)
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for editor pane's normal mode")
	}
	helpPaneInputTree, err := input.ConstructInputTree(
		map[input.Keyspec]action.Action{
			"?": action.NewSimple(func() string { return "close help" }, func() {
				controller.data.ShowHelp = false
			}),
		},
	)
	if err != nil {
		stderrLogger.Fatal().Err(err).Msg("failed to construct input tree for help pane")
	}
	helpPane := panes.NewHelpPane(
		ui.NewConstrainedRenderer(renderer, helpDimensions),
		helpDimensions,
		stylesheet,
		func() bool { return controller.data.ShowHelp },
		processors.NewModalInputProcessor(helpPaneInputTree),
	)
	editorPane := panes.NewEventEditorPane(
		ui.NewConstrainedRenderer(renderer, editorDimensions),
		cursorWrangler,
		editorDimensions,
		stylesheet,
		func() bool { return controller.data.EventEditor.Active },
		func() string { return controller.data.EventEditor.TmpEventInfo.Name },
		controller.data.EventEditor.GetMode,
		func() int { return controller.data.EventEditor.CursorPos },
		processors.NewModalInputProcessor(editorNormalModeTree),
	)
	pushEditorAsRootSubpane = func() { controller.rootPane.PushSubpane(editorPane) }
	editorStartInsertMode = func() {
		editorPane.ApplyModalOverlay(editorInsertMode)
		controller.data.EventEditor.SetMode(input.TextEditModeInsert)
	}
	editorLeaveInsertMode = func() {
		editorPane.PopModalOverlay()
		controller.data.EventEditor.SetMode(input.TextEditModeNormal)
	}

	rootPane := panes.NewRootPane(
		renderer,
		cursorWrangler,
		screenDimensions,

		dayViewMainPane,
		weekViewMainPane,
		monthViewMainPane,

		panes.NewSummaryPane(
			ui.NewConstrainedRenderer(renderer, screenDimensions),
			screenDimensions,
			stylesheet,
			func() bool { return controller.data.ShowSummary },
			func() string {
				dateString := ""
				switch controller.data.ActiveView() {
				case ui.ViewDay:
					dateString = controller.data.CurrentDate.ToString()
				case ui.ViewWeek:
					start, end := controller.data.CurrentDate.Week()
					dateString = fmt.Sprintf("week %s..%s", start.ToString(), end.ToString())
				case ui.ViewMonth:
					dateString = fmt.Sprintf("%s %d", controller.data.CurrentDate.ToGotime().Month().String(), controller.data.CurrentDate.Year)
				}
				return fmt.Sprintf("SUMMARY (%s)", dateString)
			},
			func() []*model.Day {
				switch controller.data.ActiveView() {
				case ui.ViewDay:
					result := make([]*model.Day, 1)
					result[0] = controller.data.Days.GetDay(controller.data.CurrentDate)
					return result
				case ui.ViewWeek:
					result := make([]*model.Day, 7)
					start, end := controller.data.CurrentDate.Week()
					for current, i := start, 0; current != end.Next(); current = current.Next() {
						result[i] = controller.data.Days.GetDay(current)
						i++
					}
					return result
				case ui.ViewMonth:
					start, end := controller.data.CurrentDate.MonthBounds()
					result := make([]*model.Day, end.Day)
					for current, i := start, 0; current != end.Next(); current = current.Next() {
						result[i] = controller.data.Days.GetDay(current)
						i++
					}
					return result
				default:
					panic("unknown view in summary data gathering")
				}
			},
			&categoryStyling,
			processors.NewModalInputProcessor(summaryPaneInputTree),
		),
		panes.NewLogPane(
			ui.NewConstrainedRenderer(renderer, screenDimensions),
			screenDimensions,
			stylesheet,
			func() bool { return controller.data.ShowLog },
			func() string { return "LOG" },
			&potatolog.GlobalMemoryLogReaderWriter,
		),
		helpPane,

		panes.NewPerfPane(
			ui.NewConstrainedRenderer(renderer, func() (x, y, w, h int) { return 2, 2, 50, 2 }),
			func() (x, y, w, h int) { return 2, 2, 50, 2 },
			func() bool { return controller.data.ShowDebug },
			&controller.data.RenderTimes,
			&controller.data.EventProcessingTimes,
		),
		processors.NewModalInputProcessor(rootPaneInputTree),
		dayViewMainPane,
	)
	controller.data.ActiveView = rootPane.GetView
	rootPaneInputTree.Root.Children[input.Key{Key: tcell.KeyESC}] = &input.Node{
		Action: action.NewSimple(func() string { return "view up" }, func() {
			rootPane.ViewUp()
			controller.loadDaysForView(controller.data.ActiveView())
		}),
	}
	rootPaneInputTree.Root.Children[input.Key{Key: tcell.KeyRune, Ch: 'i'}] = &input.Node{
		Action: action.NewSimple(func() string { return "view down" }, func() {
			rootPane.ViewDown()
			controller.loadDaysForView(controller.data.ActiveView())
		}),
	}

	helpContentRegister = func() {
		helpPane.Content = rootPane.GetHelp()
	}

	controller.data.EventEditor.SetMode(input.TextEditModeNormal)
	controller.data.EventEditMode = edit.EventEditModeNormal

	coordinatesProvided := (envData.Latitude != "" && envData.Longitude != "")
	owmApiKeyProvided := (envData.OwmApiKey != "")

	// intialize weather handler if geographic location and api key provided
	if coordinatesProvided && owmApiKeyProvided {
		controller.data.Weather = *weather.NewHandler(envData.Latitude, envData.Longitude, envData.OwmApiKey)
	} else {
		if !owmApiKeyProvided {
			log.Error().Msg("no OWM API key provided -> no weather data")
		}
		if !coordinatesProvided {
			log.Error().Msg("no lat-/longitude provided -> no weather data")
		}
	}

	// process latitude longitude
	// TODO
	var suntimes model.SunTimes
	if !coordinatesProvided {
		log.Error().Msg("could not fetch lat-&longitude -> no sunrise/-set times known")
	} else {
		latF, parseErrLat := strconv.ParseFloat(envData.Latitude, 64)
		lonF, parseErrLon := strconv.ParseFloat(envData.Longitude, 64)
		if parseErrLon != nil || parseErrLat != nil {
			log.Error().
				Interface("lon-parse-error", parseErrLon).
				Interface("lat-parse-error", parseErrLat).
				Msg("could not parse longitude/latitude")
		} else {
			suntimes = date.GetSunTimes(latF, lonF)
		}
	}

	controller.tmpStatusYOffsetGetter = func() int { _, y, _, _ := statusDimensions(); return y }
	controller.data.EnvData = envData
	controller.screenEvents = renderer.GetEventPollable()

	controller.fhMutex.Lock()
	defer controller.fhMutex.Unlock()
	controller.FileHandlers = make(map[model.Date]*storage.FileHandler)
	controller.FileHandlers[date] = storage.NewFileHandler(controller.data.EnvData.BaseDirPath + "/days/" + date.ToString())

	controller.data.CurrentDate = date
	if controller.FileHandlers[date] == nil {
		controller.data.Days.AddDay(date, &model.Day{}, &suntimes)
	} else {
		controller.data.Days.AddDay(date, controller.FileHandlers[date].Read(controller.data.Categories), &suntimes)
	}

	controller.rootPane = rootPane
	controller.data.CurrentCategory.Name = "default"

	controller.loadDaysForView(controller.data.ActiveView())

	controller.timestampGuesser = func(cursorX, cursorY int) model.Timestamp {
		_, yOffset, _, _ := dayViewEventsPaneDimensions()
		return controller.data.MainTimelineViewParams.TimeAtY(yOffset + cursorY)
	}

	controller.initializedScreen = renderer
	controller.syncer = renderer

	controller.data.MouseEditState = edit.MouseEditStateNone

	return &controller
}

func (t *Controller) ScrollUp(by int) {
	eventviewTopRow := 0
	if t.data.MainTimelineViewParams.ScrollOffset-by >= eventviewTopRow {
		t.data.MainTimelineViewParams.ScrollOffset -= by
	} else {
		t.ScrollTop()
	}
}

func (t *Controller) ScrollDown(by int) {
	eventviewBottomRow := t.tmpStatusYOffsetGetter()
	if t.data.MainTimelineViewParams.ScrollOffset+by+eventviewBottomRow <= (24 * t.data.MainTimelineViewParams.NRowsPerHour) {
		t.data.MainTimelineViewParams.ScrollOffset += by
	} else {
		t.ScrollBottom()
	}
}

func (t *Controller) ScrollTop() {
	t.data.MainTimelineViewParams.ScrollOffset = 0
}

func (t *Controller) ScrollBottom() {
	eventviewBottomRow := t.tmpStatusYOffsetGetter()
	t.data.MainTimelineViewParams.ScrollOffset = 24*t.data.MainTimelineViewParams.NRowsPerHour - eventviewBottomRow
}

func (t *Controller) abortEdit() {
	t.data.MouseEditState = edit.MouseEditStateNone
	t.data.MouseEditedEvent = nil
	t.data.EventEditor.Active = false
	t.rootPane.PopSubpane()
}

func (t *Controller) endEdit() {
	t.data.MouseEditState = edit.MouseEditStateNone
	t.data.MouseEditedEvent = nil
	if t.data.EventEditor.Active {
		t.data.EventEditor.Active = false
		tmp := t.data.EventEditor.TmpEventInfo
		t.data.EventEditor.Original.Name = tmp.Name
	}
	t.data.GetCurrentDay().UpdateEventOrder()
	t.rootPane.PopSubpane()
}

func (t *Controller) startMouseMove(eventsInfo *ui.EventsPanePositionInfo) {
	t.data.MouseEditState = edit.MouseEditStateMoving
	t.data.MouseEditedEvent = eventsInfo.Event
	t.data.CurrentMoveStartingOffsetMinutes = eventsInfo.Event.Start.DurationInMinutesUntil(eventsInfo.Time)
}

func (t *Controller) startMouseResize(eventsInfo *ui.EventsPanePositionInfo) {
	t.data.MouseEditState = edit.MouseEditStateResizing
	t.data.MouseEditedEvent = eventsInfo.Event
}

func (t *Controller) startMouseEventCreation(info *ui.EventsPanePositionInfo) {
	// find out cursor time
	start := info.Time

	log.Debug().Str("position-time", info.Time.ToString()).Msg("creation called")

	// create event at time with cat etc.
	e := model.Event{}
	e.Cat = t.data.CurrentCategory
	e.Name = ""
	e.Start = start
	e.End = start.OffsetMinutes(+10)

	err := t.data.GetCurrentDay().AddEvent(&e)
	if err != nil {
		log.Error().Err(err).Interface("event", e).Msg("error occurred adding event")
	} else {
		t.data.MouseEditedEvent = &e
		t.data.MouseEditState = edit.MouseEditStateResizing
	}
}

func (t *Controller) goToDay(newDate model.Date) {
	log.Debug().Str("new-date", newDate.ToString()).Msg("going to new date")

	t.data.CurrentDate = newDate
	t.loadDaysForView(t.data.ActiveView())
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
			latF, parseErrLat := strconv.ParseFloat(t.data.EnvData.Latitude, 64)
			lonF, parseErrLon := strconv.ParseFloat(t.data.EnvData.Longitude, 64)
			if parseErrLon != nil || parseErrLat != nil {
				log.Error().
					Interface("lon-parse-error", parseErrLon).
					Interface("lat-parse-error", parseErrLat).
					Msg("could not parse longitude/latitude")
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

func (t *Controller) writeModel() {
	go func() {
		t.fhMutex.RLock()
		t.FileHandlers[t.data.CurrentDate].Write(t.data.GetCurrentDay())
		t.fhMutex.RUnlock()
	}()
}

func (t *Controller) updateCursorPos(x, y int) {
	t.data.CursorPos.X, t.data.CursorPos.Y = x, y
}

func (t *Controller) handleMouseNoneEditEvent(e *tcell.EventMouse) {
	t.data.MouseMode = true

	// get new position
	x, y := e.Position()
	t.updateCursorPos(x, y)

	positionInfo := t.rootPane.GetPositionInfo(x, y)
	if positionInfo == nil {
		return
	}

	buttons := e.Buttons()

	switch positionInfo := positionInfo.(type) {
	case *ui.StatusPanePositionInfo:

	case *ui.WeatherPanePositionInfo:
		switch buttons {
		case tcell.WheelUp:
			t.ScrollUp(1)
		case tcell.WheelDown:
			t.ScrollDown(1)
		}

	case *ui.TimelinePanePositionInfo:
		switch buttons {
		case tcell.WheelUp:
			t.ScrollUp(1)
		case tcell.WheelDown:
			t.ScrollDown(1)
		}

	case *ui.EventsPanePositionInfo:
		eventsInfo := positionInfo

		// if button clicked, handle
		switch buttons {
		case tcell.Button3:
			t.data.GetCurrentDay().RemoveEvent(eventsInfo.Event)
		case tcell.Button2:
			event := eventsInfo.Event
			if event != nil && eventsInfo.Time.IsAfter(event.Start) {
				t.data.GetCurrentDay().SplitEvent(event, eventsInfo.Time)
			}

		case tcell.Button1:
			// we've clicked while not editing
			// now we need to check where the cursor is and either start event
			// creation, resizing or moving
			switch eventsInfo.EventBoxPart {
			case ui.EventBoxNowhere:
				t.startMouseEventCreation(eventsInfo)
			case ui.EventBoxBottomRight:
				t.startMouseResize(eventsInfo)
			case ui.EventBoxInterior:
				t.startMouseMove(eventsInfo)
			case ui.EventBoxTopEdge:
				t.data.EventEditor.Activate(eventsInfo.Event)
			}

		case tcell.WheelUp:
			t.ScrollUp(1)

		case tcell.WheelDown:
			t.ScrollDown(1)

		}

	case *ui.ToolsPanePositionInfo:
		toolsInfo := positionInfo
		switch buttons {
		case tcell.Button1:
			cat := toolsInfo.Category
			if cat != nil {
				t.data.CurrentCategory = *cat
			}
		}

	}
}

func (t *Controller) handleMouseResizeEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		x, y := e.Position()

		buttons := e.Buttons()

		switch buttons {
		case tcell.Button1:
			cursorTime := t.timestampGuesser(x, y)
			visualCursorTime := cursorTime.OffsetMinutes(int(t.data.MainTimelineViewParams.DurationOfHeight(1) / time.Minute))
			event := t.data.MouseEditedEvent

			var err error
			err = t.data.GetCurrentDay().ResizeTo(event, visualCursorTime)
			if err != nil {
				log.Warn().Err(err).Msg("unable to resize")
			}

		case tcell.ButtonNone:
			t.endEdit()
		}

		t.updateCursorPos(x, y)
	}
}

func (t *Controller) handleMouseMoveEditEvent(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventMouse:
		x, y := e.Position()

		buttons := e.Buttons()

		switch buttons {
		case tcell.Button1:
			cursorTime := t.timestampGuesser(x, y)
			t.data.GetCurrentDay().MoveSingleEventTo(t.data.MouseEditedEvent, cursorTime.OffsetMinutes(-t.data.CurrentMoveStartingOffsetMinutes))
		case tcell.ButtonNone:
			t.endEdit()
		}

		t.updateCursorPos(x, y)
	}
}

func (c *Controller) updateWeather() {
	go func() {
		err := c.data.Weather.Update()
		if err != nil {
			log.Error().Err(err).Msg("could not update weather data")
		} else {
			log.Debug().Msg("successfully retrieved weather data")
		}
		c.controllerEvents <- ControllerEventRender
	}()
}

type ControllerEvent int

const (
	ControllerEventExit ControllerEvent = iota
	ControllerEventRender
	ControllerEventTaskEditorExit
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

// Run
func (t *Controller) Run() {
	log.Info().Msg("dayplan TUI started")

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
					t.data.RenderTimes.Add(uint64(end.Sub(start).Microseconds()))

				case ControllerEventTaskEditorExit:
					if t.data.TaskEditor == nil {
						log.Warn().Msgf("got task editor exit event, but no task editor active; likely logic error")
					} else {
						t.data.TaskEditor = nil
						t.rootPane.PopSubpane()
						log.Debug().Msgf("removed (presumed) task-editor subpane from root")
						go func() { t.controllerEvents <- ControllerEventRender }()
					}

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

			{
				switch e := ev.(type) {
				case *tcell.EventKey:
					t.data.MouseMode = false
					t.data.MouseEditState = edit.MouseEditStateNone

					key := input.KeyFromTcellEvent(e)
					inputApplied := t.rootPane.ProcessInput(key)
					if !inputApplied {
						log.Warn().Str("key", key.ToDebugString()).Msg("could not apply key input")
					}

				case *tcell.EventMouse:
					t.data.MouseMode = true

					// get new position
					x, y := e.Position()
					t.updateCursorPos(x, y)

					switch t.data.MouseEditState {
					case edit.MouseEditStateNone:
						t.handleMouseNoneEditEvent(e)
					case edit.MouseEditStateResizing:
						t.handleMouseResizeEditEvent(ev)
					case edit.MouseEditStateMoving:
						t.handleMouseMoveEditEvent(ev)
					}

				case *tcell.EventResize:
					t.syncer.NeedsSync()

				}
			}

			end := time.Now()
			t.data.EventProcessingTimes.Add(uint64(end.Sub(start).Microseconds()))

			t.controllerEvents <- ControllerEventRender
		}
	}()

	wg.Wait()
}
