package providers

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/storage"
)

const notSameDayEventErrorMsg = string("event does not start and end on the same day")

var fileDateNamingRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// FilesDataProvider ...
type FilesDataProvider struct {
	BasePath string

	fhMutex      sync.RWMutex
	FileHandlers map[model.Date]*fileHandler

	log zerolog.Logger
}

// NewFilesDataProvider ...
func NewFilesDataProvider(
	basePath string,
) (*FilesDataProvider, error) {

	result := &FilesDataProvider{
		BasePath:     basePath,
		fhMutex:      sync.RWMutex{},
		FileHandlers: make(map[model.Date]*fileHandler),
		log:          log.With().Str("component", "files-data-provider").Logger(),
	}

	return result, nil
}

// TODO: doc getFileHandler
func (p *FilesDataProvider) getFileHandler(date model.Date) (*fileHandler, error) {

	// check if already loaded
	{
		p.fhMutex.RLock()
		if fh, ok := p.FileHandlers[date]; ok {
			p.fhMutex.RUnlock()
			p.log.Trace().Msgf("found already loaded file handler for '%s'", date.String())
			return fh, nil
		}
		p.fhMutex.RUnlock()
	}

	p.log.Trace().Msgf("file handler for '%s' not yet loaded, loading...", date.String())
	defer p.log.Trace().Msgf("loaded file handler for '%s'", date.String())

	// add the new handler
	p.fhMutex.Lock()
	defer p.fhMutex.Unlock()
	fh, err := newFileHandlerWithDataReadFromDisk(p.BasePath, date)
	if err != nil {
		return nil, fmt.Errorf("could not load file handler for '%s' (%w)", date.String(), err)
	}
	p.FileHandlers[date] = fh
	p.log.Trace().Msgf("file handler for '%s' added to cache", date.String())

	return fh, nil
}

// AddEvent ...
// TODO: doc AddEvent
func (p *FilesDataProvider) AddEvent(e model.Event) error {
	if !eventStartsAndEndsOnSameDate(&e) {
		return fmt.Errorf(notSameDayEventErrorMsg)
	}
	d := model.DateFromGotime(e.Start)
	fh, err := p.getFileHandler(d)
	if err != nil {
		return fmt.Errorf("error loading file handler for date (%w)", err)
	}
	fh.AddEvent(&e)
	return nil
}

// TODO: doc RemoveEvent
func (p *FilesDataProvider) RemoveEvent(storage.EventIdentifier) error {
	p.log.Fatal().Msg("TODO IMPL(RemoveEvent)")
	return nil
}

// TODO: doc RemoveEvents
func (p *FilesDataProvider) RemoveEvents([]storage.EventIdentifier) error {
	p.log.Fatal().Msg("TODO IMPL(RemoveEvents)")
	return nil
}

// GetEventAfter retrieves the first event after the specified time.
func (p *FilesDataProvider) GetEventAfter(t time.Time) (*model.Event, error) {
	p.log.Debug().Msgf("getting first event after %s", t.String())
	defer p.log.Debug().Msgf("done getting first event after %s", t.String())

	availableDates, err := p.getAvailableDates()
	if err != nil {
		return nil, fmt.Errorf("error getting available dates (%w)", err)
	}
	p.log.Trace().Msgf("have %d available dates", len(availableDates))

	sort.Sort(model.DateSlice(availableDates))

	dateForT := model.DateFromGotime(t)

	for _, d := range availableDates {
		if d.IsBefore(dateForT) {
			p.log.Trace().Msgf("skipping date '%s' because it is before the target time", d.String())
			continue
		}
		p.log.Trace().Msgf("getting file handler for date '%s'", d.String())
		fh, err := p.getFileHandler(d)
		if err != nil {
			return nil, fmt.Errorf("error getting file handler for date '%s', which should not happen since the file should exist (%w)", d.String(), err)
		}
		for _, event := range fh.data.Events {
			if event.Start == t || event.Start.After(t) {
				p.log.Trace().Msgf("found event starting after target time: %s", event.String())
				return event, nil
			}
		}
	}
	return nil, nil
}

