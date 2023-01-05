package model

import (
	"fmt"
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

// ToPrintableFormat returns this TimesheetEntry in its printable (CSV) format.
func (e *TimesheetEntry) ToPrintableFormat() string {
	return fmt.Sprintf(
		"%s,%s,%s",
		e.Start.ToString(),
		e.BreakDuration.String(),
		e.End.ToString(),
	)
}

// IsEmpty is a helper to identify empty timesheet entries.
func (e *TimesheetEntry) IsEmpty() bool {
	return e.Start == e.End
}
