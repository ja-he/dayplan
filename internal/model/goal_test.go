package model_test

import (
	"testing"

	"github.com/ja-he/dayplan/internal/model"
)

func TestGoalForRange(t *testing.T) {
	t.Run("realistic workweek", func(t *testing.T) {
		pentecost := model.Date{2024, 5, 20}
		startOfWeek := model.Date{2024, 5, 20}
		endOfWeek := model.Date{2024, 5, 26}
		g := model.WorkweekGoal{
			Monday:    0,
			Tuesday:   5,
			Wednesday: 5,
			Thursday:  5,
			Friday:    5,
			Saturday:  0,
			Sunday:    0,
			Except: map[model.Date]struct{}{
				pentecost: {},
			},
		}
		d := model.GoalForRange(&g, startOfWeek, endOfWeek)
		if d != 20 {
			t.Errorf("expected 20, got %d", d)
		}
	})
}
