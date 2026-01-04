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

	// diskMtime is the file modification time when we last read from disk.
	// Used to detect if the file has been modified externally.
	diskMtime time.Time

	// Tracking of local modifications for merge-on-write strategy
	locallyModifiedEvents map[model.EventID]struct{}
	locallyAddedEvents    map[model.EventID]struct{}
	locallyRemovedEvents  map[model.EventID]struct{}

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

// DiskHasChanged checks if the file on disk has been modified since we last read it.
// This can be used to detect external changes made by other processes.
func (h *fileHandler) DiskHasChanged() (bool, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.diskHasChangedUnsafe()
}

// diskHasChangedUnsafe is the internal version that assumes mutex is already held.
func (h *fileHandler) diskHasChangedUnsafe() (bool, error) {
	info, err := os.Stat(h.Filename())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File doesn't exist. If we never read (diskMtime is zero), no change.
			// If we did read and file is gone, that's a change.
			return !h.diskMtime.IsZero(), nil
		}
		return false, fmt.Errorf("error checking file modification time (%w)", err)
	}
	return info.ModTime().After(h.diskMtime), nil
}

// clearLocalModifications resets the tracking of local changes.
// Called after a successful write that has merged and persisted all changes.
func (h *fileHandler) clearLocalModifications() {
	h.locallyModifiedEvents = make(map[model.EventID]struct{})
	h.locallyAddedEvents = make(map[model.EventID]struct{})
	h.locallyRemovedEvents = make(map[model.EventID]struct{})
}

func newFileHandlerWithDataReadFromDisk(basePath string, date model.Date) (*fileHandler, error) {
	f := fileHandler{
		basePath:              basePath,
		date:                  date,
		locallyModifiedEvents: make(map[model.EventID]struct{}),
		locallyAddedEvents:    make(map[model.EventID]struct{}),
		locallyRemovedEvents:  make(map[model.EventID]struct{}),
	}
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

	// Check if we have local changes
	hasLocalChanges := !h.onDiskIsUpToDate()

	// Check if disk has external changes
	hasExternalChanges, err := h.diskHasChangedUnsafe()
	if err != nil {
		log.Warn().Err(err).Msgf("Could not check for external changes for %s, proceeding with write", h.date.String())
		hasExternalChanges = false
	}

	if !hasLocalChanges && !hasExternalChanges {
		log.Debug().Msgf("Skipping write of %s because no local or external changes.", h.date.String())
		return nil
	}

	return h.writeUnsafe()
}

func (h *fileHandler) writeUnsafe() error {
	filename := h.Filename()
	tempFilename := filename + ".tmp"

	// Open the target file with exclusive lock for reading current state
	targetFile, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("could not open file '%s' for locking (%w)", filename, err)
	}
	defer targetFile.Close()

	// Acquire exclusive lock
	if err := lockFileExclusive(targetFile); err != nil {
		return fmt.Errorf("could not acquire exclusive lock on '%s' (%w)", filename, err)
	}
	defer unlockFile(targetFile)

	// Merge with current disk state
	mergedEvents, err := h.mergeWithDiskState(targetFile)
	if err != nil {
		return fmt.Errorf("could not merge with disk state (%w)", err)
	}

	// Write to temp file
	tempFile, err := os.OpenFile(tempFilename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("could not open temp file '%s' (%w)", tempFilename, err)
	}

	writer := bufio.NewWriter(tempFile)
	var errs []error
	for _, e := range mergedEvents {
		_, err = writer.WriteString(e.String() + "\n")
		if err != nil {
			errs = append(errs, fmt.Errorf("Unable to write (%w)", err))
		}
	}
	if err := writer.Flush(); err != nil {
		errs = append(errs, fmt.Errorf("Unable flush (%w)", err))
	}
	if err := tempFile.Close(); err != nil {
		errs = append(errs, fmt.Errorf("Unable close temp file (%w)", err))
	}
	if errs != nil {
		os.Remove(tempFilename) // Clean up temp file on error
		return fmt.Errorf("Unable to write data to temp file '%s' (%w)", tempFilename, errors.Join(errs...))
	}

	// Atomic rename: temp file -> target file
	if err := os.Rename(tempFilename, filename); err != nil {
		os.Remove(tempFilename) // Clean up temp file on error
		return fmt.Errorf("could not rename temp file to target '%s' (%w)", filename, err)
	}

	// Update disk mtime and our in-memory state
	info, err := os.Stat(filename)
	if err != nil {
		// Non-fatal: file was written successfully, but we couldn't get mtime
		log.Warn().Err(err).Msgf("could not stat file '%s' after write", filename)
	} else {
		h.diskMtime = info.ModTime()
	}

	// Update in-memory data to match what we wrote (merged events)
	h.data.Events = mergedEvents
	h.data.UpdateEventOrder()

	// Clear local modification tracking since everything is now persisted
	h.clearLocalModifications()

	h.updateLastWrite()
	return nil
}

