package providers

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/storage"
)

const notSameDayEventErrorMsg = string("event does not start and end on the same day")

// FilesDataProvider ...
type FilesDataProvider struct {
	BasePath string

	fhMutex      sync.RWMutex
	FileHandlers map[model.Date]*FileHandler

	categories []model.Category
}

// NewFilesDataProvider ...
func NewFilesDataProvider(
	basePath string,
	categories []model.Category,
) (*FilesDataProvider, error) {

	result := &FilesDataProvider{
		BasePath:     basePath,
		fhMutex:      sync.RWMutex{},
		FileHandlers: make(map[model.Date]*FileHandler),
		categories:   categories,
	}

	return result, nil
}

// TODO: doc getFileHandler
func (p *FilesDataProvider) getFileHandler(date model.Date) (*FileHandler, error) {

	// check if already loaded
	{
		p.fhMutex.RLock()
		if fh, ok := p.FileHandlers[date]; ok {
			return fh, nil
		}
		p.fhMutex.RUnlock()
	}

	// add the new handler
	p.fhMutex.Lock()
	fh := NewFileHandler(p.BasePath, date)
	p.FileHandlers[date] = fh
	p.fhMutex.Unlock()

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
	panic("TODO IMPL(RemoveEvent)")
}

// TODO: doc RemoveEvents
func (p *FilesDataProvider) RemoveEvents([]storage.EventIdentifier) error {
	panic("TODO IMPL(RemoveEvents)")
}

// TODO: doc GetEventAfter
func (p *FilesDataProvider) GetEventAfter(time.Time) (*model.Event, error) {
	panic("TODO IMPL(GetEventAfter)")
}

// TODO: doc GetEventBefore
func (p *FilesDataProvider) GetEventBefore(time.Time) (*model.Event, error) {
	panic("TODO IMPL(GetEventBefore)")
}

// TODO: doc GetPrecedingEvent
func (p *FilesDataProvider) GetPrecedingEvent(storage.EventIdentifier, time.Time) (*model.Event, error) {
	panic("TODO IMPL(GetPrecedingEvent)")
}

// TODO: doc GetFollowingEvent
func (p *FilesDataProvider) GetFollowingEvent(storage.EventIdentifier, time.Time) (*model.Event, error) {
	panic("TODO IMPL(GetFollowingEvent)")
}

// TODO: doc GetEventsCoveringTimerange
func (p *FilesDataProvider) GetEventsCoveringTimerange(start, end time.Time) ([]*model.Event, error) {
	panic("TODO IMPL(GetEventsCoveringTimerange)")
}

// TODO: doc SplitEvent
func (p *FilesDataProvider) SplitEvent(storage.EventIdentifier, time.Time) error {
	panic("TODO IMPL(SplitEvent)")
}

// TODO: doc SetEventStart
func (p *FilesDataProvider) SetEventStart(storage.EventIdentifier, time.Time) error {
	panic("TODO IMPL(SetEventStart)")
}

// TODO: doc SetEventEnd
func (p *FilesDataProvider) SetEventEnd(storage.EventIdentifier, time.Time) error {
	panic("TODO IMPL(SetEventEnd)")
}

// TODO: doc SetEventTimes
func (p *FilesDataProvider) SetEventTimes(storage.EventIdentifier, time.Time, time.Time) error {
	panic("TODO IMPL(SetEventTimes)")
}

// TODO: doc OffsetEventStart
func (p *FilesDataProvider) OffsetEventStart(storage.EventIdentifier, time.Duration) error {
	panic("TODO IMPL(OffsetEventStart)")
}

// TODO: doc OffsetEventEnd
func (p *FilesDataProvider) OffsetEventEnd(storage.EventIdentifier, time.Duration) error {
	panic("TODO IMPL(OffsetEventEnd)")
}

// TODO: doc OffsetEventTimes
func (p *FilesDataProvider) OffsetEventTimes(storage.EventIdentifier, time.Duration) error {
	panic("TODO IMPL(OffsetEventTimes)")
}

