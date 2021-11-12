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
