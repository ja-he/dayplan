package model

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

type ByName []Category

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type Category struct {
	Name     string
	Priority int
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

func NewEvent(s string, knownCategories map[string]*Category) *Event {
	var e Event

	args := strings.SplitN(s, "|", 4)
	startString := args[0]
	endString := args[1]
	catString := args[2]
	nameString := args[3]

	e.Start = *NewTimestamp(startString)
	e.End = *NewTimestamp(endString)

	maybeCategory, categoryKnown := knownCategories[catString]

	e.Name = nameString
	if categoryKnown {
		e.Cat = *maybeCategory
	} else {
		e.Cat.Name = catString
	}

	return &e
}

func (e *Event) toString() string {
	start := e.Start.ToString()
	end := e.End.ToString()
	cat := e.Cat.Name
	name := e.Name

	return (start + "|" + end + "|" + cat + "|" + name)
}

type ByStartConsideringDuration []Event

func (a ByStartConsideringDuration) Len() int      { return len(a) }
func (a ByStartConsideringDuration) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByStartConsideringDuration) Less(i, j int) bool {
	secondStartsLater := a[j].Start.IsAfter(a[i].Start)
	sameStart := a[i].Start == a[j].Start
	secondEndEarlier := a[i].End.IsAfter(a[j].End)

	return secondStartsLater || (sameStart && secondEndEarlier)
}

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
	sort.Sort(ByStartConsideringDuration(day.Events))
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

// Sum up the event durations of a given day per category.
// Time cannot be counted multiple times, so if multiple events overlap, only
// one of them can have the time of the overlap counted. The prioritization for
// this is according to category priority.
func (day *Day) SumUpByCategory() map[Category]int {
	result := make(map[Category]int)

	flattened := day.Clone()
	flattened.Flatten()
	for i := range flattened.Events {
		event := &flattened.Events[i]
		result[event.Cat] += event.Duration()
	}

	return result
}

// "Flattens" the events of a given day, i.E. ensures that no overlapping
// events exist. It does this by e.g. trimming one of two overlapping events or
// splitting a less prioritized event if it had a higher-priority event occur
// during it as shown here:
//
//     +-------+         +-------+
//     | a     |         | a     |    (`a` lower prio than `B`)
//     |   +-----+       +-------+
//     |   | B   |  ~~>  | B     |
//     |   +-----+       +-------+
//     |       |         | a     |
//     +-------+         +-------+
//
//     +-------+         +-------+
//     | a     |         | a     |    (`a` lower prio than `B`)
//     |   +-----+       +-------+
//     |   | B   |  ~~>  | B     |
//     +---|     |       |       |
//         +-----+       +-------+
//
func (day *Day) Flatten() {
	if len(day.Events) < 2 {
		return
	}

	current := 0
	next := 1

	for current < len(day.Events) && next < len(day.Events) {
		day.UpdateEventOrder()

		if day.Events[next].IsContainedIn(&day.Events[current]) {
			if day.Events[next].Cat.Priority > day.Events[current].Cat.Priority {
				// clone the current event for the remainder after the next event
				currentRemainder := day.Events[current]
				currentRemainder.Start = day.Events[next].End
				// TODO: obviously a hack, but I want to get rid of EventIDs anyway
				currentRemainder.ID = EventID(rand.Int())

				// trim the current until the next event
				day.Events[current].End = day.Events[next].Start

				// easiest to just append
				if currentRemainder.Duration() > 0 {
					day.Events = append(day.Events, currentRemainder)
				}

				// if the current now has become zero-length, remove it (in which case
				// we don't need to move the indices), else move the indices one up
				if day.Events[current].Duration() == 0 {
					day.Events = append(day.Events[:current], day.Events[current+1:]...)
				} else {
					current = next
					next += 1
				}
			} else {
				// if not of higher priority, simply remove
				day.Events = append(day.Events[:next], day.Events[next+1:]...)
			}
		} else if day.Events[next].StartsDuring(&day.Events[current]) {
			if day.Events[next].Cat.Priority > day.Events[current].Cat.Priority {
				// trim current
				day.Events[current].End = day.Events[next].Start
				if day.Events[current].Duration() == 0 {
					// remove current
					day.Events = append(day.Events[:current], day.Events[next:]...)
				} else {
					// move on
					current = next
					next += 1
				}
			} else if day.Events[next].Cat.Name == day.Events[current].Cat.Name {
				// lengthen current, remove next
				day.Events[current].End = day.Events[next].End
				day.Events = append(day.Events[:next], day.Events[next+1:]...)
			} else {
				// shorten next
				day.Events[next].Start = day.Events[current].End
			}
		} else {
			current = next
			next += 1
		}
	}
}

// Whether one event A contains another B, i.E.
// - B's start is _not before_ A's start and
// - B's end is _not after_ A's end
func (b *Event) IsContainedIn(a *Event) bool {
	return b.StartsDuring(a) &&
		!(b.End.IsAfter(a.End))
}

// Whether one event B starts during another A.
func (b *Event) StartsDuring(a *Event) bool {
	if a.Start.IsAfter(b.Start) {
		return false
	}

	return a.End.IsAfter(b.Start)
}
