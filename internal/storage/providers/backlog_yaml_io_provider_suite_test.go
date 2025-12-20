package providers_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/storage"
	"github.com/ja-he/dayplan/internal/storage/providers"
)

// TestBacklogYamlIoProvider_Suite runs the comprehensive BacklogProvider test suite
// against the BacklogYamlIoProvider implementation.
func TestBacklogYamlIoProvider_Suite(t *testing.T) {
	factory := func(t *testing.T) storage.BacklogProvider {
		// Create a temporary file for this test
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "backlog.yaml")

		provider, err := providers.NewBacklogYamlIoProvider(filePath)
		if err != nil {
			t.Fatalf("Failed to create BacklogYamlIoProvider: %v", err)
		}

		// Initialize with empty state
		if err := provider.Load(); err != nil {
			// If load fails, that's ok for a new file
			// Some tests expect an empty state
		}

		return provider
	}

	RunBacklogProviderTests(t, factory)
}

// TestBacklogYamlIoProvider_PersistenceAcrossInstances tests that data persists
// when creating multiple provider instances pointing to the same file.
func TestBacklogYamlIoProvider_PersistenceAcrossInstances(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "backlog.yaml")

	// Create first provider and add tasks
	provider1, err := providers.NewBacklogYamlIoProvider(filePath)
	if err != nil {
		t.Fatalf("Failed to create first provider: %v", err)
	}

	if err := provider1.Load(); err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("Load failed: %v", err)
		}
	}

	task := &model.Task{Name: "Persistent Task", Category: "work"}
	id, err := provider1.InsertBack(task, nil)
	if err != nil {
		t.Fatalf("Failed to insert task: %v", err)
	}

	if err := provider1.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Create second provider instance and load
	provider2, err := providers.NewBacklogYamlIoProvider(filePath)
	if err != nil {
		t.Fatalf("Failed to create second provider: %v", err)
	}

	if err := provider2.Load(); err != nil {
		t.Fatalf("Failed to load in second provider: %v", err)
	}

	// Verify the task exists (note: ID will be different after load)
	foundTask := false
	provider2.WithRoots(func(roots []model.ReadableTask) {
		for _, root := range roots {
			if root.GetName() == "Persistent Task" && root.GetCategory() == "work" {
				foundTask = true
				break
			}
		}
	})

	if !foundTask {
		t.Error("Task not found in second provider instance after load")
	}

	_ = id // ID changes after reload, so we can't verify it matches
}
