package model

import (
	"fmt"
	"time"

	"github.com/ja-he/dayplan/src/config"
)

// Goal defines a time goal.
// It can be queried, for any given date, what duration the goal requires.
type Goal interface {
	Requires(Date) time.Duration
}

// RangedGoal is a Goal that is defined by an expected total duration over a
// bounded period of time.
type RangedGoal struct {
	Start Date
	End   Date
	Time  time.Duration
}

// Requires returns the duration required for the given date.
//
// It is (Time/ DAYSINRANGE(Start, End)) for any day in range, 0 otherwise.
func (g *RangedGoal) Requires(date Date) time.Duration {
	if date.IsBefore(g.Start) || date.IsAfter(g.End) {
		return 0
	} else {
		return g.Time / time.Duration(g.Start.DaysUntil(g.End)+1)
	}
}

// NewRangedGoalFromConfig constructs a new RangedGoal from config data.
func NewRangedGoalFromConfig(cfg config.RangedGoal) (*RangedGoal, error) {

	start, err := FromString(cfg.Start)
	end, err := FromString(cfg.End)

	duration, err := time.ParseDuration(cfg.Time)
	if err != nil {
		return nil, err
	} else {
		return &RangedGoal{
			Start: start,
			End:   end,
			Time:  duration,
		}, nil
	}
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
	monday, mondayErr := time.ParseDuration(cfg.Monday)
	tuesday, tuesdayErr := time.ParseDuration(cfg.Tuesday)
	wednesday, wednesdayErr := time.ParseDuration(cfg.Wednesday)
	thursday, thursdayErr := time.ParseDuration(cfg.Thursday)
	friday, fridayErr := time.ParseDuration(cfg.Friday)
	saturday, saturdayErr := time.ParseDuration(cfg.Saturday)
	sunday, sundayErr := time.ParseDuration(cfg.Sunday)

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
