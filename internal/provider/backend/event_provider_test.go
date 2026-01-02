package backend_test

import (
	"testing"
	"time"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/provider"
)

// EventProviderFactory is a function type that creates a new EventProvider instance for testing.
// Each test will get a fresh instance to ensure isolation.
type EventProviderFactory func(t *testing.T) provider.EventProvider

// EventProviderTestOptions configures which tests to run.
// This allows implementations to opt out of tests for unsupported features.
type EventProviderTestOptions struct {
	// SkipCrossDayTests skips tests that involve events spanning multiple days.
	// Set to true for implementations that don't support cross-day events.
	SkipCrossDayTests bool
}

// RunEventProviderTests runs a comprehensive test suite against any EventProvider implementation.
// Pass a factory function that creates a fresh instance for each test.
//
// Example usage:
//
//	func TestMyEventProvider(t *testing.T) {
//	    factory := func(t *testing.T) provider.EventProvider {
//	        provider, err := NewMyEventProvider()
//	        require.NoError(t, err)
//	        return provider
//	    }
//	    RunEventProviderTests(t, factory, EventProviderTestOptions{})
//	}
func RunEventProviderTests(t *testing.T, factory EventProviderFactory, opts EventProviderTestOptions) {
	t.Run("AddEvent", func(t *testing.T) { testAddEvent(t, factory, opts) })
	t.Run("RemoveEvent", func(t *testing.T) { testRemoveEvent(t, factory) })
	t.Run("RemoveEvents", func(t *testing.T) { testRemoveEvents(t, factory) })
	t.Run("GetEvent", func(t *testing.T) { testGetEvent(t, factory) })
	t.Run("GetEventAfter", func(t *testing.T) { testGetEventAfter(t, factory) })
	t.Run("GetEventBefore", func(t *testing.T) { testGetEventBefore(t, factory) })
	t.Run("GetPrecedingEvent", func(t *testing.T) { testGetPrecedingEvent(t, factory) })
	t.Run("GetFollowingEvent", func(t *testing.T) { testGetFollowingEvent(t, factory) })
	t.Run("GetEventsCoveringTimerange", func(t *testing.T) { testGetEventsCoveringTimerange(t, factory) })
	t.Run("SplitEvent", func(t *testing.T) { testSplitEvent(t, factory) })
	t.Run("SetEventStart", func(t *testing.T) { testSetEventStart(t, factory, opts) })
	t.Run("SetEventEnd", func(t *testing.T) { testSetEventEnd(t, factory, opts) })
	t.Run("SetEventTimes", func(t *testing.T) { testSetEventTimes(t, factory, opts) })
	t.Run("OffsetEventStart", func(t *testing.T) { testOffsetEventStart(t, factory, opts) })
	t.Run("OffsetEventEnd", func(t *testing.T) { testOffsetEventEnd(t, factory, opts) })
	t.Run("OffsetEventTimes", func(t *testing.T) { testOffsetEventTimes(t, factory, opts) })
	t.Run("SnapEventStart", func(t *testing.T) { testSnapEventStart(t, factory) })
	t.Run("SnapEventEnd", func(t *testing.T) { testSnapEventEnd(t, factory) })
	t.Run("SnapEventTimes", func(t *testing.T) { testSnapEventTimes(t, factory) })
	t.Run("SnapEventStartPreserveDuration", func(t *testing.T) { testSnapEventStartPreserveDuration(t, factory) })
	t.Run("SnapEventEndPreserveDuration", func(t *testing.T) { testSnapEventEndPreserveDuration(t, factory) })
	t.Run("SetEventName", func(t *testing.T) { testSetEventName(t, factory) })
	t.Run("SetEventCategory", func(t *testing.T) { testSetEventCategory(t, factory) })
	t.Run("SetEventAllData", func(t *testing.T) { testSetEventAllData(t, factory, opts) })
	t.Run("SumUpTimespanByCategory", func(t *testing.T) { testSumUpTimespanByCategory(t, factory) })
	t.Run("CommitState", func(t *testing.T) { testCommitState(t, factory) })
	t.Run("DataProviderInfo", func(t *testing.T) { testDataProviderInfo(t, factory) })
}

// Helper function to create a simple event
func createEvent(name string, category model.CategoryName, start, end time.Time) model.Event {
	return model.Event{
		Name:     name,
		Category: category,
		Start:    start,
		End:      end,
	}
}

// Reference date for tests - a fixed point in time
var testBaseDate = time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)

// Helper to create times on the same day
func timeOnDay(hour, minute int) time.Time {
	return testBaseDate.Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
}

// Helper to create times on a different day (offset from base)
func timeOnDayOffset(dayOffset, hour, minute int) time.Time {
	return testBaseDate.AddDate(0, 0, dayOffset).Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
}

// Test AddEvent functionality
func testAddEvent(t *testing.T, factory EventProviderFactory, opts EventProviderTestOptions) {
	t.Run("basic add", func(t *testing.T) {
		p := factory(t)

		event := createEvent("Test Event", "work", timeOnDay(10, 0), timeOnDay(12, 0))
		id, err := p.AddEvent(event)
		if err != nil {
			t.Fatalf("AddEvent failed: %v", err)
		}
		if id == "" {
			t.Error("Expected non-empty ID")
		}

		// Verify event was added
		retrieved, err := p.GetEvent(id)
		if err != nil {
			t.Fatalf("GetEvent failed: %v", err)
		}
		if retrieved.Name != "Test Event" {
			t.Errorf("Expected name 'Test Event', got '%s'", retrieved.Name)
		}
		if retrieved.Category != "work" {
			t.Errorf("Expected category 'work', got '%s'", retrieved.Category)
		}
	})

	t.Run("add multiple events", func(t *testing.T) {
		p := factory(t)

		id1, err := p.AddEvent(createEvent("Event 1", "cat1", timeOnDay(8, 0), timeOnDay(9, 0)))
		if err != nil {
			t.Fatalf("AddEvent 1 failed: %v", err)
		}
		id2, err := p.AddEvent(createEvent("Event 2", "cat2", timeOnDay(10, 0), timeOnDay(11, 0)))
		if err != nil {
			t.Fatalf("AddEvent 2 failed: %v", err)
		}
		id3, err := p.AddEvent(createEvent("Event 3", "cat3", timeOnDay(14, 0), timeOnDay(15, 0)))
		if err != nil {
			t.Fatalf("AddEvent 3 failed: %v", err)
		}

		// Verify all events exist
		if id1 == id2 || id2 == id3 || id1 == id3 {
			t.Error("Expected unique IDs for each event")
		}

		events, err := p.GetEventsCoveringTimerange(timeOnDay(0, 0), timeOnDay(23, 59))
		if err != nil {
			t.Fatalf("GetEventsCoveringTimerange failed: %v", err)
		}
		if len(events) != 3 {
			t.Errorf("Expected 3 events, got %d", len(events))
		}
	})

	t.Run("add event with existing ID", func(t *testing.T) {
		p := factory(t)

		event1 := createEvent("Event 1", "cat1", timeOnDay(8, 0), timeOnDay(9, 0))
		id1, err := p.AddEvent(event1)
		if err != nil {
			t.Fatalf("AddEvent 1 failed: %v", err)
		}

		// Try to add another event with the same ID
		event2 := createEvent("Event 2", "cat2", timeOnDay(10, 0), timeOnDay(11, 0))
		event2.ID = id1
		_, err = p.AddEvent(event2)
		// Behavior may vary - some implementations may error, others may overwrite
		// This test just verifies it doesn't panic
	})

	if !opts.SkipCrossDayTests {
		t.Run("add cross-day event", func(t *testing.T) {
			p := factory(t)

			event := createEvent("Overnight", "sleep", timeOnDay(22, 0), timeOnDayOffset(1, 6, 0))
			_, err := p.AddEvent(event)
			// Cross-day support varies by implementation
			if err != nil {
				t.Skipf("Cross-day events not supported: %v", err)
			}
		})
	}
}

