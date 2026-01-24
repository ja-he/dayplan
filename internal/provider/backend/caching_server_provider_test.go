package backend

import (
	"os"
	"testing"
	"time"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func timeptr(t time.Time) *time.Time {
	r := new(time.Time)
	*r = t
	return r
}

var doesNotMatterFallbackTime = time.Time{} // for now

// TestCachingServerClientDataProvider_TwoClientSync tests that changes sync between two clients.
func TestCachingServerClientDataProvider_TwoClientSync(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Create two separate SQLite databases
	dbFile1, err := os.CreateTemp("", "dayplan-test-client1-*.db")
	require.NoError(t, err)
	dbPath1 := dbFile1.Name()
	dbFile1.Close()
	defer os.Remove(dbPath1)

	dbFile2, err := os.CreateTemp("", "dayplan-test-client2-*.db")
	require.NoError(t, err)
	dbPath2 := dbFile2.Name()
	dbFile2.Close()
	defer os.Remove(dbPath2)

	serverURL := "http://localhost:8080"

	// Create client 1
	client1, err := NewCachingServerClientDataProvider(
		CachingServerClientConfig{DBPath: dbPath1, ServerURL: serverURL},
		nil,
	)
	require.NoError(t, err)
	defer client1.Close()

	// Create client 2
	client2, err := NewCachingServerClientDataProvider(
		CachingServerClientConfig{DBPath: dbPath2, ServerURL: serverURL},
		nil,
	)
	require.NoError(t, err)
	defer client2.Close()

	// Both clients login
	require.NoError(t, client1.Login("testuser", "testpass123"))
	require.NoError(t, client2.Login("testuser", "testpass123"))

	// Initial sync for both
	<-client1.TriggerSync()
	<-client2.TriggerSync()

	// Client 1 creates an event
	event := model.Event{
		Name:     "Shared Event",
		Category: "meeting",
		Start:    time.Now().UTC().Add(time.Hour).Truncate(time.Second),
		End:      timeptr(time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)),
	}
	eventID, err := client1.AddEvent(event)
	require.NoError(t, err)
	t.Logf("Client 1 created event: %s", eventID)

	// Client 1 syncs to push the event
	err = <-client1.TriggerSync()
	require.NoError(t, err)

	// Client 2 syncs to pull the event
	err = <-client2.TriggerSync()
	require.NoError(t, err)

	// Client 2 should now have the event
	retrievedEvent, err := client2.GetEvent(eventID)
	require.NoError(t, err)
	assert.Equal(t, "Shared Event", retrievedEvent.Name)
	assert.Equal(t, model.CategoryName("meeting"), retrievedEvent.Category)
	t.Logf("Client 2 retrieved event: %+v", retrievedEvent)

	// Client 2 updates the event
	err = client2.SetEventName(eventID, "Updated Shared Event")
	require.NoError(t, err)

	// Client 2 syncs to push the update
	err = <-client2.TriggerSync()
	require.NoError(t, err)

	// Client 1 syncs to pull the update
	err = <-client1.TriggerSync()
	require.NoError(t, err)

	// Client 1 should see the updated event
	updatedEvent, err := client1.GetEvent(eventID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Shared Event", updatedEvent.Name)
	t.Logf("Client 1 sees updated event: %+v", updatedEvent)

	// Client 1 deletes the event
	err = client1.RemoveEvent(eventID)
	require.NoError(t, err)

	// Client 1 syncs to push the deletion
	err = <-client1.TriggerSync()
	require.NoError(t, err)

	// Client 2 syncs to pull the deletion
	err = <-client2.TriggerSync()
	require.NoError(t, err)

	// Client 2 should not find the event
	_, err = client2.GetEvent(eventID)
	assert.Error(t, err, "Event should be deleted")
	t.Log("Client 2 confirms event is deleted")
}

// Integration test - requires server running at localhost:8080
// Run with: go test -v -tags=integration ./internal/provider/backend/...

func TestCachingServerClientDataProvider_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Use a temp file for the SQLite database
	dbFile, err := os.CreateTemp("", "dayplan-test-*.db")
	require.NoError(t, err)
	dbPath := dbFile.Name()
	dbFile.Close()
	defer os.Remove(dbPath)

	// Create provider
	cfg := CachingServerClientConfig{
		DBPath:    dbPath,
		ServerURL: "http://localhost:8080",
	}

	provider, err := NewCachingServerClientDataProvider(cfg, nil)
	require.NoError(t, err)
	defer provider.Close()

	// Test Login
	t.Run("Login", func(t *testing.T) {
		err := provider.Login("testuser", "testpass123")
		require.NoError(t, err)

		status := provider.SyncStatus()
		t.Logf("Status after login: %+v", status)
	})

	// Wait for initial sync
	time.Sleep(2 * time.Second)

	// Test AddEvent
	var eventID model.EventID
	t.Run("AddEvent", func(t *testing.T) {
		event := model.Event{
			Name:     "Test Event",
			Category: "work",
			Start:    time.Now().UTC().Truncate(time.Second),
			End:      timeptr(time.Now().UTC().Add(time.Hour).Truncate(time.Second)),
		}

		id, err := provider.AddEvent(event)
		require.NoError(t, err)
		assert.NotEmpty(t, id)
		eventID = id
		t.Logf("Created event with ID: %s", id)
	})

	// Test GetEvent
	t.Run("GetEvent", func(t *testing.T) {
		event, err := provider.GetEvent(eventID)
		require.NoError(t, err)
		assert.Equal(t, "Test Event", event.Name)
		assert.Equal(t, model.CategoryName("work"), event.Category)
		t.Logf("Retrieved event: %+v", event)
	})

	// Test SetEventName
	t.Run("SetEventName", func(t *testing.T) {
		err := provider.SetEventName(eventID, "Updated Event")
		require.NoError(t, err)

		event, err := provider.GetEvent(eventID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Event", event.Name)
	})

	// Test SetEventCategory
	t.Run("SetEventCategory", func(t *testing.T) {
		err := provider.SetEventCategory(eventID, "meeting")
		require.NoError(t, err)

		event, err := provider.GetEvent(eventID)
		require.NoError(t, err)
		assert.Equal(t, model.CategoryName("meeting"), event.Category)
	})

	// Test GetEventsCoveringTimerange
	t.Run("GetEventsCoveringTimerange", func(t *testing.T) {
		start := time.Now().UTC().Add(-time.Hour)
		end := time.Now().UTC().Add(2 * time.Hour)

		events, err := provider.GetEventsCoveringTimerange(start, end)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(events), 1)
		t.Logf("Found %d events in timerange", len(events))
	})

	// Test SplitEvent
	var secondEventID model.EventID
	t.Run("SplitEvent", func(t *testing.T) {
		event, err := provider.GetEvent(eventID)
		require.NoError(t, err)

		splitTime := event.Start.Add(30 * time.Minute)
		err = provider.SplitEvent(eventID, splitTime)
		require.NoError(t, err)

		// Verify original event was shortened
		event, err = provider.GetEvent(eventID)
		require.NoError(t, err)
		require.NotNil(t, event.End)
		assert.Equal(t, splitTime, *event.End)

		// Find the new event
		events, err := provider.GetEventsCoveringTimerange(
			splitTime,
			splitTime.Add(time.Hour),
		)
		require.NoError(t, err)
		for _, e := range events {
			if e.ID != eventID {
				secondEventID = e.ID
				break
			}
		}
		assert.NotEmpty(t, secondEventID, "Should have created second event")
		t.Logf("Split created second event: %s", secondEventID)
	})

	// Test OffsetEventTimes
	t.Run("OffsetEventTimes", func(t *testing.T) {
		event, err := provider.GetEvent(eventID)
		require.NoError(t, err)
		originalStart := event.Start

		newStart, newEnd, err := provider.OffsetEventTimes(eventID, 15*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, originalStart.Add(15*time.Minute), newStart)
		t.Logf("Offset event to: %s - %s", newStart, newEnd)
	})

	// Test RemoveEvent
	t.Run("RemoveEvent", func(t *testing.T) {
		err := provider.RemoveEvent(secondEventID)
		require.NoError(t, err)

		_, err = provider.GetEvent(secondEventID)
		assert.Error(t, err, "Event should not be found after deletion")
	})

	// Test TriggerSync
	t.Run("TriggerSync", func(t *testing.T) {
		errCh := provider.TriggerSync()
		err := <-errCh
		require.NoError(t, err)

		status := provider.SyncStatus()
		t.Logf("Status after sync: Online=%v, Pending=%d, Conflicts=%d",
			status.Online, status.PendingChanges, status.ConflictCount)
	})

	// Test FullyCommitted
	t.Run("FullyCommitted", func(t *testing.T) {
		// Wait for sync to complete
		time.Sleep(2 * time.Second)

		committed, err := provider.FullyCommitted()
		require.NoError(t, err)
		t.Logf("FullyCommitted: %v", committed)
	})

	// Test RemoveEvent on the first event
	t.Run("RemoveFirstEvent", func(t *testing.T) {
		err := provider.RemoveEvent(eventID)
		require.NoError(t, err)

		_, err = provider.GetEvent(eventID)
		assert.Error(t, err, "Event should not be found after deletion")
	})

	// Test Logout
	t.Run("Logout", func(t *testing.T) {
		err := provider.Logout()
		require.NoError(t, err)
	})
}

