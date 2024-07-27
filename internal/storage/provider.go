package storage

import (
	"time"

	"github.com/ja-he/dayplan/internal/model"
)

type EventIdentifier = *model.Event // TODO: need an ID of some sort for this

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
	AddEvent(model.Event) error

	RemoveEvent(EventIdentifier) error
	RemoveEvents([]EventIdentifier) error

	GetEventAfter(time.Time) (*model.Event, error)
	GetEventBefore(time.Time) (*model.Event, error)
	GetPrecedingEvent(EventIdentifier, time.Time) (*model.Event, error)
	GetFollowingEvent(EventIdentifier, time.Time) (*model.Event, error)
	GetEventsCoveringTimerange(start, end time.Time) ([]*model.Event, error)

	SplitEvent(EventIdentifier, time.Time) error
	SetEventStart(EventIdentifier, time.Time) error
	SetEventEnd(EventIdentifier, time.Time) error
	SetEventTimes(EventIdentifier, time.Time, time.Time) error
	OffsetEventStart(EventIdentifier, time.Duration) error
	OffsetEventEnd(EventIdentifier, time.Duration) error
	OffsetEventTimes(EventIdentifier, time.Duration) error
	SnapEventStart(EventIdentifier, time.Duration) error
	SnapEventEnd(EventIdentifier, time.Duration) error
	SnapEventTimes(EventIdentifier, time.Duration) error
	SetEventTitle(EventIdentifier, string) error
	SetEventCategory(EventIdentifier, model.Category) error
	SetEventAllData(EventIdentifier, model.Event) error

	SumUpTimespanByCategory(start time.Time, end time.Time) map[string]time.Duration

	// need something here for mutability, e.g. constructing an editor...

	CommitState() error
}

// SunTimesProvider
type SunTimesProvider interface {
	Get(model.Date) model.SunTimes
}
