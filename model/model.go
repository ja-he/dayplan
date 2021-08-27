package model

import (
	"strings"

	"dayplan/timestamp"
)

type Category struct {
	Name string
}

type Event struct {
	Start, End timestamp.Timestamp
	Name       string
	Cat        Category
}

func NewEvent(s string) *Event {
	var e Event

	args := strings.SplitN(s, "|", 4)
	startString := args[0]
	endString := args[1]
	catString := args[2]
	nameString := args[3]

	e.Start = *timestamp.NewTimestamp(startString)
	e.End = *timestamp.NewTimestamp(endString)

	e.Name = nameString
	e.Cat.Name = catString

	return &e
}

type ByStart []Event

func (a ByStart) Len() int           { return len(a) }
func (a ByStart) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByStart) Less(i, j int) bool { return a[j].Start.IsAfter(a[i].Start) }

func (e *Event) Move(offset timestamp.TimeOffset) {
	e.Start = e.Start.Offset(offset)
	e.End = e.End.Offset(offset)
}

func (e *Event) Snap(minuteResolution int) {
	e.Start.Snap(minuteResolution)
	e.End.Snap(minuteResolution)
}

type Model struct {
	Events []Event
}

func (m *Model) AddEvent(e Event) {
	m.Events = append(m.Events, e)
}
