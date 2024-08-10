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
