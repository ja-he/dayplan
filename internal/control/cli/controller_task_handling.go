package cli

import (
	"fmt"
	"time"

	"github.com/ja-he/dayplan/internal/model"
)

func (c *Controller) goToPrevTask() {
	defer func() { c.ensureBacklogTaskVisible(c.data.CurrentTask) }()
	if c.data.CurrentTask == nil {
		firstTaskID, err := c.backlogProvider.GetFirstChildTaskID(nil)
		if err != nil {
			c.log.Error().Err(err).Msg("Unable to get first task ID")
		}
		c.data.CurrentTask = firstTaskID
		return
	}

	locationContext, err := c.backlogProvider.GetLocationContext(*c.data.CurrentTask)
	if err != nil {
		c.log.Error().Err(err).Str("current-task", string(*c.data.CurrentTask)).Msg("Unable to get location context for current task.")
		return
	}
	if locationContext.Previous == nil {
		c.log.Warn().Msg("No previous task available to switch to.")
		return
	}
	c.data.CurrentTask = locationContext.Previous
}
func (c *Controller) goToNextTask() {
	defer func() { c.ensureBacklogTaskVisible(c.data.CurrentTask) }()
	if c.data.CurrentTask == nil {
		firstTaskID, err := c.backlogProvider.GetFirstChildTaskID(nil)
		if err != nil {
			c.log.Error().Err(err).Msg("Unable to get last task ID")
		}
		c.data.CurrentTask = firstTaskID
		return
	}

	locationContext, err := c.backlogProvider.GetLocationContext(*c.data.CurrentTask)
	if err != nil {
		c.log.Error().Err(err).Str("current-task", string(*c.data.CurrentTask)).Msg("Unable to get location context for current task.")
		return
	}
	if locationContext.Next == nil {
		c.log.Warn().Msg("No next task available to switch to.")
		return
	}
	c.data.CurrentTask = locationContext.Next
}
func (c *Controller) stepOutToParentTask() {
	if c.data.CurrentTask == nil {
		c.log.Debug().Msg("No current task to step out from.")
		return
	}
	locationContext, err := c.backlogProvider.GetLocationContext(*c.data.CurrentTask)
	if err != nil {
		c.log.Error().Err(err).Str("current-task", string(*c.data.CurrentTask)).Msg("Unable to get location context.")
		return
	}
	if locationContext.GetParentIDOrNil() == nil {
		c.log.Debug().Msg("could not find parent, so not changing current task")
		return
	}
	c.data.CurrentTask = locationContext.GetParentIDOrNil()
	c.ensureBacklogTaskVisible(c.data.CurrentTask)
}
func (c *Controller) insertTaskBeforeCurrent() {
	if c.data.CurrentTask == nil {
		c.log.Debug().Msgf("asked to add a task before to nil current task, adding as first")
		var newTask model.ReadableTask = &model.Task{
			Name:     "", // user should be hinted to change this quite quickly, i.e. via immediate editor activation
			Category: c.data.CurrentCategory,
		}
		newTaskID, err := c.backlogProvider.InsertFront(newTask, nil)
		if err != nil {
			c.log.Error().Err(err).Msg("Unable to insert new event before current.")
			return
		}
		c.log.Info().Msgf("Created new task at front '%s'", newTaskID)
		c.data.CurrentTask = &newTaskID
		c.createAndEnableTaskEditor(newTaskID)
		return
	}

	locationContext, err := c.backlogProvider.GetLocationContext(*c.data.CurrentTask)
	if err != nil {
		c.log.Error().Err(err).Str("current-task", string(*c.data.CurrentTask)).Msg("Unable to get location context.")
		return
	}
	newTaskCategory := c.data.CurrentCategory
	if locationContext.GetParentIDOrNil() != nil {
		cat, err := c.backlogProvider.GetCategory(*locationContext.GetParentIDOrNil())
		if err != nil {
			c.log.Error().Err(err).Msg("Unable to get category.")
			return
		}
		newTaskCategory = cat
	}
	var newTask model.ReadableTask = &model.Task{
		Name:     "", // user should be hinted to change this quite quickly, i.e. via immediate editor activation
		Category: newTaskCategory,
	}
	newTaskID, err := c.backlogProvider.InsertBefore(newTask, *c.data.CurrentTask)
	if err != nil {
		c.log.Error().Err(err).Msg("Unable to insert new event before current.")
		return
	}
	c.data.CurrentTask = &newTaskID
	c.createAndEnableTaskEditor(newTaskID)
}
func (c *Controller) insertTaskAfterCurrent() {
	if c.data.CurrentTask == nil {
		c.log.Debug().Msgf("asked to add a task after to nil current task, adding as first")
		var newTask model.ReadableTask = &model.Task{
			Name:     "", // user should be hinted to change this quite quickly, i.e. via immediate editor activation
			Category: c.data.CurrentCategory,
		}
		newTaskID, err := c.backlogProvider.InsertBack(newTask, nil)
		if err != nil {
			c.log.Error().Err(err).Msg("Unable to insert new event after current.")
			return
		}
		c.data.CurrentTask = &newTaskID
		c.createAndEnableTaskEditor(newTaskID)
		return
	}

	locationContext, err := c.backlogProvider.GetLocationContext(*c.data.CurrentTask)
	if err != nil {
		c.log.Error().Err(err).Str("current-task", string(*c.data.CurrentTask)).Msg("Unable to get location context.")
		return
	}
	newTaskCategory := c.data.CurrentCategory
	if locationContext.GetParentIDOrNil() != nil {
		cat, err := c.backlogProvider.GetCategory(*locationContext.GetParentIDOrNil())
		if err != nil {
			c.log.Error().Err(err).Msg("Unable to get category.")
			return
		}
		newTaskCategory = cat
	}
	var newTask model.ReadableTask = &model.Task{
		Name:     "", // user should be hinted to change this quite quickly, i.e. via immediate editor activation
		Category: newTaskCategory,
	}
	newTaskID, err := c.backlogProvider.InsertAfter(newTask, *c.data.CurrentTask)
	if err != nil {
		c.log.Error().Err(err).Msg("Unable to insert new event after current.")
		return
	}
	c.data.CurrentTask = &newTaskID
	c.createAndEnableTaskEditor(newTaskID)
}
func (c *Controller) addSubtaskOfCurrent() {
	if c.data.CurrentTask == nil {
		c.log.Warn().Msg("No current task.")
		return
	}

	currentTaskCategory, err := c.backlogProvider.GetCategory(*c.data.CurrentTask)
	if err != nil {
		c.log.Error().Err(err).Msg("Unable to get current task category.")
		return
	}
	var newTask model.ReadableTask = &model.Task{
		Name:     "", // user should be hinted to change this quite quickly, i.e. via immediate editor activation
		Category: currentTaskCategory,
	}
	newTaskID, err := c.backlogProvider.InsertBack(newTask, c.data.CurrentTask)
	if err != nil {
		c.log.Error().Err(err).Msg("Unable to insert new event as child (at back) of current.")
		return
	}
	c.data.CurrentTask = &newTaskID
	c.createAndEnableTaskEditor(newTaskID)
}
func (c *Controller) writeBacklog() {
	err := c.backlogProvider.Save()
	if err != nil {
		c.log.Error().Err(err).Msg("Unable to write backlog.")
		return
	}
	c.log.Info().Msg("Wrote backlog successfully.")
}

