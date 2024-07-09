package model

import (
	"fmt"
	"time"

	"github.com/ja-he/dayplan/internal/config"
)

// Goal defines a time goal.
// It can be queried, for any given date, what duration the goal requires.
type Goal interface {
	Requires(Date) time.Duration
}

// RangedGoal is a Goal that is defined by any number of ranges and the
// expected duration for each.
type RangedGoal struct {
	Entries []rangedGoalEntry
}

// rangedGoalEntry is a Goal that is defined by an expected total duration over a
// bounded period of time.
type rangedGoalEntry struct {
	Start Date
	End   Date
	Time  time.Duration
}

// Requires returns the duration required for the given date.
//
// It is (Time/ DAYSINRANGE(Start, End)) for any day in range, 0 otherwise.
func (g *RangedGoal) Requires(date Date) time.Duration {
	for _, e := range g.Entries {
		// if within range of entry, return the duration (proportional to range length)
		if !date.IsBefore(e.Start) && !date.IsAfter(e.End) {
			return e.Time / time.Duration(e.Start.DaysUntil(e.End)+1)
		}
	}

	return 0
}

// NewRangedGoalFromConfig constructs a new RangedGoal from config data.
func NewRangedGoalFromConfig(cfg []config.RangedGoal) (*RangedGoal, error) {
	result := RangedGoal{}

	for i := range cfg {
		start, err := FromString(cfg[i].Start)
		if err != nil {
			return nil, fmt.Errorf("error parsing start date of range no. %d: %w", i, err)
		}
		end, err := FromString(cfg[i].End)
		if err != nil {
			return nil, fmt.Errorf("error parsing start date of range no. %d: %w", i, err)
		}
		duration, err := time.ParseDuration(cfg[i].Time)
		if err != nil {
			return nil, fmt.Errorf("error parsing duration of range no. %d: %w", i, err)
		} else {
			for j := 0; j < i; j++ {
				if !((start.IsBefore(result.Entries[j].Start) && end.IsBefore(result.Entries[j].Start)) || (start.IsAfter(result.Entries[j].End) && end.IsAfter(result.Entries[j].End))) {
					return nil, fmt.Errorf("range no. %d defined overlaps with range no. %d", i, j)
				}
			}

			result.Entries = append(result.Entries, rangedGoalEntry{
				Start: start,
				End:   end,
				Time:  duration,
			})
		}
	}

	return &result, nil
}

// WorkweekGoal is a goal that defines the duration per day of the week.
type WorkweekGoal struct {
	Monday    time.Duration
	Tuesday   time.Duration
	Wednesday time.Duration
	Thursday  time.Duration
	Friday    time.Duration
	Saturday  time.Duration
	Sunday    time.Duration
}

// Requires returns the duration required for the given date.
//
// It is just the duration defined for the date's weekday.
func (g *WorkweekGoal) Requires(date Date) time.Duration {
	switch date.ToWeekday() {
	case time.Monday:
		return g.Monday
	case time.Tuesday:
		return g.Tuesday
	case time.Wednesday:
		return g.Wednesday
	case time.Thursday:
		return g.Thursday
	case time.Friday:
		return g.Friday
	case time.Saturday:
		return g.Saturday
	case time.Sunday:
		return g.Sunday
	default:
		panic(fmt.Sprintf("unknown weekday %d", date.ToWeekday()))
	}
}

// NewWorkweekGoalFromConfig constructs a new WorkweekGoal from config data.
func NewWorkweekGoalFromConfig(cfg config.WorkweekGoal) (*WorkweekGoal, error) {
	var monday, tuesday, wednesday, thursday, friday, saturday, sunday time.Duration
	var mondayErr, tuesdayErr, wednesdayErr, thursdayErr, fridayErr, saturdayErr, sundayErr error

	// parse any provided durations (else defaults to 0)
	if cfg.Monday != "" {
		monday, mondayErr = time.ParseDuration(cfg.Monday)
	}
	if cfg.Tuesday != "" {
		tuesday, tuesdayErr = time.ParseDuration(cfg.Tuesday)
	}
	if cfg.Wednesday != "" {
		wednesday, wednesdayErr = time.ParseDuration(cfg.Wednesday)
	}
	if cfg.Thursday != "" {
		thursday, thursdayErr = time.ParseDuration(cfg.Thursday)
	}
	if cfg.Friday != "" {
		friday, fridayErr = time.ParseDuration(cfg.Friday)
	}
	if cfg.Saturday != "" {
		saturday, saturdayErr = time.ParseDuration(cfg.Saturday)
	}
	if cfg.Sunday != "" {
		sunday, sundayErr = time.ParseDuration(cfg.Sunday)
	}

	// return valid config, unless errors occurred during parsing
	switch {
	case mondayErr != nil:
		return nil, mondayErr
	case tuesdayErr != nil:
		return nil, tuesdayErr
	case wednesdayErr != nil:
		return nil, wednesdayErr
	case thursdayErr != nil:
		return nil, thursdayErr
	case fridayErr != nil:
		return nil, fridayErr
	case saturdayErr != nil:
		return nil, saturdayErr
	case sundayErr != nil:
		return nil, sundayErr
	default:
		return &WorkweekGoal{
			Monday:    monday,
			Tuesday:   tuesday,
			Wednesday: wednesday,
			Thursday:  thursday,
			Friday:    friday,
			Saturday:  saturday,
			Sunday:    sunday,
		}, nil
	}
}

// GoalForRange is a helper to sum up the duration for the given range expected
// by the given Goal.
func GoalForRange(goal Goal, startDate, endDate Date) time.Duration {
	sum := time.Duration(0)

	currentDate := startDate
	for currentDate != endDate.Next() {
		sum += goal.Requires(currentDate)
		currentDate = currentDate.Next()
	}

	return sum
}