// Test RemoveEvent functionality
func testRemoveEvent(t *testing.T, factory EventProviderFactory) {
	t.Run("remove existing event", func(t *testing.T) {
		p := factory(t)

		id, err := p.AddEvent(createEvent("To Remove", "cat", timeOnDay(10, 0), timeOnDay(12, 0)))
		if err != nil {
			t.Fatalf("AddEvent failed: %v", err)
		}

		err = p.RemoveEvent(id)
		if err != nil {
			t.Errorf("RemoveEvent failed: %v", err)
		}

		// Verify event is gone
		_, err = p.GetEvent(id)
		if err == nil {
			t.Error("Expected error when getting removed event")
		}
	})

	t.Run("remove non-existent event", func(t *testing.T) {
		p := factory(t)

		err := p.RemoveEvent("non-existent-id")
		if err == nil {
			t.Error("Expected error when removing non-existent event")
		}
	})

	t.Run("remove from multiple events", func(t *testing.T) {
		p := factory(t)

		id1, _ := p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))
		id2, _ := p.AddEvent(createEvent("Event 2", "cat", timeOnDay(10, 0), timeOnDay(11, 0)))
		id3, _ := p.AddEvent(createEvent("Event 3", "cat", timeOnDay(14, 0), timeOnDay(15, 0)))

		err := p.RemoveEvent(id2)
		if err != nil {
			t.Errorf("RemoveEvent failed: %v", err)
		}

		// Verify id2 is gone but others remain
		_, err = p.GetEvent(id2)
		if err == nil {
			t.Error("Expected error when getting removed event")
		}

		_, err = p.GetEvent(id1)
		if err != nil {
			t.Errorf("Event 1 should still exist: %v", err)
		}

		_, err = p.GetEvent(id3)
		if err != nil {
			t.Errorf("Event 3 should still exist: %v", err)
		}
	})
}

// Test RemoveEvents functionality
func testRemoveEvents(t *testing.T, factory EventProviderFactory) {
	t.Run("remove multiple events", func(t *testing.T) {
		p := factory(t)

		id1, _ := p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))
		id2, _ := p.AddEvent(createEvent("Event 2", "cat", timeOnDay(10, 0), timeOnDay(11, 0)))
		id3, _ := p.AddEvent(createEvent("Event 3", "cat", timeOnDay(14, 0), timeOnDay(15, 0)))

		err := p.RemoveEvents([]model.EventID{id1, id3})
		if err != nil {
			t.Errorf("RemoveEvents failed: %v", err)
		}

		// Verify id1 and id3 are gone but id2 remains
		_, err = p.GetEvent(id1)
		if err == nil {
			t.Error("Expected error when getting removed event 1")
		}

		_, err = p.GetEvent(id3)
		if err == nil {
			t.Error("Expected error when getting removed event 3")
		}

		_, err = p.GetEvent(id2)
		if err != nil {
			t.Errorf("Event 2 should still exist: %v", err)
		}
	})

	t.Run("remove empty list", func(t *testing.T) {
		p := factory(t)

		err := p.RemoveEvents([]model.EventID{})
		if err != nil {
			t.Errorf("RemoveEvents with empty list failed: %v", err)
		}
	})

	t.Run("remove with non-existent ID in list", func(t *testing.T) {
		p := factory(t)

		id1, _ := p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))

		err := p.RemoveEvents([]model.EventID{id1, "non-existent"})
		if err == nil {
			t.Error("Expected error when removing with non-existent ID")
		}
	})
}

// Test GetEvent functionality
func testGetEvent(t *testing.T, factory EventProviderFactory) {
	t.Run("get existing event", func(t *testing.T) {
		p := factory(t)

		originalEvent := createEvent("My Event", "work", timeOnDay(10, 0), timeOnDay(12, 0))
		id, err := p.AddEvent(originalEvent)
		if err != nil {
			t.Fatalf("AddEvent failed: %v", err)
		}

		retrieved, err := p.GetEvent(id)
		if err != nil {
			t.Fatalf("GetEvent failed: %v", err)
		}

		if retrieved.Name != originalEvent.Name {
			t.Errorf("Name mismatch: expected '%s', got '%s'", originalEvent.Name, retrieved.Name)
		}
		if retrieved.Category != originalEvent.Category {
			t.Errorf("Category mismatch: expected '%s', got '%s'", originalEvent.Category, retrieved.Category)
		}
		if !retrieved.Start.Equal(originalEvent.Start) {
			t.Errorf("Start mismatch: expected %v, got %v", originalEvent.Start, retrieved.Start)
		}
		if !retrieved.End.Equal(originalEvent.End) {
			t.Errorf("End mismatch: expected %v, got %v", originalEvent.End, retrieved.End)
		}
	})

	t.Run("get non-existent event", func(t *testing.T) {
		p := factory(t)

		_, err := p.GetEvent("non-existent-id")
		if err == nil {
			t.Error("Expected error when getting non-existent event")
		}
	})

	t.Run("get returns independent copy", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Original", "cat", timeOnDay(10, 0), timeOnDay(12, 0)))

		retrieved1, _ := p.GetEvent(id)
		retrieved1.Name = "Modified"

		retrieved2, _ := p.GetEvent(id)
		if retrieved2.Name != "Original" {
			t.Error("Modifying retrieved event should not affect stored event")
		}
	})
}

