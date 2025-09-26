package providers

import (
	"errors"
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
	"github.com/ja-he/dayplan/internal/storage"
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

	categoryProvider storage.CategoryProvider

	log zerolog.Logger
}

// NewFilesDataProvider ...
func NewFilesDataProvider(
	basePath string,
	categoryProvider storage.CategoryProvider,
) (*FilesDataProvider, error) {

	result := &FilesDataProvider{
		BasePath:         basePath,
		fhMutex:          sync.RWMutex{},
		FileHandlers:     make(map[model.Date]*fileHandler),
		eventsDateMap:    make(map[model.EventID]model.Date),
		categoryProvider: categoryProvider,
		log:              log.Level(zerolog.WarnLevel).With().Str("component", "files-data-provider").Logger(),
	}
	result.log.Debug().Msgf("created new files data provider with base path '%s'", basePath)

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

func (p *FilesDataProvider) setEventDateInMap(id model.EventID, date model.Date) {
	p.eventsDateMapMtx.Lock()
	p.eventsDateMap[id] = date
	p.eventsDateMapMtx.Unlock()
}

func (p *FilesDataProvider) getEventDateFromMap(id model.EventID) (model.Date, bool) {
	p.eventsDateMapMtx.RLock()
	d, ok := p.eventsDateMap[id]
	p.eventsDateMapMtx.RUnlock()
	return d, ok
}

// AddEvent ...
// TODO: doc AddEvent
func (p *FilesDataProvider) AddEvent(e model.Event) (model.EventID, error) {
	if e.ID == "" {
		generatedID := filesProviderIDGenerator()
		e.ID = generatedID
		p.log.Debug().Msgf("generated ID '%s' for event", generatedID)
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
	if err := fh.AddEvent(&e); err != nil {
		return "", fmt.Errorf("Unable to add event to day's (%s) file handler (%w).", d, err)
	}
	return e.ID, nil
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
	e, err := p.GetEventRaw(id)
	if err != nil {
		return nil, err
	}
	clonedEvent := e.Clone()
	return &clonedEvent, nil
}
func (p *FilesDataProvider) GetEventRaw(id model.EventID) (*model.Event, error) {
	_, e, err := p.getEventWithFH(id)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (p *FilesDataProvider) getEventWithFH(id model.EventID) (*fileHandler, *model.Event, error) {
	if !filesProviderIDValidator(id) {
		return nil, nil, fmt.Errorf("invalid event ID")
	}

	p.log.Debug().Msgf("getting event with ID '%s'", id)
	defer p.log.Debug().Msgf("done getting event with ID '%s'", id)

	d, ok := p.getEventDateFromMap(id)

	if ok {
		p.log.Trace().Msgf("found event ID '%s' in map for date %s", id, d)
		fh, err := p.getFileHandler(d)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting file handler for date '%s' (%w)", d.String(), err)
		}
		e, err := fh.GetEvent(id)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting event with ID '%s' from file handler %s (%w)", id, fh.date, err)
		}
		return fh, e, nil
	}

	p.log.Trace().Msgf("will look for event with ID '%s' in files", id)
	fh, e, d, err := p.getYetUnfoundEvent(id)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	p.log.Trace().Msgf("found date '%s' for event with ID '%s', will add to map", d.String(), id)
	p.setEventDateInMap(id, d)

	return fh, e, nil

}

func (p *FilesDataProvider) getYetUnfoundEvent(id model.EventID) (*fileHandler, *model.Event, model.Date, error) {
	availableDates, err := p.getAvailableDates()
	if err != nil {
		return nil, nil, model.Date{}, fmt.Errorf("error getting available dates (%w)", err)
	}
	p.log.Trace().Msgf("have %d available dates", len(availableDates))

	for _, d := range availableDates {
		p.log.Trace().Msgf("getting file handler for date '%s'", d.String())
		fh, err := p.getFileHandler(d)
		if err != nil {
			return nil, nil, model.Date{}, fmt.Errorf("error getting file handler for date '%s', which should not happen since the file should exist (%w)", d.String(), err)
		}
		for _, event := range fh.data.Events {
			if event.ID == id {
				p.log.Trace().Msgf("found event with ID '%s'", id)
				return fh, event, d, nil
			}
		}
	}
	return nil, nil, model.Date{}, fmt.Errorf("event with ID '%s' not found", id)
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
	d, ok := p.getEventDateFromMap(id)
	if !ok {
		var err error
		_, _, d, err = p.getYetUnfoundEvent(id)
		if err != nil {
			return nil, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
		}
		p.setEventDateInMap(id, d)
	}
	p.log.Debug().Msgf("found date '%s' for event with ID '%s'", d.String(), id)

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
	dateIndex := -1
	for i, date := range availableDates {
		if date == d {
			dateIndex = i
			break
		}
	}
	if dateIndex == -1 {
		return nil, fmt.Errorf("could not find date '%s' in available dates even though it should be available", d.String())
	}
	for i := dateIndex - 1; i >= 0; i-- {
		precedingDate := availableDates[i]
		e, err := p.getLastEventFromFH(precedingDate)
		if err != nil {
			return nil, fmt.Errorf("error getting preceding event for event with ID '%s' (%w)", id, err)
		}
		if e != nil {
			return e, nil
		}
	}

	return nil, nil
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
	d, ok := p.getEventDateFromMap(id)
	if !ok {
		p.log.Trace().Msgf("have to find date for event with ID '%s'", id)
		var err error
		_, _, d, err = p.getYetUnfoundEvent(id)
		if err != nil {
			return nil, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
		}
		p.setEventDateInMap(id, d)
	}
	log.Debug().Msgf("found date '%s' for event with ID '%s'", d.String(), id)

	// get following event from file handler, if possible
	e, err := p.getNextEventFromFH(d, id)
	if err != nil {
		return nil, fmt.Errorf("error getting next event for event with ID '%s' (%w)", id, err)
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
	dateIndex := -1
	for i, date := range availableDates {
		if date == d {
			dateIndex = i
			break
		}
	}
	if dateIndex == -1 {
		return nil, fmt.Errorf("could not find date '%s' in available dates even though it should be available", d.String())
	}

	for i := dateIndex + 1; i < len(availableDates); i++ {
		followingDate := availableDates[i]
		e, err := p.getFirstEventFromFH(followingDate)
		if err != nil {
			return nil, fmt.Errorf("error getting next event for event with ID '%s' (%w)", id, err)
		}
		if e != nil {
			return e, nil
		}
	}

	return nil, nil
}

func (p *FilesDataProvider) getFirstEventFromFH(d model.Date) (*model.Event, error) {
	fh, err := p.getFileHandler(d)
	if err != nil {
		return nil, fmt.Errorf("error getting file handler for date '%s' (%w)", d.String(), err)
	}
	fh.mutex.Lock()
	defer fh.mutex.Unlock()
	if len(fh.data.Events) == 0 {
		return nil, nil
	}
	return fh.data.Events[0], nil
}

func (p *FilesDataProvider) getLastEventFromFH(d model.Date) (*model.Event, error) {
	fh, err := p.getFileHandler(d)
	if err != nil {
		return nil, fmt.Errorf("error getting file handler for date '%s' (%w)", d.String(), err)
	}
	fh.mutex.Lock()
	defer fh.mutex.Unlock()
	if len(fh.data.Events) == 0 {
		return nil, nil
	}
	return fh.data.Events[len(fh.data.Events)-1], nil
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
		startDate := model.DateFromGotime(start)
		endDate := model.DateFromGotime(end)
		if end.Hour() == 0 && end.Minute() == 0 && end.Second() == 0 {
			endDate = endDate.Prev()
		}
		p.log.Debug().Msgf("getting file handlers for dates %s to %s", startDate.String(), endDate.String())

		availableDates, err := p.getAvailableDates()
		if err != nil {
			return nil, fmt.Errorf("error getting available dates (%w)", err)
		}
		for _, d := range availableDates {
			p.log.Trace().Msgf("checking date '%s'", d.String())
			if !d.IsBefore(startDate) && !d.IsAfter(endDate) {
				fh, err := p.getFileHandler(d)
				if err != nil {
					return nil, fmt.Errorf("error getting file handler for date %s (%w)", startDate.String(), err)
				}
				result = append(result, fh)
			}
		}

		return result, nil
	}()
	if err != nil {
		return nil, fmt.Errorf("error getting file handlers for timerange (%w)", err)
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
func (p *FilesDataProvider) SplitEvent(id model.EventID, splitTime time.Time) error {

	e, err := p.GetEvent(id)
	if err != nil {
		return fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	p.log.Debug().Msgf("will try to split event '%s' (%s til %s) at %s", id, e.Start, e.End, splitTime.String())

	if !(splitTime.After(e.Start) && splitTime.Before(e.End)) {
		return fmt.Errorf("split time is not between start and end time of event in question")
	}

	fh, err := p.getFileHandler(model.DateFromGotime(e.Start))
	if err != nil {
		return fmt.Errorf("error loading file handler for date (%w)", err)
	}

	firstHalfEvent := e
	secondHalfEvent := *e

	firstHalfEvent.End = splitTime
	secondHalfEvent.Start = splitTime
	secondHalfEvent.ID = filesProviderIDGenerator()

	err = fh.UpdateEvent(firstHalfEvent)
	if err != nil {
		return fmt.Errorf("error updating first half event (%w)", err)
	}
	err = fh.AddEvent(&secondHalfEvent)
	if err != nil {
		return fmt.Errorf("error adding second half event (%w)", err)
	}

	return nil
}

// SetEventStart sets the start time of an event with a specific ID.
func (p *FilesDataProvider) SetEventStart(id model.EventID, start time.Time) error {
	e, err := p.GetEvent(id)
	if err != nil {
		return fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	if !start.Before(e.End) {
		return fmt.Errorf("start time is not before end time")
	}

	// Ensure start and end are on the same date
	if !timesOnSameDate(start, e.End) {
		return fmt.Errorf(notSameDayEventErrorMsg)
	}

	e.Start = start

	fh, err := p.getFileHandler(model.DateFromGotime(start))
	if err != nil {
		return fmt.Errorf("error loading file handler for date (%w)", err)
	}

	if err := fh.UpdateEvent(e); err != nil {
		return fmt.Errorf("TODO (%w)", err)
	}
	return nil
}

// SetEventEnd sets the end time of an event with a specific ID.
func (p *FilesDataProvider) SetEventEnd(id model.EventID, end time.Time) error {
	e, err := p.GetEvent(id)
	if err != nil {
		return fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	if !e.Start.Before(end) {
		return fmt.Errorf("start time is not before end time")
	}

	// Ensure start and end are on the same date
	if !timesOnSameDate(e.Start, end) {
		return fmt.Errorf(notSameDayEventErrorMsg)
	}

	e.End = end
	fh, err := p.getFileHandler(model.DateFromGotime(end))
	if err != nil {
		return fmt.Errorf("error loading file handler for date (%w)", err)
	}

	if err := fh.UpdateEvent(e); err != nil {
		return fmt.Errorf("TODO (%w)", err)
	}
	return nil
}

// TODO: doc SetEventTimes
func (p *FilesDataProvider) SetEventTimes(id model.EventID, newStart time.Time, newEnd time.Time) error {
	if !newStart.Before(newEnd) {
		return fmt.Errorf("start time is not before end time")
	}

	if !timesOnSameDate(newStart, newEnd) {
		return fmt.Errorf(notSameDayEventErrorMsg)
	}

	fh, e, err := p.getEventWithFH(id)
	if err != nil {
		return fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	// if new times are on the same date as old times we just update the event
	// with the file handler for that date
	if timesOnSameDate(e.Start, newStart) {
		e.Start = newStart
		e.End = newEnd

		if err := fh.UpdateEvent(e); err != nil {
			return fmt.Errorf("TODO (%w)", err)
		}
		return nil
	}

	// otherwise we need to remove the event from this file handler and add it to
	// the new one
	if err := fh.RemoveEvent(id); err != nil {
		return fmt.Errorf("error removing event with ID '%s' from file handler (%w)", id, err)
	}
	tryToAddBackDueToError := func() {
		if err := fh.AddEvent(e); err != nil {
			p.log.Warn().Msgf("error adding event back to file handler after (another) error: %v", err)
		}
	}

	eventClone := *e
	eventClone.Start = newStart
	eventClone.End = newEnd

	newDate := model.DateFromGotime(newStart)
	newFH, err := p.getFileHandler(newDate)
	if err != nil {
		tryToAddBackDueToError()
		return fmt.Errorf("error loading file handler for date (%w)", err)
	}
	if err := newFH.AddEvent(&eventClone); err != nil {
		tryToAddBackDueToError()
		return fmt.Errorf("error adding event to new file handler (%w)", err)
	}

	p.setEventDateInMap(id, newDate)

	return nil
}

// OffsetEventStart offsets the start time of an event with the specified ID by a duration.
func (p *FilesDataProvider) OffsetEventStart(id model.EventID, offset time.Duration) (time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	newStart := e.Start.Add(offset)
	if !newStart.Before(e.End) {
		return time.Time{}, fmt.Errorf("resulting start time would not be before end time")
	}

	if !timesOnSameDate(newStart, e.End) {
		return time.Time{}, fmt.Errorf(notSameDayEventErrorMsg)
	}

	e.Start = newStart

	fh, err := p.getFileHandler(model.DateFromGotime(e.Start))
	if err != nil {
		return time.Time{}, fmt.Errorf("error loading file handler for date (%w)", err)
	}

	if err := fh.UpdateEvent(e); err != nil {
		return time.Time{}, fmt.Errorf("TODO (%w)", err)
	}
	return e.Start, nil
}

// OffsetEventEnd offsets the end time of an event with the specified ID by a duration.
func (p *FilesDataProvider) OffsetEventEnd(id model.EventID, offset time.Duration) (time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	newEnd := e.End.Add(offset)
	if !e.Start.Before(newEnd) {
		return time.Time{}, fmt.Errorf("resulting end time would not be after start time")
	}

	if !timesOnSameDate(e.Start, newEnd) {
		return time.Time{}, fmt.Errorf(notSameDayEventErrorMsg)
	}

	e.End = newEnd

	fh, err := p.getFileHandler(model.DateFromGotime(e.Start))
	if err != nil {
		return time.Time{}, fmt.Errorf("error loading file handler for date (%w)", err)
	}

	if err := fh.UpdateEvent(e); err != nil {
		return time.Time{}, fmt.Errorf("Could not update event (%w)", err)
	}
	return e.End, nil
}

// OffsetEventTimes offsets both the start and end times of an event with the specified ID by a duration.
func (p *FilesDataProvider) OffsetEventTimes(id model.EventID, offset time.Duration) (time.Time, time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	newStart := e.Start.Add(offset)
	newEnd := e.End.Add(offset)

	// Ensure start and end are on the same date
	if !timesOnSameDate(newStart, newEnd) {
		return time.Time{}, time.Time{}, fmt.Errorf(notSameDayEventErrorMsg)
	}

	fh, err := p.getFileHandler(model.DateFromGotime(e.Start))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error loading file handler for date (%w)", err)
	}

	// The times are on the same date, but the new times are not on the same date as the old times.
	moveToOtherDay := !timesOnSameDate(e.Start, newEnd)

	e.Start = newStart
	e.End = newEnd

	if moveToOtherDay {
		oldFileHandler := fh
		newFileHandler, err := p.getFileHandler(model.DateFromGotime(newStart))

		err = oldFileHandler.RemoveEvent(e.ID)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("Unable to remove event %s from file handler %s (%w)", e.ID, oldFileHandler.date, err)
		}

		addErr := newFileHandler.AddEvent(e)
		if addErr != nil {
			addErr2 := oldFileHandler.AddEvent(e)
			if addErr2 != nil {
				return time.Time{}, time.Time{}, fmt.Errorf("Unable to add event %s to FH %s and then unable to even re-add to %s (%w; %w).", e.ID, newFileHandler.date, oldFileHandler.date, addErr, addErr2)
			}
			return time.Time{}, time.Time{}, fmt.Errorf("Unable to add event %s to FH %s (%w).", e.ID, newFileHandler.date, addErr)
		}
		p.setEventDateInMap(e.ID, newFileHandler.date)

	} else {
		err = fh.UpdateEvent(e)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("unable to update event with new times (%w)", err)
		}
	}

	return e.Start, e.End, nil
}

// SnapEventStart snaps the start time of an event with the specified ID to the nearest interval.
func (p *FilesDataProvider) SnapEventStart(id model.EventID, interval time.Duration) (time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	newStart := snapToInterval(e.Start, interval)

	if !timesOnSameDate(newStart, e.End) {
		return time.Time{}, fmt.Errorf(notSameDayEventErrorMsg)
	}

	if !newStart.Before(e.End) {
		return time.Time{}, fmt.Errorf("resulting start time would not be before end time")
	}

	e.Start = newStart

	fh, err := p.getFileHandler(model.DateFromGotime(e.Start))
	if err != nil {
		return time.Time{}, fmt.Errorf("error loading file handler for date (%w)", err)
	}

	if err := fh.UpdateEvent(e); err != nil {
		return time.Time{}, fmt.Errorf("TODO (%w)", err)
	}
	return e.Start, nil
}

// SnapEventEnd snaps the end time of an event with the specified ID to the nearest interval.
func (p *FilesDataProvider) SnapEventEnd(id model.EventID, interval time.Duration) (time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	newEnd := snapToInterval(e.End, interval)

	if !eventStartsAndEndsOnSameDate(e) {
		return time.Time{}, fmt.Errorf(notSameDayEventErrorMsg)
	}

	if !e.Start.Before(newEnd) {
		return time.Time{}, fmt.Errorf("resulting end time would not be after start time")
	}

	e.End = newEnd

	fh, err := p.getFileHandler(model.DateFromGotime(e.End))
	if err != nil {
		return time.Time{}, fmt.Errorf("error loading file handler for date (%w)", err)
	}

	if err := fh.UpdateEvent(e); err != nil {
		return time.Time{}, fmt.Errorf("TODO (%w)", err)
	}
	return e.End, nil
}

// SnapEventTimes snaps both the start and end times of an event with the specified ID to the nearest interval.
func (p *FilesDataProvider) SnapEventTimes(id model.EventID, interval time.Duration) (time.Time, time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}

	newStart := snapToInterval(e.Start, interval)
	newEnd := snapToInterval(e.End, interval)

	if !timesOnSameDate(newStart, newEnd) {
		return time.Time{}, time.Time{}, fmt.Errorf(notSameDayEventErrorMsg)
	}

	if !newStart.Before(newEnd) {
		return time.Time{}, time.Time{}, fmt.Errorf("resulting start time would not be before end time")
	}

	e.Start, e.End = newStart, newEnd

	fh, err := p.getFileHandler(model.DateFromGotime(e.Start))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("error loading file handler for date (%w)", err)
	}

	if err := fh.UpdateEvent(e); err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("TODO (%w)", err)
	}
	return e.Start, e.End, nil
}

func (p *FilesDataProvider) SetEventName(id model.EventID, newName string) error {
	fh, e, err := p.getEventWithFH(id)
	if err != nil {
		return fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}
	// TODO: might be advisable to check that the new name matches some character set
	eClone := *e
	eClone.Name = newName
	err = fh.UpdateEvent(&eClone)
	if err != nil {
		return fmt.Errorf("error updating event with ID '%s' (%w)", id, err)
	}
	return nil
}

// TODO: doc SetEventCategory
func (p *FilesDataProvider) SetEventCategory(id model.EventID, newCatName model.CategoryName) error {
	fh, e, err := p.getEventWithFH(id)
	if err != nil {
		return fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}
	// TODO: might be advisable to check that the new name matches some character set
	eClone := *e
	eClone.Category = newCatName
	err = fh.UpdateEvent(&eClone)
	if err != nil {
		return fmt.Errorf("error updating event with ID '%s' (%w)", id, err)
	}
	return nil
}

// TODO: doc SetEventAllData
func (p *FilesDataProvider) SetEventAllData(id model.EventID, newEventData model.Event) error {
	if newEventData.ID != "" && newEventData.ID != id {
		return fmt.Errorf("new event data has different ID than specified")
	}
	if !timesOnSameDate(newEventData.Start, newEventData.End) {
		return fmt.Errorf(notSameDayEventErrorMsg)
	}

	newEventData.ID = id // ensure ID is correct
	fh, e, err := p.getEventWithFH(id)
	if err != nil {
		return fmt.Errorf("error getting event with ID '%s' (%w)", id, err)
	}
	if timesOnSameDate(e.Start, newEventData.Start) {
		err = fh.UpdateEvent(&newEventData)
		if err != nil {
			return fmt.Errorf("error updating event with ID '%s' (%w)", id, err)
		}
		return nil
	}

	err = fh.RemoveEvent(id)
	if err != nil {
		return fmt.Errorf("error removing event with ID '%s' (%w)", id, err)
	}
	addBackDueToError := func() {
		if err := fh.AddEvent(e); err != nil {
			p.log.Warn().Msgf("error adding event back to file handler after (another) error: %v", err)
		}
	}

	newStartDate := model.DateFromGotime(newEventData.Start)
	newFH, err := p.getFileHandler(newStartDate)
	if err != nil {
		addBackDueToError()
		return fmt.Errorf("error loading file handler for date (%w)", err)
	}

	err = newFH.AddEvent(&newEventData)
	if err != nil {
		addBackDueToError()
		return fmt.Errorf("error adding event to new file handler (%w)", err)
	}

	p.setEventDateInMap(id, newStartDate)

	return nil
}

// CommitState iterates all file handlers and compels them to commit their
// state to disk if necessary.
func (p *FilesDataProvider) CommitState() error {
	var errs []error
	for _, fh := range p.FileHandlers {
		err := fh.Write()
		if err != nil {
			errs = append(errs, fmt.Errorf("Unable to write %s (%w).", fh.Filename(), err))
		}
	}
	return errors.Join(errs...)
}

// FullyCommitted returns whether each file handler has its state fully
// committed to disk. In the case that on any file handler a change was made
// that has not yet been written, this will return false.
func (p *FilesDataProvider) FullyCommitted() (bool, error) {
	p.fhMutex.RLock()
	defer p.fhMutex.RUnlock()
	for _, fh := range p.FileHandlers {
		if !fh.OnDiskIsUpToDate() {
			return false, nil
		}
	}
	return true, nil
}

// TODO: doc SumUpTimespanByCategory
func (p *FilesDataProvider) SumUpTimespanByCategory(start time.Time, end time.Time) (map[model.CategoryName]time.Duration, error) {
	fullEventList := model.EventList{
		Events: []*model.Event{},
	}

	allDates, err := p.getAvailableDates()
	if err != nil {
		return nil, fmt.Errorf("error getting available dates (%w)", err)
	}

	sort.Sort(model.DateSlice(allDates))

	firstDateIndex, afterLastDateIndex := -1, -1
	for i, d := range allDates {
		if firstDateIndex == -1 {
			if !d.IsAfter(model.DateFromGotime(start)) {
				firstDateIndex = i
			}
		}
		if d.IsAfter(model.DateFromGotime(end)) {
			afterLastDateIndex = i
			break
		}
	}
	if firstDateIndex == -1 || afterLastDateIndex == -1 {
		return nil, fmt.Errorf("could not find first or last date in available dates")
	}

	datesInRange := allDates[firstDateIndex:afterLastDateIndex]

	for _, d := range datesInRange {
		fh, err := p.getFileHandler(d)
		if err != nil {
			return nil, fmt.Errorf("error getting file handler for date '%s' (%w)", d.String(), err)
		}
		for _, e := range fh.data.Events {
			fullEventList.AddEvent(e)
		}
	}

	summary := fullEventList.SumUpByCategory(func(category model.CategoryName) int {
		c := p.categoryProvider.GetCategory(category)
		if c == nil {
			p.log.Warn().Msgf("category '%s' not found in category provider", category)
			return 0
		}
		return c.Priority
	})

	return summary, nil
}

func eventStartsAndEndsOnSameDate(e *model.Event) bool {
	return timesOnSameDate(e.Start, e.End)
}

func timesOnSameDate(a, b time.Time) bool {
	if a.After(b) {
		a, b = b, a
	}
	if isMidnight(b) && (!isMidnight(a) || a.YearDay() < b.YearDay()) {
		b = b.Add(-1)
	}
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}
func isMidnight(t time.Time) bool {
	return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
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

	// get all handler's dates
	p.fhMutex.RLock()
	for d := range p.FileHandlers {
		dates = append(dates, d)
	}
	p.fhMutex.RUnlock()

	// add remaining dates (files with no handlers yet)
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
		p.fhMutex.RLock()
		_, haveFH := p.FileHandlers[d]
		p.fhMutex.RUnlock()
		if !haveFH {
			dates = append(dates, d)
		}
	}
	return dates, nil

}

func snapToInterval(t time.Time, interval time.Duration) time.Time {
	startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	durationFromStartOfDay := t.Sub(startOfDay)

	intervals := durationFromStartOfDay / interval
	remainder := durationFromStartOfDay % interval

	if remainder < interval/2 {
		return startOfDay.Add(intervals * interval)
	}
	return startOfDay.Add((intervals + 1) * interval)
}
