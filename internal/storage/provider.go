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
	AddEvent(model.Event) (model.EventID, error)

	RemoveEvent(model.EventID) error
	RemoveEvents([]model.EventID) error

	GetEvent(model.EventID) (*model.Event, error)
	GetEventAfter(time.Time) (*model.Event, error)
	GetEventBefore(time.Time) (*model.Event, error)
	GetPrecedingEvent(model.EventID) (*model.Event, error)
	GetFollowingEvent(model.EventID) (*model.Event, error)
	GetEventsCoveringTimerange(start, end time.Time) ([]*model.Event, error)

	SplitEvent(model.EventID, time.Time) error
	SetEventStart(model.EventID, time.Time) error
	SetEventEnd(model.EventID, time.Time) error
	SetEventTimes(model.EventID, time.Time, time.Time) error
	OffsetEventStart(model.EventID, time.Duration) (time.Time, error)
	OffsetEventEnd(model.EventID, time.Duration) (time.Time, error)
	OffsetEventTimes(model.EventID, time.Duration) (time.Time, time.Time, error)
	SnapEventStart(model.EventID, time.Duration) (time.Time, error)
	SnapEventEnd(model.EventID, time.Duration) (time.Time, error)
	SnapEventTimes(model.EventID, time.Duration) (time.Time, time.Time, error)
	SetEventTitle(model.EventID, string) error
	SetEventCategory(model.EventID, model.Category) error
	SetEventAllData(model.EventID, model.Event) error

	SumUpTimespanByCategory(start time.Time, end time.Time) map[model.CategoryName]time.Duration

	// need something here for mutability, e.g. constructing an editor...

	CommitState() error
}

// SunTimesProvider
type SunTimesProvider interface {
	Get(model.Date) model.SunTimes
}