// Test GetEventAfter functionality
func testGetEventAfter(t *testing.T, factory EventProviderFactory) {
	t.Run("find event after time", func(t *testing.T) {
		p := factory(t)

		p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))
		id2, _ := p.AddEvent(createEvent("Event 2", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))
		p.AddEvent(createEvent("Event 3", "cat", timeOnDay(16, 0), timeOnDay(18, 0)))

		event, err := p.GetEventAfter(timeOnDay(10, 0))
		if err != nil {
			t.Fatalf("GetEventAfter failed: %v", err)
		}
		if event == nil {
			t.Fatal("Expected to find an event")
		}
		if event.ID != id2 {
			t.Errorf("Expected event 2, got %s", event.Name)
		}
	})

	t.Run("no event after time", func(t *testing.T) {
		p := factory(t)

		p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))

		event, err := p.GetEventAfter(timeOnDay(20, 0))
		if err != nil {
			t.Fatalf("GetEventAfter failed: %v", err)
		}
		if event != nil {
			t.Error("Expected nil when no event after time")
		}
	})

	t.Run("event starting at exact time", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Exact", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		event, err := p.GetEventAfter(timeOnDay(12, 0))
		if err != nil {
			t.Fatalf("GetEventAfter failed: %v", err)
		}
		if event == nil {
			t.Fatal("Expected to find the event starting at exact time")
		}
		if event.ID != id {
			t.Error("Expected the event starting at exact time")
		}
	})

	t.Run("empty provider", func(t *testing.T) {
		p := factory(t)

		event, err := p.GetEventAfter(timeOnDay(12, 0))
		if err != nil {
			t.Fatalf("GetEventAfter failed: %v", err)
		}
		if event != nil {
			t.Error("Expected nil for empty provider")
		}
	})
}

// Test GetEventBefore functionality
func testGetEventBefore(t *testing.T, factory EventProviderFactory) {
	t.Run("find event before time", func(t *testing.T) {
		p := factory(t)

		id1, _ := p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))
		p.AddEvent(createEvent("Event 2", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))
		p.AddEvent(createEvent("Event 3", "cat", timeOnDay(16, 0), timeOnDay(18, 0)))

		event, err := p.GetEventBefore(timeOnDay(10, 0))
		if err != nil {
			t.Fatalf("GetEventBefore failed: %v", err)
		}
		if event == nil {
			t.Fatal("Expected to find an event")
		}
		if event.ID != id1 {
			t.Errorf("Expected event 1, got %s", event.Name)
		}
	})

	t.Run("no event before time", func(t *testing.T) {
		p := factory(t)

		p.AddEvent(createEvent("Event 1", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		event, err := p.GetEventBefore(timeOnDay(8, 0))
		if err != nil {
			t.Fatalf("GetEventBefore failed: %v", err)
		}
		if event != nil {
			t.Error("Expected nil when no event before time")
		}
	})

	t.Run("event ending at exact time", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Exact", "cat", timeOnDay(10, 0), timeOnDay(12, 0)))

		event, err := p.GetEventBefore(timeOnDay(12, 0))
		if err != nil {
			t.Fatalf("GetEventBefore failed: %v", err)
		}
		if event == nil {
			t.Fatal("Expected to find the event ending at exact time")
		}
		if event.ID != id {
			t.Error("Expected the event ending at exact time")
		}
	})
}

// Test GetPrecedingEvent functionality
func testGetPrecedingEvent(t *testing.T, factory EventProviderFactory) {
	t.Run("find preceding event", func(t *testing.T) {
		p := factory(t)

		id1, _ := p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))
		id2, _ := p.AddEvent(createEvent("Event 2", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))
		p.AddEvent(createEvent("Event 3", "cat", timeOnDay(16, 0), timeOnDay(18, 0)))

		event, err := p.GetPrecedingEvent(id2)
		if err != nil {
			t.Fatalf("GetPrecedingEvent failed: %v", err)
		}
		if event == nil {
			t.Fatal("Expected to find preceding event")
		}
		if event.ID != id1 {
			t.Errorf("Expected event 1 as preceding, got %s", event.Name)
		}
	})

	t.Run("no preceding event", func(t *testing.T) {
		p := factory(t)

		id1, _ := p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))

		event, err := p.GetPrecedingEvent(id1)
		if err != nil {
			t.Fatalf("GetPrecedingEvent failed: %v", err)
		}
		if event != nil {
			t.Error("Expected nil when no preceding event")
		}
	})

	t.Run("non-existent event ID", func(t *testing.T) {
		p := factory(t)

		_, err := p.GetPrecedingEvent("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent event ID")
		}
	})

	t.Run("preceding with overlapping events", func(t *testing.T) {
		p := factory(t)

		id1, _ := p.AddEvent(createEvent("Event 1", "cat", timeOnDay(10, 0), timeOnDay(14, 0)))
		id2, _ := p.AddEvent(createEvent("Event 2", "cat", timeOnDay(12, 0), timeOnDay(16, 0)))
		id3, _ := p.AddEvent(createEvent("Event 3", "cat", timeOnDay(14, 0), timeOnDay(18, 0)))

		// Event 2 should have Event 1 as preceding (starts before Event 2)
		event, err := p.GetPrecedingEvent(id2)
		if err != nil {
			t.Fatalf("GetPrecedingEvent failed: %v", err)
		}
		if event == nil || event.ID != id1 {
			t.Error("Expected Event 1 as preceding Event 2")
		}

		// Event 3 should have Event 2 as preceding
		event, err = p.GetPrecedingEvent(id3)
		if err != nil {
			t.Fatalf("GetPrecedingEvent failed: %v", err)
		}
		if event == nil || event.ID != id2 {
			t.Error("Expected Event 2 as preceding Event 3")
		}
	})
}

// Test GetFollowingEvent functionality
func testGetFollowingEvent(t *testing.T, factory EventProviderFactory) {
	t.Run("find following event", func(t *testing.T) {
		p := factory(t)

		id1, _ := p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))
		id2, _ := p.AddEvent(createEvent("Event 2", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))
		p.AddEvent(createEvent("Event 3", "cat", timeOnDay(16, 0), timeOnDay(18, 0)))

		event, err := p.GetFollowingEvent(id1)
		if err != nil {
			t.Fatalf("GetFollowingEvent failed: %v", err)
		}
		if event == nil {
			t.Fatal("Expected to find following event")
		}
		if event.ID != id2 {
			t.Errorf("Expected event 2 as following, got %s", event.Name)
		}
	})

	t.Run("no following event", func(t *testing.T) {
		p := factory(t)

		id1, _ := p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))

		event, err := p.GetFollowingEvent(id1)
		if err != nil {
			t.Fatalf("GetFollowingEvent failed: %v", err)
		}
		if event != nil {
			t.Error("Expected nil when no following event")
		}
	})

	t.Run("non-existent event ID", func(t *testing.T) {
		p := factory(t)

		_, err := p.GetFollowingEvent("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent event ID")
		}
	})
}

