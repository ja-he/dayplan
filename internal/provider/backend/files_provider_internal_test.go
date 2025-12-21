package backend

import (
	"testing"
	"time"
)

type TestTimesOnSameDateTestDataEntry struct {
	ParamA         time.Time
	ParamB         time.Time
	ExpectedResult bool
}

func TestTimesOnSameDate(t *testing.T) {
	data := []TestTimesOnSameDateTestDataEntry{
		{time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), true},
		{time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 15, 0, 0, 1, time.UTC), true},
		{time.Date(2023, 1, 1, 15, 0, 0, 1, time.UTC), time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), true},
		{time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 1, time.UTC), true},
		{time.Date(2023, 1, 1, 0, 0, 0, 1, time.UTC), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{time.Date(2023, 1, 1, 23, 59, 59, 999999999, time.UTC), time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), true},
		{time.Date(2023, 1, 1, 23, 50, 0, 0, time.UTC), time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), true},
		{time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), time.Date(2023, 1, 2, 15, 0, 0, 0, time.UTC), false},
		{time.Date(2023, 1, 2, 15, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), false},
		{time.Date(2023, 1, 1, 00, 0, 0, 0, time.UTC), time.Date(2023, 1, 2, 00, 0, 0, 0, time.UTC), true},
		{time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), time.Date(2023, 1, 2, 00, 0, 0, 0, time.UTC), true},
		{time.Date(2022, 1, 1, 15, 0, 0, 0, time.UTC), time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), false},
	}

	for i, entry := range data {
		actualResult := timesOnSameDate(entry.ParamA, entry.ParamB)
		if actualResult != entry.ExpectedResult {
			t.Fatalf("[testcase=%d] Expected timesOnSameDate(%s, %s) to yield %t but got %t.", i, entry.ParamA, entry.ParamB, entry.ExpectedResult, actualResult)
		}
	}
}
