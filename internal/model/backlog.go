package model

import (
	"fmt"
	"io"
	"time"
)

// Backlog holds Tasks which can be popped out of the backlog to a concrete
// timeslot.
// Backlogging Tasks is a planning mechanism; the Backlog can be seen as a
// to-do list.
type Backlog struct {
	Tasks []Task `yaml:"tasks"` // TODO: can we 'inline' this?
}

// A Task remains to be done (or dropped) but is not yet scheduled.
// It has a name and belongs to a category (by name);
// it can further have a duration (estimate), a deadline (due date) and
// subtasks.
type Task struct {
	Name     string         `yaml:"name"`
	Category string         `yaml:"category"`
	Duration *time.Duration `yaml:"duration,omitempty"`
	Deadline *Timestamp     `yaml:"deadline,omitempty"`
	Subtasks []Task         `yaml:"subtasks,omitempty"`
}

// Write writes the Backlog to the given io.Writer, e.g., an opened file.
func (b *Backlog) Write(w io.Writer) error {
	// TODO
	return fmt.Errorf("unimplemented")
}

// Read reads and deserializes a backlog from the io.Reader and returns the
// backlog.
func BacklogFromReader(r io.Reader) (*Backlog, error) {
	// TODO
	return nil, fmt.Errorf("unimplemented")
}
