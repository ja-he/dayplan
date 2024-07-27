package model

import (
	"time"
)

// ...
type Event struct {
	Name  string    `dpedit:"name"`
	Cat   Category  `dpedit:"category"` // TODO: change to just category name
	Start time.Time `dpedit:",ignore"`
	End   time.Time `dpedit:",ignore"`
}

// ...
func (e *Event) Duration() time.Duration {
	return e.End.Sub(e.Start)
}

// ...
func (e *Event) Clone() Event {
	return Event{
		Name:  e.Name,
		Cat:   e.Cat,
		Start: e.Start,
		End:   e.End,
	}
}

func (e *Event) String() string {
	start := e.Start.String()
	end := e.End.String()
	catName := e.Cat.Name
	eventName := e.Name

	return (start + "|" + end + "|" + string(catName) + "|" + eventName)
}

// ...
type ByStartConsideringDuration []*Event

func (a ByStartConsideringDuration) Len() int      { return len(a) }
func (a ByStartConsideringDuration) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByStartConsideringDuration) Less(i, j int) bool {
	secondStartsLater := a[j].Start.After(a[i].Start)
	sameStart := a[i].Start == a[j].Start
	secondEndEarlier := a[i].End.After(a[j].End)

	return secondStartsLater || (sameStart && secondEndEarlier)
}

// Whether one event A contains another B, i.E.
// - B's start is _not before_ A's start and
// - B's end is _not after_ A's end
func (b *Event) IsContainedIn(a *Event) bool {
	return b.StartsDuring(a) &&
		!(b.End.After(a.End))
}

// Whether one event B starts during another A.
func (b *Event) StartsDuring(a *Event) bool {
	if a.Start.After(b.Start) {
		return false
	}

	return a.End.After(b.Start)
}