// mergeWithDiskState reads the current disk state and merges our local changes with it.
// Returns the merged list of events to be written.
func (h *fileHandler) mergeWithDiskState(f *os.File) ([]*model.Event, error) {
	// Build a map of our local events for quick lookup
	localEventsByID := make(map[model.EventID]*model.Event)
	for _, e := range h.data.Events {
		localEventsByID[e.ID] = e
	}

	// Read current disk state
	diskEvents, err := h.parseEventsFromFile(f)
	if err != nil {
		return nil, fmt.Errorf("could not read disk events (%w)", err)
	}

	// Merge: start with disk events, apply our changes
	var merged []*model.Event
	seenIDs := make(map[model.EventID]struct{})

	for _, diskEvent := range diskEvents {
		// Skip events we've removed locally
		if _, removed := h.locallyRemovedEvents[diskEvent.ID]; removed {
			continue
		}

		// Use our version if we modified it, otherwise keep disk version
		if _, modified := h.locallyModifiedEvents[diskEvent.ID]; modified {
			if localEvent, ok := localEventsByID[diskEvent.ID]; ok {
				merged = append(merged, localEvent)
			}
			// If not in local map, it was removed - skip
		} else if _, added := h.locallyAddedEvents[diskEvent.ID]; added {
			// This shouldn't happen (disk event in our added set), but handle it
			if localEvent, ok := localEventsByID[diskEvent.ID]; ok {
				merged = append(merged, localEvent)
			}
		} else {
			// Event from disk that we haven't touched - keep disk version
			merged = append(merged, diskEvent)
		}
		seenIDs[diskEvent.ID] = struct{}{}
	}

	// Add our locally added events (not seen on disk)
	for addedID := range h.locallyAddedEvents {
		if _, seen := seenIDs[addedID]; !seen {
			if localEvent, ok := localEventsByID[addedID]; ok {
				merged = append(merged, localEvent)
			}
		}
	}

	return merged, nil
}

// parseEventsFromFile reads events from an already-open and locked file.
func (h *fileHandler) parseEventsFromFile(f *os.File) ([]*model.Event, error) {
	// Seek to beginning of file
	if _, err := f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("could not seek to beginning of file (%w)", err)
	}

	var events []*model.Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		if s == "" {
			continue // Skip empty lines
		}
		var e *model.Event
		var parseErr error
		if strings.Count(s, "|") == 3 {
			e = newEventFromDaywiseFileLineLegacy(h.date, s)
		} else {
			e, parseErr = newEventFromDaywiseFileLineNew(s)
		}
		if parseErr != nil {
			return nil, fmt.Errorf("unable to parse event line '%s' (%w)", s, parseErr)
		}
		if e.ID == "" {
			// Legacy format without ID - generate one
			e.ID = filesProviderIDGenerator()
		} else if !filesProviderIDValidator(e.ID) {
			return nil, fmt.Errorf("invalid event ID '%s'", e.ID)
		}
		events = append(events, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file (%w)", err)
	}
	return events, nil
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
	h.locallyAddedEvents[e.ID] = struct{}{}
	return nil
}

// RemoveEvent ...
func (h *fileHandler) RemoveEvent(id model.EventID) error {
	defer h.updateLastChange()
	h.mutex.Lock()
	defer h.mutex.Unlock()

	indexOfEvent := -1
	for i, ev := range h.data.Events {
		if ev.ID == id {
			indexOfEvent = i
			break
		}
	}
	if indexOfEvent == -1 {
		return fmt.Errorf("event with ID '%s' not found", id)
	}
	h.data.Events = append(h.data.Events[:indexOfEvent], h.data.Events[indexOfEvent+1:]...)

	// Track the removal. If it was a locally added event, just remove from added set.
	// Otherwise, mark as removed so we don't re-add it from disk on merge.
	if _, wasAdded := h.locallyAddedEvents[id]; wasAdded {
		delete(h.locallyAddedEvents, id)
	} else {
		h.locallyRemovedEvents[id] = struct{}{}
	}
	// If it was modified, remove from modified set since it's now gone
	delete(h.locallyModifiedEvents, id)

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

	// Track modification (unless it's a locally added event, which is already tracked)
	if _, wasAdded := h.locallyAddedEvents[e.ID]; !wasAdded {
		h.locallyModifiedEvents[e.ID] = struct{}{}
	}

	return nil
}

// readFromDisk reads events from the file on disk with proper locking.
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
			// File doesn't exist - start with empty data, no mtime to track
			h.diskMtime = time.Time{}
			return nil
		}
		return fmt.Errorf("could not read file '%s' from disk (%w)", h.Filename(), err)
	}
	defer f.Close()

	// Acquire shared lock for reading
	if err := lockFileShared(f); err != nil {
		return fmt.Errorf("could not acquire shared lock on '%s' (%w)", h.Filename(), err)
	}
	defer unlockFile(f)

	// Get file modification time for stale detection
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("could not stat file '%s' (%w)", h.Filename(), err)
	}
	h.diskMtime = info.ModTime()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		var e *model.Event
		var parseErr error
		if strings.Count(s, "|") == 3 {
			e = newEventFromDaywiseFileLineLegacy(h.date, s)
		} else {
			e, parseErr = newEventFromDaywiseFileLineNew(s)
		}
		if parseErr != nil {
			return fmt.Errorf("Unable to parse event line '%s' (%w)", s, parseErr)
		}
		if e.ID == "" {
			newID := filesProviderIDGenerator()
			log.Debug().
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

	e.Start = model.DateAndTimestampToGotime(date, startTime, time.Local).UTC()
	e.End = model.DateAndTimestampToGotime(date, endTime, time.Local).UTC()

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
