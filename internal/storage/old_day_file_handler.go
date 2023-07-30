package storage

import (
	"bufio"
	"log"
	"os"
	"sync"

	"github.com/ja-he/dayplan/internal/model"
)

type FileHandler struct {
	mutex    sync.Mutex
	filename string
}

func NewFileHandler(filename string) *FileHandler {
	f := FileHandler{filename: filename}
	return &f
}

func (h *FileHandler) Write(day *model.Day) {
	h.mutex.Lock()
	f, err := os.OpenFile(h.filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("error opening file '%s'", h.filename)
	}

	writer := bufio.NewWriter(f)
	for _, line := range day.ToSlice() {
		_, _ = writer.WriteString(line + "\n")
	}
	writer.Flush()
	f.Close()
	h.mutex.Unlock()
}

func (h *FileHandler) Read(knownCategories []model.Category) *model.Day {
	day := model.NewDay()

	h.mutex.Lock()
	f, err := os.Open(h.filename)
	fileExists := (err == nil)
	if fileExists {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			s := scanner.Text()
			day.AddEvent(model.NewEvent(s, knownCategories))
		}
		f.Close()
	}
	h.mutex.Unlock()

	return day
}