// Unit tests for navigation that don't require a server
// These test the local SQLite navigation logic directly

// createLocalProvider creates a CachingServerClientDataProvider with a temp database for testing.
// It doesn't require a server connection - just tests local operations.
func createLocalProvider(t *testing.T) *CachingServerClientDataProvider {
	t.Helper()
	dbFile, err := os.CreateTemp("", "dayplan-nav-test-*.db")
	require.NoError(t, err)
	dbPath := dbFile.Name()
	dbFile.Close()
	t.Cleanup(func() { os.Remove(dbPath) })

	cfg := CachingServerClientConfig{
		DBPath:    dbPath,
		ServerURL: "http://localhost:9999", // Fake URL - we won't connect
	}

	provider, err := NewCachingServerClientDataProvider(cfg, nil)
	require.NoError(t, err)
	t.Cleanup(func() { provider.Close() })

	return provider
}

// Helper to create times on a test day
func testTimeOnDay(hour, minute int) time.Time {
	return time.Date(2023, 6, 15, hour, minute, 0, 0, time.UTC)
}

func TestCachingServerClientDataProvider_NavigationSymmetry(t *testing.T) {
	p := createLocalProvider(t)

	// Create events
	idA, err := p.AddEvent(model.Event{
		Name:     "A",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(11, 0)),
	})
	require.NoError(t, err)

	idB, err := p.AddEvent(model.Event{
		Name:     "B",
		Category: "cat",
		Start:    testTimeOnDay(12, 0),
		End:      timeptr(testTimeOnDay(13, 0)),
	})
	require.NoError(t, err)

	idC, err := p.AddEvent(model.Event{
		Name:     "C",
		Category: "cat",
		Start:    testTimeOnDay(14, 0),
		End:      timeptr(testTimeOnDay(15, 0)),
	})
	require.NoError(t, err)

	// Test: next(A) should be B
	nextA, err := p.GetFollowingEvent(idA, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextA)
	assert.Equal(t, idB, nextA.ID, "next(A) should be B")

	// Test: prev(B) should be A (symmetry)
	prevB, err := p.GetPrecedingEvent(idB, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevB)
	assert.Equal(t, idA, prevB.ID, "prev(B) should be A")

	// Test: next(B) should be C
	nextB, err := p.GetFollowingEvent(idB, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextB)
	assert.Equal(t, idC, nextB.ID, "next(B) should be C")

	// Test: prev(C) should be B (symmetry)
	prevC, err := p.GetPrecedingEvent(idC, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevC)
	assert.Equal(t, idB, prevC.ID, "prev(C) should be B")
}

