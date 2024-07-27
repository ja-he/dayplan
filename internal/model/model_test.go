package model_test

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/ja-he/dayplan/internal/model"
)

var baseDate = time.Date(2022, 11, 13, 0, 0, 0, 0, time.UTC)

func TestStartsDuring(t *testing.T) {
	{
		testcase := "starts during"

		// +-----+
		// | a +---+
		// |   | b |
		// +---|   |
		//     +---+

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(6 * time.Hour), End: baseDate.Add(7*time.Hour + 30*time.Minute)}

		expected := true
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts after"

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(6*time.Hour + 40*time.Minute), End: baseDate.Add(7*time.Hour + 30*time.Minute)}

		expected := false
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts at the same time"

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(7*time.Hour + 30*time.Minute)}

		expected := true
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts flush"

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(6*time.Hour + 30*time.Minute), End: baseDate.Add(7*time.Hour + 30*time.Minute)}

		expected := false
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts during (contained, should not matter)"

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(5*time.Hour + 55*time.Minute), End: baseDate.Add(6*time.Hour + 20*time.Minute)}

		expected := true
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts before"

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(4*time.Hour + 50*time.Minute), End: baseDate.Add(7*time.Hour + 30*time.Minute)}

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

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(5*time.Hour + 55*time.Minute), End: baseDate.Add(6*time.Hour + 20*time.Minute)}

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

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(6*time.Hour + 40*time.Minute), End: baseDate.Add(7*time.Hour + 30*time.Minute)}

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

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(6*time.Hour + 30*time.Minute), End: baseDate.Add(7*time.Hour + 30*time.Minute)}

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

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(5*time.Hour + 55*time.Minute), End: baseDate.Add(6*time.Hour + 40*time.Minute)}

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

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(5*time.Hour + 55*time.Minute), End: baseDate.Add(6*time.Hour + 40*time.Minute)}

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

		a := &model.Event{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}
		b := &model.Event{Name: "Get Started", Cat: model.Category{Name: "work"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)}

		expected := true
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
}

func TestSumUpByCategory(t *testing.T) {
	{
		testcase := "single event"
		eventList := model.EventList{}
		eventList.Events = []*model.Event{
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)},
		}
		expected := map[model.Category]int{
			{
				Name:       "eating",
				Priority:   0,
				Goal:       nil,
				Deprecated: false,
			}: 40,
		}
		result := eventList.SumUpByCategory()
		if !reflect.DeepEqual(result, expected) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, result)
		}
	}
	{
		testcase := "multiple events of same category"
		eventList := model.EventList{}
		eventList.Events = []*model.Event{
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)},
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(11*time.Hour + 30*time.Minute), End: baseDate.Add(12*time.Hour + 10*time.Minute)},
			{Name: "Dinner", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(18*time.Hour + 15*time.Minute), End: baseDate.Add(19 * time.Hour)},
		}
		expected := map[model.Category]int{
			{
				Name:       "eating",
				Priority:   0,
				Goal:       nil,
				Deprecated: false,
			}: 125,
		}
		result := eventList.SumUpByCategory()
		if !reflect.DeepEqual(result, expected) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, result)
		}
	}
	{
		testcase := "multiple events of different categories"
		eventList := model.EventList{}
		eventList.Events = []*model.Event{
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)},
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(11*time.Hour + 30*time.Minute), End: baseDate.Add(12*time.Hour + 10*time.Minute)},
			{Name: "Dinner", Cat: model.Category{Name: "cooking"}, Start: baseDate.Add(18*time.Hour + 15*time.Minute), End: baseDate.Add(19 * time.Hour)},
		}
		expected := map[model.Category]int{
			{
				Name:       "eating",
				Priority:   0,
				Goal:       nil,
				Deprecated: false,
			}: 80,
			{
				Name:       "cooking",
				Priority:   0,
				Goal:       nil,
				Deprecated: false,
			}: 45,
		}
		result := eventList.SumUpByCategory()
		if !reflect.DeepEqual(result, expected) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, result)
		}
	}
	{
		testcase := "events that overlap partially"
		eventList := model.EventList{}
		eventList.Events = []*model.Event{
			{Name: "A1", Cat: model.Category{Name: "a"}, Start: baseDate.Add(1 * time.Hour), End: baseDate.Add(2 * time.Hour)},
			{Name: "A2", Cat: model.Category{Name: "a"}, Start: baseDate.Add(1*time.Hour + 30*time.Minute), End: baseDate.Add(2*time.Hour + 30*time.Minute)},
		}
		expected := map[model.Category]int{
			{
				Name:       "a",
				Priority:   0,
				Goal:       nil,
				Deprecated: false,
			}: 90,
		}
		result := eventList.SumUpByCategory()
		if !reflect.DeepEqual(result, expected) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, result)
		}
	}
	{
		testcase := "one event that contains another"
		eventList := model.EventList{}
		eventList.Events = []*model.Event{
			{Name: "A main", Cat: model.Category{Name: "a"}, Start: baseDate.Add(1 * time.Hour), End: baseDate.Add(2 * time.Hour)},
			{Name: "A subevent", Cat: model.Category{Name: "a"}, Start: baseDate.Add(1*time.Hour + 15*time.Minute), End: baseDate.Add(1*time.Hour + 45*time.Minute)},
		}
		expected := map[model.Category]int{
			{
				Name:       "a",
				Priority:   0,
				Goal:       nil,
				Deprecated: false,
			}: 60,
		}
		result := eventList.SumUpByCategory()
		if !reflect.DeepEqual(result, expected) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, result)
		}
	}
}

