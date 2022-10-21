package model

import (
	"log"
	"testing"
	"time"
)

func TestValid(t *testing.T) {
	{
		testcase := "regular date"
		date := Date{1949, 12, 21}
		expect := true

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
	{
		testcase := "first of year"
		date := Date{2020, 01, 01}
		expect := true

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
	{
		testcase := "last of year"
		date := Date{2020, 12, 31}
		expect := true

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
	{
		testcase := "last of 30 day month"
		date := Date{2020, 11, 30}
		expect := true

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
	{
		testcase := "last of regular february"
		date := Date{2020, 2, 28}
		expect := true

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
	{
		testcase := "last of leap year february"
		date := Date{2004, 2, 29}
		expect := true

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
	{
		testcase := "zero month"
		date := Date{2004, 0, 10}
		expect := false

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
	{
		testcase := "zero day"
		date := Date{2004, 4, 0}
		expect := false

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
	{
		testcase := "31st november"
		date := Date{2021, 11, 31}
		expect := false

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
	{
		testcase := "32nd october"
		date := Date{2021, 10, 32}
		expect := false

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
	{
		testcase := "29th of regular february"
		date := Date{2021, 2, 29}
		expect := false

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
	{
		testcase := "30th of leap year february"
		date := Date{2004, 2, 30}
		expect := false

		if date.Valid() != expect {
			log.Fatalf("Date validation testcase '%s' (date %s) failed, should be %t but isn't",
				testcase, date.ToString(), expect)
		}
	}
}

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

func TestIsAfter(t *testing.T) {
	{
		testcase := "basic true case"
		a := Date{2021, 12, 19}
		b := Date{2021, 12, 14}

		expected := true
		result := a.IsAfter(b)

		if result != expected {
			log.Fatalf("testcase '%s' for `IsAfter` failed: should be %t but isn't",
				testcase, expected)
		}
	}
	{
		testcase := "basic false case"
		a := Date{2021, 12, 14}
		b := Date{2021, 12, 19}

		expected := false
		result := a.IsAfter(b)

		if result != expected {
			log.Fatalf("testcase '%s' for `IsAfter` failed: should be %t but isn't",
				testcase, expected)
		}
	}
}

func TestDaysUntil(t *testing.T) {
	{
		testcase := "basic case"
		a := Date{2021, 12, 14}
		b := Date{2021, 12, 19}

		expected := 5
		result := a.DaysUntil(b)

		if result != expected {
			log.Fatalf("testcase '%s' for `DaysUntil` failed: got %d, expected %d",
				testcase, result, expected)
		}
	}
	{
		testcase := "zero case"
		a := Date{2021, 12, 14}
		b := Date{2021, 12, 14}

		expected := 0
		result := a.DaysUntil(b)

		if result != expected {
			log.Fatalf("testcase '%s' for `DaysUntil` failed: got %d, expected %d",
				testcase, result, expected)
		}
	}
	{
		testcase := "different month"
		a := Date{2021, 11, 14}
		b := Date{2021, 12, 14}

		expected := 30
		result := a.DaysUntil(b)

		if result != expected {
			log.Fatalf("testcase '%s' for `DaysUntil` failed: got %d, expected %d",
				testcase, result, expected)
		}
	}
	{
		testcase := "different year"
		a := Date{2020, 11, 14}
		b := Date{2021, 12, 14}

		expected := 365 + 30
		result := a.DaysUntil(b)

		if result != expected {
			log.Fatalf("testcase '%s' for `DaysUntil` failed: got %d, expected %d",
				testcase, result, expected)
		}
	}
	{
		testcase := "leap year through feb 29th"
		a := Date{2020, 1, 30}
		b := Date{2020, 3, 30}

		expected := 1 + 29 + 30
		result := a.DaysUntil(b)

		if result != expected {
			log.Fatalf("testcase '%s' for `DaysUntil` failed: got %d, expected %d",
				testcase, result, expected)
		}
	}
	{
		testcase := "same day"
		a := Date{2021, 12, 14}
		b := Date{2021, 12, 14}

		expected := false
		result := a.IsAfter(b)

		if result != expected {
			log.Fatalf("testcase '%s' for `IsAfter` failed: should be %t but isn't",
				testcase, expected)
		}
	}
	{
		testcase := "one day later"
		a := Date{2021, 12, 14}
		b := Date{2021, 12, 15}

		expected := false
		result := a.IsAfter(b)

		if result != expected {
			log.Fatalf("testcase '%s' for `IsAfter` failed: should be %t but isn't",
				testcase, expected)
		}
	}
}

func TestGetDayInWeek(t *testing.T) {
	{
		testcase := "getting Monday for a Wednesday"
		input := Date{2022, 3, 9}

		expected := Date{2022, 3, 7}
		result := input.GetDayInWeek(0)

		if result != expected {
			log.Fatalf("testcase '%s' for `GetDayInWeek` failed: expected %s, got %s",
				testcase, expected.ToString(), result.ToString())
		}
	}

	// TODO
}

func TestGetDayInMonth(t *testing.T) {
	{
		testcase := "getting 1. of month for for a some date"
		input := Date{2022, 3, 9}

		expected := Date{2022, 3, 1}
		result := input.GetDayInMonth(0)

		if result != expected {
			log.Fatalf("testcase '%s' for `GetDayInMonth` failed: expected %s, got %s",
				testcase, expected.ToString(), result.ToString())
		}
	}
}
