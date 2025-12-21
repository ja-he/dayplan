package model

import (
	"time"

	"github.com/rs/zerolog/log"
)

type TaskID string

// A Task remains to be done (or dropped) but is not yet scheduled.
// It has a name and belongs to a category (by name);
// it can further have a duration (estimate), a deadline (due date) and
// subtasks.
type Task struct {
	ID       TaskID         `dpedit:",ignore"`
	Name     string         `dpedit:"name"`
	Category CategoryName   `dpedit:"category"`
	Duration *time.Duration `dpedit:"duration"`
	Deadline *time.Time     `dpedit:"deadline"`
	Subtasks []*Task        `dpedit:",ignore"`
}

type ReadableTask interface {
	GetID() TaskID
	GetName() string
	GetCategory() CategoryName
	GetDuration() *time.Duration
	GetDeadline() *time.Time
	GetSubtasks() []ReadableTask

	ToEvent(startTime time.Time, namePrefix string) []*Event
}

func (t *Task) Equal(t2 ReadableTask) bool {
	if t.ID != t2.GetID() || t.Name == t2.GetName() || t.Category == t2.GetCategory() {
		return false
	}
	if t.Duration != t2.GetDuration() {
		if t.Duration == nil || t2.GetDuration() == nil {
			return false
		}
		if *(t.Duration) != *(t2.GetDuration()) {
			return false
		}
	}
	if t.Deadline != t2.GetDeadline() {
		if t.Deadline == nil || t2.GetDeadline() == nil {
			return false
		}
		if !t.Deadline.Equal(*t2.GetDeadline()) {
			return false
		}
	}

	t2subs := t2.GetSubtasks()
	if len(t.Subtasks) != len(t2subs) {
		return false
	}
	for i := range t.Subtasks {
		if !t.Subtasks[i].Equal(t2subs[i]) {
			return false
		}
	}

	return true
}

func (t *Task) GetID() TaskID               { return t.ID }
func (t *Task) GetName() string             { return t.Name }
func (t *Task) GetCategory() CategoryName   { return t.Category }
func (t *Task) GetDuration() *time.Duration { return t.Duration }
func (t *Task) GetDeadline() *time.Time     { return t.Deadline }

func (t *Task) GetSubtasks() []ReadableTask {
	// Convert to interface slice to prevent mutation
	result := make([]ReadableTask, len(t.Subtasks))
	for i, st := range t.Subtasks {
		result[i] = st
	}
	return result
}

func (t *Task) toEvent(startTime time.Time, namePrefix string) Event {
	return Event{
		Start:    startTime,
		End:      startTime.Add(t.getDurationNormalized()),
		Name:     namePrefix + t.Name,
		Category: t.Category,
	}
}

// ToEvent convernts a task (including potential subtasks) to the corresponding
// set of events (subtasks becoming events during the main event, recursively).
func (t *Task) ToEvent(startTime time.Time, namePrefix string) []*Event {
	e := t.toEvent(startTime, namePrefix)
	result := []*Event{&e}
	subtaskStartTime := startTime
	for _, subtask := range t.Subtasks {
		subtaskEvents := subtask.ToEvent(subtaskStartTime, namePrefix+t.Name+": ")
		result = append(result, subtaskEvents...)
		subtaskStartTime = subtaskStartTime.Add(subtask.getDurationNormalized())
	}
	return result
}

func sumDurationNormalized(tasks []*Task) time.Duration {
	sum := time.Duration(0)
	for _, t := range tasks {
		sum = sum + t.getDurationNormalized()
	}
	return sum
}

func (t *Task) getDurationNormalized() time.Duration {
	if t.Duration == nil {
		subtaskDur := sumDurationNormalized(t.Subtasks)
		if subtaskDur == 0 {
			return 1 * time.Hour
		} else {
			return subtaskDur
		}
	} else {
		return *t.Duration
	}
}

// TasksByDeadline is a sort interface to stort tasks by their deadlines
type TasksByDeadline []*Task

func (a TasksByDeadline) Len() int      { return len(a) }
func (a TasksByDeadline) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a TasksByDeadline) Less(i, j int) bool {
	switch {

	case a[i].Deadline == nil && a[j].Deadline == nil: // neither deadlines
		return a[i].Name < a[j].Name

	case a[i].Deadline == nil && a[j].Deadline != nil: // only second deadline
		return false

	case a[i].Deadline != nil && a[j].Deadline == nil: // only first deadline
		return true

	case a[i].Deadline != nil && a[j].Deadline != nil: // both deadlines
		return a[i].Deadline.Before(*a[j].Deadline)

	}

	log.Fatal().Msg("this is impossible to reach, how did you do it?")
	return true
}