func TestFlatten(t *testing.T) {
	{
		testcase := "single event"
		day := &model.EventList{}
		day.Events = []*model.Event{
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)},
		}
		dayExpected := day
		day.Flatten()
		if !reflect.DeepEqual(day.Events, dayExpected.Events) {
			log.Fatalf("test case '%s' failed:\n%#v", testcase, day)
		}
	}
	{
		testcase := "doubled event"
		day := &model.EventList{}
		day.Events = []*model.Event{
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)},
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)},
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
		input := &model.EventList{}
		input.Events = []*model.Event{
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)},
			{Name: "Other", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(6 * time.Hour), End: baseDate.Add(7 * time.Hour)},
		}
		expected := &model.EventList{}
		expected.Events = []*model.Event{
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(7 * time.Hour)},
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(input, expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		testcase := "contained event of same cat"
		input := &model.EventList{}
		input.Events = []*model.Event{
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(7 * time.Hour)},
			{Name: "Other", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(6 * time.Hour), End: baseDate.Add(6*time.Hour + 30*time.Minute)},
		}
		expected := &model.EventList{}
		expected.Events = []*model.Event{
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(7 * time.Hour)},
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(input, expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		testcase := "overlap with higher priority (low then high)"
		input := &model.EventList{}
		input.Events = []*model.Event{
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6*time.Hour + 30*time.Minute)},
			{Name: "Work", Cat: model.Category{Name: "work"}, Start: baseDate.Add(6 * time.Hour), End: baseDate.Add(8 * time.Hour)},
		}
		expected := &model.EventList{}
		expected.Events = []*model.Event{
			{Name: "Breakfast", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(5*time.Hour + 50*time.Minute), End: baseDate.Add(6 * time.Hour)},
			{Name: "Work", Cat: model.Category{Name: "work"}, Start: baseDate.Add(6 * time.Hour), End: baseDate.Add(8 * time.Hour)},
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(input, expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		testcase := "overlap with higher priority (high then low)"
		input := &model.EventList{}
		input.Events = []*model.Event{
			{Name: "Work", Cat: model.Category{Name: "work"}, Start: baseDate.Add(9 * time.Hour), End: baseDate.Add(12 * time.Hour)},
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(11*time.Hour + 30*time.Minute), End: baseDate.Add(12*time.Hour + 30*time.Minute)},
		}
		expected := &model.EventList{}
		expected.Events = []*model.Event{
			{Name: "Work", Cat: model.Category{Name: "work"}, Start: baseDate.Add(9 * time.Hour), End: baseDate.Add(12 * time.Hour)},
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(12*time.Hour + 30*time.Minute)},
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(input, expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		testcase := "low prio contained in higher prio"
		input := &model.EventList{}
		input.Events = []*model.Event{
			{Name: "Work", Cat: model.Category{Name: "work"}, Start: baseDate.Add(9 * time.Hour), End: baseDate.Add(14 * time.Hour)},
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(12*time.Hour + 30*time.Minute)},
		}
		expected := &model.EventList{}
		expected.Events = []*model.Event{
			{Name: "Work", Cat: model.Category{Name: "work"}, Start: baseDate.Add(9 * time.Hour), End: baseDate.Add(14 * time.Hour)},
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(input, expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		testcase := "high prio contained in lower prio"
		input := &model.EventList{}
		input.Events = []*model.Event{
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(13 * time.Hour)},
			{Name: "Check that one Email quickly", Cat: model.Category{Name: "work"}, Start: baseDate.Add(12*time.Hour + 25*time.Minute), End: baseDate.Add(12*time.Hour + 35*time.Minute)},
		}
		expected := &model.EventList{}
		expected.Events = []*model.Event{
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(12*time.Hour + 25*time.Minute)},
			{Name: "Check that one Email quickly", Cat: model.Category{Name: "work"}, Start: baseDate.Add(12*time.Hour + 25*time.Minute), End: baseDate.Add(12*time.Hour + 35*time.Minute)},
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(12*time.Hour + 35*time.Minute), End: baseDate.Add(13 * time.Hour)},
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(input, expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		testcase := "high prio contained in lower prio such that lower former becomes zero-length"
		input := &model.EventList{}
		input.Events = []*model.Event{
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(13 * time.Hour)},
			{Name: "Get suckered into checking that thing real quick", Cat: model.Category{Name: "work"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(12*time.Hour + 10*time.Minute)},
		}
		expected := &model.EventList{}
		expected.Events = []*model.Event{
			{Name: "Get suckered into checking that thing real quick", Cat: model.Category{Name: "work"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(12*time.Hour + 10*time.Minute)},
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(12*time.Hour + 10*time.Minute), End: baseDate.Add(13 * time.Hour)},
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(input, expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		testcase := "high prio contained in lower prio such that lower latter becomes zero-length"
		input := &model.EventList{}
		input.Events = []*model.Event{
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(13 * time.Hour)},
			{Name: "Reply to that one email even though it could wait 15 minutes", Cat: model.Category{Name: "work"}, Start: baseDate.Add(12*time.Hour + 40*time.Minute), End: baseDate.Add(13 * time.Hour)},
		}
		expected := &model.EventList{}
		expected.Events = []*model.Event{
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(12*time.Hour + 40*time.Minute)},
			{Name: "Reply to that one email even though it could wait 15 minutes", Cat: model.Category{Name: "work"}, Start: baseDate.Add(12*time.Hour + 40*time.Minute), End: baseDate.Add(13 * time.Hour)},
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(input, expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		testcase := "high prio contained in lower prio such that lower former becomes zero-length, but sort is needed"
		input := &model.EventList{}
		input.Events = []*model.Event{
			{Name: "A", Cat: model.Category{Name: "a"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(13 * time.Hour)},
			{Name: "B", Cat: model.Category{Name: "b"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(12*time.Hour + 20*time.Minute)},
			{Name: "C", Cat: model.Category{Name: "c"}, Start: baseDate.Add(12*time.Hour + 10*time.Minute), End: baseDate.Add(12*time.Hour + 30*time.Minute)},
		}
		expected := &model.EventList{}
		expected.Events = []*model.Event{
			{Name: "B", Cat: model.Category{Name: "b"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(12*time.Hour + 10*time.Minute)},
			{Name: "C", Cat: model.Category{Name: "c"}, Start: baseDate.Add(12*time.Hour + 10*time.Minute), End: baseDate.Add(12*time.Hour + 30*time.Minute)},
			{Name: "A", Cat: model.Category{Name: "a"}, Start: baseDate.Add(12*time.Hour + 30*time.Minute), End: baseDate.Add(13 * time.Hour)},
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(input, expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
	{
		testcase := "high prio starting right at start of lower prio such that lower becomes zero-length"
		input := &model.EventList{}
		input.Events = []*model.Event{
			{Name: "Lunch", Cat: model.Category{Name: "eating"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(13 * time.Hour)},
			{Name: "Work through lunch break and beyond", Cat: model.Category{Name: "work"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(15 * time.Hour)},
		}
		expected := &model.EventList{}
		expected.Events = []*model.Event{
			{Name: "Work through lunch break and beyond", Cat: model.Category{Name: "work"}, Start: baseDate.Add(12 * time.Hour), End: baseDate.Add(15 * time.Hour)},
		}

		input.Flatten()
		if !eventsEqualExceptingIDs(input, expected) {
			log.Fatalf("test case '%s' failed, expected (a) but got (b)\n (a): %#v\n (b): %#v", testcase, expected, input)
		}
	}
}

// comparison helper
func eventsEqualExceptingIDs(a, b *model.EventList) bool {
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