// Test GetEventsCoveringTimerange functionality
func testGetEventsCoveringTimerange(t *testing.T, factory EventProviderFactory) {
	t.Run("get events in range", func(t *testing.T) {
		p := factory(t)

		p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))
		p.AddEvent(createEvent("Event 2", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))
		p.AddEvent(createEvent("Event 3", "cat", timeOnDay(16, 0), timeOnDay(18, 0)))

		events, err := p.GetEventsCoveringTimerange(timeOnDay(10, 0), timeOnDay(15, 0))
		if err != nil {
			t.Fatalf("GetEventsCoveringTimerange failed: %v", err)
		}
		if len(events) != 1 {
			t.Errorf("Expected 1 event fully in range, got %d", len(events))
		}
	})

	t.Run("get events in range with partials", func(t *testing.T) {
		p := factory(t)

		p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))
		p.AddEvent(createEvent("Event 2", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))
		p.AddEvent(createEvent("Event 3", "cat", timeOnDay(16, 0), timeOnDay(18, 0)))

		events, err := p.GetEventsCoveringTimerange(timeOnDay(10, 0), timeOnDay(17, 0))
		if err != nil {
			t.Fatalf("GetEventsCoveringTimerange failed: %v", err)
		}
		if len(events) != 2 {
			t.Errorf("Expected 2 events fully in range, got %d", len(events))
		}
	})

	t.Run("empty range", func(t *testing.T) {
		p := factory(t)

		p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))

		events, err := p.GetEventsCoveringTimerange(timeOnDay(12, 0), timeOnDay(14, 0))
		if err != nil {
			t.Fatalf("GetEventsCoveringTimerange failed: %v", err)
		}
		if len(events) != 0 {
			t.Errorf("Expected 0 events, got %d", len(events))
		}
	})

	t.Run("all events in range", func(t *testing.T) {
		p := factory(t)

		p.AddEvent(createEvent("Event 1", "cat", timeOnDay(8, 0), timeOnDay(9, 0)))
		p.AddEvent(createEvent("Event 2", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))
		p.AddEvent(createEvent("Event 3", "cat", timeOnDay(16, 0), timeOnDay(18, 0)))

		events, err := p.GetEventsCoveringTimerange(timeOnDay(0, 0), timeOnDay(23, 59))
		if err != nil {
			t.Fatalf("GetEventsCoveringTimerange failed: %v", err)
		}
		if len(events) != 3 {
			t.Errorf("Expected 3 events, got %d", len(events))
		}
	})

	t.Run("invalid range", func(t *testing.T) {
		p := factory(t)

		_, err := p.GetEventsCoveringTimerange(timeOnDay(14, 0), timeOnDay(10, 0))
		if err == nil {
			t.Error("Expected error for invalid range (end before start)")
		}
	})

	t.Run("empty time range", func(t *testing.T) {
		p := factory(t)

		_, err := p.GetEventsCoveringTimerange(timeOnDay(12, 0), timeOnDay(12, 0))
		if err == nil {
			t.Error("Expected error for empty range (start equals end)")
		}
	})
}

// Test SplitEvent functionality
func testSplitEvent(t *testing.T, factory EventProviderFactory) {
	t.Run("basic split", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("To Split", "cat", timeOnDay(10, 0), timeOnDay(14, 0)))
		splitTime := timeOnDay(12, 0)

		err := p.SplitEvent(id, splitTime)
		if err != nil {
			t.Fatalf("SplitEvent failed: %v", err)
		}

		// Original event should end at split time
		original, err := p.GetEvent(id)
		if err != nil {
			t.Fatalf("GetEvent failed: %v", err)
		}
		if !original.End.Equal(splitTime) {
			t.Errorf("Original event should end at split time, got %v", original.End)
		}

		// Should have two events now
		events, _ := p.GetEventsCoveringTimerange(timeOnDay(0, 0), timeOnDay(23, 59))
		if len(events) != 2 {
			t.Errorf("Expected 2 events after split, got %d", len(events))
		}
	})

	t.Run("split at start", func(t *testing.T) {
		p := factory(t)

		start := timeOnDay(10, 0)
		id, _ := p.AddEvent(createEvent("To Split", "cat", start, timeOnDay(14, 0)))

		err := p.SplitEvent(id, start)
		if err == nil {
			t.Error("Expected error when splitting at start time")
		}
	})

	t.Run("split at end", func(t *testing.T) {
		p := factory(t)

		end := timeOnDay(14, 0)
		id, _ := p.AddEvent(createEvent("To Split", "cat", timeOnDay(10, 0), end))

		err := p.SplitEvent(id, end)
		if err == nil {
			t.Error("Expected error when splitting at end time")
		}
	})

	t.Run("split outside range", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("To Split", "cat", timeOnDay(10, 0), timeOnDay(14, 0)))

		err := p.SplitEvent(id, timeOnDay(8, 0))
		if err == nil {
			t.Error("Expected error when splitting before start")
		}

		err = p.SplitEvent(id, timeOnDay(16, 0))
		if err == nil {
			t.Error("Expected error when splitting after end")
		}
	})

	t.Run("split non-existent event", func(t *testing.T) {
		p := factory(t)

		err := p.SplitEvent("non-existent", timeOnDay(12, 0))
		if err == nil {
			t.Error("Expected error when splitting non-existent event")
		}
	})
}

// Test SetEventStart functionality
func testSetEventStart(t *testing.T, factory EventProviderFactory, opts EventProviderTestOptions) {
	t.Run("basic set start", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))
		newStart := timeOnDay(10, 0)

		err := p.SetEventStart(id, newStart)
		if err != nil {
			t.Fatalf("SetEventStart failed: %v", err)
		}

		event, _ := p.GetEvent(id)
		if !event.Start.Equal(newStart) {
			t.Errorf("Expected start %v, got %v", newStart, event.Start)
		}
	})

	t.Run("set start after end", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		err := p.SetEventStart(id, timeOnDay(16, 0))
		if err == nil {
			t.Error("Expected error when setting start after end")
		}
	})

	t.Run("set start equal to end", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		err := p.SetEventStart(id, timeOnDay(14, 0))
		if err == nil {
			t.Error("Expected error when setting start equal to end")
		}
	})

	t.Run("set start for non-existent event", func(t *testing.T) {
		p := factory(t)

		err := p.SetEventStart("non-existent", timeOnDay(10, 0))
		if err == nil {
			t.Error("Expected error for non-existent event")
		}
	})

	if !opts.SkipCrossDayTests {
		t.Run("set start to different day", func(t *testing.T) {
			p := factory(t)

			id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

			err := p.SetEventStart(id, timeOnDayOffset(-1, 12, 0))
			// Behavior varies by implementation
			if err != nil {
				t.Skipf("Cross-day events not supported: %v", err)
			}
		})
	}
}