func TestCachingServerClientDataProvider_FullyNestedEvents(t *testing.T) {
	p := createLocalProvider(t)

	// Outer: 10:00-14:00 (contains Inner)
	// Inner: 11:00-13:00 (fully inside Outer)
	// After: 15:00-16:00
	idOuter, err := p.AddEvent(model.Event{
		Name:     "Outer",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(14, 0)),
	})
	require.NoError(t, err)

	idInner, err := p.AddEvent(model.Event{
		Name:     "Inner",
		Category: "cat",
		Start:    testTimeOnDay(11, 0),
		End:      timeptr(testTimeOnDay(13, 0)),
	})
	require.NoError(t, err)

	idAfter, err := p.AddEvent(model.Event{
		Name:     "After",
		Category: "cat",
		Start:    testTimeOnDay(15, 0),
		End:      timeptr(testTimeOnDay(16, 0)),
	})
	require.NoError(t, err)

	// Total order: Outer (10:00, 14:00), Inner (11:00, 13:00), After (15:00, 16:00)

	// Test forward: Outer -> Inner -> After
	nextOuter, err := p.GetFollowingEvent(idOuter, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextOuter, "Expected Inner as next after Outer")
	assert.Equal(t, idInner, nextOuter.ID, "next(Outer) should be Inner")

	nextInner, err := p.GetFollowingEvent(idInner, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextInner, "Expected After as next after Inner")
	assert.Equal(t, idAfter, nextInner.ID, "next(Inner) should be After")

	// Test backward: After -> Inner -> Outer
	prevAfter, err := p.GetPrecedingEvent(idAfter, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevAfter)
	assert.Equal(t, idInner, prevAfter.ID, "prev(After) should be Inner")

	prevInner, err := p.GetPrecedingEvent(idInner, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevInner)
	assert.Equal(t, idOuter, prevInner.ID, "prev(Inner) should be Outer")
}

func TestCachingServerClientDataProvider_MultiLevelNesting(t *testing.T) {
	p := createLocalProvider(t)

	// L1: 10:00-18:00 (outermost)
	// L2: 11:00-17:00 (inside L1)
	// L3: 12:00-16:00 (inside L2)
	// After: 19:00-20:00
	idL1, err := p.AddEvent(model.Event{
		Name:     "L1",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(18, 0)),
	})
	require.NoError(t, err)

	idL2, err := p.AddEvent(model.Event{
		Name:     "L2",
		Category: "cat",
		Start:    testTimeOnDay(11, 0),
		End:      timeptr(testTimeOnDay(17, 0)),
	})
	require.NoError(t, err)

	idL3, err := p.AddEvent(model.Event{
		Name:     "L3",
		Category: "cat",
		Start:    testTimeOnDay(12, 0),
		End:      timeptr(testTimeOnDay(16, 0)),
	})
	require.NoError(t, err)

	idAfter, err := p.AddEvent(model.Event{
		Name:     "After",
		Category: "cat",
		Start:    testTimeOnDay(19, 0),
		End:      timeptr(testTimeOnDay(20, 0)),
	})
	require.NoError(t, err)

	// Total order: L1, L2, L3, After
	allIDs := []model.EventID{idL1, idL2, idL3, idAfter}
	names := []string{"L1", "L2", "L3", "After"}

	// Forward navigation: L1 -> L2 -> L3 -> After
	for i := 0; i < len(allIDs)-1; i++ {
		next, err := p.GetFollowingEvent(allIDs[i], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, next, "Expected event after %s", names[i])
		assert.Equal(t, allIDs[i+1], next.ID, "next(%s) should be %s", names[i], names[i+1])
	}

	// Backward navigation: After -> L3 -> L2 -> L1
	for i := len(allIDs) - 1; i > 0; i-- {
		prev, err := p.GetPrecedingEvent(allIDs[i], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, prev, "Expected event before %s", names[i])
		assert.Equal(t, allIDs[i-1], prev.ID, "prev(%s) should be %s", names[i], names[i-1])
	}
}

