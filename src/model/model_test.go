package model

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"
)

func TestStartsDuring(t *testing.T) {
	{
		testcase := "starts during"

		// +-----+
		// | a +---+
		// |   | b |
		// +---|   |
		//     +---+

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("06:00|07:30|work|Get Started", make([]Category, 0))

		expected := true
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts after"

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("06:40|07:30|work|Get Started", make([]Category, 0))

		expected := false
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts at the same time"

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("05:50|07:30|work|Get Started", make([]Category, 0))

		expected := true
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts flush"

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("06:30|07:30|work|Get Started", make([]Category, 0))

		expected := false
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts during (contained, should not matter)"

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("05:55|06:20|work|Get Started", make([]Category, 0))

		expected := true
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts before"

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("04:50|07:30|work|Get Started", make([]Category, 0))

		expected := false
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
}

func TestIsContainedIn(t *testing.T) {
	{
		testcase := "is contained"

		// +-----+
		// | a +---+
		// |   | b |
		// |   |   |
		// |   +---+
		// +-----+

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("05:55|06:20|work|Get Started", make([]Category, 0))

		expected := true
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts after"

		// +-----+
		// | a   |
		// |     |
		// |     |
		// +-----+
		//
		// +-----+
		// | b   |
		// |     |
		// +-----+

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("06:40|07:30|work|Get Started", make([]Category, 0))

		expected := false
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "is fully flush"

		// +-----+
		// | a   |
		// |     |
		// |     |
		// +-----+
		// | b   |
		// |     |
		// +-----+

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("06:30|07:30|work|Get Started", make([]Category, 0))

		expected := false
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "only starts during but ends after"

		// +-----+
		// | a +---+
		// |   | b |
		// +---|   |
		//     +---+

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("05:55|06:40|work|Get Started", make([]Category, 0))

		expected := false
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts before ends during"

		//     +---+
		// +---| b |
		// | a |   |
		// |   +---+
		// +----+

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("05:55|06:40|work|Get Started", make([]Category, 0))

		expected := false
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "perfectly flush"

		// +---+---+
		// | a | b |
		// |   |   |
		// +---+---+

		a := NewEvent("05:50|06:30|eating|Breakfast", make([]Category, 0))
		b := NewEvent("05:50|06:30|work|Get Started", make([]Category, 0))

		expected := true
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
}

func TestSumUpByCategory(t *testing.T) {
	defaultEmptyCategories := make([]Category, 0)
	{
		testcase := "single event"
		model := NewDay()
		model.Events = []*Event{
			NewEvent("05:50|06:30|eating|Breakfast", defaultEmptyCategories),
		}
		expected := map[Category]int{
			{"eating", 0}: 40,
		}
		result := model.SumUpByCategory()
		if !reflect.DeepEqual(result, expected) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, result)
		}
	}
	{
		testcase := "multiple events of same category"
		model := NewDay()
		model.Events = []*Event{
			NewEvent("05:50|06:30|eating|Breakfast", defaultEmptyCategories),
			NewEvent("11:30|12:10|eating|Lunch", defaultEmptyCategories),
			NewEvent("18:15|19:00|eating|Dinner", defaultEmptyCategories),
		}
		expected := map[Category]int{
			{"eating", 0}: 125,
		}
		result := model.SumUpByCategory()
		if !reflect.DeepEqual(result, expected) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, result)
		}
	}
	{
		testcase := "multiple events of different categories"
		model := NewDay()
		model.Events = []*Event{
			NewEvent("05:50|06:30|eating|Breakfast", defaultEmptyCategories),
			NewEvent("11:30|12:10|eating|Lunch", defaultEmptyCategories),
			NewEvent("18:15|19:00|cooking|Dinner", defaultEmptyCategories),
		}
		expected := map[Category]int{
			{"eating", 0}:  80,
			{"cooking", 0}: 45,
		}
		result := model.SumUpByCategory()
		if !reflect.DeepEqual(result, expected) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, result)
		}
	}
	{
		testcase := "events that overlap partially"
		model := NewDay()
		model.Events = []*Event{
			NewEvent("01:00|02:00|a|A1", defaultEmptyCategories),
			NewEvent("01:30|02:30|a|A2", defaultEmptyCategories),
		}
		expected := map[Category]int{
			{"a", 0}: 90,
		}
		result := model.SumUpByCategory()
		if !reflect.DeepEqual(result, expected) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, result)
		}
	}
	{
		testcase := "one event that contains another"
		model := NewDay()
		model.Events = []*Event{
			NewEvent("01:00|02:00|a|A main", defaultEmptyCategories),
			NewEvent("01:15|01:45|a|A subevent", defaultEmptyCategories),
		}
		expected := map[Category]int{
			{"a", 0}: 60,
		}
		result := model.SumUpByCategory()
		if !reflect.DeepEqual(result, expected) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, result)
		}
	}
}

