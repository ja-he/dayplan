package model

import (
	"fmt"
	"sort"
	"strings"
)

type ByName []Category

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type Category struct {
	Name string
}

type EventID int

type Event struct {
	ID         EventID
	Start, End Timestamp
	Name       string
	Cat        Category
}

func (e *Event) Duration() int {
	return e.Start.DurationInMinutesUntil(e.End)
}

func NewEvent(s string) *Event {
	var e Event

	args := strings.SplitN(s, "|", 4)
	startString := args[0]
	endString := args[1]
	catString := args[2]
	nameString := args[3]

	e.Start = *NewTimestamp(startString)
	e.End = *NewTimestamp(endString)

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

func (e *Event) Move(offset TimeOffset) {
	e.Start = e.Start.Offset(offset)
	e.End = e.End.Offset(offset)
}

func (e *Event) Snap(minuteResolution int) {
	e.Start.Snap(minuteResolution)
	e.End.Snap(minuteResolution)
}

type Day struct {
	Events []Event
	idseq  func() EventID
}

func (day *Day) ToSlice() []string {
	var data []string
	for _, e := range day.Events {
		data = append(data, e.toString())
	}
	return data
}

func NewDay() *Day {
	day := Day{}
	day.idseq = idseq()
	return &day
}

func NewDayWithEvents(events []Event) *Day {
	day := NewDay()
	for i := range events {
		day.AddEvent(events[i])
	}
	return day
}

func idseq() func() EventID {
	next := 0
	return func() EventID {
		next++
		return EventID(next)
	}
}

func (day *Day) RemoveEvent(id EventID) {
	if id != 0 {
		index := -1
		for i := range day.Events {
			if day.Events[i].ID == id {
				index = i
				break
			}
		}
		if index == -1 {
			panic(fmt.Sprintf("element with id %d not found", id))
		}
		day.Events = append(day.Events[:index], day.Events[index+1:]...)
	}
}

func (day *Day) AddEvent(e Event) EventID {
	if !(e.End.IsAfter(e.Start)) {
		fmt.Println("refusing to add negative length event")
		return 0
	}
	e.ID = day.idseq()
	day.Events = append(day.Events, e)
	day.UpdateEventOrder()
	return e.ID
}

func (day *Day) UpdateEventOrder() {
	sort.Sort(ByStart(day.Events))
}

func (day *Day) getEvent(id EventID, getFollowing bool) []*Event {
	for i := range day.Events {
		e := &day.Events[i]
		if (*e).ID == id {
			if getFollowing {
				fromID := []*Event{}
				for j := i; j < len(day.Events); j++ {
					fromID = append(fromID, &day.Events[j])
				}
				return fromID
			} else {
				return []*Event{e}
			}
		}
	}
	panic(fmt.Sprintf("error getting event for id '%d' from model", id))
}

func (day *Day) GetEvent(id EventID) *Event {
	e := day.getEvent(id, false)[0]
	return e
}

func (day *Day) GetEventsFrom(id EventID) []*Event {
	f := day.getEvent(id, true)
	return f
}

func (day *Day) SplitEvent(id EventID, timestamp Timestamp) {
	originalEvent := day.GetEvent(id)

	secondEvent := Event{
		Name:  originalEvent.Name,
		Cat:   originalEvent.Cat,
		Start: timestamp,
		End:   originalEvent.End,
	}

	originalEvent.End = timestamp

	if !originalEvent.End.IsAfter(originalEvent.Start) ||
		!secondEvent.End.IsAfter(secondEvent.Start) {
		fmt.Println("warning: an event has become invalid through split")
	}

	day.AddEvent(secondEvent)
}

// TODO: obsolete?
func (day *Day) OffsetEnd(id EventID, offset TimeOffset) {
	e := day.GetEvent(id)
	e.End = e.End.Offset(offset)
	if e.Start.IsAfter(e.End) {
		panic("start after end!")
	}
}
func (day *Day) SetEnd(id EventID, end Timestamp) {
	e := day.GetEvent(id)
	if e.Start.IsAfter(end) {
		panic("start after end!")
	}
	e.End = end
}
func (day *Day) SetTimes(id EventID, start, end Timestamp) {
	if start.IsAfter(end) {
		panic("start after end!")
	}
	e := day.GetEvent(id)
	e.Start = start
	e.End = end
}

func (day *Day) Clone() *Day {
	cloned := NewDayWithEvents(day.Events)
	return cloned
}

func (day *Day) SumUpByCategory() map[Category]int {
	result := make(map[Category]int)

	for i := range day.Events {
		event := &day.Events[i]
		category := event.Cat
		duration := event.Duration()
		result[category] += duration
	}

	return result
}

func (later *Event) IsContainedIn(earlier *Event) bool {
	return later.StartsDuring(earlier) &&
		!(later.End.IsAfter(earlier.End))
}

func (later *Event) StartsDuring(earlier *Event) bool {
	// verify later/earlier input
	if earlier.Start.IsAfter(later.Start) {
		return false
	}

	return earlier.End.IsAfter(later.Start)
}
