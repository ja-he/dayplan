package model

import (
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
