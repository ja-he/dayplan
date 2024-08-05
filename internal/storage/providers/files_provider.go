package providers

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/model"
)

const notSameDayEventErrorMsg = string("event does not start and end on the same day")

var fileDateNamingRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

var filesProviderIDGenerator = func() model.EventID {
	return uuid.NewString()
}
var filesProviderIDValidator = func(id model.EventID) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// FilesDataProvider ...
type FilesDataProvider struct {
	BasePath string

	fhMutex      sync.RWMutex
	FileHandlers map[model.Date]*fileHandler

	eventsDateMapMtx sync.RWMutex
	eventsDateMap    map[model.EventID]model.Date

	log zerolog.Logger
}

// NewFilesDataProvider ...
func NewFilesDataProvider(
	basePath string,
) (*FilesDataProvider, error) {

	result := &FilesDataProvider{
		BasePath:      basePath,
		fhMutex:       sync.RWMutex{},
		FileHandlers:  make(map[model.Date]*fileHandler),
		eventsDateMap: make(map[model.EventID]model.Date),
		log:           log.With().Str("component", "files-data-provider").Logger(),
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
func (p *FilesDataProvider) AddEvent(e model.Event) (model.EventID, error) {
	if e.ID == "" {
		e.ID = filesProviderIDGenerator()
	} else {
		if !filesProviderIDValidator(e.ID) {
			return "", fmt.Errorf("invalid event ID")
		}
	}

	if !eventStartsAndEndsOnSameDate(&e) {
		return "", fmt.Errorf(notSameDayEventErrorMsg)
	}
	d := model.DateFromGotime(e.Start)
	fh, err := p.getFileHandler(d)
	if err != nil {
		return "", fmt.Errorf("error loading file handler for date (%w)", err)
	}
	fh.AddEvent(&e)
	return filesProviderIDGenerator(), nil
}

// RemoveEvent removes an event with the specified ID.
func (p *FilesDataProvider) RemoveEvent(id model.EventID) error {
	if !filesProviderIDValidator(id) {
		return fmt.Errorf("invalid event ID")
	}

	e, err := p.GetEvent(id)
	if err != nil {
		return fmt.Errorf("error getting event with ID '%s' for removal (%w)", id, err)
	}

	d := model.DateFromGotime(e.Start)
	fh, err := p.getFileHandler(d)
	if err != nil {
		return fmt.Errorf("error loading file handler for date (%w)", err)
	}

	fh.RemoveEvent(id)

	p.eventsDateMapMtx.Lock()
	delete(p.eventsDateMap, id)
	p.eventsDateMapMtx.Unlock()

	return nil
}

// RemoveEvents removes multiple events by their IDs.
func (p *FilesDataProvider) RemoveEvents(ids []model.EventID) error {
	for _, id := range ids {
		if err := p.RemoveEvent(id); err != nil {
			return fmt.Errorf("error removing event with ID '%s' (%w)", id, err)
		}
	}
	return nil
}

// GetEvent retrieves the event with the specified ID.
func (p *FilesDataProvider) GetEvent(id model.EventID) (*model.Event, error) {
	if !filesProviderIDValidator(id) {
		return nil, fmt.Errorf("invalid event ID")
	}

	p.log.Debug().Msgf("getting event with ID '%s'", id)
	defer p.log.Debug().Msgf("done getting event with ID '%s'", id)

	p.eventsDateMapMtx.RLock()
	d, ok := p.eventsDateMap[id]
	p.eventsDateMapMtx.RUnlock()

	if ok {
		p.log.Trace().Msgf("found event ID '%s' in map", id)
		fh, err := p.getFileHandler(d)
		if err != nil {
			return nil, fmt.Errorf("error getting file handler for date '%s' (%w)", d.String(), err)
		}
		e, err := fh.GetEvent(id)
		if err != nil {
			return nil, fmt.Errorf("error getting event with ID '%s' from file handler (%w)", id, err)
		}
		return e, nil
	}

	p.log.Trace().Msgf("will look for event with ID '%s' in files", id)
	e, d, err := p.getYetUnfoundEvent(id)
	if err != nil {
		return nil, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	p.log.Trace().Msgf("found date '%s' for event with ID '%s', will add to map", d.String(), id)
	p.eventsDateMapMtx.Lock()
	p.eventsDateMap[id] = d
	p.eventsDateMapMtx.Unlock()

	return e, nil

}

func (p *FilesDataProvider) getYetUnfoundEvent(id model.EventID) (*model.Event, model.Date, error) {
	availableDates, err := p.getAvailableDates()
	if err != nil {
		return nil, model.Date{}, fmt.Errorf("error getting available dates (%w)", err)
	}
	p.log.Trace().Msgf("have %d available dates", len(availableDates))

	for _, d := range availableDates {
		p.log.Trace().Msgf("getting file handler for date '%s'", d.String())
		fh, err := p.getFileHandler(d)
		if err != nil {
			return nil, model.Date{}, fmt.Errorf("error getting file handler for date '%s', which should not happen since the file should exist (%w)", d.String(), err)
		}
		for _, event := range fh.data.Events {
			if event.ID == id {
				p.log.Trace().Msgf("found event with ID '%s'", id)
				return event, d, nil
			}
		}
	}
	return nil, model.Date{}, fmt.Errorf("event with ID '%s' not found", id)
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

// GetPrecedingEvent retrieves the event immediately preceding the event with the specified ID.
func (p *FilesDataProvider) GetPrecedingEvent(id model.EventID) (*model.Event, error) {
	if !filesProviderIDValidator(id) {
		return nil, fmt.Errorf("invalid event ID")
	}

	// find out date for event
	p.eventsDateMapMtx.RLock()
	d, ok := p.eventsDateMap[id]
	p.eventsDateMapMtx.RUnlock()
	if !ok {
		var err error
		_, d, err = p.getYetUnfoundEvent(id)
		if err != nil {
			return nil, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
		}
		p.eventsDateMapMtx.Lock()
		p.eventsDateMap[id] = d
		p.eventsDateMapMtx.Unlock()
	}
	log.Debug().Msgf("found date '%s' for event with ID '%s'", d.String(), id)

	// get preceding event from file handler, if possible
	e, err := p.getPrevEventFromFH(d, id)
	if err != nil {
		return nil, fmt.Errorf("error getting preceding event for event with ID '%s' (%w)", id, err)
	}
	if e != nil {
		return e, nil
	}

	// get preceding event from the closesd previous day
	availableDates, err := p.getAvailableDates()
	if err != nil {
		return nil, fmt.Errorf("error getting available dates (%w)", err)
	}
	sort.Sort(model.DateSlice(availableDates))
	for i, date := range availableDates {
		if date == d {
			if i == 0 {
				// there is no previous available date
				return nil, nil
			}
			previousDate := availableDates[i-1]
			e, err := p.getPrevEventFromFH(previousDate, id)
			if err != nil {
				return nil, fmt.Errorf("error getting preceding event for event with ID '%s' (%w)", id, err)
			}
			return e, nil
		}
	}
	return nil, fmt.Errorf("could not find date '%s' in available dates even though it should be available", d.String())
}

func (p *FilesDataProvider) getPrevEventFromFH(d model.Date, id model.EventID) (*model.Event, error) {
	fh, err := p.getFileHandler(d)
	if err != nil {
		return nil, fmt.Errorf("error getting file handler for date '%s' (%w)", d.String(), err)
	}
	fh.mutex.Lock()
	defer fh.mutex.Unlock()
	eventIndex := -1
	for i, e := range fh.data.Events {
		if e.ID == id {
			eventIndex = i
			break
		}
	}
	if eventIndex == -1 {
		return nil, fmt.Errorf("event with ID '%s' not found in file handler for date '%s'", id, d.String())
	}
	if eventIndex == 0 {
		return nil, nil
	}
	return fh.data.Events[eventIndex-1], nil
}

func (p *FilesDataProvider) getNextEventFromFH(d model.Date, id model.EventID) (*model.Event, error) {
	fh, err := p.getFileHandler(d)
	if err != nil {
		return nil, fmt.Errorf("error getting file handler for date '%s' (%w)", d.String(), err)
	}
	fh.mutex.Lock()
	defer fh.mutex.Unlock()
	eventIndex := -1
	for i, e := range fh.data.Events {
		if e.ID == id {
			eventIndex = i
			break
		}
	}
	if eventIndex == -1 {
		return nil, fmt.Errorf("event with ID '%s' not found in file handler for date '%s'", id, d.String())
	}
	if eventIndex == len(fh.data.Events)-1 {
		return nil, nil
	}
	return fh.data.Events[eventIndex+1], nil
}

// TODO: doc GetFollowingEvent
func (p *FilesDataProvider) GetFollowingEvent(id model.EventID) (*model.Event, error) {
	if !filesProviderIDValidator(id) {
		return nil, fmt.Errorf("invalid event ID")
	}

	// find out date for event
	p.eventsDateMapMtx.RLock()
	d, ok := p.eventsDateMap[id]
	p.eventsDateMapMtx.RUnlock()
	if !ok {
		var err error
		_, d, err = p.getYetUnfoundEvent(id)
		if err != nil {
			return nil, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
		}
		p.eventsDateMapMtx.Lock()
		p.eventsDateMap[id] = d
		p.eventsDateMapMtx.Unlock()
	}
	log.Debug().Msgf("found date '%s' for event with ID '%s'", d.String(), id)

	// get following event from file handler, if possible
	e, err := p.getNextEventFromFH(d, id)
	if err != nil {
		return nil, fmt.Errorf("error getting preceding event for event with ID '%s' (%w)", id, err)
	}
	if e != nil {
		return e, nil
	}

	// get preceding event from the closesd previous day
	availableDates, err := p.getAvailableDates()
	if err != nil {
		return nil, fmt.Errorf("error getting available dates (%w)", err)
	}
	sort.Sort(model.DateSlice(availableDates))
	for i, date := range availableDates {
		if date == d {
			if i == len(availableDates)-1 {
				// there is no next available date
				return nil, nil
			}
			nextDate := availableDates[i+1]
			e, err := p.getPrevEventFromFH(nextDate, id)
			if err != nil {
				return nil, fmt.Errorf("error getting next event for event with ID '%s' (%w)", id, err)
			}
			return e, nil
		}
	}
	return nil, fmt.Errorf("could not find date '%s' in available dates even though it should be available", d.String())

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
func (p *FilesDataProvider) SplitEvent(model.EventID, time.Time) error {
	p.log.Fatal().Msg("TODO IMPL(SplitEvent)")
	return nil
}

// TODO: doc SetEventStart
func (p *FilesDataProvider) SetEventStart(model.EventID, time.Time) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventStart)")
	return nil
}

// TODO: doc SetEventEnd
func (p *FilesDataProvider) SetEventEnd(model.EventID, time.Time) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventEnd)")
	return nil
}

// TODO: doc SetEventTimes
func (p *FilesDataProvider) SetEventTimes(model.EventID, time.Time, time.Time) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventTimes)")
	return nil
}

