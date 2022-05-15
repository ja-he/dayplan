package model

import (
	"fmt"
	"strings"
)

type Event struct {
	Start, End Timestamp
	Name       string
	Cat        Category
}

func (e *Event) Duration() int {
	return e.Start.DurationInMinutesUntil(e.End)
}

func NewEvent(s string, knownCategories []Category) *Event {
	var e Event

	args := strings.SplitN(s, "|", 4)
	startString := args[0]
	endString := args[1]
	catString := args[2]
	nameString := args[3]

	e.Start = *NewTimestamp(startString)
	e.End = *NewTimestamp(endString)

	var maybeCategory *Category
	for i := range knownCategories {
		if knownCategories[i].Name == catString {
			maybeCategory = &knownCategories[i]
		}
	}

	e.Name = nameString
	if maybeCategory != nil {
		e.Cat = *maybeCategory
	} else {
		e.Cat.Name = catString
	}

	return &e
}

func (e *Event) Clone() *Event {
	return &Event{
		Name:  e.Name,
		Cat:   e.Cat,
		Start: e.Start,
		End:   e.End,
	}
}

func (e *Event) toString() string {
	start := e.Start.ToString()
	end := e.End.ToString()
	cat := e.Cat.Name
	name := e.Name

	return (start + "|" + end + "|" + cat + "|" + name)
}

type ByStartConsideringDuration []*Event

func (a ByStartConsideringDuration) Len() int      { return len(a) }
func (a ByStartConsideringDuration) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByStartConsideringDuration) Less(i, j int) bool {
	secondStartsLater := a[j].Start.IsAfter(a[i].Start)
	sameStart := a[i].Start == a[j].Start
	secondEndEarlier := a[i].End.IsAfter(a[j].End)

	return secondStartsLater || (sameStart && secondEndEarlier)
}

func (e *Event) MoveBy(duration int) error {
	if e.CanMoveBy(duration) {
		e.Start = e.Start.OffsetMinutes(duration)
		e.End = e.End.OffsetMinutes(duration)
		return nil
	} else {
		return fmt.Errorf("moving event %s by %d would cross day boundary", e.toString(), duration)
	}
}

func (e *Event) MoveTo(newStart Timestamp) error {
	if e.CanMoveTo(newStart) {
		delta := e.Start.DurationInMinutesUntil(newStart)
		e.Start = newStart
		e.End = e.End.OffsetMinutes(delta)
		return nil
	} else {
		return fmt.Errorf("moving event %s to %s would cross day boundary", e.toString(), newStart.ToString())
	}
}

func (event *Event) CanMoveBy(minutes int) bool {
	fullDayMinutes := 24 * 60

	switch {
	case minutes >= fullDayMinutes || minutes <= -fullDayMinutes:
		return false

	case minutes > 0:
		return event.Start.OffsetMinutes(minutes).IsAfter(event.Start) && event.End.OffsetMinutes(minutes).IsAfter(event.End)

	case minutes < 0:
		return event.Start.OffsetMinutes(minutes).IsBefore(event.Start) && event.End.OffsetMinutes(minutes).IsBefore(event.End)

	default:
		return true
	}
}

func (event *Event) CanMoveTo(newStart Timestamp) bool {
	return event.CanMoveBy(newStart.DurationInMinutesUntil(event.Start))
}

func (e *Event) Snap(minuteResolution int) {
	e.Start.Snap(minuteResolution)
	e.End.Snap(minuteResolution)
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