// Test SetEventEnd functionality
func testSetEventEnd(t *testing.T, factory EventProviderFactory, opts EventProviderTestOptions) {
	t.Run("basic set end", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))
		newEnd := timeOnDay(16, 0)

		err := p.SetEventEnd(id, newEnd)
		if err != nil {
			t.Fatalf("SetEventEnd failed: %v", err)
		}

		event, _ := p.GetEvent(id)
		if !event.End.Equal(newEnd) {
			t.Errorf("Expected end %v, got %v", newEnd, event.End)
		}
	})

	t.Run("set end before start", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		err := p.SetEventEnd(id, timeOnDay(10, 0))
		if err == nil {
			t.Error("Expected error when setting end before start")
		}
	})

	t.Run("set end equal to start", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		err := p.SetEventEnd(id, timeOnDay(12, 0))
		if err == nil {
			t.Error("Expected error when setting end equal to start")
		}
	})

	if !opts.SkipCrossDayTests {
		t.Run("set end to midnight", func(t *testing.T) {
			p := factory(t)

			id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(22, 0), timeOnDay(23, 0)))
			midnight := timeOnDayOffset(1, 0, 0)

			err := p.SetEventEnd(id, midnight)
			if err != nil {
				t.Fatalf("SetEventEnd to midnight failed: %v", err)
			}

			event, _ := p.GetEvent(id)
			if !event.End.Equal(midnight) {
				t.Errorf("Expected end at midnight, got %v", event.End)
			}
		})
	}

	if !opts.SkipCrossDayTests {
		t.Run("set end to different day", func(t *testing.T) {
			p := factory(t)

			id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

			err := p.SetEventEnd(id, timeOnDayOffset(1, 12, 0))
			// Behavior varies by implementation
			if err != nil {
				t.Skipf("Cross-day events not supported: %v", err)
			}
		})
	}
}

// Test SetEventTimes functionality
func testSetEventTimes(t *testing.T, factory EventProviderFactory, opts EventProviderTestOptions) {
	t.Run("basic set times", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))
		newStart := timeOnDay(10, 0)
		newEnd := timeOnDay(16, 0)

		err := p.SetEventTimes(id, newStart, newEnd)
		if err != nil {
			t.Fatalf("SetEventTimes failed: %v", err)
		}

		event, _ := p.GetEvent(id)
		if !event.Start.Equal(newStart) {
			t.Errorf("Expected start %v, got %v", newStart, event.Start)
		}
		if !event.End.Equal(newEnd) {
			t.Errorf("Expected end %v, got %v", newEnd, event.End)
		}
	})

	t.Run("move to different day", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))
		newStart := timeOnDayOffset(5, 10, 0)
		newEnd := timeOnDayOffset(5, 16, 0)

		err := p.SetEventTimes(id, newStart, newEnd)
		if err != nil {
			t.Fatalf("SetEventTimes failed: %v", err)
		}

		event, _ := p.GetEvent(id)
		if !event.Start.Equal(newStart) {
			t.Errorf("Expected start %v, got %v", newStart, event.Start)
		}
	})

	t.Run("invalid start after end", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		err := p.SetEventTimes(id, timeOnDay(16, 0), timeOnDay(14, 0))
		if err == nil {
			t.Error("Expected error when start is after end")
		}
	})

	t.Run("invalid start equals end", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		err := p.SetEventTimes(id, timeOnDay(12, 0), timeOnDay(12, 0))
		if err == nil {
			t.Error("Expected error when start equals end")
		}
	})

	if !opts.SkipCrossDayTests {
		t.Run("set times crossing days", func(t *testing.T) {
			p := factory(t)

			id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

			err := p.SetEventTimes(id, timeOnDay(22, 0), timeOnDayOffset(1, 6, 0))
			if err != nil {
				t.Skipf("Cross-day events not supported: %v", err)
			}
		})
	}
}

// Test OffsetEventStart functionality
func testOffsetEventStart(t *testing.T, factory EventProviderFactory, opts EventProviderTestOptions) {
	t.Run("positive offset", func(t *testing.T) {
		p := factory(t)

		originalStart := timeOnDay(12, 0)
		id, _ := p.AddEvent(createEvent("Event", "cat", originalStart, timeOnDay(14, 0)))

		newStart, err := p.OffsetEventStart(id, time.Hour)
		if err != nil {
			t.Fatalf("OffsetEventStart failed: %v", err)
		}

		expectedStart := originalStart.Add(time.Hour)
		if !newStart.Equal(expectedStart) {
			t.Errorf("Expected start %v, got %v", expectedStart, newStart)
		}

		event, _ := p.GetEvent(id)
		if !event.Start.Equal(expectedStart) {
			t.Errorf("Event start mismatch")
		}
	})

	t.Run("negative offset", func(t *testing.T) {
		p := factory(t)

		originalStart := timeOnDay(12, 0)
		id, _ := p.AddEvent(createEvent("Event", "cat", originalStart, timeOnDay(14, 0)))

		newStart, err := p.OffsetEventStart(id, -time.Hour)
		if err != nil {
			t.Fatalf("OffsetEventStart failed: %v", err)
		}

		expectedStart := originalStart.Add(-time.Hour)
		if !newStart.Equal(expectedStart) {
			t.Errorf("Expected start %v, got %v", expectedStart, newStart)
		}
	})

	t.Run("offset past end", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		_, err := p.OffsetEventStart(id, 3*time.Hour)
		if err == nil {
			t.Error("Expected error when offset moves start past end")
		}
	})

	t.Run("offset to exactly end", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		_, err := p.OffsetEventStart(id, 2*time.Hour)
		if err == nil {
			t.Error("Expected error when offset moves start to exactly end")
		}
	})

	if !opts.SkipCrossDayTests {
		t.Run("offset to different day", func(t *testing.T) {
			p := factory(t)

			id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(2, 0), timeOnDay(4, 0)))

			_, err := p.OffsetEventStart(id, -3*time.Hour)
			if err != nil {
				t.Skipf("Cross-day events not supported: %v", err)
			}
		})
	}
}

