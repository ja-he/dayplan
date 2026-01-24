package model

import (
	"time"
)

type EventID = string

// ...
type Event struct {
	ID       EventID      `dpedit:",ignore"`
	Name     string       `dpedit:"name"`
	Category CategoryName `dpedit:"category"`
	Start    time.Time    `dpedit:",ignore"`
	End      *time.Time   `dpedit:",ignore"`
}

// ...
func (e *Event) Duration(fallbackEnd time.Time) time.Duration {
	if e.End == nil {
		return fallbackEnd.Sub(e.Start)
	}
	return e.End.Sub(e.Start)
}

func (e *Event) String() string {
	id := e.ID
	start := e.Start.Format(time.RFC3339)
	end := ""
	if e.End != nil {
		end = e.End.Format(time.RFC3339)
	}
	catName := e.Category
	eventName := e.Name

	return (id + "|" + start + "|" + end + "|" + string(catName) + "|" + eventName)
}

// ...
type ByStartConsideringEnd []*Event

func (a ByStartConsideringEnd) Len() int      { return len(a) }
func (a ByStartConsideringEnd) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByStartConsideringEnd) Less(i, j int) bool {
	if a[j].Start.After(a[i].Start) {
		return true
	}
	if a[i].Start == a[j].Start {
		if a[i].End == nil && a[j].End == nil {
			// neither has end
			return a[i].Category < a[j].Category && a[i].Name < a[j].Name && a[i].ID < a[j].ID
		}
		if a[i].End == nil {
			// first has no end
			return false
		}
		if a[i].End == nil {
			// second has no end
			return true
		}
		// both have end

		if a[i].End.Equal(*a[j].End) {
			return a[i].Category < a[j].Category && a[i].Name < a[j].Name && a[i].ID < a[j].ID
		}
		return a[j].End.Before(*a[i].End)
	}
	return false
}

// Whether one event A contains another B, i.E.
// - B's start is _not before_ A's start and
// - B's end is _not after_ A's end
func (b *Event) IsContainedIn(a *Event) bool {
	if a.End == nil || b.End == nil {
		return false
	}
	return b.StartsDuring(a) &&
		!(b.End.After(*a.End))
}

// Whether one event B starts during another A.
func (b *Event) StartsDuring(a *Event) bool {
	if a.Start.After(b.Start) {
		return false
	}

	return a.End.After(b.Start)
}

func (e Event) Clone() Event {
	return Event{
		ID:       e.ID,
		Name:     e.Name,
		Category: e.Category,
		Start:    e.Start,
		End:      e.End,
	}
}