func (c *Controller) scrollBacklogToTopOrBottom(toTop bool) {
	var parentOfCurrent *model.TaskID = nil

	if c.data.CurrentTask != nil {
		ctx, err := c.backlogProvider.GetLocationContext(*c.data.CurrentTask)
		if err != nil {
			c.log.Error().Err(err).Msgf("Unable to get parent location context of current task '%s'.", *c.data.CurrentTask)
			return
		}
		parentOfCurrent = ctx.GetParentIDOrNil()
	}

	var t *model.TaskID
	var err error
	var firstOrLast string
	if toTop {
		t, err = c.backlogProvider.GetFirstChildTaskID(parentOfCurrent)
		firstOrLast = "first"
	} else {
		t, err = c.backlogProvider.GetLastChildTaskID(parentOfCurrent)
		firstOrLast = "last"
	}
	if err != nil {
		c.log.Error().Err(err).Msgf("Unable to get %s child-of-current or root task from backlog.", firstOrLast)
		return
	}
	c.data.CurrentTask = t
}
func (c *Controller) ScrollBacklogToTop() {
	defer func() { c.ensureBacklogTaskVisible(c.data.CurrentTask) }()
	c.scrollBacklogToTopOrBottom(true)
}
func (c *Controller) ScrollBacklogToBottom() {
	defer func() { c.ensureBacklogTaskVisible(c.data.CurrentTask) }()
	c.scrollBacklogToTopOrBottom(false)
}