// Test OffsetEventEnd functionality
func testOffsetEventEnd(t *testing.T, factory EventProviderFactory, opts EventProviderTestOptions) {
	t.Run("positive offset", func(t *testing.T) {
		p := factory(t)

		originalEnd := timeOnDay(14, 0)
		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), originalEnd))

		newEnd, err := p.OffsetEventEnd(id, time.Hour)
		if err != nil {
			t.Fatalf("OffsetEventEnd failed: %v", err)
		}

		expectedEnd := originalEnd.Add(time.Hour)
		if !newEnd.Equal(expectedEnd) {
			t.Errorf("Expected end %v, got %v", expectedEnd, newEnd)
		}
	})

	t.Run("negative offset", func(t *testing.T) {
		p := factory(t)

		originalEnd := timeOnDay(14, 0)
		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), originalEnd))

		newEnd, err := p.OffsetEventEnd(id, -time.Hour)
		if err != nil {
			t.Fatalf("OffsetEventEnd failed: %v", err)
		}

		expectedEnd := originalEnd.Add(-time.Hour)
		if !newEnd.Equal(expectedEnd) {
			t.Errorf("Expected end %v, got %v", expectedEnd, newEnd)
		}
	})

	t.Run("offset before start", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		_, err := p.OffsetEventEnd(id, -3*time.Hour)
		if err == nil {
			t.Error("Expected error when offset moves end before start")
		}
	})

	t.Run("offset to midnight", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(22, 0), timeOnDay(23, 0)))

		newEnd, err := p.OffsetEventEnd(id, time.Hour)
		if err != nil {
			t.Fatalf("OffsetEventEnd to midnight failed: %v", err)
		}

		expectedEnd := timeOnDayOffset(1, 0, 0)
		if !newEnd.Equal(expectedEnd) {
			t.Errorf("Expected end at midnight %v, got %v", expectedEnd, newEnd)
		}
	})

	if !opts.SkipCrossDayTests {
		t.Run("offset to different day", func(t *testing.T) {
			p := factory(t)

			id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(22, 0), timeOnDay(23, 0)))

			_, err := p.OffsetEventEnd(id, 2*time.Hour)
			if err != nil {
				t.Skipf("Cross-day events not supported: %v", err)
			}
		})
	}
}

// Test OffsetEventTimes functionality
func testOffsetEventTimes(t *testing.T, factory EventProviderFactory, opts EventProviderTestOptions) {
	t.Run("positive offset", func(t *testing.T) {
		p := factory(t)

		originalStart := timeOnDay(12, 0)
		originalEnd := timeOnDay(14, 0)
		id, _ := p.AddEvent(createEvent("Event", "cat", originalStart, originalEnd))

		newStart, newEnd, err := p.OffsetEventTimes(id, time.Hour)
		if err != nil {
			t.Fatalf("OffsetEventTimes failed: %v", err)
		}

		expectedStart := originalStart.Add(time.Hour)
		expectedEnd := originalEnd.Add(time.Hour)
		if !newStart.Equal(expectedStart) {
			t.Errorf("Expected start %v, got %v", expectedStart, newStart)
		}
		if !newEnd.Equal(expectedEnd) {
			t.Errorf("Expected end %v, got %v", expectedEnd, newEnd)
		}
	})

	t.Run("zero offset", func(t *testing.T) {
		p := factory(t)

		originalStart := timeOnDay(12, 0)
		originalEnd := timeOnDay(14, 0)
		id, _ := p.AddEvent(createEvent("Event", "cat", originalStart, originalEnd))

		newStart, newEnd, err := p.OffsetEventTimes(id, 0)
		if err != nil {
			t.Fatalf("OffsetEventTimes failed: %v", err)
		}

		if !newStart.Equal(originalStart) {
			t.Errorf("Expected start unchanged")
		}
		if !newEnd.Equal(originalEnd) {
			t.Errorf("Expected end unchanged")
		}
	})

	t.Run("preserves duration", func(t *testing.T) {
		p := factory(t)

		originalStart := timeOnDay(12, 0)
		originalEnd := timeOnDay(14, 0)
		originalDuration := originalEnd.Sub(originalStart)
		id, _ := p.AddEvent(createEvent("Event", "cat", originalStart, originalEnd))

		newStart, newEnd, err := p.OffsetEventTimes(id, 2*time.Hour)
		if err != nil {
			t.Fatalf("OffsetEventTimes failed: %v", err)
		}

		newDuration := newEnd.Sub(newStart)
		if newDuration != originalDuration {
			t.Errorf("Duration changed from %v to %v", originalDuration, newDuration)
		}
	})

	if !opts.SkipCrossDayTests {
		t.Run("offset to different day", func(t *testing.T) {
			p := factory(t)

			id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(22, 0), timeOnDay(23, 0)))

			_, _, err := p.OffsetEventTimes(id, 2*time.Hour)
			if err != nil {
				t.Skipf("Cross-day events not supported: %v", err)
			}
		})
	}
}

// Test SnapEventStart functionality
func testSnapEventStart(t *testing.T, factory EventProviderFactory) {
	t.Run("snap to 15 minutes", func(t *testing.T) {
		p := factory(t)

		// 12:34 should snap to 12:30
		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 34), timeOnDay(14, 0)))

		newStart, err := p.SnapEventStart(id, 15*time.Minute)
		if err != nil {
			t.Fatalf("SnapEventStart failed: %v", err)
		}

		expectedStart := timeOnDay(12, 30)
		if !newStart.Equal(expectedStart) {
			t.Errorf("Expected start %v, got %v", expectedStart, newStart)
		}
	})

	t.Run("snap to hour rounds up", func(t *testing.T) {
		p := factory(t)

		// 12:34 should snap to 13:00 (rounds up at half)
		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 34), timeOnDay(15, 0)))

		newStart, err := p.SnapEventStart(id, time.Hour)
		if err != nil {
			t.Fatalf("SnapEventStart failed: %v", err)
		}

		expectedStart := timeOnDay(13, 0)
		if !newStart.Equal(expectedStart) {
			t.Errorf("Expected start %v, got %v", expectedStart, newStart)
		}
	})

	t.Run("snap would put start at or after end", func(t *testing.T) {
		p := factory(t)

		// 12:34 snapping to hour gives 13:00 which is after end 12:40
		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 34), timeOnDay(12, 40)))

		_, err := p.SnapEventStart(id, time.Hour)
		if err == nil {
			t.Error("Expected error when snap would put start at or after end")
		}
	})
}