func TestCachingServerClientDataProvider_SameStartDifferentEnd(t *testing.T) {
	p := createLocalProvider(t)

	// All start at 10:00, but end at different times
	// Order should be: Long (ends 18:00), Medium (ends 14:00), Short (ends 12:00)
	idShort, err := p.AddEvent(model.Event{
		Name:     "Short",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(12, 0)),
	})
	require.NoError(t, err)

	idMedium, err := p.AddEvent(model.Event{
		Name:     "Medium",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(14, 0)),
	})
	require.NoError(t, err)

	idLong, err := p.AddEvent(model.Event{
		Name:     "Long",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(18, 0)),
	})
	require.NoError(t, err)

	idAfter, err := p.AddEvent(model.Event{
		Name:     "After",
		Category: "cat",
		Start:    testTimeOnDay(19, 0),
		End:      timeptr(testTimeOnDay(20, 0)),
	})
	require.NoError(t, err)

	// Total order: Long (end DESC for same start), Medium, Short, After

	// Verify forward navigation from Long
	nextLong, err := p.GetFollowingEvent(idLong, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextLong)
	assert.Equal(t, idMedium, nextLong.ID, "next(Long) should be Medium")

	nextMedium, err := p.GetFollowingEvent(idMedium, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextMedium)
	assert.Equal(t, idShort, nextMedium.ID, "next(Medium) should be Short")

	nextShort, err := p.GetFollowingEvent(idShort, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextShort)
	assert.Equal(t, idAfter, nextShort.ID, "next(Short) should be After")

	// Verify backward navigation: After -> Short -> Medium -> Long
	prevAfter, err := p.GetPrecedingEvent(idAfter, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevAfter)
	assert.Equal(t, idShort, prevAfter.ID, "prev(After) should be Short")

	prevShort, err := p.GetPrecedingEvent(idShort, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevShort)
	assert.Equal(t, idMedium, prevShort.ID, "prev(Short) should be Medium")

	prevMedium, err := p.GetPrecedingEvent(idMedium, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevMedium)
	assert.Equal(t, idLong, prevMedium.ID, "prev(Medium) should be Long")
}

func TestCachingServerClientDataProvider_RoundTripNavigation(t *testing.T) {
	p := createLocalProvider(t)

	// Create a chain of events
	idA, err := p.AddEvent(model.Event{
		Name:     "A",
		Category: "cat",
		Start:    testTimeOnDay(8, 0),
		End:      timeptr(testTimeOnDay(9, 0)),
	})
	require.NoError(t, err)

	_, err = p.AddEvent(model.Event{
		Name:     "B",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(11, 0)),
	})
	require.NoError(t, err)

	_, err = p.AddEvent(model.Event{
		Name:     "C",
		Category: "cat",
		Start:    testTimeOnDay(12, 0),
		End:      timeptr(testTimeOnDay(13, 0)),
	})
	require.NoError(t, err)

	_, err = p.AddEvent(model.Event{
		Name:     "D",
		Category: "cat",
		Start:    testTimeOnDay(14, 0),
		End:      timeptr(testTimeOnDay(15, 0)),
	})
	require.NoError(t, err)

	_, err = p.AddEvent(model.Event{
		Name:     "E",
		Category: "cat",
		Start:    testTimeOnDay(16, 0),
		End:      timeptr(testTimeOnDay(17, 0)),
	})
	require.NoError(t, err)

	// Navigate forward 3 steps
	current := idA
	for i := 0; i < 3; i++ {
		next, err := p.GetFollowingEvent(current, doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, next, "Forward navigation failed at step %d", i)
		current = next.ID
	}

	// Now navigate backward 3 steps
	for i := 0; i < 3; i++ {
		prev, err := p.GetPrecedingEvent(current, doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, prev, "Backward navigation failed at step %d", i)
		current = prev.ID
	}

	// Should be back at A
	assert.Equal(t, idA, current, "Round-trip failed: expected to return to A")
}

func TestCachingServerClientDataProvider_ComplexOverlapping(t *testing.T) {
	p := createLocalProvider(t)

	// A: 10:00-14:00
	// B: 10:30-11:30 (inside A, after A starts)
	// C: 12:00-13:00 (inside A, after B)
	// D: 13:30-15:00 (overlaps A's end)
	// E: 16:00-17:00 (after all)

	idA, err := p.AddEvent(model.Event{
		Name:     "A",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(14, 0)),
	})
	require.NoError(t, err)

	idB, err := p.AddEvent(model.Event{
		Name:     "B",
		Category: "cat",
		Start:    testTimeOnDay(10, 30),
		End:      timeptr(testTimeOnDay(11, 30)),
	})
	require.NoError(t, err)

	idC, err := p.AddEvent(model.Event{
		Name:     "C",
		Category: "cat",
		Start:    testTimeOnDay(12, 0),
		End:      timeptr(testTimeOnDay(13, 0)),
	})
	require.NoError(t, err)

	idD, err := p.AddEvent(model.Event{
		Name:     "D",
		Category: "cat",
		Start:    testTimeOnDay(13, 30),
		End:      timeptr(testTimeOnDay(15, 0)),
	})
	require.NoError(t, err)

	idE, err := p.AddEvent(model.Event{
		Name:     "E",
		Category: "cat",
		Start:    testTimeOnDay(16, 0),
		End:      timeptr(testTimeOnDay(17, 0)),
	})
	require.NoError(t, err)

	// Total order: A, B, C, D, E
	allIDs := []model.EventID{idA, idB, idC, idD, idE}
	names := []string{"A", "B", "C", "D", "E"}

	// Forward navigation should visit all in order
	current := idA
	visited := []model.EventID{idA}
	for i := 0; i < 10; i++ {
		next, err := p.GetFollowingEvent(current, doesNotMatterFallbackTime)
		require.NoError(t, err)
		if next == nil {
			break
		}
		visited = append(visited, next.ID)
		current = next.ID
	}

	assert.Equal(t, 5, len(visited), "Expected to visit 5 events")

	// Verify navigation symmetry for all adjacent pairs
	for i := 0; i < len(allIDs)-1; i++ {
		next, err := p.GetFollowingEvent(allIDs[i], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, allIDs[i+1], next.ID, "next(%s) should be %s", names[i], names[i+1])

		prev, err := p.GetPrecedingEvent(allIDs[i+1], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, prev)
		assert.Equal(t, allIDs[i], prev.ID, "prev(%s) should be %s (symmetry)", names[i+1], names[i])
	}
}

