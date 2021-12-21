package model

import (
	"log"
	"testing"
	"time"
)

func TestToWeekday(t *testing.T) {
	{
		date := Date{2021, 11, 12}
		expected := time.Friday
		result := date.ToWeekday()
		if result != expected {
			log.Fatalf("%s should be weekday %s not %s", date.ToString(), ToString(expected), ToString(result))
		}
	}
}

func TestWeek(t *testing.T) {
	{
		date := Date{2021, 11, 12}
		expStart, expEnd := Date{2021, 11, 8}, Date{2021, 11, 14}
		resStart, resEnd := date.Week()
		if resStart != expStart || resEnd != expEnd {
			log.Fatalf("%s is bounded by (%s,%s) not (%s,%s)", date.ToString(), expStart.ToString(), expEnd.ToString(), resStart.ToString(), resEnd.ToString())
		}
	}
}

func TestMonthBounds(t *testing.T) {
	{
		testcase := "in 30 day month"

		date := Date{
			Year:  2021,
			Month: 11,
			Day:   12,
		}

		expStart, expEnd := Date{2021, 11, 1}, Date{2021, 11, 30}
		resStart, resEnd := date.MonthBounds()
		if resStart != expStart || resEnd != expEnd {
			log.Fatalf("[testcase '%s' failed]: %s is bounded by (%s,%s) not (%s,%s)", testcase, date.ToString(), expStart.ToString(), expEnd.ToString(), resStart.ToString(), resEnd.ToString())
		}
	}
	{
		testcase := "in 31 day month"

		date := Date{
			Year:  2021,
			Month: 12,
			Day:   14,
		}

		expStart, expEnd := Date{2021, 12, 1}, Date{2021, 12, 31}
		resStart, resEnd := date.MonthBounds()
		if resStart != expStart || resEnd != expEnd {
			log.Fatalf("[testcase '%s' failed]: %s is bounded by (%s,%s) not (%s,%s)", testcase, date.ToString(), expStart.ToString(), expEnd.ToString(), resStart.ToString(), resEnd.ToString())
		}
	}
	{
		testcase := "in february"

		date := Date{
			Year:  2021,
			Month: 2,
			Day:   4,
		}

		expStart, expEnd := Date{2021, 2, 1}, Date{2021, 2, 28}
		resStart, resEnd := date.MonthBounds()
		if resStart != expStart || resEnd != expEnd {
			log.Fatalf("[testcase '%s' failed]: %s is bounded by (%s,%s) not (%s,%s)", testcase, date.ToString(), expStart.ToString(), expEnd.ToString(), resStart.ToString(), resEnd.ToString())
		}
	}
	{
		testcase := "in february in leap year"

		date := Date{
			Year:  2004,
			Month: 2,
			Day:   4,
		}

		expStart, expEnd := Date{2004, 2, 1}, Date{2004, 2, 29}
		resStart, resEnd := date.MonthBounds()
		if resStart != expStart || resEnd != expEnd {
			log.Fatalf("[testcase '%s' failed]: %s is bounded by (%s,%s) not (%s,%s)", testcase, date.ToString(), expStart.ToString(), expEnd.ToString(), resStart.ToString(), resEnd.ToString())
		}
	}
}
