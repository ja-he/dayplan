package providers

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/ja-he/dayplan/internal/model"
)

type fileHandler struct {
	mutex sync.Mutex

	basePath string
	date     model.Date

	data model.EventList
}

func newFileHandlerWithDataReadFromDisk(basePath string, date model.Date) (*fileHandler, error) {
	f := fileHandler{basePath: basePath, date: date}
	err := f.readFromDisk()
	if err != nil {
		return nil, fmt.Errorf("could not read file from disk (%w)", err)
	}
	return &f, nil
}

func (h *fileHandler) Write() error {
	h.mutex.Lock()
	filename := h.Filename()
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("could not open file '%s' (%w)", filename, err)
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
func (h *fileHandler) Filename() string {
	return path.Join(h.basePath, h.date.String())
}

// AddEvent ...
func (h *fileHandler) AddEvent(e *model.Event) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	err := h.data.AddEvent(e)
	if err != nil {
		return fmt.Errorf("error adding event to file handler (%w)", err)
	}
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
		h.data.AddEvent(newEventFromDaywiseFileLine(h.date, s))
	}

	return nil
}

func newEventFromDaywiseFileLine(date model.Date, line string) *model.Event {
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
	e.Cat.Name = catString

	return &e
}
