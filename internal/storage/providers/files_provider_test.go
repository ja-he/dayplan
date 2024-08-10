package providers_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/storage"
	"github.com/ja-he/dayplan/internal/storage/providers"
)

func TestFilesProvider(t *testing.T) {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out: &testWriter{
			logFunc: t.Log,
		},
	})

	tempDir := t.TempDir()
	var p storage.DataProvider
	var err error
	p, err = providers.NewFilesDataProvider(tempDir)
	assert.Nil(t, err)

	t.Run("create-event", func(t *testing.T) {
		yearZero := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
		yearTenK := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		evs, err := p.GetEventsCoveringTimerange(yearZero, yearTenK)
		assert.Nil(t, err)
		assert.Empty(t, evs)
		id, err := p.AddEvent(model.Event{
			Name:     "thing",
			Category: "cat",
			Start:    time.Date(2021, 1, 1, 14, 30, 0, 0, time.UTC),
			End:      time.Date(2021, 1, 1, 16, 45, 0, 0, time.UTC),
		})
		assert.Nil(t, err)
		assert.True(t, validateUUID(id), "id '%s' is not a valid UUID", id)
		evs, err = p.GetEventsCoveringTimerange(yearZero, yearTenK)
		assert.Nil(t, err)
		assert.Len(t, evs, 1)
		assert.Equal(t, "thing", evs[0].Name)
		assert.Equal(t, model.CategoryName("cat"), evs[0].Category)
	})

	t.Run("remove-event", func(t *testing.T) {
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
			assert.Equal(t, event.Start, time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC))
		})

		t.Run("try-after-end", func(t *testing.T) {
			err = p.SetEventStart(id, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, event.Start, time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC))
		})

		t.Run("try-equal-end", func(t *testing.T) {
			err = p.SetEventStart(id, time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, event.Start, time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC))
		})

		// for he files provider we expect this not to work because it does not
		// support this.
		t.Run("try-different-date", func(t *testing.T) {
			err = p.SetEventStart(id, time.Date(2022, 12, 31, 14, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, event.Start, time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC))
		})
	})

	t.Run("set-event-end", func(t *testing.T) {
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
			assert.Equal(t, event.End, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
		})

		t.Run("try-before-start", func(t *testing.T) {
			err = p.SetEventEnd(id, time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, event.End, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
		})

		t.Run("try-equal-start", func(t *testing.T) {
			err = p.SetEventEnd(id, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, event.End, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
		})

		// for the files provider we expect this not to work because it does not
		// support this.
		t.Run("try-different-date", func(t *testing.T) {
			err = p.SetEventEnd(id, time.Date(2023, 1, 2, 14, 0, 0, 0, time.UTC))
			assert.NotNil(t, err)
			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, event.End, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
		})
	})

	t.Run("offset-event-start", func(t *testing.T) {
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
			assert.Equal(t, newStart, time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC))

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, event.Start, time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC))
		})

		t.Run("before-end", func(t *testing.T) {
			newStart, err := p.OffsetEventStart(id, -1*time.Minute)
			assert.Nil(t, err)
			assert.Equal(t, newStart, time.Date(2023, 1, 1, 12, 59, 0, 0, time.UTC))

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, event.Start, time.Date(2023, 1, 1, 12, 59, 0, 0, time.UTC))
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

		t.Run("invalid-duration", func(t *testing.T) {
			_, err := p.OffsetEventEnd(id, -3*time.Hour)
			assert.NotNil(t, err)
		})
	})

	t.Run("offset-event-times", func(t *testing.T) {
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
			assert.Equal(t, newStart, time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC))
			assert.Equal(t, newEnd, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))

			event, err := p.GetEvent(id)
			assert.Nil(t, err)
			assert.Equal(t, event.Start, time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC))
			assert.Equal(t, event.End, time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC))
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