// TODO: doc GetEventBefore
func (p *FilesDataProvider) GetEventBefore(t time.Time) (*model.Event, error) {
	p.log.Debug().Msgf("getting last event before %s", t.String())
	defer p.log.Debug().Msgf("done getting last event before %s", t.String())

	availableDates, err := p.getAvailableDates()
	if err != nil {
		return nil, fmt.Errorf("error getting available dates (%w)", err)
	}
	p.log.Trace().Msgf("have %d available dates", len(availableDates))

	sort.Sort(sort.Reverse(model.DateSlice(availableDates)))

	dateForT := model.DateFromGotime(t)

	for _, d := range availableDates {
		if d.IsAfter(dateForT) {
			p.log.Trace().Msgf("skipping date '%s' because it is after the target time", d.String())
			continue
		}
		p.log.Trace().Msgf("getting file handler for date '%s'", d.String())
		fh, err := p.getFileHandler(d)
		if err != nil {
			return nil, fmt.Errorf("error getting file handler for date '%s', which should not happen since the file should exist (%w)", d.String(), err)
		}
		for i := len(fh.data.Events) - 1; i >= 0; i-- {
			event := fh.data.Events[i]
			if event.End == t || event.End.Before(t) {
				p.log.Trace().Msgf("found event ending before target time: %s", event.String())
				return event, nil
			}
		}
	}
	return nil, nil
}

// TODO: doc GetPrecedingEvent
func (p *FilesDataProvider) GetPrecedingEvent(storage.EventIdentifier, time.Time) (*model.Event, error) {
	p.log.Fatal().Msg("TODO IMPL(GetPrecedingEvent)")
	return nil, nil
}

// TODO: doc GetFollowingEvent
func (p *FilesDataProvider) GetFollowingEvent(storage.EventIdentifier, time.Time) (*model.Event, error) {
	p.log.Fatal().Msg("TODO IMPL(GetFollowingEvent)")
	return nil, nil
}

// TODO: doc GetEventsCoveringTimerange
func (p *FilesDataProvider) GetEventsCoveringTimerange(start, end time.Time) ([]*model.Event, error) {
	p.log.Debug().Msgf("getting events covering timerange %s to %s", start.String(), end.String())
	defer log.Debug().Msgf("done getting events covering timerange %s to %s", start.String(), end.String())

	if end.Before(start) {
		return nil, fmt.Errorf("end time is before start time")
	}
	if start == end {
		return nil, fmt.Errorf("empty time range requested (start is end)")
	}

	fhs, err := func() ([]*fileHandler, error) {

		var result []*fileHandler
		currentDate := model.DateFromGotime(start)
		endDate := model.DateFromGotime(end)
		if end.Hour() == 0 && end.Minute() == 0 && end.Second() == 0 {
			endDate = endDate.Prev()
		}
		p.log.Debug().Msgf("getting file handlers for dates %s to %s", currentDate.String(), endDate.String())

		for !currentDate.IsAfter(endDate) {
			fh, err := p.getFileHandler(currentDate)
			if err != nil {
				return nil, fmt.Errorf("error getting file handler for date %s (%w)", currentDate.String(), err)
			}
			result = append(result, fh)
			currentDate = currentDate.Next()
		}
		return result, nil
	}()
	if err != nil {
		return nil, fmt.Errorf("error getting file handlers for timerange (%w)", err)
	}

	if len(fhs) == 0 {
		p.log.Warn().Msgf("somehow, found no file handlers for timerange %s to %s", start.String(), end.String())
		return nil, nil
	}

	p.log.Debug().Msgf("found %d file handlers for timerange %s to %s", len(fhs), start.String(), end.String())

	// NOTE:
	//   Yes, there is probably a small bit of efficiency to be gained here by
	//   only range checking on the first and last day, or treating the case of
	//   only having one day differently or ...
	//   YAGNI, for now, especially since this provider is probably on the way
	//   out.
	var events []*model.Event
	for _, fh := range fhs {
		fh.mutex.Lock()
		for _, e := range fh.data.Events {
			if !e.Start.Before(start) && !e.End.After(end) {
				events = append(events, e)
			}
		}
		fh.mutex.Unlock()
	}
	return events, nil
}

// TODO: doc SplitEvent
func (p *FilesDataProvider) SplitEvent(storage.EventIdentifier, time.Time) error {
	p.log.Fatal().Msg("TODO IMPL(SplitEvent)")
	return nil
}

