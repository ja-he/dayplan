package backend

import (
	"os"
	"testing"
	"time"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		End:      time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second),
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
			End:      time.Now().UTC().Add(time.Hour).Truncate(time.Second),
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
		assert.Equal(t, splitTime, event.End)

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
