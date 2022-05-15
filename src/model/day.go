package model

import (
	"fmt"
	"math"
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

// TODO(ja-he): remove
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

func (day *Day) ResizeBy(event *Event, delta int) error {
	err := event.ResizeBy(delta)
	if err == nil {
		day.UpdateEventOrder()
	}
	return err
}

func (day *Day) ResizeTo(event *Event, newEnd Timestamp) error {
	delta := event.End.DurationInMinutesUntil(newEnd)
	return day.ResizeBy(event, delta)
}

func (day *Day) MoveSingleEventBy(event *Event, duration int, snapMinsMod int) error {
	err := event.MoveBy(duration, snapMinsMod)
	if err == nil {
		day.UpdateEventOrder()
	}
	return err
}

func (day *Day) MoveSingleEventTo(event *Event, newStart Timestamp) error {
	err := event.MoveTo(newStart)
	if err == nil {
		day.UpdateEventOrder()
	}
	return err
}

func (day *Day) MoveEventsPushingBy(event *Event, duration int, snapMinsMod int) error {
	apply := func(actions []func()) {
		for i := range actions {
			actions[i]()
		}
	}

	getMoveIfApplicable := func(e *Event) (applicable bool, newStart, newEnd Timestamp, move func()) {
		if e.CanMoveBy(duration, snapMinsMod) {
			return true, e.Start.OffsetMinutes(duration).Snap(snapMinsMod), e.End.OffsetMinutes(duration).Snap(snapMinsMod), func() { e.MoveBy(duration, snapMinsMod) }
		} else {
			return false, newStart, newEnd, nil
		}
	}

	moves := []func(){}

	switch {
	case duration < 0:
		applicable, lastStart, _, move := getMoveIfApplicable(event)
		if applicable {
			moves = append(moves, move)
			for _, preceding := range day.getEventsBefore(event) {
				if preceding.End.IsAfter(lastStart) {
					applicable, lastStart, _, move = getMoveIfApplicable(preceding)
					if applicable {
						moves = append(moves, move)
					} else {
						return fmt.Errorf("cannot move event %s by %d", preceding.toString(), duration)
					}
				} else {
					break
				}
			}
			apply(moves)
			return nil
		} else {
			return fmt.Errorf("cannot move event %s by %d", event.toString(), duration)
		}

	case duration > 0:
		applicable, _, lastEnd, move := getMoveIfApplicable(event)
		if applicable {
			moves = append(moves, move)
			for _, follower := range day.getEventsAfter(event) {
				if follower.Start.IsBefore(lastEnd) {
					applicable, _, lastEnd, move = getMoveIfApplicable(follower)
					if applicable {
						moves = append(moves, move)
					} else {
						return fmt.Errorf("cannot move event %s by %d", follower.toString(), duration)
					}
				} else {
					break
				}
			}
			apply(moves)
			return nil
		} else {
			return fmt.Errorf("cannot move event %s by %d", event.toString(), duration)
		}

	default:
		return nil
	}
}

func (day *Day) SnapEnd(event *Event, resolution int) error {
	newEnd := event.End.Snap(resolution)
	if math.Abs(float64(newEnd.DurationInMinutesUntil(event.End))) > float64(resolution) {
		return fmt.Errorf(
			"snapping %s to %d min resolution is illegal (would snap end to %s)",
			event.toString(), resolution, event.End.Snap(resolution).ToString(),
		)
	}
	event.End = newEnd
	day.UpdateEventOrder()
	return nil
}

func (day *Day) getEventsAfter(event *Event) []*Event {
	result := []*Event{}

	found := false
	for i := range day.Events {
		if found {
			result = append(result, day.Events[i])
		}

		if day.Events[i] == event {
			found = true
		}
	}

	return result
}

func (day *Day) getEventsBefore(event *Event) []*Event {
	result := []*Event{}

	found := false
	for i := range day.Events {
		if found {
			result = append(result, day.Events[len(day.Events)-1-i])
		}

		if day.Events[len(day.Events)-1-i] == event {
			found = true
		}
	}

	return result
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
