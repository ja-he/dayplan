package potatolog

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type LogEntry struct {
	At       time.Time
	Location string
	Type     string
	Message  string
}

type Log struct {
	mutex   sync.Mutex
	entries []LogEntry
}

func (l *Log) Add(entryType, message string) {
	_, path, line, ok := runtime.Caller(1)
	file := filepath.Base(path)
	location := "[irretrievable]"
	if ok {
		location = fmt.Sprintf("%s:%d", file, line)
	}
	l.mutex.Lock()
	l.entries = append(l.entries, LogEntry{time.Now(), location, entryType, message})
	l.mutex.Unlock()
}

func (l *Log) Get() []LogEntry {
	return l.entries
}