// TODO: doc OffsetEventStart
func (p *FilesDataProvider) OffsetEventStart(model.EventID, time.Duration) (time.Time, error) {
	p.log.Fatal().Msg("TODO IMPL(OffsetEventStart)")
	return time.Time{}, nil
}

// TODO: doc OffsetEventEnd
func (p *FilesDataProvider) OffsetEventEnd(model.EventID, time.Duration) (time.Time, error) {
	p.log.Fatal().Msg("TODO IMPL(OffsetEventEnd)")
	return time.Time{}, nil
}

// TODO: doc OffsetEventTimes
func (p *FilesDataProvider) OffsetEventTimes(model.EventID, time.Duration) (time.Time, time.Time, error) {
	p.log.Fatal().Msg("TODO IMPL(OffsetEventTimes)")
	return time.Time{}, time.Time{}, nil
}

// TODO: doc SnapEventStart
func (p *FilesDataProvider) SnapEventStart(model.EventID, time.Duration) (time.Time, error) {
	p.log.Fatal().Msg("TODO IMPL(SnapEventStart)")
	return time.Time{}, nil
}

// TODO: doc SnapEventEnd
func (p *FilesDataProvider) SnapEventEnd(model.EventID, time.Duration) (time.Time, error) {
	p.log.Fatal().Msg("TODO IMPL(SnapEventEnd)")
	return time.Time{}, nil
}

