package model

import (
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Backlog holds Tasks which can be popped out of the backlog to a concrete
// timeslot.
// Backlogging Tasks is a planning mechanism; the Backlog can be seen as a
// to-do list.
type Backlog struct {
	Tasks []Task
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
	Subtasks []Task
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
		return nil, fmt.Errorf("unable to read from reader (%s)", err.Error())
	}

	stored := BacklogStored{}
	err = yaml.Unmarshal(data, &stored)
	if err != nil {
		return nil, fmt.Errorf("yaml unmarshaling error (%s)", err.Error())
	}
	log.Debug().Int("N-Cats", len(stored.TasksByCategory)).Msg("read storeds")

	var mapSubtasks func(cat string, tasks []BaseTask) []Task
	toTask := func(cat string, b BaseTask) Task {
		return Task{
			Name:     b.Name,
			Category: categoryGetter(cat),
			Duration: b.Duration,
			Deadline: b.Deadline,
			Subtasks: mapSubtasks(cat, b.Subtasks),
		}
	}
	mapSubtasks = func(cat string, tasks []BaseTask) []Task {
		result := []Task{}
		for _, t := range tasks {
			result = append(result, toTask(cat, t))
		}
		return result
	}

	b := &Backlog{Tasks: []Task{}}
	for cat, tasks := range stored.TasksByCategory {
		for _, task := range tasks {
			b.Tasks = append(b.Tasks, toTask(cat, task))
		}
	}

	return b, nil
}