package storage

import (
	"time"

	"github.com/ja-he/dayplan/internal/model"
)

// DataProvider is the abstracted data provider, which can be implemented over
// various storage systems.
//
// The data provider's responsibility are as follows:
//   - track events
//   - provide useful queries over events
//   - handle whatever backend backs the storage
//     I think my idea here is that the creation of the concrete povider will
//     potentially configure the desired behaviour of automatically storing
//     changes to the backend or waiting for commits.
type DataProvider interface {
	AddEvent(e model.Event) error

	RemoveEvent(e model.Event) error

	GetEventsCoveringTimerange(start, end time.Time) ([]model.Event, error)

	// need something here for mutability, e.g. constructing an editor...

	CommitState() error
}
