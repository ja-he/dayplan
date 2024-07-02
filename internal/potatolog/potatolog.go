package potatolog

import (
	"encoding/json"
	"fmt"
	"sync"
)

// LogEntry is a single log entry.
type LogEntry = map[string]any

// GlobalMemoryLogReaderWriter is a global MemoryLogReaderWriter.
var GlobalMemoryLogReaderWriter = MemoryLogReaderWriter{
	mtx: sync.Mutex{},
	log: []LogEntry{},
}

// MemoryLogReaderWriter is a simple in-memory log reader and writer.
type MemoryLogReaderWriter struct {
	mtx sync.Mutex
	log []LogEntry
}

// Write appends a log entry to the log.
func (w *MemoryLogReaderWriter) Write(p []byte) (int, error) {
	entry := LogEntry{}
	err := json.Unmarshal(p, &entry)
	if err != nil {
		return 0, fmt.Errorf("could not unmarshal log entry (err:%s) (input:'%s')", err.Error(), string(p))
	}

	w.mtx.Lock()
	defer w.mtx.Unlock()
	w.log = append(w.log, entry)
	return len(p), nil
}

// Get returns the log.
func (w *MemoryLogReaderWriter) Get() []LogEntry {
	return w.log
}

// LogReader allows reading access to a log.
type LogReader interface {
	Get() []LogEntry
}
