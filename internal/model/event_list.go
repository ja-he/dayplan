package model

import (
	"fmt"
	"sort"
	"time"
)

// EventList is a list of events I suppose.
type EventList struct {
	Events []*Event
}

// ...
func (l *EventList) RemoveEvent(event *Event) error {
	if event != nil {
		index := -1
		for i := range l.Events {
			if l.Events[i] == event {
				index = i
				break
			}
		}
		if index == -1 {
			return fmt.Errorf("event %s not found for removal", event.String())
		}
		l.Events = append(l.Events[:index], l.Events[index+1:]...)
	}
	return nil
}

func (l *EventList) GetEventByID(id EventID) *Event {
	for _, e := range l.Events {
		if e.ID == id {
			return e
		}
	}
	return nil
}

// ...
func (l *EventList) AddEvent(e *Event) error {
	if !(e.End.After(e.Start)) {
		return fmt.Errorf("refusing to add negative length event %s", e.String())
	}
	l.Events = append(l.Events, e)
	l.UpdateEventOrder()
	return nil
}

// ...
func (l *EventList) GetPrevEventBefore(t time.Time) *Event {
	for i := range l.Events {
		e := l.Events[len(l.Events)-1-i]
		if t.After(e.End) {
			return e
		}
	}
	return nil
}

// ...
func (l *EventList) GetNextEventAfter(t time.Time) *Event {
	for _, e := range l.Events {
		if e.Start.After(t) {
			return e
		}
	}
	return nil
}

// ...
func (l *EventList) UpdateEventOrder() {
	sort.Sort(ByStartConsideringDuration(l.Events))
}

// ...
func (l *EventList) SplitEvent(originalEvent *Event, timestamp time.Time) error {
	if !(timestamp.After(originalEvent.Start) && originalEvent.End.After(timestamp)) {
		return fmt.Errorf("timestamp %s outside event %s", timestamp.String(), originalEvent.String())
	}

	secondEvent := Event{
		Name:     originalEvent.Name,
		Category: originalEvent.Category,
		Start:    timestamp,
		End:      originalEvent.End,
	}
	originalEvent.End = timestamp

	l.AddEvent(&secondEvent)
	return nil
}

// ...
func (l *EventList) ResizeBy(event *Event, delta time.Duration) error {
	newEnd := event.End.Add(delta)
	if !newEnd.After(event.Start) {
		return fmt.Errorf("refusing to resize event %s to negative length", event.String())
	}
	event.End = newEnd
	l.UpdateEventOrder()
	return nil
}

// ...
func (l *EventList) ResizeTo(event *Event, newEnd time.Time) error {
	delta := event.End.Sub(newEnd)
	return l.ResizeBy(event, delta)
}

// ...
func (l *EventList) MoveSingleEventBy(event *Event, duration time.Duration, snapMod time.Duration) {
	newStart := event.Start.Add(duration)
	// TODO: snap
	newEnd := event.End.Add(event.Start.Sub(newStart))
	event.Start = newStart
	event.End = newEnd
	l.UpdateEventOrder()
}

func (l *EventList) MoveSingleEventTo(event *Event, newStart time.Time) {
	delta := event.Start.Sub(newStart)
	event.Start = event.Start.Add(delta)
	event.End = event.End.Add(delta)
	l.UpdateEventOrder()
}

func (l *EventList) getEventsAfter(event *Event) []*Event {
	result := []*Event{}

	found := false
	for i := range l.Events {
		if found {
			result = append(result, l.Events[i])
		}

		if l.Events[i] == event {
			found = true
		}
	}

	return result
}

func (l *EventList) getEventsBefore(event *Event) []*Event {
	result := []*Event{}

	found := false
	for i := range l.Events {
		if found {
			result = append(result, l.Events[len(l.Events)-1-i])
		}

		if l.Events[len(l.Events)-1-i] == event {
			found = true
		}
	}

	return result
}

func cloneEvent(e *Event) Event {
	return Event{
		ID:       e.ID,
		Name:     e.Name,
		Category: e.Category,
		Start:    e.Start,
		End:      e.End,
	}
}

func (l *EventList) Clone() EventList {
	cloned := make([]*Event, len(l.Events))
	for i := range l.Events {
		c := cloneEvent(l.Events[i])
		cloned[i] = &c
	}
	return EventList{Events: cloned}
}

