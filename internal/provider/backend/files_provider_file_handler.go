package backend

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ja-he/dayplan/internal/model"
)

type fileHandler struct {
	mutex sync.Mutex

	basePath string
	date     model.Date

	lastChange time.Time
	lastWrite  time.Time

	data model.EventList
}

func (h *fileHandler) updateLastChange() {
	h.lastChange = time.Now()
}
func (h *fileHandler) updateLastWrite() {
	h.lastWrite = time.Now()
}
func (h *fileHandler) onDiskIsUpToDate() bool {
	return h.lastChange.IsZero() || (h.lastChange.Before(h.lastWrite))
}

func newFileHandlerWithDataReadFromDisk(basePath string, date model.Date) (*fileHandler, error) {
	f := fileHandler{basePath: basePath, date: date}
	err := f.readFromDisk()
	if err != nil {
		return nil, fmt.Errorf("could not read file from disk (%w)", err)
	}
	return &f, nil
}

func (h *fileHandler) OnDiskIsUpToDate() bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.onDiskIsUpToDate()
}
func (h *fileHandler) Write() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.onDiskIsUpToDate() {
		log.Debug().Msgf("Skipping write of %s because up to date.", h.date.String())
		return nil
	}
	return h.writeUnsafe()
}

func (h *fileHandler) writeUnsafe() error {
	filename := h.Filename()
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("could not open file '%s' (%w)", filename, err)
	}

	writer := bufio.NewWriter(f)
	var errs []error
	for _, e := range h.data.Events {
		_, err = writer.WriteString(e.String() + "\n")
		if err != nil {
			errs = append(errs, fmt.Errorf("Unable to write (%w)", err))
		}
	}
	if err := writer.Flush(); err != nil {
		errs = append(errs, fmt.Errorf("Unable flush (%w)", err))
	}
	if err := f.Close(); err != nil {
		errs = append(errs, fmt.Errorf("Unable close file (%w)", err))
	}
	if errs != nil {
		return fmt.Errorf("Unable to write data to file '%s' (%w)", filename, errors.Join(errs...))
	}

	h.updateLastWrite()
	return nil
}

// Filename ...
func (h *fileHandler) Filename() string {
	return path.Join(h.basePath, h.date.String())
}

// AddEvent ...
func (h *fileHandler) AddEvent(e *model.Event) error {
	defer h.updateLastChange()
	h.mutex.Lock()
	defer h.mutex.Unlock()

	err := h.data.AddEvent(e)
	if err != nil {
		return fmt.Errorf("error adding event to file handler (%w)", err)
	}
	return nil
}

// RemoveEvent ...
func (h *fileHandler) RemoveEvent(e model.EventID) error {
	defer h.updateLastChange()
	h.mutex.Lock()
	defer h.mutex.Unlock()

	indexOfEvent := -1
	for i, ev := range h.data.Events {
		if ev.ID == e {
			indexOfEvent = i
			break
		}
	}
	if indexOfEvent == -1 {
		return fmt.Errorf("event with ID '%s' not found", e)
	}
	h.data.Events = append(h.data.Events[:indexOfEvent], h.data.Events[indexOfEvent+1:]...)
	return nil
}

func (h *fileHandler) GetEvent(id model.EventID) (*model.Event, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	e := h.data.GetEventByID(id)
	if e == nil {
		return nil, fmt.Errorf("event with ID '%s' not found", id)
	}
	return e, nil
}

// UpdateEvent updates an existing event identified by its ID.
func (h *fileHandler) UpdateEvent(e *model.Event) error {
	defer h.updateLastChange()
	h.mutex.Lock()
	defer h.mutex.Unlock()

	indexOfEvent := -1
	for i, ev := range h.data.Events {
		if ev.ID == e.ID {
			indexOfEvent = i
			break
		}
	}
	if indexOfEvent == -1 {
		return fmt.Errorf("event with ID '%s' not found", e.ID)
	}

	// Update the event details
	h.data.Events[indexOfEvent] = e
	h.data.UpdateEventOrder()

	return nil
}

// Read ...
func (h *fileHandler) readFromDisk() error {
	if len(h.data.Events) != 0 {
		// warn
	}

	h.data = model.EventList{
		Events: make([]*model.Event, 0),
	}

	f, err := os.Open(h.Filename())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// TODO: tell that have loaded as empty
			return nil
		}
		return fmt.Errorf("could not read file '%s' from disk (%w)", h.Filename(), err)
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		var e *model.Event
		var err error
		if strings.Count(s, "|") == 3 {
			e = newEventFromDaywiseFileLineLegacy(h.date, s)
		} else {
			e, err = newEventFromDaywiseFileLineNew(s)
		}
		if err != nil {
			return fmt.Errorf("Unable to parse event line '%s' (%w)", s, err)
		}
		if e.ID == "" {
			newID := filesProviderIDGenerator()
			log.Warn().
				Str("e.Name", e.Name).
				Stringer("e.Start", e.Start).
				Stringer("e.End", e.End).
				Msgf("generated temporary (until write) event ID '%s' to cope with legacy format", newID)
			e.ID = newID
		} else if !filesProviderIDValidator(e.ID) {
			return fmt.Errorf("invalid event ID '%s' in file '%s'", e.ID, h.Filename())
		}
		h.data.AddEvent(e)
	}

	return nil
}

func newEventFromDaywiseFileLineLegacy(date model.Date, line string) *model.Event {
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

	e.Name = nameString
	e.Category = model.CategoryName(catString)

	return &e
}

func newEventFromDaywiseFileLineNew(line string) (*model.Event, error) {
	var e model.Event

	args := strings.SplitN(line, "|", 5)
	idString := args[0]
	startString := args[1]
	endString := args[2]
	catString := args[3]
	nameString := args[4]

	var err error

	e.Start, err = time.Parse(time.RFC3339, startString)
	if err != nil {
		return nil, fmt.Errorf("Could not parse start timestamp '%s' (%w).", startString, err)
	}
	e.End, err = time.Parse(time.RFC3339, endString)
	if err != nil {
		return nil, fmt.Errorf("Could not parse end timestamp '%s' (%w).", endString, err)
	}

	e.Name = nameString
	e.Category = model.CategoryName(catString)

	e.ID = idString

	return &e, nil
}
