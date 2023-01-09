package model

import (
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Backlog holds Tasks which can be popped out of the backlog to a concrete
// timeslot.
// Backlogging Tasks is a planning mechanism; the Backlog can be seen as a
// to-do list.
type Backlog struct {
	Tasks []*Task
	Mtx   sync.RWMutex
}

// A Task remains to be done (or dropped) but is not yet scheduled.
// It has a name and belongs to a category (by name);
// it can further have a duration (estimate), a deadline (due date) and
// subtasks.
type Task struct {
	Name     string
	Category Category
	Duration *time.Duration
	Deadline *time.Time
	Subtasks []*Task
}

// BacklogStored.
type BacklogStored struct {
	TasksByCategory map[string][]BaseTask `yaml:",inline"`
}

// BaseTask.
type BaseTask struct {
	Name     string         `yaml:"name"`
	Duration *time.Duration `yaml:"duration,omitempty"`
	Deadline *time.Time     `yaml:"deadline,omitempty"`
	Subtasks []BaseTask     `yaml:"subtasks,omitempty"`
}

// Write writes the Backlog to the given io.Writer, e.g., an opened file.
func (b *Backlog) Write(w io.Writer) error {
	// TODO
	return fmt.Errorf("unimplemented")
}

// Read reads and deserializes a backlog from the io.Reader and returns the
// backlog.
func BacklogFromReader(r io.Reader, categoryGetter func(string) Category) (*Backlog, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return &Backlog{}, fmt.Errorf("unable to read from reader (%s)", err.Error())
	}

	stored := BacklogStored{}
	err = yaml.Unmarshal(data, &stored)
	if err != nil {
		return &Backlog{}, fmt.Errorf("yaml unmarshaling error (%s)", err.Error())
	}
	log.Debug().Int("N-Cats", len(stored.TasksByCategory)).Msg("read storeds")

	var mapSubtasks func(cat string, tasks []BaseTask) []*Task
	toTask := func(cat string, b BaseTask) *Task {
		return &Task{
			Name:     b.Name,
			Category: categoryGetter(cat),
			Duration: b.Duration,
			Deadline: b.Deadline,
			Subtasks: mapSubtasks(cat, b.Subtasks),
		}
	}
	mapSubtasks = func(cat string, tasks []BaseTask) []*Task {
		result := []*Task{}
		for _, t := range tasks {
			result = append(result, toTask(cat, t))
		}
		sort.Sort(ByDeadline(result))
		return result
	}

	b := &Backlog{Tasks: []*Task{}}
	for cat, tasks := range stored.TasksByCategory {
		for _, task := range tasks {
			b.Tasks = append(b.Tasks, toTask(cat, task))
		}
	}
	sort.Sort(ByDeadline(b.Tasks))

	return b, nil
}

// Pop the given task out from wherever it is in this backlog, returning
// that location (by surrounding tasks and parentage).
func (b *Backlog) Pop(task *Task) (prev *Task, next *Task, parentage []*Task, err error) {
	var indexAmongTasks int
	prev, next, parentage, indexAmongTasks, err = b.Locate(task)
	if err != nil {
		return
	}
	parentTasks := func() *[]*Task {
		if len(parentage) > 0 {
			return &parentage[0].Subtasks
		} else {
			return &b.Tasks
		}
	}()

	b.Mtx.Lock()
	defer b.Mtx.Unlock()
	*parentTasks = append((*parentTasks)[:indexAmongTasks], (*parentTasks)[indexAmongTasks+1:]...)
	return
}

// Locate the given task, i.e. give its neighbors and parentage.
// Returns an error when the task cannot be found.
func (b *Backlog) Locate(task *Task) (prev *Task, next *Task, parentage []*Task, index int, err error) {

	var locateRecursive func(t *Task, l []*Task, p []*Task) (prev *Task, next *Task, parentage []*Task, index int, err error)
	locateRecursive = func(t *Task, l []*Task, p []*Task) (prev *Task, next *Task, parentage []*Task, index int, err error) {
		for i, currentTask := range l {
			if currentTask == t {
				if i > 0 {
					prev = l[i-1]
				}
				if i < len(l)-1 {
					next = l[i+1]
				}
				parentage = p
				index = i
				err = nil
				return
			}
			maybePrev, maybeNext, maybeParentage, maybeIndex, maybeErr := locateRecursive(t, currentTask.Subtasks, append([]*Task{currentTask}, p...))
			if maybeErr == nil {
				prev, next, parentage, index, err = maybePrev, maybeNext, maybeParentage, maybeIndex, maybeErr
				return
			}
		}

		return nil, nil, nil, -1, fmt.Errorf("not found")
	}

	b.Mtx.RLock()
	defer b.Mtx.RUnlock()
	return locateRecursive(task, b.Tasks, nil)
}

func (t *Task) toEvent(startTime time.Time, namePrefix string) Event {
	return Event{
		Start: *NewTimestampFromGotime(startTime),
		End: *NewTimestampFromGotime(
			func() time.Time {
				return startTime.Add(t.getDurationNormalized())
			}(),
		),
		Name: namePrefix + t.Name,
		Cat:  t.Category,
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

type ByDeadline []*Task

func (a ByDeadline) Len() int      { return len(a) }
func (a ByDeadline) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a ByDeadline) Less(i, j int) bool {
	switch {

	case a[i].Deadline == nil && a[j].Deadline == nil: // neither deadlines
		if a[i].Category.Priority != a[i].Category.Priority {
			return a[i].Category.Priority > a[j].Category.Priority
		}
		return true

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