func TestCachingServerClientDataProvider_SameEndDifferentStart(t *testing.T) {
	p := createLocalProvider(t)

	// All end at 14:00, but start at different times
	// Outer: 10:00-14:00
	// Middle: 11:00-14:00
	// Inner: 12:00-14:00
	idOuter, err := p.AddEvent(model.Event{
		Name:     "Outer",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(14, 0)),
	})
	require.NoError(t, err)

	idMiddle, err := p.AddEvent(model.Event{
		Name:     "Middle",
		Category: "cat",
		Start:    testTimeOnDay(11, 0),
		End:      timeptr(testTimeOnDay(14, 0)),
	})
	require.NoError(t, err)

	idInner, err := p.AddEvent(model.Event{
		Name:     "Inner",
		Category: "cat",
		Start:    testTimeOnDay(12, 0),
		End:      timeptr(testTimeOnDay(14, 0)),
	})
	require.NoError(t, err)

	idAfter, err := p.AddEvent(model.Event{
		Name:     "After",
		Category: "cat",
		Start:    testTimeOnDay(15, 0),
		End:      timeptr(testTimeOnDay(16, 0)),
	})
	require.NoError(t, err)

	// Total order: Outer (10:00), Middle (11:00), Inner (12:00), After (15:00)
	allIDs := []model.EventID{idOuter, idMiddle, idInner, idAfter}
	names := []string{"Outer", "Middle", "Inner", "After"}

	// Verify forward navigation
	for i := 0; i < len(allIDs)-1; i++ {
		next, err := p.GetFollowingEvent(allIDs[i], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, allIDs[i+1], next.ID, "next(%s) should be %s", names[i], names[i+1])
	}

	// Verify backward navigation
	for i := len(allIDs) - 1; i > 0; i-- {
		prev, err := p.GetPrecedingEvent(allIDs[i], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, prev)
		assert.Equal(t, allIDs[i-1], prev.ID, "prev(%s) should be %s", names[i], names[i-1])
	}
}

func TestCachingServerClientDataProvider_FirstLastEventBoundaries(t *testing.T) {
	p := createLocalProvider(t)

	// Create events
	idFirst, err := p.AddEvent(model.Event{
		Name:     "First",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(11, 0)),
	})
	require.NoError(t, err)

	_, err = p.AddEvent(model.Event{
		Name:     "Middle",
		Category: "cat",
		Start:    testTimeOnDay(12, 0),
		End:      timeptr(testTimeOnDay(13, 0)),
	})
	require.NoError(t, err)

	idLast, err := p.AddEvent(model.Event{
		Name:     "Last",
		Category: "cat",
		Start:    testTimeOnDay(14, 0),
		End:      timeptr(testTimeOnDay(15, 0)),
	})
	require.NoError(t, err)

	// First event should have no predecessor
	prevFirst, err := p.GetPrecedingEvent(idFirst, doesNotMatterFallbackTime)
	require.NoError(t, err)
	assert.Nil(t, prevFirst, "First event should have no predecessor")

	// Last event should have no successor
	nextLast, err := p.GetFollowingEvent(idLast, doesNotMatterFallbackTime)
	require.NoError(t, err)
	assert.Nil(t, nextLast, "Last event should have no successor")
}

func TestCachingServerClientDataProvider_IdenticalStartAndEnd_IDTiebreaker(t *testing.T) {
	p := createLocalProvider(t)

	// Create events with identical start and end times
	// The ID ordering should determine the order
	// Using explicit IDs to control the order
	idA, err := p.AddEvent(model.Event{
		ID:       "aaa-event",
		Name:     "A",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(12, 0)),
	})
	require.NoError(t, err)

	idB, err := p.AddEvent(model.Event{
		ID:       "bbb-event",
		Name:     "B",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(12, 0)),
	})
	require.NoError(t, err)

	idC, err := p.AddEvent(model.Event{
		ID:       "ccc-event",
		Name:     "C",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(12, 0)),
	})
	require.NoError(t, err)

	// Total order should be: A (aaa), B (bbb), C (ccc) due to ID ASC tiebreaker

	// Forward: A -> B -> C
	nextA, err := p.GetFollowingEvent(idA, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextA)
	assert.Equal(t, idB, nextA.ID, "next(A) should be B (ID tiebreaker)")

	nextB, err := p.GetFollowingEvent(idB, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextB)
	assert.Equal(t, idC, nextB.ID, "next(B) should be C (ID tiebreaker)")

	// Backward: C -> B -> A
	prevC, err := p.GetPrecedingEvent(idC, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevC)
	assert.Equal(t, idB, prevC.ID, "prev(C) should be B (ID tiebreaker)")

	prevB, err := p.GetPrecedingEvent(idB, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevB)
	assert.Equal(t, idA, prevB.ID, "prev(B) should be A (ID tiebreaker)")
}

