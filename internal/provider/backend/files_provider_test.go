package backend_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/provider"
	"github.com/ja-he/dayplan/internal/provider/backend"
)

func TestFilesProvider(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out: &testWriter{
			logFunc: t.Log,
		},
	})

	yearZero := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	yearTenK := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	fresh := func(t *testing.T) provider.EventProvider {
		tempDir := t.TempDir()
		var p provider.EventProvider
		var err error
		p, err = backend.NewFilesDataProvider(tempDir, &backend.CPPOC{M: make(map[model.CategoryName]*model.Category)})
		assert.Nil(t, err)

		allEventsBefore, err := p.GetEventsCoveringTimerange(yearZero, yearTenK)
		assert.Nil(t, err, "could not get events after removing all events")
		assert.Empty(t, allEventsBefore, "not all events were removed")
		return p
	}
	refreshCleanEvents := func(t *testing.T, p provider.EventProvider) {
		allEventsBefore, err := p.GetEventsCoveringTimerange(yearZero, yearTenK)
		assert.Nil(t, err, "could not get all events before test")
		for _, e := range allEventsBefore {
			err = p.RemoveEvent(e.ID)
			assert.Nil(t, err, "could not remove event %s", e.ID)
		}
		allEventsBefore, err = p.GetEventsCoveringTimerange(yearZero, yearTenK)
		assert.Nil(t, err, "could not get events after removing all events")
		assert.Empty(t, allEventsBefore, "not all events were removed")
	}

	t.Run("create-event", func(t *testing.T) {
		p := fresh(t)

		id, err := p.AddEvent(model.Event{
			Name:     "thing",
			Category: "cat",
			Start:    time.Date(2021, 1, 1, 14, 30, 0, 0, time.UTC),
			End:      time.Date(2021, 1, 1, 16, 45, 0, 0, time.UTC),
		})
		assert.Nil(t, err)
		assert.True(t, validateUUID(id), "id '%s' is not a valid UUID", id)
		evs, err := p.GetEventsCoveringTimerange(yearZero, yearTenK)
		assert.Nil(t, err)
		assert.Len(t, evs, 1)
		assert.Equal(t, "thing", evs[0].Name)
		assert.Equal(t, model.CategoryName("cat"), evs[0].Category)
	})

	t.Run("remove-event", func(t *testing.T) {
		p := fresh(t)

		id, err := p.AddEvent(model.Event{
			Name:     "another event",
			Category: "remove-test",
			Start:    time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err)

		// Remove the event
		err = p.RemoveEvent(id)
		assert.Nil(t, err)

		event, err := p.GetEvent(id)
		assert.NotNil(t, err)
		assert.Nil(t, event)
	})

	t.Run("set-event-start", func(t *testing.T) {
		p := fresh(t)

		id, err := p.AddEvent(model.Event{
			Name:     "test event",
			Category: "test",
			Start:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err)

		t.Run("basic", func(t *testing.T) {
			err = p.SetEventStart(id, time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC))
			assert.Nil(t, err)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC), event.Start)
		})

		t.Run("try-after-end", func(t *testing.T) {
			err = p.SetEventStart(id, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC), event.Start)
		})

		t.Run("try-equal-end", func(t *testing.T) {
			err = p.SetEventStart(id, time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC), event.Start)
		})

		// for he files provider we expect this not to work because it does not
		// support this.
		t.Run("try-different-date", func(t *testing.T) {
			err = p.SetEventStart(id, time.Date(2022, 12, 31, 14, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC), event.Start)
		})
	})

	t.Run("set-event-end", func(t *testing.T) {
		p := fresh(t)

		id, err := p.AddEvent(model.Event{
			Name:     "test event",
			Category: "test",
			Start:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err)

		t.Run("basic", func(t *testing.T) {
			err = p.SetEventEnd(id, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
			assert.Nil(t, err)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), event.End)
		})

		t.Run("try-before-start", func(t *testing.T) {
			err = p.SetEventEnd(id, time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), event.End)
		})

		t.Run("try-equal-start", func(t *testing.T) {
			err = p.SetEventEnd(id, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), event.End)
		})

		// for the files provider we expect this not to work because it does not
		// support this.
		t.Run("try-different-date", func(t *testing.T) {
			err = p.SetEventEnd(id, time.Date(2023, 1, 2, 14, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), event.End)
		})
	})

	t.Run("offset-event-start", func(t *testing.T) {
		p := fresh(t)

		id, err := p.AddEvent(model.Event{
			Name:     "test event",
			Category: "test",
			Start:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err)

		t.Run("basic", func(t *testing.T) {
			newStart, err := p.OffsetEventStart(id, 1*time.Hour)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC), newStart)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC), event.Start)
		})

		t.Run("before-end", func(t *testing.T) {
			newStart, err := p.OffsetEventStart(id, -1*time.Minute)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 12, 59, 0, 0, time.UTC), newStart)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 12, 59, 0, 0, time.UTC), event.Start)
		})

		t.Run("invalid-duration", func(t *testing.T) {
			e, err := p.GetEvent(id)
			assert.Nil(t, err)
			invalidDuration := e.End.Sub(e.Start) + (10 * time.Minute)
			t.Logf("using invalid duration of %s to offset start", invalidDuration)

			_, err = p.OffsetEventStart(id, invalidDuration)
			assert.NotNil(t, err)
		})
	})

	t.Run("offset-event-end", func(t *testing.T) {
		p := fresh(t)

		id, err := p.AddEvent(model.Event{
			Name:     "test event",
			Category: "test",
			Start:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err)

		t.Run("basic", func(t *testing.T) {
			newEnd, err := p.OffsetEventEnd(id, 1*time.Hour)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), newEnd)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), event.End)
		})

		t.Run("after-start", func(t *testing.T) {
			e, err := p.GetEvent(id)
			assert.Nil(t, err)
			endBefore := e.End

			newEnd, err := p.OffsetEventEnd(id, -1*time.Minute)
			expectedEnd := endBefore.Add(-1 * time.Minute)
			assert.Nil(t, err)
			assert.Equal(t, expectedEnd, newEnd)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedEnd, event.End)
		})

		t.Run("invalid-duration-would-move-end-before-start", func(t *testing.T) {
			e, err := p.GetEvent(id)
			endBefore := e.End
			assert.Nil(t, err)
			invalidDuration := -(e.End.Sub(e.Start) + (10 * time.Minute))

			_, err = p.OffsetEventEnd(id, invalidDuration)
			assert.NotNil(t, err)

			e, err = p.GetEvent(id)

			assert.Equal(t, endBefore, e.End, "event end was changed despite error")
		})

		t.Run("resize exactly to midnight", func(t *testing.T) {
			e, err := p.GetEvent(id)
			assert.Nil(t, err, "event was not retrieved: %s", err)

			endBefore := e.End
			midnight := e.End.Truncate(24 * time.Hour).Add(24 * time.Hour)
			durationTilMidnight := midnight.Sub(e.End)

			_, err = p.OffsetEventEnd(id, durationTilMidnight)
			assert.Nil(t, err, "could not offset event end: %s", err)

			e, err = p.GetEvent(id)
			assert.Nil(t, err, "event was not retrieved: %s", err)

			assert.NotEqual(t, endBefore, e.End, "event end was not changed")
			assert.Equal(t, midnight, e.End, "event end was not changed to exactly midnight")

			_, err = p.OffsetEventEnd(id, -durationTilMidnight)
			assert.Nil(t, err, "could not offset event end: %s", err)
		})

		t.Run("invalid then valid", func(t *testing.T) {
			e, err := p.GetEvent(id)
			assert.Nil(t, err, "event was not found")
			t.Logf("Found event %s (%s).", id, e.ID)

			endBefore := e.End
			invalidDuration := -(e.End.Sub(e.Start) + (10 * time.Minute))
			midnight := e.End.Truncate(24 * time.Hour).Add(24 * time.Hour)
			validDuration := midnight.Sub(e.End)

			_, err = p.OffsetEventEnd(id, invalidDuration)
			assert.NotNil(t, err, "no error occurred despite changing event end to before start")

			e, err = p.GetEvent(id)
			assert.Nil(t, err, "event was not found")
			t.Logf("Found event %s (%s) again.", id, e.ID)
			assert.Equal(t, endBefore, e.End, "event end was changed despite error")

			_, err = p.OffsetEventEnd(id, validDuration)
			assert.Nil(t, err, "an error was detected when resizing the event to end at midnight at end of current day")
		})

	})

	t.Run("offset-event-times", func(t *testing.T) {
		p := fresh(t)

		id, err := p.AddEvent(model.Event{
			Name:     "test event",
			Category: "test",
			Start:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err)

		t.Run("basic", func(t *testing.T) {
			newStart, newEnd, err := p.OffsetEventTimes(id, 1*time.Hour)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC), newStart)
			assert.Equal(t, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), newEnd)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC), event.Start)
			assert.Equal(t, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), event.End)
		})

		t.Run("no-duration-change", func(t *testing.T) {
			e, err := p.GetEvent(id)
			assert.Nil(t, err)
			startBefore, endBefore := e.Start, e.End

			reportedNewStart, reportedNewEnd, err := p.OffsetEventTimes(id, 0)
			assert.Nil(t, err)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			startAfter, endAfter := event.Start, event.End

			assert.Equal(t, startBefore, startAfter)
			assert.Equal(t, endBefore, endAfter)
			assert.Equal(t, startBefore, reportedNewStart)
			assert.Equal(t, endBefore, reportedNewEnd)
		})
	})

	t.Run("get-preceding-event", func(t *testing.T) {
		t.Run("simple", func(t *testing.T) {
			p := fresh(t)

			id1, err := p.AddEvent(model.Event{
				Name:     "first event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add first event")

			id2, err := p.AddEvent(model.Event{
				Name:     "second event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 18, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add second event")

			id3, err := p.AddEvent(model.Event{
				Name:     "third event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 20, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 22, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add third event")

			id4, err := p.AddEvent(model.Event{
				Name:     "fourth event (different date)",
				Category: "test",
				Start:    time.Date(2023, 2, 12, 14, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 2, 12, 16, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add fourth event")

			t.Run("preceding-event-present", func(t *testing.T) {
				precedingEvent, err := p.GetPrecedingEvent(id2)
				assert.Nil(t, err, "error retrieving preceding event")
				assert.NotNil(t, precedingEvent, "preceding event should not be nil")
				assert.Equal(t, "first event", precedingEvent.Name, "preceding event name mismatch")
				assert.Equal(t, id1, precedingEvent.ID, "preceding event ID mismatch")
			})

			t.Run("no-preceding-event", func(t *testing.T) {
				precedingEvent, err := p.GetPrecedingEvent(id1)
				assert.Nil(t, err, "error retrieving preceding event")
				assert.Nil(t, precedingEvent, "preceding event should be nil")
			})

			t.Run("preceding-event-for-third", func(t *testing.T) {
				precedingEvent, err := p.GetPrecedingEvent(id3)
				assert.Nil(t, err, "error retrieving preceding event")
				assert.NotNil(t, precedingEvent, "preceding event should not be nil")
				assert.Equal(t, "second event", precedingEvent.Name, "preceding event name mismatch")
				assert.Equal(t, id2, precedingEvent.ID, "preceding event ID mismatch")
			})

			t.Run("preceding-on-different-date", func(t *testing.T) {
				precedingEvent, err := p.GetPrecedingEvent(id4)
				assert.Nil(t, err, "error retrieving preceding event")
				assert.NotNil(t, precedingEvent, "preceding event should not be nil")
				assert.Equal(t, "third event", precedingEvent.Name, "preceding event name mismatch")
				assert.Equal(t, id3, precedingEvent.ID, "preceding event ID mismatch")
			})

			// this is a test case that could trip up the file-based data provider
			// because it might store a handler for that date and then blindly return
			// the last event for that date (none, so nil) even though it should
			// check previous dates instead.
			t.Run("preceding-on-different-date-with-previously-existing-event-inbetween", func(t *testing.T) {
				inbetweenerID, err := p.AddEvent(model.Event{
					Name:     "inbetweener event",
					Category: "test",
					Start:    time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC),
					End:      time.Date(2023, 1, 15, 14, 0, 0, 0, time.UTC),
				})
				assert.Nil(t, err, "could not add inbetweener event")
				err = p.RemoveEvent(inbetweenerID)
				assert.Nil(t, err, "could not remove inbetweener event")

				precedingEvent, err := p.GetPrecedingEvent(id4)
				assert.Nil(t, err, "error retrieving preceding event")
				assert.NotNil(t, precedingEvent, "preceding event should not be nil")
				assert.Equal(t, "third event", precedingEvent.Name, "preceding event name mismatch")
				assert.Equal(t, id3, precedingEvent.ID, "preceding event ID mismatch")
			})

			t.Run("non-existent-event-id", func(t *testing.T) {
				invalidID := model.EventID("non-existent-id")
				precedingEvent, err := p.GetPrecedingEvent(invalidID)
				assert.NotNil(t, err, "error should occur for non-existent event ID")
				assert.Nil(t, precedingEvent, "preceding event should be nil for non-existent event ID")
			})

		})

		t.Run("get-preceding-event-overlapping", func(t *testing.T) {
			p := fresh(t)

			// Add some events to work with
			id1, err := p.AddEvent(model.Event{
				Name:     "first event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add first event")

			id2, err := p.AddEvent(model.Event{
				Name:     "second event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add second event")

			id3, err := p.AddEvent(model.Event{
				Name:     "third event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 17, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add third event")

			id4, err := p.AddEvent(model.Event{
				Name:     "fourth event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 17, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 18, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add fourth event")

			t.Run("preceding-event-present", func(t *testing.T) {
				precedingEvent, err := p.GetPrecedingEvent(id3)
				assert.Nil(t, err, "error retrieving preceding event")
				assert.NotNil(t, precedingEvent, "preceding event should not be nil")
				assert.Equal(t, "second event", precedingEvent.Name, "preceding event name mismatch")
				assert.Equal(t, id2, precedingEvent.ID, "preceding event ID mismatch")
			})

			t.Run("no-preceding-event", func(t *testing.T) {
				precedingEvent, err := p.GetPrecedingEvent(id1)
				assert.Nil(t, err, "error retrieving preceding event")
				assert.Nil(t, precedingEvent, "preceding event should be nil")
			})

			t.Run("preceding-event-for-fourth", func(t *testing.T) {
				precedingEvent, err := p.GetPrecedingEvent(id4)
				assert.Nil(t, err, "error retrieving preceding event")
				assert.NotNil(t, precedingEvent, "preceding event should not be nil")
				assert.Equal(t, "third event", precedingEvent.Name, "preceding event name mismatch")
				assert.Equal(t, id3, precedingEvent.ID, "preceding event ID mismatch")
			})

			t.Run("if two events overlap, the one that starts earlier will be considered preceding", func(t *testing.T) {
				precedingEvent, err := p.GetPrecedingEvent(id2)
				assert.Nil(t, err, "error retrieving preceding event")
				assert.NotNil(t, precedingEvent, "preceding event should not be nil")
				assert.Equal(t, "first event", precedingEvent.Name, "preceding event name mismatch")
				assert.Equal(t, id1, precedingEvent.ID, "preceding event ID mismatch")
			})

			t.Run("non-existent-event-id", func(t *testing.T) {
				invalidID := model.EventID("non-existent-id")
				precedingEvent, err := p.GetPrecedingEvent(invalidID)
				assert.NotNil(t, err, "error should occur for non-existent event ID")
				assert.Nil(t, precedingEvent, "preceding event should be nil for non-existent event ID")
			})
		})

	})

	t.Run("get-following-event", func(t *testing.T) {
		t.Run("simple", func(t *testing.T) {
			p := fresh(t)

			id1, err := p.AddEvent(model.Event{
				Name:     "first event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add first event")

			id2, err := p.AddEvent(model.Event{
				Name:     "second event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 18, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add second event")

			id3, err := p.AddEvent(model.Event{
				Name:     "third event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 20, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 22, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add third event")

			id4, err := p.AddEvent(model.Event{
				Name:     "fourth event (different date)",
				Category: "test",
				Start:    time.Date(2023, 2, 12, 14, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 2, 12, 16, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add fourth event")

			t.Run("following-event-present", func(t *testing.T) {
				followingEvent, err := p.GetFollowingEvent(id1)
				assert.Nil(t, err, "error retrieving following event")
				assert.NotNil(t, followingEvent, "following event should not be nil")
				assert.Equal(t, "second event", followingEvent.Name, "following event name mismatch")
				assert.Equal(t, id2, followingEvent.ID, "following event ID mismatch")
			})

			t.Run("no-following-event", func(t *testing.T) {
				followingEvent, err := p.GetFollowingEvent(id4)
				assert.Nil(t, err, "error retrieving following event")
				assert.Nil(t, followingEvent, "following event should be nil")
			})

			t.Run("following-event-for-second", func(t *testing.T) {
				followingEvent, err := p.GetFollowingEvent(id2)
				assert.Nil(t, err, "error retrieving following event")
				assert.NotNil(t, followingEvent, "following event should not be nil")
				assert.Equal(t, "third event", followingEvent.Name, "following event name mismatch")
				assert.Equal(t, id3, followingEvent.ID, "following event ID mismatch")
			})

			t.Run("following-on-different-date", func(t *testing.T) {
				followingEvent, err := p.GetFollowingEvent(id3)
				assert.Nil(t, err, "error retrieving following event")
				assert.NotNil(t, followingEvent, "following event should not be nil")
				assert.Equal(t, "fourth event (different date)", followingEvent.Name, "following event name mismatch")
				assert.Equal(t, id4, followingEvent.ID, "following event ID mismatch")
			})

			t.Run("non-existent-event-id", func(t *testing.T) {
				invalidID := model.EventID("non-existent-id")
				followingEvent, err := p.GetFollowingEvent(invalidID)
				assert.NotNil(t, err, "error should occur for non-existent event ID")
				assert.Nil(t, followingEvent, "following event should be nil for non-existent event ID")
			})

		})

		t.Run("overlap", func(t *testing.T) {
			p := fresh(t)

			// Add some events to work with
			id1, err := p.AddEvent(model.Event{
				Name:     "first event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add first event")

			id2, err := p.AddEvent(model.Event{
				Name:     "second event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add second event")

			id3, err := p.AddEvent(model.Event{
				Name:     "third event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 17, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add third event")

			id4, err := p.AddEvent(model.Event{
				Name:     "fourth event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 17, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 18, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add fourth event")

			t.Run("following-event-present", func(t *testing.T) {
				followingEvent, err := p.GetFollowingEvent(id1)
				assert.Nil(t, err, "error retrieving following event")
				assert.NotNil(t, followingEvent, "following event should not be nil")
				assert.Equal(t, "second event", followingEvent.Name, "following event name mismatch")
				assert.Equal(t, id2, followingEvent.ID, "following event ID mismatch")
			})

			t.Run("no-following-event", func(t *testing.T) {
				followingEvent, err := p.GetFollowingEvent(id4)
				assert.Nil(t, err, "error retrieving following event")
				assert.Nil(t, followingEvent, "following event should be nil")
			})

			t.Run("following-event-for-second", func(t *testing.T) {
				followingEvent, err := p.GetFollowingEvent(id2)
				assert.Nil(t, err, "error retrieving following event")
				assert.NotNil(t, followingEvent, "following event should not be nil")
				assert.Equal(t, "third event", followingEvent.Name, "following event name mismatch")
				assert.Equal(t, id3, followingEvent.ID, "following event ID mismatch")
			})

			t.Run("if two events overlap, the one that ends later will be considered following", func(t *testing.T) {
				followingEvent, err := p.GetFollowingEvent(id2)
				assert.Nil(t, err, "error retrieving following event")
				assert.NotNil(t, followingEvent, "following event should not be nil")
				assert.Equal(t, "third event", followingEvent.Name, "following event name mismatch")
				assert.Equal(t, id3, followingEvent.ID, "following event ID mismatch")
			})

			t.Run("non-existent-event-id", func(t *testing.T) {
				invalidID := model.EventID("non-existent-id")
				followingEvent, err := p.GetFollowingEvent(invalidID)
				assert.NotNil(t, err, "error should occur for non-existent event ID")
				assert.Nil(t, followingEvent, "following event should be nil for non-existent event ID")
			})
		})
	})

	t.Run("get-event-before", func(t *testing.T) {
		p := fresh(t)

		// Add some events to work with
		id1, err := p.AddEvent(model.Event{
			Name:     "first event",
			Category: "test",
			Start:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err, "could not add first event")

		_, err = p.AddEvent(model.Event{
			Name:     "second event",
			Category: "test",
			Start:    time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 18, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err, "could not add second event")

		_, err = p.AddEvent(model.Event{
			Name:     "third event",
			Category: "test",
			Start:    time.Date(2023, 1, 1, 20, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 22, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err, "could not add third event")

		t.Run("event-before", func(t *testing.T) {
			event, err := p.GetEventBefore(time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
			assert.Nil(t, err, "error retrieving event before specified time")
			assert.NotNil(t, event, "event should not be nil")
			assert.Equal(t, "first event", event.Name, "event name mismatch")
			assert.Equal(t, id1, event.ID, "event ID mismatch")
		})

		t.Run("no-event-before", func(t *testing.T) {
			event, err := p.GetEventBefore(time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC))
			assert.Nil(t, err, "error retrieving event before specified time")
			assert.Nil(t, event, "event should be nil")
		})

		t.Run("multiple-events-ending-at-same-time", func(t *testing.T) {
			id4, err := p.AddEvent(model.Event{
				Name:     "overlapping event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add overlapping event")

			event, err := p.GetEventBefore(time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
			assert.Nil(t, err, "error retrieving event before specified time")
			assert.NotNil(t, event, "event should not be nil")
			assert.Equal(t, id4, event.ID, "event ID mismatch")
		})
	})

	t.Run("get-event-after", func(t *testing.T) {
		p := fresh(t)

		// Add some events to work with
		_, err := p.AddEvent(model.Event{
			Name:     "first event",
			Category: "test",
			Start:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err, "could not add first event")

		id2, err := p.AddEvent(model.Event{
			Name:     "second event",
			Category: "test",
			Start:    time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 18, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err, "could not add second event")

		_, err = p.AddEvent(model.Event{
			Name:     "third event",
			Category: "test",
			Start:    time.Date(2023, 1, 1, 20, 0, 0, 0, time.UTC),
			End:      time.Date(2023, 1, 1, 22, 0, 0, 0, time.UTC),
		})
		assert.Nil(t, err, "could not add third event")

		t.Run("event-after", func(t *testing.T) {
			event, err := p.GetEventAfter(time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
			assert.Nil(t, err, "error retrieving event after specified time")
			assert.NotNil(t, event, "event should not be nil")
			assert.Equal(t, "second event", event.Name, "event name mismatch")
			assert.Equal(t, id2, event.ID, "event ID mismatch")
		})

		t.Run("no-event-after", func(t *testing.T) {
			event, err := p.GetEventAfter(time.Date(2023, 1, 1, 22, 0, 0, 0, time.UTC))
			assert.Nil(t, err, "error retrieving event after specified time")
			assert.Nil(t, event, "event should be nil")
		})

		t.Run("multiple-events-starting-at-same-time", func(t *testing.T) {
			_, err := p.AddEvent(model.Event{
				Name:     "concurrent event",
				Category: "test",
				Start:    time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC),
				End:      time.Date(2023, 1, 1, 17, 0, 0, 0, time.UTC),
			})
			assert.Nil(t, err, "could not add concurrent event")

			event, err := p.GetEventAfter(time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
			assert.Nil(t, err, "error retrieving event after specified time")
			assert.NotNil(t, event, "event should not be nil")
			assert.Equal(t, "second event", event.Name, "event name mismatch (second event should be longer)")
			assert.Equal(t, id2, event.ID, "event ID mismatch")
		})
	})

	setupWithSingleEventTimes := func(t *testing.T, p provider.EventProvider, start time.Time, end time.Time) string {
		refreshCleanEvents(t, p)
		id, err := p.AddEvent(model.Event{
			Name:     "test event",
			Category: "test",
			Start:    start,
			End:      end,
		})
		assert.Nil(t, err)
		return id
	}

	t.Run("snap-event-start", func(t *testing.T) {
		p := fresh(t)

		t.Run("basic", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 12, 34, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 13, 17, 0, 0, time.UTC),
			)

			newStart, err := p.SnapEventStart(id, 15*time.Minute)
			assert.Nil(t, err)
			expectedStart := time.Date(2023, 1, 1, 12, 30, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedStart, newStart)
			assert.Equal(t, expectedStart, event.Start)
		})

		t.Run("snap-start-to-nearest-hour", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 12, 34, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 13, 50, 0, 0, time.UTC),
			)

			newStart, err := p.SnapEventStart(id, 1*time.Hour)
			assert.Nil(t, err)
			expectedStart := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedStart, newStart)
			assert.Equal(t, expectedStart, event.Start)
		})

		t.Run("invalid-snap-start-after-end", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 12, 34, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 12, 40, 0, 0, time.UTC),
			)

			_, err := p.SnapEventStart(id, 1*time.Hour) // should snap 12:34 to 13:00 which is after end 12:40
			assert.NotNil(t, err)
		})

		t.Run("snap-start-edge-case", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 14, 29, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 15, 30, 0, 0, time.UTC),
			)

			newStart, err := p.SnapEventStart(id, 1*time.Hour)
			assert.Nil(t, err)
			expectedStart := time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedStart, newStart)
			assert.Equal(t, expectedStart, event.Start)
		})

		t.Run("snap-start-edge-case-round-up-at-half", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 14, 30, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC),
			)

			newStart, err := p.SnapEventStart(id, 1*time.Hour)
			assert.Nil(t, err)
			expectedStart := time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedStart, newStart)
			assert.Equal(t, expectedStart, event.Start)
		})

		t.Run("snap-start-edge-case-round-up", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 14, 31, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC),
			)

			newStart, err := p.SnapEventStart(id, 1*time.Hour)
			assert.Nil(t, err)
			expectedStart := time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedStart, newStart)
			assert.Equal(t, expectedStart, event.Start)
		})
	})

	t.Run("snap-event-end", func(t *testing.T) {
		p := fresh(t)

		t.Run("basic", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 12, 14, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 13, 12, 0, 0, time.UTC),
			)

			newEnd, err := p.SnapEventEnd(id, 15*time.Minute)
			assert.Nil(t, err)
			expectedEnd := time.Date(2023, 1, 1, 13, 15, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedEnd, newEnd)
			assert.Equal(t, expectedEnd, event.End)
		})

		t.Run("snap-end-to-nearest-hour", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 12, 14, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 13, 12, 0, 0, time.UTC),
			)

			newEnd, err := p.SnapEventEnd(id, 1*time.Hour)
			assert.Nil(t, err)
			expectedEnd := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedEnd, newEnd)
			assert.Equal(t, expectedEnd, event.End)
		})

		t.Run("invalid-snap-end-before-start", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 12, 34, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 12, 40, 0, 0, time.UTC),
			)

			_, err := p.SnapEventEnd(id, -1*time.Hour)
			assert.NotNil(t, err)
		})

		t.Run("snap-end-edge-case-round-down-below-half", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 14, 29, 59, 59, time.UTC),
			)

			newEnd, err := p.SnapEventEnd(id, 1*time.Hour)
			assert.Nil(t, err)
			expectedEnd := time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedEnd, newEnd)
			assert.Equal(t, expectedEnd, event.End)
		})

		t.Run("snap-end-edge-case-round-up-at-half", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 14, 30, 0, 0, time.UTC),
			)

			newEnd, err := p.SnapEventEnd(id, 1*time.Hour)
			assert.Nil(t, err)
			expectedEnd := time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedEnd, newEnd)
			assert.Equal(t, expectedEnd, event.End)
		})

		t.Run("snap-end-edge-case-round-up", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 14, 31, 0, 0, time.UTC),
			)

			newEnd, err := p.SnapEventEnd(id, 1*time.Hour)
			assert.Nil(t, err)
			expectedEnd := time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedEnd, newEnd)
			assert.Equal(t, expectedEnd, event.End)
		})
	})

	t.Run("snap-event-times", func(t *testing.T) {
		p := fresh(t)

		t.Run("basic", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 12, 34, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 13, 17, 0, 0, time.UTC),
			)

			newStart, newEnd, err := p.SnapEventTimes(id, 15*time.Minute)
			assert.Nil(t, err)
			expectedStart := time.Date(2023, 1, 1, 12, 30, 0, 0, time.UTC)
			expectedEnd := time.Date(2023, 1, 1, 13, 15, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedStart, newStart)
			assert.Equal(t, expectedEnd, newEnd)
			assert.Equal(t, expectedStart, event.Start)
			assert.Equal(t, expectedEnd, event.End)
		})

		t.Run("snap-times-to-nearest-hour", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 12, 34, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 13, 40, 0, 0, time.UTC),
			)

			newStart, newEnd, err := p.SnapEventTimes(id, 1*time.Hour)
			assert.Nil(t, err)
			expectedStart := time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC)
			expectedEnd := time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedStart, newStart)
			assert.Equal(t, expectedEnd, newEnd)
			assert.Equal(t, expectedStart, event.Start)
			assert.Equal(t, expectedEnd, event.End)
		})

		t.Run("invalid-snap-times-start-after-end", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 12, 50, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 13, 10, 0, 0, time.UTC),
			)

			_, _, err := p.SnapEventTimes(id, 1*time.Hour)
			assert.NotNil(t, err)
		})

		t.Run("snap-times-edge-case", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p,
				time.Date(2023, 1, 1, 14, 29, 0, 0, time.UTC),
				time.Date(2023, 1, 1, 14, 31, 0, 0, time.UTC),
			)

			newStart, newEnd, err := p.SnapEventTimes(id, 1*time.Hour)
			assert.Nil(t, err)
			expectedStart := time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC)
			expectedEnd := time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, expectedStart, newStart)
			assert.Equal(t, expectedEnd, newEnd)
			assert.Equal(t, expectedStart, event.Start)
			assert.Equal(t, expectedEnd, event.End)
		})
	})

	t.Run("split-event", func(t *testing.T) {
		p := fresh(t)

		originalStart := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
		originalEnd := time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC)

		t.Run("basic", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p, originalStart, originalEnd)
			splitTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

			err := p.SplitEvent(id, splitTime)
			assert.Nil(t, err)

			// Verify the original event
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, originalStart, event.Start)
			assert.Equal(t, splitTime, event.End)

			// Verify the new split event
			events, err := p.GetEventsCoveringTimerange(yearZero, yearTenK)
			assert.Nil(t, err)
			assert.Len(t, events, 2)
			assert.Equal(t, id, events[0].ID)
			assert.NotEqual(t, id, events[1].ID)
			assert.Equal(t, originalStart, events[0].Start)
			assert.Equal(t, splitTime, events[0].End)
			assert.Equal(t, splitTime, events[1].Start)
			assert.Equal(t, originalEnd, events[1].End)
		})

		t.Run("split-at-start", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p, originalStart, originalEnd)
			splitTime := originalStart

			err := p.SplitEvent(id, splitTime)
			assert.NotNil(t, err)

			// Verify the original event
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, originalStart, event.Start)
			assert.Equal(t, originalEnd, event.End)

			// Verifiy that there is no other created event
			allEvents, err := p.GetEventsCoveringTimerange(yearZero, yearTenK)
			assert.Nil(t, err)
			assert.Equal(t, len(allEvents), 1)

		})

		t.Run("split-at-end", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p, originalStart, originalEnd)
			splitTime := originalEnd

			err := p.SplitEvent(id, splitTime)
			assert.NotNil(t, err)

			// Verify the original event
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, originalStart, event.Start)
			assert.Equal(t, originalEnd, event.End)

			// Verifiy that there is no other created event
			allEvents, err := p.GetEventsCoveringTimerange(yearZero, yearTenK)
			assert.Nil(t, err)
			assert.Equal(t, len(allEvents), 1)

		})

		t.Run("invalid-split-time-before-start", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p, originalStart, originalEnd)
			splitTime := originalStart.Add(-1 * time.Minute)

			err := p.SplitEvent(id, splitTime)
			assert.NotNil(t, err)

			// Verify the original event
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, originalStart, event.Start)
			assert.Equal(t, originalEnd, event.End)

			// Verifiy that there is no other created event
			allEvents, err := p.GetEventsCoveringTimerange(yearZero, yearTenK)
			assert.Nil(t, err)
			assert.Equal(t, len(allEvents), 1)
		})

		t.Run("invalid-split-time-after-end", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p, originalStart, originalEnd)
			splitTime := originalEnd.Add(1 * time.Minute)

			err := p.SplitEvent(id, splitTime)
			assert.NotNil(t, err)
		})

		t.Run("invalid-split-same-as-end", func(t *testing.T) {
			id := setupWithSingleEventTimes(t, p, originalStart, originalEnd)
			splitTime := originalEnd

			err := p.SplitEvent(id, splitTime)
			assert.NotNil(t, err)

			// Verify the original event
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, originalStart, event.Start)
			assert.Equal(t, originalEnd, event.End)

			// Verifiy that there is no other created event
			allEvents, err := p.GetEventsCoveringTimerange(yearZero, yearTenK)
			assert.Nil(t, err)
			assert.Equal(t, len(allEvents), 1)
		})
	})

	t.Run("set-event-times", func(t *testing.T) {
		p := fresh(t)
		setupWithSingleEvent := func(t *testing.T, start time.Time, end time.Time) string {
			refreshCleanEvents(t, p)
			id, err := p.AddEvent(model.Event{
				Name:     "test event",
				Category: "test",
				Start:    start,
				End:      end,
			})
			assert.Nil(t, err)
			return id
		}

		originalStart := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		originalEnd := time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC)

		t.Run("valid-change-with-same-date", func(t *testing.T) {
			id := setupWithSingleEvent(t, originalStart, originalEnd)
			newStart := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
			newEnd := time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC)

			err := p.SetEventTimes(id, newStart, newEnd)
			assert.Nil(t, err)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, newStart, event.Start)
			assert.Equal(t, newEnd, event.End)
		})

		t.Run("valid-change-to-different-date", func(t *testing.T) {
			id := setupWithSingleEvent(t, originalStart, originalEnd)
			newStart := time.Date(2023, 1, 2, 10, 0, 0, 0, time.UTC)
			newEnd := time.Date(2023, 1, 2, 16, 0, 0, 0, time.UTC)

			err := p.SetEventTimes(id, newStart, newEnd)
			assert.Nil(t, err)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, newStart, event.Start)
			assert.Equal(t, newEnd, event.End)
		})

		t.Run("invalid-same-start-and-end", func(t *testing.T) {
			id := setupWithSingleEvent(t, originalStart, originalEnd)
			newTime := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)

			err := p.SetEventTimes(id, newTime, newTime)
			assert.NotNil(t, err)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, originalStart, event.Start)
			assert.Equal(t, originalEnd, event.End)
		})

		t.Run("invalid-start-after-end", func(t *testing.T) {
			id := setupWithSingleEvent(t, originalStart, originalEnd)
			newStart := time.Date(2023, 1, 1, 18, 0, 0, 0, time.UTC)
			newEnd := time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC)

			err := p.SetEventTimes(id, newStart, newEnd)
			assert.NotNil(t, err)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, originalStart, event.Start)
			assert.Equal(t, originalEnd, event.End)
		})

		t.Run("invalid-start-to-different-date", func(t *testing.T) {
			id := setupWithSingleEvent(t, originalStart, originalEnd)
			newStart := time.Date(2023, 1, 5, 18, 0, 0, 0, time.UTC)
			newEnd := time.Date(2023, 1, 6, 16, 0, 0, 0, time.UTC)

			err := p.SetEventTimes(id, newStart, newEnd)
			assert.NotNil(t, err)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, originalStart, event.Start)
			assert.Equal(t, originalEnd, event.End)
		})

		t.Run("invalid-end-to-different-date", func(t *testing.T) {
			id := setupWithSingleEvent(t, originalStart, originalEnd)
			newStart := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
			newEnd := time.Date(2023, 1, 2, 14, 0, 0, 0, time.UTC)

			err := p.SetEventTimes(id, newStart, newEnd)
			assert.NotNil(t, err)

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, originalStart, event.Start)
			assert.Equal(t, originalEnd, event.End)
		})
	})

	t.Run("change-and-write", func(t *testing.T) {
		p := fresh(t)

		fullyCommitted, err := p.FullyCommitted()
		assert.Nil(t, err)
		assert.True(t, fullyCommitted)

		err = p.CommitState()
		assert.Nil(t, err)

		fullyCommitted, err = p.FullyCommitted()
		assert.Nil(t, err)
		assert.True(t, fullyCommitted)

		p.AddEvent(model.Event{
			Name:     "thing",
			Category: "cat",
			Start:    time.Date(2021, 1, 1, 14, 30, 0, 0, time.UTC),
			End:      time.Date(2021, 1, 1, 16, 45, 0, 0, time.UTC),
		})

		fullyCommitted, err = p.FullyCommitted()
		assert.Nil(t, err)
		assert.False(t, fullyCommitted)

		err = p.CommitState()
		assert.Nil(t, err)

		fullyCommitted, err = p.FullyCommitted()
		assert.Nil(t, err)
		assert.True(t, fullyCommitted)
	})

}

type testWriter struct {
	logFunc func(args ...any)
}

func (w testWriter) Write(p []byte) (n int, err error) {
	w.logFunc(strings.TrimSpace(string(p)))
	return len(p), nil
}

func validateUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
