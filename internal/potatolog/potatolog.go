package potatolog

import (
	"encoding/json"
	"fmt"
	"sync"
)

type LogEntry = map[string]string

var GlobalMemoryLogReaderWriter = MemoryLogReaderWriter{
	mtx: sync.Mutex{},
	log: []LogEntry{},
}

type MemoryLogReaderWriter struct {
	mtx sync.Mutex
	log []LogEntry
}

func (w *MemoryLogReaderWriter) Write(p []byte) (int, error) {
	entry := LogEntry{}
	err := json.Unmarshal(p, &entry)
	if err != nil {
		return 0, fmt.Errorf("could not unmarshal log entry (%s)", err.Error())
	}

	w.mtx.Lock()
	defer w.mtx.Unlock()
	w.log = append(w.log, entry)
	return len(p), nil
}

func (w *MemoryLogReaderWriter) Get() []LogEntry {
	return w.log
}

// LogReader allows reading access to a log.
type LogReader interface {
	Get() []LogEntry
}
