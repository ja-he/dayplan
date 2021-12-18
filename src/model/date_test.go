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
