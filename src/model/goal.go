package model

import (
	"fmt"
	"time"
)

type Goal interface {
	Requires(Date) time.Duration
}

type RangedGoal struct {
	Start Date
	End   Date
	Time  time.Duration
}

func (g *RangedGoal) Requires(date Date) time.Duration {
	if date.IsBefore(g.Start) || date.IsAfter(g.End) {
		return 0
	} else {
		return g.Time / time.Duration(g.Start.DaysUntil(g.End)+1)
	}
}

type WorkweekGoal struct {
	Monday    time.Duration
	Tuesday   time.Duration
	Wednesday time.Duration
	Thursday  time.Duration
	Friday    time.Duration
	Saturday  time.Duration
	Sunday    time.Duration
}

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