// TODO: doc SnapEventTimes
func (p *FilesDataProvider) SnapEventTimes(model.EventID, time.Duration) (time.Time, time.Time, error) {
	p.log.Fatal().Msg("TODO IMPL(SnapEventTimes)")
	return time.Time{}, time.Time{}, nil
}

// TODO: doc SetEventTitle
func (p *FilesDataProvider) SetEventTitle(model.EventID, string) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventTitle)")
	return nil
}

// TODO: doc SetEventCategory
func (p *FilesDataProvider) SetEventCategory(model.EventID, model.CategoryName) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventCategory)")
	return nil
}

// TODO: doc SetEventAllData
func (p *FilesDataProvider) SetEventAllData(model.EventID, model.Event) error {
	p.log.Fatal().Msg("TODO IMPL(SetEventAllData)")
	return nil
}

// TODO: doc CommitState
func (p *FilesDataProvider) CommitState() error {
	p.log.Fatal().Msg("TODO IMPL(CommitState)")
	return nil
}

// TODO: doc SumUpTimespanByCategory
func (p *FilesDataProvider) SumUpTimespanByCategory(start time.Time, end time.Time) (map[model.CategoryName]time.Duration, error) {
	p.log.Fatal().Msg("TODO IMPL(SumUpTimespanByCategory)")
	return nil, nil
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