// TODO: doc SetEventStart
func (p *FilesDataProvider) SetEventStart(storage.EventIdentifier, time.Time) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventStart)")
	return nil
}

// TODO: doc SetEventEnd
func (p *FilesDataProvider) SetEventEnd(storage.EventIdentifier, time.Time) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventEnd)")
	return nil
}

// TODO: doc SetEventTimes
func (p *FilesDataProvider) SetEventTimes(storage.EventIdentifier, time.Time, time.Time) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventTimes)")
	return nil
}

// TODO: doc OffsetEventStart
func (p *FilesDataProvider) OffsetEventStart(storage.EventIdentifier, time.Duration) error {
	p.log.Fatal().Msg("TODO IMPL(OffsetEventStart)")
	return nil
}

// TODO: doc OffsetEventEnd
func (p *FilesDataProvider) OffsetEventEnd(storage.EventIdentifier, time.Duration) error {
	p.log.Fatal().Msg("TODO IMPL(OffsetEventEnd)")
	return nil
}

// TODO: doc OffsetEventTimes
func (p *FilesDataProvider) OffsetEventTimes(storage.EventIdentifier, time.Duration) error {
	p.log.Fatal().Msg("TODO IMPL(OffsetEventTimes)")
	return nil
}

// TODO: doc SnapEventStart
func (p *FilesDataProvider) SnapEventStart(storage.EventIdentifier, time.Duration) error {
	p.log.Fatal().Msg("TODO IMPL(SnapEventStart)")
	return nil
}

// TODO: doc SnapEventEnd
func (p *FilesDataProvider) SnapEventEnd(storage.EventIdentifier, time.Duration) error {
	p.log.Fatal().Msg("TODO IMPL(SnapEventEnd)")
	return nil
}

// TODO: doc SnapEventTimes
func (p *FilesDataProvider) SnapEventTimes(storage.EventIdentifier, time.Duration) error {
	p.log.Fatal().Msg("TODO IMPL(SnapEventTimes)")
	return nil
}

// TODO: doc SetEventTitle
func (p *FilesDataProvider) SetEventTitle(storage.EventIdentifier, string) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventTitle)")
	return nil
}

// TODO: doc SetEventCategory
func (p *FilesDataProvider) SetEventCategory(storage.EventIdentifier, model.Category) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventCategory)")
	return nil
}

// TODO: doc SetEventAllData
func (p *FilesDataProvider) SetEventAllData(storage.EventIdentifier, model.Event) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventAllData)")
	return nil
}

// TODO: doc CommitState
func (p *FilesDataProvider) CommitState() error {
	p.log.Fatal().Msg("TODO IMPL(CommitState)")
	return nil
}

// TODO: doc SumUpTimespanByCategory
func (p *FilesDataProvider) SumUpTimespanByCategory(start time.Time, end time.Time) map[model.CategoryName]time.Duration {
	p.log.Fatal().Msg("TODO IMPL(SumUpTimespanByCategory)")
	return nil
}

func eventStartsAndEndsOnSameDate(e *model.Event) bool {
	return timesOnSameDate(e.Start, e.End)
}

func timesOnSameDate(a, b time.Time) bool {
	return a.Year() != b.Year() || a.YearDay() != b.YearDay()
}

// NOTE: this function is fine, but its use could be improved, because we really should only need to call this once
func (p *FilesDataProvider) getAvailableDates() ([]model.Date, error) {
	p.log.Debug().Msg("getting available dates")
	defer p.log.Debug().Msg("done getting available dates")

	files, err := os.ReadDir(p.BasePath)
	if err != nil {
		return nil, fmt.Errorf("error reading directory (%w)", err)
	}
	var dates []model.Date
	for _, f := range files {
		if f.IsDir() {
			p.log.Trace().Msgf("skipping directory '%s'", f.Name())
			continue
		}
		if !fileDateNamingRegex.MatchString(f.Name()) {
			p.log.Trace().Msgf("skipping non-date file '%s'", f.Name())
			continue
		}
		d, err := model.DateFromString(f.Name())
		if err != nil {
			return nil, fmt.Errorf("error parsing date from file name '%s' (%w)", f.Name(), err)
		}
		dates = append(dates, d)
	}
	return dates, nil

}