// Test SnapEventEnd functionality
func testSnapEventEnd(t *testing.T, factory EventProviderFactory) {
	t.Run("snap to 15 minutes", func(t *testing.T) {
		p := factory(t)

		// 13:12 should snap to 13:15
		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(13, 12)))

		newEnd, err := p.SnapEventEnd(id, 15*time.Minute)
		if err != nil {
			t.Fatalf("SnapEventEnd failed: %v", err)
		}

		expectedEnd := timeOnDay(13, 15)
		if !newEnd.Equal(expectedEnd) {
			t.Errorf("Expected end %v, got %v", expectedEnd, newEnd)
		}
	})

	t.Run("snap to hour rounds down", func(t *testing.T) {
		p := factory(t)

		// 13:12 should snap to 13:00
		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(13, 12)))

		newEnd, err := p.SnapEventEnd(id, time.Hour)
		if err != nil {
			t.Fatalf("SnapEventEnd failed: %v", err)
		}

		expectedEnd := timeOnDay(13, 0)
		if !newEnd.Equal(expectedEnd) {
			t.Errorf("Expected end %v, got %v", expectedEnd, newEnd)
		}
	})

	t.Run("snap would put end at or before start", func(t *testing.T) {
		p := factory(t)

		// 12:12 snapping to hour gives 12:00 which is before start 12:10
		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 10), timeOnDay(12, 12)))

		_, err := p.SnapEventEnd(id, time.Hour)
		if err == nil {
			t.Error("Expected error when snap would put end at or before start")
		}
	})
}

// Test SnapEventTimes functionality
func testSnapEventTimes(t *testing.T, factory EventProviderFactory) {
	t.Run("snap both to 15 minutes", func(t *testing.T) {
		p := factory(t)

		// 12:34-13:17 should snap to 12:30-13:15
		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 34), timeOnDay(13, 17)))

		newStart, newEnd, err := p.SnapEventTimes(id, 15*time.Minute)
		if err != nil {
			t.Fatalf("SnapEventTimes failed: %v", err)
		}

		expectedStart := timeOnDay(12, 30)
		expectedEnd := timeOnDay(13, 15)
		if !newStart.Equal(expectedStart) {
			t.Errorf("Expected start %v, got %v", expectedStart, newStart)
		}
		if !newEnd.Equal(expectedEnd) {
			t.Errorf("Expected end %v, got %v", expectedEnd, newEnd)
		}
	})

	t.Run("snap would make start >= end", func(t *testing.T) {
		p := factory(t)

		// 12:50-13:10 snapping to hour gives 13:00-13:00
		id, _ := p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 50), timeOnDay(13, 10)))

		_, _, err := p.SnapEventTimes(id, time.Hour)
		if err == nil {
			t.Error("Expected error when snap would make start >= end")
		}
	})
}

// Test SnapEventStartPreserveDuration functionality
func testSnapEventStartPreserveDuration(t *testing.T, factory EventProviderFactory) {
	t.Run("preserve duration on snap", func(t *testing.T) {
		p := factory(t)

		originalStart := timeOnDay(12, 34)
		originalEnd := timeOnDay(14, 34)
		originalDuration := originalEnd.Sub(originalStart)
		id, _ := p.AddEvent(createEvent("Event", "cat", originalStart, originalEnd))

		newStart, newEnd, err := p.SnapEventStartPreseveDuration(id, 15*time.Minute)
		if err != nil {
			t.Fatalf("SnapEventStartPreserveDuration failed: %v", err)
		}

		newDuration := newEnd.Sub(newStart)
		if newDuration != originalDuration {
			t.Errorf("Duration changed from %v to %v", originalDuration, newDuration)
		}

		// Start should be snapped
		expectedStart := timeOnDay(12, 30)
		if !newStart.Equal(expectedStart) {
			t.Errorf("Expected start %v, got %v", expectedStart, newStart)
		}
	})
}

// Test SnapEventEndPreserveDuration functionality
func testSnapEventEndPreserveDuration(t *testing.T, factory EventProviderFactory) {
	t.Run("preserve duration on snap", func(t *testing.T) {
		p := factory(t)

		originalStart := timeOnDay(12, 17)
		originalEnd := timeOnDay(14, 17)
		originalDuration := originalEnd.Sub(originalStart)
		id, _ := p.AddEvent(createEvent("Event", "cat", originalStart, originalEnd))

		newStart, newEnd, err := p.SnapEventEndPreseveDuration(id, 15*time.Minute)
		if err != nil {
			t.Fatalf("SnapEventEndPreserveDuration failed: %v", err)
		}

		newDuration := newEnd.Sub(newStart)
		if newDuration != originalDuration {
			t.Errorf("Duration changed from %v to %v", originalDuration, newDuration)
		}

		// End should be snapped
		expectedEnd := timeOnDay(14, 15)
		if !newEnd.Equal(expectedEnd) {
			t.Errorf("Expected end %v, got %v", expectedEnd, newEnd)
		}
	})
}

// Test SetEventName functionality
func testSetEventName(t *testing.T, factory EventProviderFactory) {
	t.Run("basic set name", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Original Name", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		err := p.SetEventName(id, "New Name")
		if err != nil {
			t.Fatalf("SetEventName failed: %v", err)
		}

		event, _ := p.GetEvent(id)
		if event.Name != "New Name" {
			t.Errorf("Expected name 'New Name', got '%s'", event.Name)
		}
	})

	t.Run("set empty name", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Original Name", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		err := p.SetEventName(id, "")
		if err != nil {
			t.Fatalf("SetEventName to empty failed: %v", err)
		}

		event, _ := p.GetEvent(id)
		if event.Name != "" {
			t.Errorf("Expected empty name, got '%s'", event.Name)
		}
	})

	t.Run("set name for non-existent event", func(t *testing.T) {
		p := factory(t)

		err := p.SetEventName("non-existent", "New Name")
		if err == nil {
			t.Error("Expected error for non-existent event")
		}
	})
}

// Test SetEventCategory functionality
func testSetEventCategory(t *testing.T, factory EventProviderFactory) {
	t.Run("basic set category", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Event", "old-cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		err := p.SetEventCategory(id, "new-cat")
		if err != nil {
			t.Fatalf("SetEventCategory failed: %v", err)
		}

		event, _ := p.GetEvent(id)
		if event.Category != "new-cat" {
			t.Errorf("Expected category 'new-cat', got '%s'", event.Category)
		}
	})

	t.Run("set category for non-existent event", func(t *testing.T) {
		p := factory(t)

		err := p.SetEventCategory("non-existent", "new-cat")
		if err == nil {
			t.Error("Expected error for non-existent event")
		}
	})
}

