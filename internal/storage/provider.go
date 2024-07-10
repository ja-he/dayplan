package storage

import (
	"time"

	"github.com/ja-he/dayplan/internal/model"
)

// DataProvider is the abstracted data provider, which can be implemented over
// various storage systems.
type DataProvider interface {
	AddEvent(e model.Event) error

	RemoveEvent(e model.Event) error

	GetEventsCoveringTimerange(start, end time.Time) ([]model.Event, error)

	// need something here for mutability, e.g. constructing an editor...
}