func TestCachingServerClientDataProvider_SameStartAndEnd_ReversedCreationOrder(t *testing.T) {
	p := createLocalProvider(t)

	// Create events in reverse alphabetical order to ensure ID tiebreaker works correctly
	idC, err := p.AddEvent(model.Event{
		ID:       "ccc-event",
		Name:     "C",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(12, 0)),
	})
	require.NoError(t, err)

	idB, err := p.AddEvent(model.Event{
		ID:       "bbb-event",
		Name:     "B",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(12, 0)),
	})
	require.NoError(t, err)

	idA, err := p.AddEvent(model.Event{
		ID:       "aaa-event",
		Name:     "A",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(12, 0)),
	})
	require.NoError(t, err)

	// Total order should still be: A (aaa), B (bbb), C (ccc) regardless of creation order

	// Forward: A -> B -> C
	nextA, err := p.GetFollowingEvent(idA, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextA)
	assert.Equal(t, idB, nextA.ID, "next(A) should be B")

	nextB, err := p.GetFollowingEvent(idB, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextB)
	assert.Equal(t, idC, nextB.ID, "next(B) should be C")

	// Backward: C -> B -> A
	prevC, err := p.GetPrecedingEvent(idC, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevC)
	assert.Equal(t, idB, prevC.ID, "prev(C) should be B")

	prevB, err := p.GetPrecedingEvent(idB, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevB)
	assert.Equal(t, idA, prevB.ID, "prev(B) should be A")
}

func TestCachingServerClientDataProvider_DeepNesting4Levels(t *testing.T) {
	p := createLocalProvider(t)

	// Very deeply nested events
	// L1: 08:00-20:00 (outermost)
	// L2: 09:00-19:00
	// L3: 10:00-18:00
	// L4: 11:00-17:00 (innermost)
	// After: 21:00-22:00

	idL1, err := p.AddEvent(model.Event{
		Name:     "L1",
		Category: "cat",
		Start:    testTimeOnDay(8, 0),
		End:      timeptr(testTimeOnDay(20, 0)),
	})
	require.NoError(t, err)

	idL2, err := p.AddEvent(model.Event{
		Name:     "L2",
		Category: "cat",
		Start:    testTimeOnDay(9, 0),
		End:      timeptr(testTimeOnDay(19, 0)),
	})
	require.NoError(t, err)

	idL3, err := p.AddEvent(model.Event{
		Name:     "L3",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(18, 0)),
	})
	require.NoError(t, err)

	idL4, err := p.AddEvent(model.Event{
		Name:     "L4",
		Category: "cat",
		Start:    testTimeOnDay(11, 0),
		End:      timeptr(testTimeOnDay(17, 0)),
	})
	require.NoError(t, err)

	idAfter, err := p.AddEvent(model.Event{
		Name:     "After",
		Category: "cat",
		Start:    testTimeOnDay(21, 0),
		End:      timeptr(testTimeOnDay(22, 0)),
	})
	require.NoError(t, err)

	// Total order: L1, L2, L3, L4, After
	allIDs := []model.EventID{idL1, idL2, idL3, idL4, idAfter}
	names := []string{"L1", "L2", "L3", "L4", "After"}

	// Forward navigation
	for i := 0; i < len(allIDs)-1; i++ {
		next, err := p.GetFollowingEvent(allIDs[i], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, next, "Expected event after %s", names[i])
		assert.Equal(t, allIDs[i+1], next.ID, "next(%s) should be %s", names[i], names[i+1])
	}

	// Backward navigation
	for i := len(allIDs) - 1; i > 0; i-- {
		prev, err := p.GetPrecedingEvent(allIDs[i], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, prev, "Expected event before %s", names[i])
		assert.Equal(t, allIDs[i-1], prev.ID, "prev(%s) should be %s", names[i], names[i-1])
	}

	// Verify you can navigate from innermost (L4) all the way back to outermost (L1)
	current := idL4
	for steps := 0; steps < 3; steps++ {
		prev, err := p.GetPrecedingEvent(current, doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, prev, "Should be able to navigate back from L4, step %d", steps)
		current = prev.ID
	}
	assert.Equal(t, idL1, current, "After navigating back 3 steps from L4, should be at L1")
}

