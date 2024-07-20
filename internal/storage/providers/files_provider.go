package providers

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/ja-he/dayplan/internal/model"
)

type FilesDataProvider struct {
	BasePath string

	fhMutex      sync.RWMutex
	FileHandlers map[model.Date]*FileHandler

	categories []model.Category
}

// NewFilesDataProvider ...
func NewFilesDataProvider(
	basePath string,
	initialDate model.Date,
	categories []model.Category,
) (*FilesDataProvider, error) {

	result := &FilesDataProvider{
		BasePath:     basePath,
		fhMutex:      sync.RWMutex{},
		FileHandlers: make(map[model.Date]*FileHandler),
		categories:   categories,
	}

	result.loadFileHandlerForDate(initialDate)

	return result, nil
}

func (p *FilesDataProvider) loadFileHandlerForDate(date model.Date) error {

	// check if already loaded
	{
		p.fhMutex.RLock()
		if _, ok := p.FileHandlers[date]; ok {
			return nil
		}
		p.fhMutex.RUnlock()
	}

	// add the new handler
	p.fhMutex.Lock()
	defer p.fhMutex.Unlock()
	p.FileHandlers[date] = NewFileHandler(p.BasePath, date)

	return nil
}

type FileHandler struct {
	mutex sync.Mutex

	basePath string
	date     model.Date

	data []*model.Event
}

func NewFileHandler(basePath string, date model.Date) *FileHandler {
	f := FileHandler{basePath: basePath, date: date}
	return &f
}

func (h *FileHandler) Write() error {
	if h.data == nil {
		return fmt.Errorf("have not yet read data, refusing to write")
	}

	h.mutex.Lock()
	filename := h.Filename()
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("error opening file '%s'", filename)
	}

	writer := bufio.NewWriter(f)
	// TODO: don't ignore the errors, obviously
	for _, e := range h.data {
		_, _ = writer.WriteString(e.String() + "\n")
	}
	writer.Flush()
	f.Close()
	h.mutex.Unlock()

	return nil
}

// ...
func (h *FileHandler) Filename() string {
	return path.Join(h.basePath, h.date.String())
}

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

func (p *FilesDataProvider) getDayFromFileHandler(date model.Date) *model.EventList {
	p.fhMutex.RLock()
	fh, ok := p.FileHandlers[date]
	p.fhMutex.RUnlock()
	if ok {
		tmp := fh.Read(p.categories)
		return tmp
	}

	newHandler := NewFileHandler(p.BasePath, date)
	p.fhMutex.Lock()
	p.FileHandlers[date] = newHandler
	p.fhMutex.Unlock()
	tmp := newHandler.Read(p.categories)
	return tmp
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
