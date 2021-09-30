package log

import (
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

func (l *Log) Add(location, entryType, message string) {
	l.mutex.Lock()
	l.entries = append(l.entries, LogEntry{time.Now(), location, entryType, message})
	l.mutex.Unlock()
}

func (l *Log) Get() []LogEntry {
	return l.entries
}