// Sum up the event durations of a given day per category.
// Time cannot be counted multiple times, so if multiple events overlap, only
// one of them can have the time of the overlap counted. The prioritization for
// this is according to category priority.
func (l *EventList) SumUpByCategory(getCategoryPriority func(CategoryName) int) map[CategoryName]time.Duration {
	result := make(map[CategoryName]time.Duration)

	flattened := l.Clone()
	flattened.Flatten(getCategoryPriority)

	for i := range flattened.Events {
		event := flattened.Events[i]
		result[event.Category] += event.Duration()
	}

	return result
}

// GetTimesheetEntry returns the TimesheetEntry for this day for a given
// category (e.g. "work").
func (l *EventList) GetTimesheetEntry(matcher func(CategoryName) bool, getCategoryPriority func(CategoryName) int) (*TimesheetEntry, error) {
	startFound := false
	var firstStart time.Time
	var lastEnd time.Time
	var dateOfAllEvents Date
	var breakDurationCumulative time.Duration

	flattened := l.Clone()
	flattened.Flatten(getCategoryPriority)

	for _, event := range flattened.Events {

		if matcher(event.Category) {

			if !startFound {
				firstStart = event.Start
				dateOfAllEvents = DateFromGotime(firstStart)
				startFound = true
			} else {
				breakDurationCumulative += lastEnd.Sub(firstStart)
			}

			dateOfStart, dateOfend := DateFromGotime(event.Start), DateFromGotime(event.End)
			if dateOfStart != dateOfAllEvents || dateOfend != dateOfAllEvents {
				return nil, fmt.Errorf("events of different days in one entry")
			}

			lastEnd = event.End

		}

	}

	return &TimesheetEntry{
		Start:         *NewTimestampFromGotime(firstStart),
		BreakDuration: breakDurationCumulative,
		End:           *NewTimestampFromGotime(lastEnd),
	}, nil
}

// Flatten "flattens" the events of a given day, i.E. ensures that no
// overlapping events exist.
// It does this by e.g. trimming one of two overlapping events or splitting a
// less prioritized event if it had a higher-priority event occur during it as
// shown here:
//
//	+-------+         +-------+
//	| a     |         | a     |    (`a` lower prio than `B`)
//	|   +-----+       +-------+
//	|   | B   |  ~~>  | B     |
//	|   +-----+       +-------+
//	|       |         | a     |
//	+-------+         +-------+
//
//	+-------+         +-------+
//	| a     |         | a     |    (`a` lower prio than `B`)
//	|   +-----+       +-------+
//	|   | B   |  ~~>  | B     |
//	+---|     |       |       |
//	    +-----+       +-------+
//
// It modifies the input in-place.
func (l *EventList) Flatten(getCategoryPriority func(CategoryName) int) {
	if len(l.Events) < 2 {
		return
	}

	current := 0
	next := 1

	for current < len(l.Events) && next < len(l.Events) {
		l.UpdateEventOrder()

		nextPrio := getCategoryPriority(l.Events[next].Category)
		currentPrio := getCategoryPriority(l.Events[current].Category)

		if l.Events[next].IsContainedIn(l.Events[current]) {
			if nextPrio > currentPrio {
				// clone the current event for the remainder after the next event
				currentRemainder := cloneEvent(l.Events[current])
				currentRemainder.Start = l.Events[next].End

				// trim the current until the next event
				l.Events[current].End = l.Events[next].Start

				// easiest to just append
				if currentRemainder.Duration() > 0 {
					l.Events = append(l.Events, &currentRemainder)
				}

				// if the current now has become zero-length, remove it (in which case
				// we don't need to move the indices), else move the indices one up
				if l.Events[current].Duration() == 0 {
					l.Events = append(l.Events[:current], l.Events[current+1:]...)
				} else {
					current = next
					next++
				}
			} else {
				// if not of higher priority, simply remove
				l.Events = append(l.Events[:next], l.Events[next+1:]...)
			}
		} else if l.Events[next].StartsDuring(l.Events[current]) {
			if nextPrio > currentPrio {
				// trim current
				l.Events[current].End = l.Events[next].Start
				if l.Events[current].Duration() == 0 {
					// remove current
					l.Events = append(l.Events[:current], l.Events[next:]...)
				} else {
					// move on
					current = next
					next++
				}
			} else if l.Events[next].Category == l.Events[current].Category {
				// lengthen current, remove next
				l.Events[current].End = l.Events[next].End
				l.Events = append(l.Events[:next], l.Events[next+1:]...)
			} else {
				// shorten next
				l.Events[next].Start = l.Events[current].End
			}
		} else {
			current = next
			next++
		}
	}
}
