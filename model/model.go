package model

import (
	"fmt"
	"sort"
	"strings"

	"dayplan/timestamp"
)

type Category struct {
	Name string
}

type EventID int

type Event struct {
	ID         EventID
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

func (e *Event) toString() string {
	start := e.Start.ToString()
	end := e.End.ToString()
	cat := e.Cat.Name
	name := e.Name

	return (start + "|" + end + "|" + cat + "|" + name)
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
	idseq  func() EventID
}

func (m *Model) ToSlice() []string {
	var data []string
	for _, e := range m.Events {
		data = append(data, e.toString())
	}
	return data
}

func NewModel() *Model {
	m := Model{}
	m.idseq = idseq()
	return &m
}

func idseq() func() EventID {
	next := 0
	return func() EventID {
		next++
		return EventID(next)
	}
}

func (m *Model) RemoveEvent(id EventID) {
	if id != 0 {
		index := -1
		for i := range m.Events {
			if m.Events[i].ID == id {
				index = i
				break
			}
		}
		if index == -1 {
			panic(fmt.Sprintf("element with id %d not found", id))
		}
		m.Events = append(m.Events[:index], m.Events[index+1:]...)
	}
}

func (m *Model) AddEvent(e Event) EventID {
	e.ID = m.idseq()
	m.Events = append(m.Events, e)
	return e.ID
}

func (m *Model) UpdateEventOrder() {
	sort.Sort(ByStart(m.Events))
}

func (m *Model) GetEvent(id EventID) *Event {
	for i := range m.Events {
		e := &m.Events[i]
		if (*e).ID == id {
			return e
		}
	}
	panic(fmt.Sprintf("error getting event for id '%d' from model", id))
}

// TODO: obsolete?
func (m *Model) OffsetEnd(id EventID, offset timestamp.TimeOffset) {
	e := m.GetEvent(id)
	e.End = e.End.Offset(offset)
	if e.Start.IsAfter(e.End) {
		panic("start after end!")
	}
}
func (m *Model) SetEnd(id EventID, end timestamp.Timestamp) {
	e := m.GetEvent(id)
	if e.Start.IsAfter(end) {
		panic("start after end!")
	}
	e.End = end
}
func (m *Model) SetTimes(id EventID, start, end timestamp.Timestamp) {
	if start.IsAfter(end) {
		panic("start after end!")
	}
	e := m.GetEvent(id)
	e.Start = start
	e.End = end
}