func TestFlatten(t *testing.T) {
	defaultEmptyCategories := make([]Category, 0)
	{
		testcase := "single event"
		day := *NewDay()
		day.Events = []*Event{
			NewEvent("05:50|06:30|eating|Breakfast", defaultEmptyCategories),
		}
		dayExpected := day
		day.Flatten()
		if !reflect.DeepEqual(day.Events, dayExpected.Events) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, day)
		}
	}
	{
		testcase := "doubled event"
		day := *NewDay()
		day.Events = []*Event{
			NewEvent("05:50|06:30|eating|Breakfast", defaultEmptyCategories),
			NewEvent("05:50|06:30|eating|Breakfast", defaultEmptyCategories),
		}
		dayExpected := day
		day.Flatten()
		if reflect.DeepEqual(day.Events, dayExpected.Events) {
			log.Fatalf("test case '%s' failed, these should be different:\n%#v\n%#v", testcase, day, dayExpected)
		}
		if len(day.Events) != 1 {
			log.Fatalf("test case '%s' failed: len is %d", testcase, len(day.Events))
		}
	}
	{
		testcase := "overlapping events of same cat"
		input := *NewDay()
		input.Events = []*Event{
			NewEvent("05:50|06:30|eating|Breakfast", defaultEmptyCategories),
			NewEvent("06:00|07:00|eating|Other", defaultEmptyCategories),
		}
		expected := *NewDay()
		expected.Events = []*Event{
			NewEvent("05:50|07:00|eating|Breakfast", defaultEmptyCategories),
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(&input, &expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		testcase := "contained event of same cat"
		input := *NewDay()
		input.Events = []*Event{
			NewEvent("05:50|07:00|eating|Breakfast", defaultEmptyCategories),
			NewEvent("06:00|06:30|eating|Other", defaultEmptyCategories),
		}
		expected := *NewDay()
		expected.Events = []*Event{
			NewEvent("05:50|07:00|eating|Breakfast", defaultEmptyCategories),
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(&input, &expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		categories := make([]Category, 0)
		categories = append(categories, Category{Name: "eating", Priority: 0})
		categories = append(categories, Category{Name: "work", Priority: 20})

		testcase := "overlap with higher priority (low then high)"
		input := *NewDay()
		input.Events = []*Event{
			NewEvent("05:50|06:30|eating|Breakfast", categories),
			NewEvent("06:00|08:00|work|Work", categories),
		}
		expected := *NewDay()
		expected.Events = []*Event{
			NewEvent("05:50|06:00|eating|Breakfast", categories),
			NewEvent("06:00|08:00|work|Work", categories),
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(&input, &expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		categories := make([]Category, 0)
		categories = append(categories, Category{Name: "eating", Priority: 0})
		categories = append(categories, Category{Name: "work", Priority: 20})

		testcase := "overlap with higher priority (high then low)"
		input := *NewDay()
		input.Events = []*Event{
			NewEvent("09:00|12:00|work|Work", categories),
			NewEvent("11:30|12:30|eating|Lunch", categories),
		}
		expected := *NewDay()
		expected.Events = []*Event{
			NewEvent("09:00|12:00|work|Work", categories),
			NewEvent("12:00|12:30|eating|Lunch", categories),
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(&input, &expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		categories := make([]Category, 0)
		categories = append(categories, Category{Name: "eating", Priority: 0})
		categories = append(categories, Category{Name: "work", Priority: 20})

		testcase := "low prio contained in higher prio"
		input := *NewDay()
		input.Events = []*Event{
			NewEvent("09:00|14:00|work|Work", categories),
			NewEvent("12:00|12:30|eating|Lunch", categories),
		}
		expected := *NewDay()
		expected.Events = []*Event{
			NewEvent("09:00|14:00|work|Work", categories),
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(&input, &expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected.ToSlice(), input.ToSlice())
		}
	}
	{
		categories := make([]Category, 0)
		categories = append(categories, Category{Name: "eating", Priority: 0})
		categories = append(categories, Category{Name: "work", Priority: 20})

		testcase := "high prio contained in lower prio"
		input := *NewDay()
		input.Events = []*Event{
			NewEvent("12:00|13:00|eating|Lunch", categories),
			NewEvent("12:25|12:35|work|Check that one Email quickly", categories),
		}
		expected := *NewDay()
		expected.Events = []*Event{
			NewEvent("12:00|12:25|eating|Lunch", categories),
			NewEvent("12:25|12:35|work|Check that one Email quickly", categories),
			NewEvent("12:35|13:00|eating|Lunch", categories),
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(&input, &expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected.ToSlice(), input.ToSlice())
		}
	}
	{
		categories := make([]Category, 0)
		categories = append(categories, Category{Name: "eating", Priority: 0})
		categories = append(categories, Category{Name: "work", Priority: 20})

		testcase := "high prio contained in lower prio such that lower former becomes zero-length"
		input := *NewDay()
		input.Events = []*Event{
			NewEvent("12:00|13:00|eating|Lunch", categories),
			NewEvent("12:00|12:10|work|Get suckered into checking that thing real quick", categories),
		}
		expected := *NewDay()
		expected.Events = []*Event{
			NewEvent("12:00|12:10|work|Get suckered into checking that thing real quick", categories),
			NewEvent("12:10|13:00|eating|Lunch", categories),
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(&input, &expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		categories := make([]Category, 0)
		categories = append(categories, Category{Name: "eating", Priority: 0})
		categories = append(categories, Category{Name: "work", Priority: 20})

		testcase := "high prio contained in lower prio such that lower latter becomes zero-length"
		input := *NewDay()
		input.Events = []*Event{
			NewEvent("12:00|13:00|eating|Lunch", categories),
			NewEvent("12:40|13:00|work|Reply to that one email even though it could wait 15 minutes", categories),
		}
		expected := *NewDay()
		expected.Events = []*Event{
			NewEvent("12:00|12:40|eating|Lunch", categories),
			NewEvent("12:40|13:00|work|Reply to that one email even though it could wait 15 minutes", categories),
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(&input, &expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		categories := make([]Category, 0)
		categories = append(categories, Category{Name: "a", Priority: 0})
		categories = append(categories, Category{Name: "b", Priority: 1})
		categories = append(categories, Category{Name: "c", Priority: 2})

		testcase := "high prio contained in lower prio such that lower former becomes zero-length, but sort is needed"
		input := *NewDay()
		input.Events = []*Event{
			NewEvent("12:00|13:00|a|A", categories),
			NewEvent("12:00|12:20|b|B", categories),
			NewEvent("12:10|12:30|c|C", categories),
		}
		expected := *NewDay()
		expected.Events = []*Event{
			NewEvent("12:00|12:10|b|B", categories),
			NewEvent("12:10|12:30|c|C", categories),
			NewEvent("12:30|13:00|a|A", categories),
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(&input, &expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		categories := make([]Category, 0)
		categories = append(categories, Category{Name: "eating", Priority: 0})
		categories = append(categories, Category{Name: "work", Priority: 20})

		testcase := "high prio starting right at start of lower prio such that lower becomes zero-length"
		input := *NewDay()
		input.Events = []*Event{
			NewEvent("12:00|13:00|eating|Lunch", categories),
			NewEvent("12:00|15:00|work|Work through lunch break and beyond", categories),
		}
		expected := *NewDay()
		expected.Events = []*Event{
			NewEvent("12:00|15:00|work|Work through lunch break and beyond", categories),
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(&input, &expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
}

// comparison helper
func eventsEqualExceptingIDs(a, b *Day) bool {
	if len(a.Events) != len(b.Events) {
		fmt.Fprintln(os.Stderr, "lengths different:", len(a.Events), len(b.Events))
		return false
	}

	for i := range a.Events {
		if a.Events[i].Name != b.Events[i].Name {
			fmt.Fprintln(os.Stderr, "Name different:", a.Events[i].Name, b.Events[i].Name)
			return false
		}
		if a.Events[i].Cat != b.Events[i].Cat {
			fmt.Fprintln(os.Stderr, "Cat different:", a.Events[i].Cat, b.Events[i].Cat)
			return false
		}
		if a.Events[i].Start != b.Events[i].Start {
			fmt.Fprintln(os.Stderr, "Start different:", a.Events[i].Start, b.Events[i].Start)
			return false
		}
		if a.Events[i].End != b.Events[i].End {
			fmt.Fprintln(os.Stderr, "End different:", a.Events[i].End, b.Events[i].End)
			return false
		}
	}

	return true
}
