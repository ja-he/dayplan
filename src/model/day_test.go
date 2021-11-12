package model

import (
	"log"
	"testing"
	"time"
)

func TestToWeekday(t *testing.T) {
	{
		day := Day{2021, 11, 12}
		expected := time.Friday
		result := day.ToWeekday()
		if result != expected {
			log.Fatalf("%s should be weekday %s not %s", day.ToString(), ToString(expected), ToString(result))
		}
	}
}

func TestWeek(t *testing.T) {
	{
		day := Day{2021, 11, 12}
		expStart, expEnd := Day{2021, 11, 8}, Day{2021, 11, 14}
		resStart, resEnd := day.Week()
		if resStart != expStart || resEnd != expEnd {
			log.Fatalf("%s is bounded by (%s,%s) not (%s,%s)", day.ToString(), expStart.ToString(), expEnd.ToString(), resStart.ToString(), resEnd.ToString())
		}
	}
}