func TestCachingServerClientDataProvider_PartiallyOverlappingChain(t *testing.T) {
	p := createLocalProvider(t)

	// Chain of partially overlapping events (like a shifted chain)
	// A: 10:00-12:00
	// B: 11:00-13:00 (overlaps A)
	// C: 12:00-14:00 (overlaps B, might overlap A)
	// D: 13:00-15:00 (overlaps C)
	// E: 14:00-16:00 (overlaps D)

	idA, err := p.AddEvent(model.Event{
		Name:     "A",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(12, 0)),
	})
	require.NoError(t, err)

	idB, err := p.AddEvent(model.Event{
		Name:     "B",
		Category: "cat",
		Start:    testTimeOnDay(11, 0),
		End:      timeptr(testTimeOnDay(13, 0)),
	})
	require.NoError(t, err)

	idC, err := p.AddEvent(model.Event{
		Name:     "C",
		Category: "cat",
		Start:    testTimeOnDay(12, 0),
		End:      timeptr(testTimeOnDay(14, 0)),
	})
	require.NoError(t, err)

	idD, err := p.AddEvent(model.Event{
		Name:     "D",
		Category: "cat",
		Start:    testTimeOnDay(13, 0),
		End:      timeptr(testTimeOnDay(15, 0)),
	})
	require.NoError(t, err)

	idE, err := p.AddEvent(model.Event{
		Name:     "E",
		Category: "cat",
		Start:    testTimeOnDay(14, 0),
		End:      timeptr(testTimeOnDay(16, 0)),
	})
	require.NoError(t, err)

	// Total order: A (10:00), B (11:00), C (12:00), D (13:00), E (14:00)
	allIDs := []model.EventID{idA, idB, idC, idD, idE}
	names := []string{"A", "B", "C", "D", "E"}

	// Forward navigation should visit all
	current := idA
	visited := []model.EventID{idA}
	for i := 0; i < 10; i++ {
		next, err := p.GetFollowingEvent(current, doesNotMatterFallbackTime)
		require.NoError(t, err)
		if next == nil {
			break
		}
		visited = append(visited, next.ID)
		current = next.ID
	}

	assert.Equal(t, 5, len(visited), "Should visit all 5 events")

	// Verify exact order and symmetry
	for i := 0; i < len(allIDs)-1; i++ {
		next, err := p.GetFollowingEvent(allIDs[i], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, allIDs[i+1], next.ID, "next(%s) should be %s", names[i], names[i+1])

		prev, err := p.GetPrecedingEvent(allIDs[i+1], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, prev)
		assert.Equal(t, allIDs[i], prev.ID, "prev(%s) should be %s (symmetry)", names[i+1], names[i])
	}
}

func TestCachingServerClientDataProvider_CrossDayEvents(t *testing.T) {
	p := createLocalProvider(t)

	// Events spanning multiple days
	day1 := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2023, 6, 16, 0, 0, 0, 0, time.UTC)

	// A: Day 1, 10:00-14:00
	idA, err := p.AddEvent(model.Event{
		Name:     "A",
		Category: "cat",
		Start:    day1.Add(10 * time.Hour),
		End:      timeptr(day1.Add(14 * time.Hour)),
	})
	require.NoError(t, err)

	// B: Day 1, 20:00 to Day 2, 02:00 (crosses midnight)
	idB, err := p.AddEvent(model.Event{
		Name:     "B",
		Category: "cat",
		Start:    day1.Add(20 * time.Hour),
		End:      timeptr(day2.Add(2 * time.Hour)),
	})
	require.NoError(t, err)

	// C: Day 2, 10:00-14:00
	idC, err := p.AddEvent(model.Event{
		Name:     "C",
		Category: "cat",
		Start:    day2.Add(10 * time.Hour),
		End:      timeptr(day2.Add(14 * time.Hour)),
	})
	require.NoError(t, err)

	// Total order: A (day1 10:00), B (day1 20:00), C (day2 10:00)

	// Forward: A -> B -> C
	nextA, err := p.GetFollowingEvent(idA, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextA)
	assert.Equal(t, idB, nextA.ID, "next(A) should be B")

	nextB, err := p.GetFollowingEvent(idB, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextB)
	assert.Equal(t, idC, nextB.ID, "next(B) should be C")

	// Backward: C -> B -> A
	prevC, err := p.GetPrecedingEvent(idC, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevC)
	assert.Equal(t, idB, prevC.ID, "prev(C) should be B")

	prevB, err := p.GetPrecedingEvent(idB, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevB)
	assert.Equal(t, idA, prevB.ID, "prev(B) should be A")
}

func TestCachingServerClientDataProvider_MidnightBoundaryEvents(t *testing.T) {
	p := createLocalProvider(t)

	day := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)

	// A: 23:00-23:59 (near end of day)
	idA, err := p.AddEvent(model.Event{
		Name:     "A",
		Category: "cat",
		Start:    day.Add(23 * time.Hour),
		End:      timeptr(day.Add(23*time.Hour + 59*time.Minute)),
	})
	require.NoError(t, err)

	// B: 00:00-01:00 (start of next day, which is actually day 16)
	nextDay := day.AddDate(0, 0, 1)
	idB, err := p.AddEvent(model.Event{
		Name:     "B",
		Category: "cat",
		Start:    nextDay,
		End:      timeptr(nextDay.Add(time.Hour)),
	})
	require.NoError(t, err)

	// Forward: A -> B
	nextA, err := p.GetFollowingEvent(idA, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextA)
	assert.Equal(t, idB, nextA.ID, "next(A) should be B")

	// Backward: B -> A
	prevB, err := p.GetPrecedingEvent(idB, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevB)
	assert.Equal(t, idA, prevB.ID, "prev(B) should be A")
}

