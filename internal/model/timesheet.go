package model

import (
	"time"
)

// A TimesheetEntry represents an entry in a common timesheet.
//
// It defines a beginning (i.e. the time you clocked in), an end (i.e. the time
// you clocked out), and the total length of breaks taken between them.
type TimesheetEntry struct {
	Start         Timestamp
	BreakDuration time.Duration
	End           Timestamp
}

// IsEmpty is a helper to identify empty timesheet entries.
func (e *TimesheetEntry) IsEmpty() bool {
	return e.Start == e.End
}