// Test SetEventAllData functionality
func testSetEventAllData(t *testing.T, factory EventProviderFactory, opts EventProviderTestOptions) {
	t.Run("update all data", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Original", "old-cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		newEvent := createEvent("Updated", "new-cat", timeOnDay(10, 0), timeOnDay(16, 0))
		err := p.SetEventAllData(id, newEvent)
		if err != nil {
			t.Fatalf("SetEventAllData failed: %v", err)
		}

		event, _ := p.GetEvent(id)
		if event.Name != "Updated" {
			t.Errorf("Expected name 'Updated', got '%s'", event.Name)
		}
		if event.Category != "new-cat" {
			t.Errorf("Expected category 'new-cat', got '%s'", event.Category)
		}
		if !event.Start.Equal(timeOnDay(10, 0)) {
			t.Errorf("Start time mismatch")
		}
		if !event.End.Equal(timeOnDay(16, 0)) {
			t.Errorf("End time mismatch")
		}
	})

	t.Run("move to different day", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Original", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		newEvent := createEvent("Moved", "cat", timeOnDayOffset(5, 10, 0), timeOnDayOffset(5, 16, 0))
		err := p.SetEventAllData(id, newEvent)
		if err != nil {
			t.Fatalf("SetEventAllData failed: %v", err)
		}

		event, _ := p.GetEvent(id)
		if !event.Start.Equal(timeOnDayOffset(5, 10, 0)) {
			t.Errorf("Start time mismatch after move")
		}
	})

	t.Run("mismatched ID", func(t *testing.T) {
		p := factory(t)

		id, _ := p.AddEvent(createEvent("Original", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		newEvent := createEvent("Updated", "cat", timeOnDay(10, 0), timeOnDay(16, 0))
		newEvent.ID = "different-id"
		err := p.SetEventAllData(id, newEvent)
		if err == nil {
			t.Error("Expected error for mismatched ID")
		}
	})

	t.Run("non-existent event", func(t *testing.T) {
		p := factory(t)

		newEvent := createEvent("Updated", "cat", timeOnDay(10, 0), timeOnDay(16, 0))
		err := p.SetEventAllData("non-existent", newEvent)
		if err == nil {
			t.Error("Expected error for non-existent event")
		}
	})

	if !opts.SkipCrossDayTests {
		t.Run("set times crossing days", func(t *testing.T) {
			p := factory(t)

			id, _ := p.AddEvent(createEvent("Original", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

			newEvent := createEvent("Updated", "cat", timeOnDay(22, 0), timeOnDayOffset(1, 6, 0))
			err := p.SetEventAllData(id, newEvent)
			if err != nil {
				t.Skipf("Cross-day events not supported: %v", err)
			}
		})
	}
}

// Test SumUpTimespanByCategory functionality
func testSumUpTimespanByCategory(t *testing.T, factory EventProviderFactory) {
	t.Run("sum up single category", func(t *testing.T) {
		p := factory(t)

		// 2 hours of work
		p.AddEvent(createEvent("Work 1", "work", timeOnDay(8, 0), timeOnDay(10, 0)))
		// 1 hour of work
		p.AddEvent(createEvent("Work 2", "work", timeOnDay(14, 0), timeOnDay(15, 0)))

		summary, err := p.SumUpTimespanByCategory(timeOnDay(0, 0), timeOnDay(23, 59))
		if err != nil {
			t.Fatalf("SumUpTimespanByCategory failed: %v", err)
		}

		workDuration := summary["work"]
		if workDuration != 3*time.Hour {
			t.Errorf("Expected 3 hours of work, got %v", workDuration)
		}
	})

	t.Run("sum up multiple categories", func(t *testing.T) {
		p := factory(t)

		p.AddEvent(createEvent("Work", "work", timeOnDay(8, 0), timeOnDay(12, 0)))        // 4 hours
		p.AddEvent(createEvent("Lunch", "break", timeOnDay(12, 0), timeOnDay(13, 0)))     // 1 hour
		p.AddEvent(createEvent("Meeting", "meeting", timeOnDay(14, 0), timeOnDay(15, 0))) // 1 hour

		summary, err := p.SumUpTimespanByCategory(timeOnDay(0, 0), timeOnDay(23, 59))
		if err != nil {
			t.Fatalf("SumUpTimespanByCategory failed: %v", err)
		}

		if summary["work"] != 4*time.Hour {
			t.Errorf("Expected 4 hours of work, got %v", summary["work"])
		}
		if summary["break"] != 1*time.Hour {
			t.Errorf("Expected 1 hour of break, got %v", summary["break"])
		}
		if summary["meeting"] != 1*time.Hour {
			t.Errorf("Expected 1 hour of meeting, got %v", summary["meeting"])
		}
	})

	t.Run("empty range different day", func(t *testing.T) {
		p := factory(t)

		// Add event on base day
		p.AddEvent(createEvent("Work", "work", timeOnDay(8, 0), timeOnDay(12, 0)))

		// Query a different day where no events exist
		summary, err := p.SumUpTimespanByCategory(timeOnDayOffset(10, 8, 0), timeOnDayOffset(10, 16, 0))
		if err != nil {
			t.Fatalf("SumUpTimespanByCategory failed: %v", err)
		}

		if len(summary) != 0 {
			t.Errorf("Expected empty summary for day with no events, got %v", summary)
		}
	})
}

// Test CommitState functionality
func testCommitState(t *testing.T, factory EventProviderFactory) {
	t.Run("commit after changes", func(t *testing.T) {
		p := factory(t)

		// Initially should be fully committed
		committed, err := p.FullyCommitted()
		if err != nil {
			t.Fatalf("FullyCommitted failed: %v", err)
		}
		if !committed {
			t.Error("New provider should be fully committed")
		}

		// Add an event
		p.AddEvent(createEvent("Event", "cat", timeOnDay(12, 0), timeOnDay(14, 0)))

		// May not be fully committed after change
		committed, _ = p.FullyCommitted()

		// Commit
		err = p.CommitState()
		if err != nil {
			t.Fatalf("CommitState failed: %v", err)
		}

		// Should be fully committed after commit
		committed, err = p.FullyCommitted()
		if err != nil {
			t.Fatalf("FullyCommitted failed: %v", err)
		}
		if !committed {
			t.Error("Should be fully committed after CommitState")
		}
	})
}

// Test DataProviderInfo functionality
func testDataProviderInfo(t *testing.T, factory EventProviderFactory) {
	t.Run("get storage location info", func(t *testing.T) {
		p := factory(t)

		info, err := p.GetStorageLocationInfo()
		if err != nil {
			t.Fatalf("GetStorageLocationInfo failed: %v", err)
		}
		if info == "" {
			t.Error("Expected non-empty storage location info")
		}
	})

	t.Run("fully committed", func(t *testing.T) {
		p := factory(t)

		committed, err := p.FullyCommitted()
		if err != nil {
			t.Fatalf("FullyCommitted failed: %v", err)
		}
		// New provider should be fully committed (no pending changes)
		if !committed {
			t.Error("New provider should be fully committed")
		}
	})
}