func TestCachingServerClientDataProvider_ShortDurationEvents(t *testing.T) {
	p := createLocalProvider(t)

	// Events with very short durations (1 minute)
	idA, err := p.AddEvent(model.Event{
		Name:     "A",
		Category: "cat",
		Start:    testTimeOnDay(10, 0),
		End:      timeptr(testTimeOnDay(10, 1)), // 1 minute
	})
	require.NoError(t, err)

	idB, err := p.AddEvent(model.Event{
		Name:     "B",
		Category: "cat",
		Start:    testTimeOnDay(10, 1),
		End:      timeptr(testTimeOnDay(10, 2)), // 1 minute, immediately after A
	})
	require.NoError(t, err)

	idC, err := p.AddEvent(model.Event{
		Name:     "C",
		Category: "cat",
		Start:    testTimeOnDay(10, 2),
		End:      timeptr(testTimeOnDay(10, 3)), // 1 minute, immediately after B
	})
	require.NoError(t, err)

	// Forward: A -> B -> C
	nextA, err := p.GetFollowingEvent(idA, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextA)
	assert.Equal(t, idB, nextA.ID, "next(A) should be B")

	nextB, err := p.GetFollowingEvent(idB, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, nextB)
	assert.Equal(t, idC, nextB.ID, "next(B) should be C")

	// Backward: C -> B -> A
	prevC, err := p.GetPrecedingEvent(idC, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevC)
	assert.Equal(t, idB, prevC.ID, "prev(C) should be B")

	prevB, err := p.GetPrecedingEvent(idB, doesNotMatterFallbackTime)
	require.NoError(t, err)
	require.NotNil(t, prevB)
	assert.Equal(t, idA, prevB.ID, "prev(B) should be A")
}

func TestCachingServerClientDataProvider_MixedNestedAndSequential(t *testing.T) {
	p := createLocalProvider(t)

	// Mix of nested and sequential events (more realistic scenario)
	// A: 08:00-12:00 (morning block)
	// B: 09:00-10:00 (nested in A)
	// C: 10:30-11:30 (nested in A)
	// D: 14:00-18:00 (afternoon block)
	// E: 15:00-16:00 (nested in D)
	// F: 20:00-21:00 (evening)

	idA, err := p.AddEvent(model.Event{
		Name:     "A",
		Category: "cat",
		Start:    testTimeOnDay(8, 0),
		End:      timeptr(testTimeOnDay(12, 0)),
	})
	require.NoError(t, err)

	idB, err := p.AddEvent(model.Event{
		Name:     "B",
		Category: "cat",
		Start:    testTimeOnDay(9, 0),
		End:      timeptr(testTimeOnDay(10, 0)),
	})
	require.NoError(t, err)

	idC, err := p.AddEvent(model.Event{
		Name:     "C",
		Category: "cat",
		Start:    testTimeOnDay(10, 30),
		End:      timeptr(testTimeOnDay(11, 30)),
	})
	require.NoError(t, err)

	idD, err := p.AddEvent(model.Event{
		Name:     "D",
		Category: "cat",
		Start:    testTimeOnDay(14, 0),
		End:      timeptr(testTimeOnDay(18, 0)),
	})
	require.NoError(t, err)

	idE, err := p.AddEvent(model.Event{
		Name:     "E",
		Category: "cat",
		Start:    testTimeOnDay(15, 0),
		End:      timeptr(testTimeOnDay(16, 0)),
	})
	require.NoError(t, err)

	idF, err := p.AddEvent(model.Event{
		Name:     "F",
		Category: "cat",
		Start:    testTimeOnDay(20, 0),
		End:      timeptr(testTimeOnDay(21, 0)),
	})
	require.NoError(t, err)

	// Total order: A, B, C, D, E, F
	allIDs := []model.EventID{idA, idB, idC, idD, idE, idF}
	names := []string{"A", "B", "C", "D", "E", "F"}

	// Verify forward navigation visits all
	current := idA
	visited := []model.EventID{idA}
	for i := 0; i < 20; i++ {
		next, err := p.GetFollowingEvent(current, doesNotMatterFallbackTime)
		require.NoError(t, err)
		if next == nil {
			break
		}
		visited = append(visited, next.ID)
		current = next.ID
	}

	assert.Equal(t, 6, len(visited), "Should visit all 6 events")

	// Verify forward/backward symmetry
	for i := 0; i < len(allIDs)-1; i++ {
		next, err := p.GetFollowingEvent(allIDs[i], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, allIDs[i+1], next.ID, "next(%s) should be %s", names[i], names[i+1])

		prev, err := p.GetPrecedingEvent(allIDs[i+1], doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, prev)
		assert.Equal(t, allIDs[i], prev.ID, "prev(%s) should be %s", names[i+1], names[i])
	}

	// Verify can navigate from innermost nested event (E) back to A
	current = idE
	for steps := 0; steps < 4; steps++ {
		prev, err := p.GetPrecedingEvent(current, doesNotMatterFallbackTime)
		require.NoError(t, err)
		require.NotNil(t, prev, "Should be able to navigate back from E, step %d", steps)
		current = prev.ID
	}
	assert.Equal(t, idA, current, "After navigating back 4 steps from E, should be at A")
}