// TODO: doc SnapEventStart
func (p *FilesDataProvider) SnapEventStart(storage.EventIdentifier, time.Duration) error {
	panic("TODO IMPL(SnapEventStart)")
}

// TODO: doc SnapEventEnd
func (p *FilesDataProvider) SnapEventEnd(storage.EventIdentifier, time.Duration) error {
	panic("TODO IMPL(SnapEventEnd)")
}

// TODO: doc SnapEventTimes
func (p *FilesDataProvider) SnapEventTimes(storage.EventIdentifier, time.Duration) error {
	panic("TODO IMPL(SnapEventTimes)")
}

// TODO: doc SetEventTitle
func (p *FilesDataProvider) SetEventTitle(storage.EventIdentifier, string) error {
	panic("TODO IMPL(SetEventTitle)")
}

// TODO: doc SetEventCategory
func (p *FilesDataProvider) SetEventCategory(storage.EventIdentifier, model.Category) error {
	panic("TODO IMPL(SetEventCategory)")
}

// TODO: doc SetEventAllData
func (p *FilesDataProvider) SetEventAllData(storage.EventIdentifier, model.Event) error {
	panic("TODO IMPL(SetEventAllData)")
}

// TODO: doc CommitState
func (p *FilesDataProvider) CommitState() error { panic("TODO IMPL(CommitState)") }

// FileHandler ...
type FileHandler struct {
	mutex sync.Mutex

	basePath string
	date     model.Date

	data model.EventList
}

// NewFileHandler ...
func NewFileHandler(basePath string, date model.Date) *FileHandler {
	f := FileHandler{basePath: basePath, date: date}
	return &f
}

func (h *FileHandler) Write() error {
	h.mutex.Lock()
	filename := h.Filename()
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("error opening file '%s'", filename)
	}

	writer := bufio.NewWriter(f)
	// TODO: don't ignore the errors, obviously
	for _, e := range h.data.Events {
		_, _ = writer.WriteString(e.String() + "\n")
	}
	writer.Flush()
	f.Close()
	h.mutex.Unlock()

	return nil
}

// Filename ...
func (h *FileHandler) Filename() string {
	return path.Join(h.basePath, h.date.String())
}

// AddEvent ...
func (h *FileHandler) AddEvent(e *model.Event) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	err := h.data.AddEvent(e)
	if err != nil {
		return fmt.Errorf("error adding event to file handler (%w)", err)
	}
	return nil
}

// Read ...
func (h *FileHandler) Read(knownCategories []model.Category) *model.EventList {
	l := model.EventList{
		Events: make([]*model.Event, 0),
	}

	h.mutex.Lock()
	f, err := os.Open(h.Filename())
	fileExists := (err == nil)
	if fileExists {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			s := scanner.Text()
			l.AddEvent(newEventFromDaywiseFileLine(h.date, s, knownCategories))
		}
		f.Close()
	}
	h.mutex.Unlock()

	return &l
}

func newEventFromDaywiseFileLine(date model.Date, line string, knownCategories []model.Category) *model.Event {
	var e model.Event

	args := strings.SplitN(line, "|", 4)
	startString := args[0]
	endString := args[1]
	catString := args[2]
	nameString := args[3]

	startTime := *model.NewTimestamp(startString)
	endTime := *model.NewTimestamp(endString)

	e.Start = model.DateAndTimestampToGotime(date, startTime)
	e.End = model.DateAndTimestampToGotime(date, endTime)

	var maybeCategory *model.Category
	for i := range knownCategories {
		if knownCategories[i].Name == catString {
			maybeCategory = &knownCategories[i]
		}
	}

	e.Name = nameString
	if maybeCategory != nil {
		e.Cat = *maybeCategory
	} else {
		e.Cat.Name = catString
	}

	return &e
}

func eventStartsAndEndsOnSameDate(e *model.Event) bool {
	return e.Start.Year() != e.End.Year() || e.Start.YearDay() != e.End.YearDay()
}
