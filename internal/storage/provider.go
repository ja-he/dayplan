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

	// AddEvent adds a new event to the data provider and returns the ID of the
	// event.
	AddEvent(model.Event) (model.EventID, error)

	// RemoveEvent removes the event with the given ID from the data provider.
	RemoveEvent(model.EventID) error
	// RemoveEvents removes the events with the given IDs from the data provider.
	RemoveEvents([]model.EventID) error

	// GetEvent returns the event data for the event with the given ID from the
	// data provider.
	GetEvent(model.EventID) (*model.Event, error)
	// GetEventAfter returns the event that starts after the given time.
	// If there is no such event, it returns nil with no error.
	// If there are multiple events starting at the same time, it will return the
	// longest one, in accordance with tima-based event ordering.
	GetEventAfter(time.Time) (*model.Event, error)
	// GetEventBefore returns the event that ends before the given time.
	// If there is no such event, it returns nil with no error.
	// If there are multiple events ending at the same time, it will return the
	// shortest one, in accordance with time-based event ordering.
	GetEventBefore(time.Time) (*model.Event, error)
	// GetPrecedingEvent returns the "preceding" event according to time-based
	// event ordering rules.
	GetPrecedingEvent(model.EventID) (*model.Event, error)
	// GetFollowingEvent returns the "following" event according to time-based
	// event ordering rules.
	GetFollowingEvent(model.EventID) (*model.Event, error)
	// GetEventsCoveringTimerange returns all events that cover the given time
	// range.
	// None of the returned elements will be nil.
	GetEventsCoveringTimerange(start, end time.Time) ([]*model.Event, error)

	// SplitEvent splits the event with the given ID at the given time.
	// The event will be split into two events, with the first event ending at
	// the given time and the second event starting at the given time.
	// The first event will retain the original ID, the second event will have a
	// new ID.
	SplitEvent(model.EventID, time.Time) error
	// SetEventStart sets the start time of the event with the given ID to the
	// given time.
	// The end time of the event will not be changed.
	// If the given start time is not before the end time of the event, an error
	// will be returned.
	SetEventStart(model.EventID, time.Time) error
	// SetEventEnd sets the end time of the event with the given ID to the given
	// time.
	// The start time of the event will not be changed.
	// If the given end time is not after the start time of the event, an error
	// will be returned.
	SetEventEnd(model.EventID, time.Time) error
	// SetEventTimes sets the start and end times of the event with the given ID
	// to the given times.
	// Unless the start time is before the end time, an error will be returned.
	SetEventTimes(model.EventID, time.Time, time.Time) error
	// OffsetEventStart offsets the start time of the event with the given ID by
	// the given duration.
	// The end time of the event will not be changed.
	// If the resulting start time is not before the end time of the event, an
	// error will be returned.
	OffsetEventStart(model.EventID, time.Duration) (time.Time, error)
	// OffsetEventEnd offsets the end time of the event with the given ID by the
	// given duration.
	// The start time of the event will not be changed.
	// If the resulting end time is not after the start time of the event, an
	// error will be returned.
	OffsetEventEnd(model.EventID, time.Duration) (time.Time, error)
	// OffsetEventTimes offsets the start and end times of the event with the
	// given ID by the given duration.
	// The duration of the event will thus not be changed, as start and end times
	// are offset by the same amount.
	OffsetEventTimes(model.EventID, time.Duration) (time.Time, time.Time, error)
	// SnapEventStart snaps the start time of the event with the given ID to the
	// nearest time that is a multiple of the given duration.
	// The end time of the event will not be changed.
	// If the resulting start time is not before the end time of the event, an
	// error will be returned.
	SnapEventStart(model.EventID, time.Duration) (time.Time, error)
	// SnapEventEnd snaps the end time of the event with the given ID to the
	// nearest time that is a multiple of the given duration.
	// The start time of the event will not be changed.
	// If the resulting end time is not after the start time of the event, an
	// error will be returned.
	SnapEventEnd(model.EventID, time.Duration) (time.Time, error)
	// SnapEventTimes snaps the start and end times of the event with the given ID
	// to the nearest times that are multiples of the given duration.
	// This may result in a change of the duration of the event.
	// If the resulting start time is not before the resulting end time of the
	// event, an error will be returned.
	SnapEventTimes(model.EventID, time.Duration) (time.Time, time.Time, error)
	// SetEventTitle sets the title of the event with the given ID to the given
	// string.
	SetEventTitle(model.EventID, string) error
	// SetEventCategory	sets the category of the event with the given ID to the
	// given category.
	SetEventCategory(model.EventID, model.CategoryName) error
	// SetEventAllData sets all data of the event with the given ID to the given
	// data.
	SetEventAllData(model.EventID, model.Event) error

	// SumUpTimespanByCategory returns the total duration of all events in the
	// given time range, grouped by category.
	SumUpTimespanByCategory(start time.Time, end time.Time) (map[model.CategoryName]time.Duration, error)

	// need something here for mutability, e.g. constructing an editor...

	CommitState() error
}

// SunTimesProvider
type SunTimesProvider interface {
	Get(model.Date) model.SunTimes
}
