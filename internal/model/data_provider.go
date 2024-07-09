package model

import "time"

// DataProvider is the abstracted data provider, which can be implemented over
// various storage systems.
type DataProvider interface {
	AddEvent(e Event) error

	RemoveEvent(e Event) error

	GetEventsCoveringTimerange(start, end time.Time) ([]Event, error)

	// need something here for mutability, e.g. constructing an editor...

}