func (c *Controller) popAndScheduleCurrentTask(when *time.Time) error {
	currentTaskID := c.data.CurrentTask
	if currentTaskID == nil {
		return fmt.Errorf("Have no current task")
	}
	taskToBeScheduled, ctx, err := c.backlogProvider.Remove(*currentTaskID)
	if err != nil {
		return fmt.Errorf("Could not find current task (%w)", err)
	}

	// update current task
	c.data.CurrentTask = func() *model.TaskID {
		switch {
		case ctx.Next != nil:
			return ctx.Next
		case ctx.Previous != nil:
			return ctx.Previous
		default:
			return ctx.GetParentIDOrNil()
		}
	}()
	// schedule task, if time for that was given
	if when != nil {
		namePrefix := ""
		c.backlogProvider.WithTasks(ctx.Parentage, func(parents []model.ReadableTask) {
			for _, parent := range parents {
				namePrefix = parent.GetName() + ": " + namePrefix
			}
		})
		newEvents := taskToBeScheduled.ToEvent(*when, namePrefix)
		for _, newEvent := range newEvents {
			_, err := c.eventsProvider.AddEvent(*newEvent)
			if err != nil {
				return fmt.Errorf("Unable to add event (%w)", err)
			}
		}
	}
	return nil
}

func (c *Controller) ScheduleCurrentTaskNow() {
	when := time.Now()
	if err := c.popAndScheduleCurrentTask(&when); err != nil {
		c.log.Error().Err(err).Msg("Unable to schedule current task now.")
	}
}
func (c *Controller) DeleteCurrentTask() {
	if err := c.popAndScheduleCurrentTask(nil); err != nil {
		// NOTE: really this is a poor use of a function called "schedule..." which actually is used to just pop and do nothing after
		c.log.Error().Err(err).Msg("Unable to 'schedule' current task.")
	}
}

func (c *Controller) StepIntoCurrentTaskSubtasks() {
	if c.data.CurrentTask == nil {
		return
	}
	err := c.backlogProvider.WithTask(*c.data.CurrentTask, func(t model.ReadableTask) {
		children := t.GetSubtasks()
		if len(children) > 0 {
			childID := children[0].GetID()
			c.data.CurrentTask = &childID
			c.ensureBacklogTaskVisible(c.data.CurrentTask)
		} else {
			c.log.Debug().Msg("current task has no subtasks, so remaining at it")
		}
	})
	if err != nil {
		c.log.Error().Err(err).Msgf("Error when executing task switching function on current task.")
	}
}

func (c *Controller) ensureBacklogTaskVisible(t *model.TaskID) {
	if t == nil {
		return
	}
	viewportLB, viewportUB := c.tasksPane.GetTaskVisibilityBounds()
	taskLB, taskUB := c.tasksPane.GetTaskUIYBounds(*t)
	c.log.Debug().Msgf("ensuring task %s visible, viewport:[%d:%d] task:[%d:%d]", *t, viewportLB, viewportUB, taskLB, taskUB)
	if taskLB < viewportLB {
		c.data.BacklogViewParams.SetScrollOffset(c.data.BacklogViewParams.GetScrollOffset() - (viewportLB - taskLB))
	} else if taskUB > viewportUB {
		c.data.BacklogViewParams.SetScrollOffset(c.data.BacklogViewParams.GetScrollOffset() - (viewportUB - taskUB))
	}
}
