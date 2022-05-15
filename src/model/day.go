package model

import (
	"fmt"
	"sort"
)

type Day struct {
	Events  []*Event
	Current *Event
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
	return &day
}

func NewDayWithEvents(events []*Event) *Day {
	day := NewDay()
	for i := range events {
		day.AddEvent(events[i].Clone())
	}
	return day
}

func (day *Day) RemoveEvent(event *Event) {
	if event != nil {
		index := -1
		for i := range day.Events {
			if day.Events[i] == event {
				index = i
				break
			}
		}
		if index == -1 {
			panic(fmt.Sprintf("event %s not found for removal", event.toString()))
		}
		day.Events = append(day.Events[:index], day.Events[index+1:]...)
		if day.Current == event {
			if index < len(day.Events) {
				day.Current = day.Events[index]
			} else if len(day.Events) > 0 {
				day.Current = day.Events[len(day.Events)-1]
			} else {
				day.Current = nil
			}
		}
	}
}

func (day *Day) AddEvent(e *Event) error {
	if !(e.End.IsAfter(e.Start)) {
		return fmt.Errorf("refusing to add negative length event %s", e.toString())
	}
	day.Events = append(day.Events, e)
	day.UpdateEventOrder()
	day.Current = e
	return nil
}

func (day *Day) GetPrevEventBefore(t Timestamp) *Event {
	for i := range day.Events {
		e := day.Events[len(day.Events)-1-i]
		if t.IsAfter(e.End) {
			return e
		}
	}
	return nil
}

func (day *Day) GetNextEventAfter(t Timestamp) *Event {
	for _, e := range day.Events {
		if e.Start.IsAfter(t) {
			return e
		}
	}
	return nil
}

func (day *Day) CurrentPrev() {
	for i := range day.Events {
		if day.Events[i] == day.Current {
			if i > 0 {
				day.Current = day.Events[i-1]
			}
			return
		}
	}

	// in case no event with ID found (e.g. because 0)
	if len(day.Events) > 0 {
		day.Current = day.Events[0]
	}
	return
}

func (day *Day) CurrentNext() {
	for i := range day.Events {
		if day.Events[i] == day.Current {
			if len(day.Events) > i+1 {
				day.Current = day.Events[i+1]
			}
			return
		}
	}

	// in case no event found (e.g. because nil)
	if len(day.Events) > 0 {
		day.Current = day.Events[0]
	}
	return
}

func (day *Day) UpdateEventOrder() {
	sort.Sort(ByStartConsideringDuration(day.Events))
}

func (day *Day) GetEventsFrom(event *Event) []*Event {
	for i := range day.Events {
		if day.Events[i] == event {
			return day.Events[i:]
		}
	}
	return make([]*Event, 0)
}

func (day *Day) SplitEvent(originalEvent *Event, timestamp Timestamp) error {
	if !(timestamp.IsAfter(originalEvent.Start) && originalEvent.End.IsAfter(timestamp)) {
		return fmt.Errorf("timestamp %s outside event %s", timestamp.ToString(), originalEvent.toString())
	}

	secondEvent := Event{
		Name:  originalEvent.Name,
		Cat:   originalEvent.Cat,
		Start: timestamp,
		End:   originalEvent.End,
	}
	originalEvent.End = timestamp

	day.AddEvent(&secondEvent)
	return nil
}

// TODO: obsolete?
func (day *Day) OffsetEnd(e *Event, offset TimeOffset) {
	e.End = e.End.Offset(offset)
	if e.Start.IsAfter(e.End) {
		panic("start after end!")
	}
}
func (day *Day) SetEnd(e *Event, end Timestamp) {
	if e.Start.IsAfter(end) {
		panic("start after end!")
	}
	e.End = end
}
func (day *Day) SetTimes(e *Event, start, end Timestamp) error {
	if start.IsAfter(end) {
		return fmt.Errorf("start after end!")
	}
	e.Start = start
	e.End = end
	day.UpdateEventOrder()
	return nil
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
		event := flattened.Events[i]
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

		if day.Events[next].IsContainedIn(day.Events[current]) {
			if day.Events[next].Cat.Priority > day.Events[current].Cat.Priority {
				// clone the current event for the remainder after the next event
				currentRemainder := day.Events[current].Clone()
				currentRemainder.Start = day.Events[next].End

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
		} else if day.Events[next].StartsDuring(day.Events[current]) {
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
