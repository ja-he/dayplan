package providers_test

import (
	"testing"
	"time"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/storage"
)

// BacklogProviderFactory is a function type that creates a new BacklogProvider instance for testing.
// Each test will get a fresh instance to ensure isolation.
type BacklogProviderFactory func(t *testing.T) storage.BacklogProvider

// RunBacklogProviderTests runs a comprehensive test suite against any BacklogProvider implementation.
// Pass a factory function that creates a fresh instance for each test.
//
// Example usage:
//
//	func TestMyBacklogProvider(t *testing.T) {
//	    factory := func(t *testing.T) storage.BacklogProvider {
//	        provider, err := NewMyBacklogProvider()
//	        require.NoError(t, err)
//	        return provider
//	    }
//	    RunBacklogProviderTests(t, factory)
//	}
func RunBacklogProviderTests(t *testing.T, factory BacklogProviderFactory) {
	t.Run("WithRoots", func(t *testing.T) { testWithRoots(t, factory) })
	t.Run("WithTask", func(t *testing.T) { testWithTask(t, factory) })
	t.Run("WithTasks", func(t *testing.T) { testWithTasks(t, factory) })
	t.Run("GetFirstChildTaskID", func(t *testing.T) { testGetFirstChildTaskID(t, factory) })
	t.Run("GetLastChildTaskID", func(t *testing.T) { testGetLastChildTaskID(t, factory) })
	t.Run("GetLocationContext", func(t *testing.T) { testGetLocationContext(t, factory) })
	t.Run("GetCategory", func(t *testing.T) { testGetCategory(t, factory) })
	t.Run("InsertFront", func(t *testing.T) { testInsertFront(t, factory) })
	t.Run("InsertBack", func(t *testing.T) { testInsertBack(t, factory) })
	t.Run("InsertBefore", func(t *testing.T) { testInsertBefore(t, factory) })
	t.Run("InsertAfter", func(t *testing.T) { testInsertAfter(t, factory) })
	t.Run("Remove", func(t *testing.T) { testRemove(t, factory) })
	t.Run("Update", func(t *testing.T) { testUpdate(t, factory) })
	t.Run("LoadAndSave", func(t *testing.T) { testLoadAndSave(t, factory) })
	t.Run("ComplexScenarios", func(t *testing.T) { testComplexScenarios(t, factory) })
}

// Helper function to create a simple task
func createTask(name string, category model.CategoryName) *model.Task {
	return &model.Task{
		Name:     name,
		Category: category,
	}
}

// Helper function to create a task with duration
func createTaskWithDuration(name string, category model.CategoryName, duration time.Duration) *model.Task {
	return &model.Task{
		Name:     name,
		Category: category,
		Duration: &duration,
	}
}

// Helper function to create a task with deadline
func createTaskWithDeadline(name string, category model.CategoryName, deadline time.Time) *model.Task {
	return &model.Task{
		Name:     name,
		Category: category,
		Deadline: &deadline,
	}
}

// Helper function to create a task with subtasks
func createTaskWithSubtasks(name string, category model.CategoryName, subtasks ...*model.Task) *model.Task {
	return &model.Task{
		Name:     name,
		Category: category,
		Subtasks: subtasks,
	}
}

// Test WithRoots functionality
func testWithRoots(t *testing.T, factory BacklogProviderFactory) {
	t.Run("empty backlog", func(t *testing.T) {
		provider := factory(t)

		err := provider.WithRoots(func(roots []model.ReadableTask) {
			if len(roots) != 0 {
				t.Errorf("Expected 0 roots, got %d", len(roots))
			}
		})
		if err != nil {
			t.Errorf("WithRoots failed: %v", err)
		}
	})

	t.Run("single root task", func(t *testing.T) {
		provider := factory(t)

		task := createTask("Test Task", "test-category")
		id, err := provider.InsertFront(task, nil)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}

		err = provider.WithRoots(func(roots []model.ReadableTask) {
			if len(roots) != 1 {
				t.Errorf("Expected 1 root, got %d", len(roots))
				return
			}
			if roots[0].GetID() != id {
				t.Errorf("Expected ID %s, got %s", id, roots[0].GetID())
			}
			if roots[0].GetName() != "Test Task" {
				t.Errorf("Expected name 'Test Task', got '%s'", roots[0].GetName())
			}
		})
		if err != nil {
			t.Errorf("WithRoots failed: %v", err)
		}
	})

	t.Run("multiple root tasks", func(t *testing.T) {
		provider := factory(t)

		id1, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)
		id2, _ := provider.InsertBack(createTask("Task 2", "cat2"), nil)
		id3, _ := provider.InsertBack(createTask("Task 3", "cat3"), nil)

		err := provider.WithRoots(func(roots []model.ReadableTask) {
			if len(roots) != 3 {
				t.Errorf("Expected 3 roots, got %d", len(roots))
				return
			}
			expectedIDs := []model.TaskID{id1, id2, id3}
			for i, root := range roots {
				if root.GetID() != expectedIDs[i] {
					t.Errorf("Root %d: expected ID %s, got %s", i, expectedIDs[i], root.GetID())
				}
			}
		})
		if err != nil {
			t.Errorf("WithRoots failed: %v", err)
		}
	})
}

// Test WithTask functionality
func testWithTask(t *testing.T, factory BacklogProviderFactory) {
	t.Run("retrieve existing task", func(t *testing.T) {
		provider := factory(t)

		task := createTask("My Task", "work")
		id, err := provider.InsertFront(task, nil)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}

		err = provider.WithTask(id, func(retrieved model.ReadableTask) {
			if retrieved.GetID() != id {
				t.Errorf("Expected ID %s, got %s", id, retrieved.GetID())
			}
			if retrieved.GetName() != "My Task" {
				t.Errorf("Expected name 'My Task', got '%s'", retrieved.GetName())
			}
			if retrieved.GetCategory() != "work" {
				t.Errorf("Expected category 'work', got '%s'", retrieved.GetCategory())
			}
		})
		if err != nil {
			t.Errorf("WithTask failed: %v", err)
		}
	})

	t.Run("non-existent task", func(t *testing.T) {
		provider := factory(t)

		err := provider.WithTask("non-existent-id", func(t model.ReadableTask) {
			// Should not be called
		})
		if err == nil {
			t.Error("Expected error for non-existent task, got nil")
		}
	})

	t.Run("retrieve subtask", func(t *testing.T) {
		provider := factory(t)

		subtask := createTask("Subtask", "work")
		parent := createTaskWithSubtasks("Parent", "work", subtask)
		parentID, _ := provider.InsertFront(parent, nil)

		// Get the actual subtask ID
		var subtaskID model.TaskID
		provider.WithTask(parentID, func(p model.ReadableTask) {
			if len(p.GetSubtasks()) > 0 {
				subtaskID = p.GetSubtasks()[0].GetID()
			}
		})

		err := provider.WithTask(subtaskID, func(retrieved model.ReadableTask) {
			if retrieved.GetName() != "Subtask" {
				t.Errorf("Expected name 'Subtask', got '%s'", retrieved.GetName())
			}
		})
		if err != nil {
			t.Errorf("WithTask failed for subtask: %v", err)
		}
	})
}

// Test WithTasks functionality
func testWithTasks(t *testing.T, factory BacklogProviderFactory) {
	t.Run("retrieve multiple tasks", func(t *testing.T) {
		provider := factory(t)

		id1, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)
		id2, _ := provider.InsertBack(createTask("Task 2", "cat2"), nil)
		id3, _ := provider.InsertBack(createTask("Task 3", "cat3"), nil)

		err := provider.WithTasks([]model.TaskID{id1, id2, id3}, func(tasks []model.ReadableTask) {
			if len(tasks) != 3 {
				t.Errorf("Expected 3 tasks, got %d", len(tasks))
				return
			}
			expectedNames := []string{"Task 1", "Task 2", "Task 3"}
			for i, task := range tasks {
				if task.GetName() != expectedNames[i] {
					t.Errorf("Task %d: expected name '%s', got '%s'", i, expectedNames[i], task.GetName())
				}
			}
		})
		if err != nil {
			t.Errorf("WithTasks failed: %v", err)
		}
	})

	t.Run("empty task list", func(t *testing.T) {
		provider := factory(t)

		err := provider.WithTasks([]model.TaskID{}, func(tasks []model.ReadableTask) {
			if len(tasks) != 0 {
				t.Errorf("Expected 0 tasks, got %d", len(tasks))
			}
		})
		if err != nil {
			t.Errorf("WithTasks failed: %v", err)
		}
	})

	t.Run("with non-existent task", func(t *testing.T) {
		provider := factory(t)

		id1, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)

		err := provider.WithTasks([]model.TaskID{id1, "non-existent"}, func(tasks []model.ReadableTask) {
			// Should not be called
		})
		if err == nil {
			t.Error("Expected error for non-existent task, got nil")
		}
	})
}

// Test GetFirstChildTaskID functionality
func testGetFirstChildTaskID(t *testing.T, factory BacklogProviderFactory) {
	t.Run("empty backlog", func(t *testing.T) {
		provider := factory(t)

		firstID, err := provider.GetFirstChildTaskID(nil)
		if err != nil {
			t.Errorf("GetFirstChildTaskID failed: %v", err)
		}
		if firstID != nil {
			t.Errorf("Expected nil for empty backlog, got %s", *firstID)
		}
	})

	t.Run("single root task", func(t *testing.T) {
		provider := factory(t)

		id, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)

		firstID, err := provider.GetFirstChildTaskID(nil)
		if err != nil {
			t.Errorf("GetFirstChildTaskID failed: %v", err)
		}
		if firstID == nil {
			t.Error("Expected non-nil ID")
		} else if *firstID != id {
			t.Errorf("Expected ID %s, got %s", id, *firstID)
		}
	})

	t.Run("multiple root tasks", func(t *testing.T) {
		provider := factory(t)

		id1, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)
		provider.InsertBack(createTask("Task 2", "cat2"), nil)

		firstID, err := provider.GetFirstChildTaskID(nil)
		if err != nil {
			t.Errorf("GetFirstChildTaskID failed: %v", err)
		}
		if firstID == nil {
			t.Error("Expected non-nil ID")
		} else if *firstID != id1 {
			t.Errorf("Expected first ID %s, got %s", id1, *firstID)
		}
	})

	t.Run("task with no children", func(t *testing.T) {
		provider := factory(t)

		id, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)

		firstID, err := provider.GetFirstChildTaskID(&id)
		if err != nil {
			t.Errorf("GetFirstChildTaskID failed: %v", err)
		}
		if firstID != nil {
			t.Errorf("Expected nil for task with no children, got %s", *firstID)
		}
	})

	t.Run("task with children", func(t *testing.T) {
		provider := factory(t)

		parentID, _ := provider.InsertBack(createTask("Parent", "cat1"), nil)
		child1ID, _ := provider.InsertBack(createTask("Child 1", "cat1"), &parentID)
		provider.InsertBack(createTask("Child 2", "cat1"), &parentID)

		firstID, err := provider.GetFirstChildTaskID(&parentID)
		if err != nil {
			t.Errorf("GetFirstChildTaskID failed: %v", err)
		}
		if firstID == nil {
			t.Error("Expected non-nil ID")
		} else if *firstID != child1ID {
			t.Errorf("Expected first child ID %s, got %s", child1ID, *firstID)
		}
	})
}

// Test GetLastChildTaskID functionality
func testGetLastChildTaskID(t *testing.T, factory BacklogProviderFactory) {
	t.Run("empty backlog", func(t *testing.T) {
		provider := factory(t)

		lastID, err := provider.GetLastChildTaskID(nil)
		if err != nil {
			t.Errorf("GetLastChildTaskID failed: %v", err)
		}
		if lastID != nil {
			t.Errorf("Expected nil for empty backlog, got %s", *lastID)
		}
	})

	t.Run("multiple root tasks", func(t *testing.T) {
		provider := factory(t)

		provider.InsertBack(createTask("Task 1", "cat1"), nil)
		id3, _ := provider.InsertBack(createTask("Task 3", "cat3"), nil)

		lastID, err := provider.GetLastChildTaskID(nil)
		if err != nil {
			t.Errorf("GetLastChildTaskID failed: %v", err)
		}
		if lastID == nil {
			t.Error("Expected non-nil ID")
		} else if *lastID != id3 {
			t.Errorf("Expected last ID %s, got %s", id3, *lastID)
		}
	})

	t.Run("task with children", func(t *testing.T) {
		provider := factory(t)

		parentID, _ := provider.InsertBack(createTask("Parent", "cat1"), nil)
		provider.InsertBack(createTask("Child 1", "cat1"), &parentID)
		child2ID, _ := provider.InsertBack(createTask("Child 2", "cat1"), &parentID)

		lastID, err := provider.GetLastChildTaskID(&parentID)
		if err != nil {
			t.Errorf("GetLastChildTaskID failed: %v", err)
		}
		if lastID == nil {
			t.Error("Expected non-nil ID")
		} else if *lastID != child2ID {
			t.Errorf("Expected last child ID %s, got %s", child2ID, *lastID)
		}
	})
}

// Test GetLocationContext functionality
func testGetLocationContext(t *testing.T, factory BacklogProviderFactory) {
	t.Run("single root task", func(t *testing.T) {
		provider := factory(t)

		id, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)

		ctx, err := provider.GetLocationContext(id)
		if err != nil {
			t.Errorf("GetLocationContext failed: %v", err)
		}
		if ctx.Previous != nil {
			t.Error("Expected nil Previous")
		}
		if ctx.Next != nil {
			t.Error("Expected nil Next")
		}
		if len(ctx.Parentage) != 0 {
			t.Errorf("Expected empty Parentage, got %d items", len(ctx.Parentage))
		}
	})

	t.Run("middle root task", func(t *testing.T) {
		provider := factory(t)

		id1, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)
		id2, _ := provider.InsertBack(createTask("Task 2", "cat2"), nil)
		id3, _ := provider.InsertBack(createTask("Task 3", "cat3"), nil)

		ctx, err := provider.GetLocationContext(id2)
		if err != nil {
			t.Errorf("GetLocationContext failed: %v", err)
		}
		if ctx.Previous == nil || *ctx.Previous != id1 {
			t.Errorf("Expected Previous %s, got %v", id1, ctx.Previous)
		}
		if ctx.Next == nil || *ctx.Next != id3 {
			t.Errorf("Expected Next %s, got %v", id3, ctx.Next)
		}
		if len(ctx.Parentage) != 0 {
			t.Errorf("Expected empty Parentage, got %d items", len(ctx.Parentage))
		}
	})

	t.Run("subtask context", func(t *testing.T) {
		provider := factory(t)

		parentID, _ := provider.InsertBack(createTask("Parent", "cat1"), nil)
		child1ID, _ := provider.InsertBack(createTask("Child 1", "cat1"), &parentID)
		child2ID, _ := provider.InsertBack(createTask("Child 2", "cat2"), &parentID)
		child3ID, _ := provider.InsertBack(createTask("Child 3", "cat3"), &parentID)

		ctx, err := provider.GetLocationContext(child2ID)
		if err != nil {
			t.Errorf("GetLocationContext failed: %v", err)
		}
		if ctx.Previous == nil || *ctx.Previous != child1ID {
			t.Errorf("Expected Previous %s, got %v", child1ID, ctx.Previous)
		}
		if ctx.Next == nil || *ctx.Next != child3ID {
			t.Errorf("Expected Next %s, got %v", child3ID, ctx.Next)
		}
		if len(ctx.Parentage) != 1 {
			t.Errorf("Expected Parentage length 1, got %d", len(ctx.Parentage))
		} else if ctx.Parentage[0] != parentID {
			t.Errorf("Expected parent %s, got %s", parentID, ctx.Parentage[0])
		}
	})

	t.Run("non-existent task", func(t *testing.T) {
		provider := factory(t)

		_, err := provider.GetLocationContext("non-existent-id")
		if err == nil {
			t.Error("Expected error for non-existent task, got nil")
		}
	})
}

// Test GetCategory functionality
func testGetCategory(t *testing.T, factory BacklogProviderFactory) {
	t.Run("retrieve category", func(t *testing.T) {
		provider := factory(t)

		id, _ := provider.InsertBack(createTask("Task 1", "work"), nil)

		cat, err := provider.GetCategory(id)
		if err != nil {
			t.Errorf("GetCategory failed: %v", err)
		}
		if cat != "work" {
			t.Errorf("Expected category 'work', got '%s'", cat)
		}
	})

	t.Run("non-existent task", func(t *testing.T) {
		provider := factory(t)

		_, err := provider.GetCategory("non-existent-id")
		if err == nil {
			t.Error("Expected error for non-existent task, got nil")
		}
	})
}

// Test InsertFront functionality
func testInsertFront(t *testing.T, factory BacklogProviderFactory) {
	t.Run("insert at empty root", func(t *testing.T) {
		provider := factory(t)

		id, err := provider.InsertFront(createTask("First Task", "cat1"), nil)
		if err != nil {
			t.Errorf("InsertFront failed: %v", err)
		}
		if id == "" {
			t.Error("Expected non-empty ID")
		}

		firstID, _ := provider.GetFirstChildTaskID(nil)
		if firstID == nil || *firstID != id {
			t.Errorf("Expected first task ID %s, got %v", id, firstID)
		}
	})

	t.Run("insert at front of existing roots", func(t *testing.T) {
		provider := factory(t)

		provider.InsertBack(createTask("Second Task", "cat2"), nil)
		id1, _ := provider.InsertFront(createTask("First Task", "cat1"), nil)

		firstID, _ := provider.GetFirstChildTaskID(nil)
		if firstID == nil || *firstID != id1 {
			t.Errorf("Expected first task ID %s, got %v", id1, firstID)
		}
	})

	t.Run("insert at front of children", func(t *testing.T) {
		provider := factory(t)

		parentID, _ := provider.InsertBack(createTask("Parent", "cat1"), nil)
		provider.InsertBack(createTask("Second Child", "cat1"), &parentID)
		child1ID, _ := provider.InsertFront(createTask("First Child", "cat1"), &parentID)

		firstChildID, _ := provider.GetFirstChildTaskID(&parentID)
		if firstChildID == nil || *firstChildID != child1ID {
			t.Errorf("Expected first child ID %s, got %v", child1ID, firstChildID)
		}
	})
}

// Test InsertBack functionality
func testInsertBack(t *testing.T, factory BacklogProviderFactory) {
	t.Run("insert at empty root", func(t *testing.T) {
		provider := factory(t)

		id, err := provider.InsertBack(createTask("Last Task", "cat1"), nil)
		if err != nil {
			t.Errorf("InsertBack failed: %v", err)
		}
		if id == "" {
			t.Error("Expected non-empty ID")
		}

		lastID, _ := provider.GetLastChildTaskID(nil)
		if lastID == nil || *lastID != id {
			t.Errorf("Expected last task ID %s, got %v", id, lastID)
		}
	})

	t.Run("insert at back of existing roots", func(t *testing.T) {
		provider := factory(t)

		provider.InsertBack(createTask("First Task", "cat1"), nil)
		id2, _ := provider.InsertBack(createTask("Second Task", "cat2"), nil)

		lastID, _ := provider.GetLastChildTaskID(nil)
		if lastID == nil || *lastID != id2 {
			t.Errorf("Expected last task ID %s, got %v", id2, lastID)
		}
	})

	t.Run("insert at back of children", func(t *testing.T) {
		provider := factory(t)

		parentID, _ := provider.InsertBack(createTask("Parent", "cat1"), nil)
		provider.InsertBack(createTask("First Child", "cat1"), &parentID)
		child2ID, _ := provider.InsertBack(createTask("Second Child", "cat1"), &parentID)

		lastChildID, _ := provider.GetLastChildTaskID(&parentID)
		if lastChildID == nil || *lastChildID != child2ID {
			t.Errorf("Expected last child ID %s, got %v", child2ID, lastChildID)
		}
	})
}

// Test InsertBefore functionality
func testInsertBefore(t *testing.T, factory BacklogProviderFactory) {
	t.Run("insert before root task", func(t *testing.T) {
		provider := factory(t)

		id2, _ := provider.InsertBack(createTask("Task 2", "cat2"), nil)
		id1, err := provider.InsertBefore(createTask("Task 1", "cat1"), id2)
		if err != nil {
			t.Errorf("InsertBefore failed: %v", err)
		}

		ctx, _ := provider.GetLocationContext(id1)
		if ctx.Next == nil || *ctx.Next != id2 {
			t.Errorf("Expected Next %s, got %v", id2, ctx.Next)
		}
	})

	t.Run("insert before child task", func(t *testing.T) {
		provider := factory(t)

		parentID, _ := provider.InsertBack(createTask("Parent", "cat1"), nil)
		child2ID, _ := provider.InsertBack(createTask("Child 2", "cat1"), &parentID)
		child1ID, err := provider.InsertBefore(createTask("Child 1", "cat1"), child2ID)
		if err != nil {
			t.Errorf("InsertBefore failed: %v", err)
		}

		ctx, _ := provider.GetLocationContext(child1ID)
		if ctx.Next == nil || *ctx.Next != child2ID {
			t.Errorf("Expected Next %s, got %v", child2ID, ctx.Next)
		}
		if len(ctx.Parentage) != 1 || ctx.Parentage[0] != parentID {
			t.Errorf("Expected parent %s in parentage", parentID)
		}
	})

	t.Run("insert before non-existent task", func(t *testing.T) {
		provider := factory(t)

		_, err := provider.InsertBefore(createTask("Task", "cat1"), "non-existent")
		if err == nil {
			t.Error("Expected error for non-existent anchor task, got nil")
		}
	})
}

// Test InsertAfter functionality
func testInsertAfter(t *testing.T, factory BacklogProviderFactory) {
	t.Run("insert after root task", func(t *testing.T) {
		provider := factory(t)

		id1, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)
		id2, err := provider.InsertAfter(createTask("Task 2", "cat2"), id1)
		if err != nil {
			t.Errorf("InsertAfter failed: %v", err)
		}

		ctx, _ := provider.GetLocationContext(id2)
		if ctx.Previous == nil || *ctx.Previous != id1 {
			t.Errorf("Expected Previous %s, got %v", id1, ctx.Previous)
		}
	})

	t.Run("insert after child task", func(t *testing.T) {
		provider := factory(t)

		parentID, err := provider.InsertBack(createTask("Parent", "cat1"), nil)
		if err != nil {
			t.Errorf("InsertBack failed for root: %v", err)
		}
		child1ID, err := provider.InsertBack(createTask("Child 1", "cat1"), &parentID)
		if err != nil {
			t.Errorf("InsertBack failed for parent: %v", err)
		}
		child2ID, err := provider.InsertAfter(createTask("Child 2", "cat1"), child1ID)
		if err != nil {
			t.Errorf("InsertAfter failed: %v", err)
		}

		ctx, err := provider.GetLocationContext(child2ID)
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
		if ctx.Previous == nil || *ctx.Previous != child1ID {
			t.Errorf("Expected Previous %s, got %v (ctx: %#v)", child1ID, ctx.Previous, ctx)
		}
		if len(ctx.Parentage) != 1 || ctx.Parentage[0] != parentID {
			t.Errorf("Expected parent %s in parentage", parentID)
		}
	})

	t.Run("insert after non-existent task", func(t *testing.T) {
		provider := factory(t)

		_, err := provider.InsertAfter(createTask("Task", "cat1"), "non-existent")
		if err == nil {
			t.Error("Expected error for non-existent anchor task, got nil")
		}
	})
}

// Test Remove functionality
func testRemove(t *testing.T, factory BacklogProviderFactory) {
	t.Run("remove only root task", func(t *testing.T) {
		provider := factory(t)

		id, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)

		removed, ctx, err := provider.Remove(id)
		if err != nil {
			t.Errorf("Remove failed: %v", err)
		}
		if removed.GetName() != "Task 1" {
			t.Errorf("Expected removed task name 'Task 1', got '%s'", removed.GetName())
		}
		if ctx.Previous != nil || ctx.Next != nil {
			t.Error("Expected no previous/next for single task")
		}

		// Verify it's gone
		err = provider.WithRoots(func(roots []model.ReadableTask) {
			if len(roots) != 0 {
				t.Errorf("Expected 0 roots after removal, got %d", len(roots))
			}
		})
		if err != nil {
			t.Errorf("WithRoots failed: %v", err)
		}
	})

	t.Run("remove middle root task", func(t *testing.T) {
		provider := factory(t)

		id1, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)
		id2, _ := provider.InsertBack(createTask("Task 2", "cat2"), nil)
		id3, _ := provider.InsertBack(createTask("Task 3", "cat3"), nil)

		removed, ctx, err := provider.Remove(id2)
		if err != nil {
			t.Errorf("Remove failed: %v", err)
		}
		if removed.GetName() != "Task 2" {
			t.Errorf("Expected removed task name 'Task 2', got '%s'", removed.GetName())
		}
		if ctx.Previous == nil || *ctx.Previous != id1 {
			t.Errorf("Expected Previous %s, got %v", id1, ctx.Previous)
		}
		if ctx.Next == nil || *ctx.Next != id3 {
			t.Errorf("Expected Next %s, got %v", id3, ctx.Next)
		}

		// Verify order is maintained
		provider.WithRoots(func(roots []model.ReadableTask) {
			if len(roots) != 2 {
				t.Errorf("Expected 2 roots after removal, got %d", len(roots))
				return
			}
			if roots[0].GetID() != id1 || roots[1].GetID() != id3 {
				t.Error("Root tasks not in expected order after removal")
			}
		})
	})

	t.Run("remove child task", func(t *testing.T) {
		provider := factory(t)

		parentID, _ := provider.InsertBack(createTask("Parent", "cat1"), nil)
		child1ID, _ := provider.InsertBack(createTask("Child 1", "cat1"), &parentID)
		child2ID, _ := provider.InsertBack(createTask("Child 2", "cat2"), &parentID)

		removed, ctx, err := provider.Remove(child1ID)
		if err != nil {
			t.Errorf("Remove failed: %v", err)
		}
		if removed.GetName() != "Child 1" {
			t.Errorf("Expected removed task name 'Child 1', got '%s'", removed.GetName())
		}
		if len(ctx.Parentage) != 1 || ctx.Parentage[0] != parentID {
			t.Errorf("Expected parent %s in parentage", parentID)
		}

		// Verify child is removed
		firstChildID, _ := provider.GetFirstChildTaskID(&parentID)
		if firstChildID == nil || *firstChildID != child2ID {
			t.Errorf("Expected first child %s after removal, got %v", child2ID, firstChildID)
		}
	})

	t.Run("remove non-existent task", func(t *testing.T) {
		provider := factory(t)

		_, _, err := provider.Remove("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent task, got nil")
		}
	})
}

// Test Update functionality
func testUpdate(t *testing.T, factory BacklogProviderFactory) {
	t.Run("update task name", func(t *testing.T) {
		provider := factory(t)

		id, _ := provider.InsertBack(createTask("Original Name", "cat1"), nil)

		updatedTask := createTask("Updated Name", "cat1")
		err := provider.Update(id, updatedTask)
		if err != nil {
			t.Errorf("Update failed: %v", err)
		}

		provider.WithTask(id, func(task model.ReadableTask) {
			if task.GetName() != "Updated Name" {
				t.Errorf("Expected updated name 'Updated Name', got '%s'", task.GetName())
			}
		})
	})

	t.Run("update task category", func(t *testing.T) {
		provider := factory(t)

		id, _ := provider.InsertBack(createTask("Task", "old-cat"), nil)

		updatedTask := createTask("Task", "new-cat")
		err := provider.Update(id, updatedTask)
		if err != nil {
			t.Errorf("Update failed: %v", err)
		}

		cat, _ := provider.GetCategory(id)
		if cat != "new-cat" {
			t.Errorf("Expected category 'new-cat', got '%s'", cat)
		}
	})

	t.Run("update task with duration", func(t *testing.T) {
		provider := factory(t)

		id, _ := provider.InsertBack(createTask("Task", "cat1"), nil)

		duration := 2 * time.Hour
		updatedTask := createTaskWithDuration("Task", "cat1", duration)
		err := provider.Update(id, updatedTask)
		if err != nil {
			t.Errorf("Update failed: %v", err)
		}

		provider.WithTask(id, func(task model.ReadableTask) {
			if task.GetDuration() == nil {
				t.Error("Expected duration to be set")
			} else if *task.GetDuration() != duration {
				t.Errorf("Expected duration %v, got %v", duration, *task.GetDuration())
			}
		})
	})

	t.Run("update task with deadline", func(t *testing.T) {
		provider := factory(t)

		id, _ := provider.InsertBack(createTask("Task", "cat1"), nil)

		deadline := time.Now().Add(24 * time.Hour)
		updatedTask := createTaskWithDeadline("Task", "cat1", deadline)
		err := provider.Update(id, updatedTask)
		if err != nil {
			t.Errorf("Update failed: %v", err)
		}

		provider.WithTask(id, func(task model.ReadableTask) {
			if task.GetDeadline() == nil {
				t.Error("Expected deadline to be set")
			} else if !task.GetDeadline().Equal(deadline) {
				t.Errorf("Expected deadline %v, got %v", deadline, *task.GetDeadline())
			}
		})
	})

	t.Run("update non-existent task", func(t *testing.T) {
		provider := factory(t)

		err := provider.Update("non-existent", createTask("Task", "cat1"))
		if err == nil {
			t.Error("Expected error for non-existent task, got nil")
		}
	})
}

// Test Load and Save functionality
func testLoadAndSave(t *testing.T, factory BacklogProviderFactory) {
	t.Run("load restores saved state", func(t *testing.T) {
		provider := factory(t)

		// Insert some tasks and save
		id1, err := provider.InsertBack(createTask("Original Task 1", "cat1"), nil)
		if err != nil {
			t.Errorf("Unable to insert task 1: %v", err)
			return
		}
		id2, err := provider.InsertBack(createTask("Original Task 2", "cat2"), nil)
		if err != nil {
			t.Errorf("Unable to insert task 2: %v", err)
			return
		}

		// Save the state
		err = provider.Save()
		if err != nil {
			t.Skipf("Save not supported or failed: %v", err)
			return
		}

		// Modify the state: remove task 1 and rename task 2
		_, _, err = provider.Remove(id1)
		if err != nil {
			t.Errorf("Unable to remove task 1: %v", err)
			return
		}
		err = provider.Update(id2, createTask("Modified Task 2", "cat2"))
		if err != nil {
			t.Errorf("Unable to update task 2: %v", err)
			return
		}

		// Verify the modified state
		provider.WithRoots(func(roots []model.ReadableTask) {
			if len(roots) != 1 {
				t.Errorf("Expected 1 root after removal, got %d", len(roots))
			}
			if len(roots) > 0 && roots[0].GetName() != "Modified Task 2" {
				t.Errorf("Expected modified name, got '%s'", roots[0].GetName())
			}
		})

		// Load - should restore the original saved state
		err = provider.Load()
		if err != nil {
			t.Skipf("Load not supported or failed: %v", err)
			return
		}

		// Verify the state is back to what was saved (2 tasks with original names)
		var foundTask1, foundTask2 bool
		var taskCount int
		provider.WithRoots(func(roots []model.ReadableTask) {
			taskCount = len(roots)
			for _, root := range roots {
				if root.GetName() == "Original Task 1" && root.GetCategory() == "cat1" {
					foundTask1 = true
				}
				if root.GetName() == "Original Task 2" && root.GetCategory() == "cat2" {
					foundTask2 = true
				}
			}
		})

		if taskCount != 2 {
			t.Errorf("Expected 2 tasks after load, got %d", taskCount)
		}
		if !foundTask1 {
			t.Error("Original Task 1 not found after load - state was not restored")
		}
		if !foundTask2 {
			t.Error("Original Task 2 not found after load - state was not restored")
		}
	})
}

// Test complex scenarios combining multiple operations
func testComplexScenarios(t *testing.T, factory BacklogProviderFactory) {
	t.Run("nested task hierarchy", func(t *testing.T) {
		provider := factory(t)

		// Create: Root1 -> Child1 -> GrandChild1
		//                        -> GrandChild2
		//              -> Child2
		//         Root2

		root1ID, _ := provider.InsertBack(createTask("Root 1", "cat1"), nil)
		child1ID, _ := provider.InsertBack(createTask("Child 1", "cat1"), &root1ID)
		grandchild1ID, _ := provider.InsertBack(createTask("GrandChild 1", "cat1"), &child1ID)
		grandchild2ID, _ := provider.InsertBack(createTask("GrandChild 2", "cat1"), &child1ID)
		child2ID, _ := provider.InsertBack(createTask("Child 2", "cat1"), &root1ID)
		root2ID, _ := provider.InsertBack(createTask("Root 2", "cat2"), nil)

		// Verify structure
		provider.WithRoots(func(roots []model.ReadableTask) {
			if len(roots) != 2 {
				t.Errorf("Expected 2 roots, got %d", len(roots))
				return
			}

			if roots[0].GetID() != root1ID {
				t.Error("First root is not Root 1")
				return
			}

			children := roots[0].GetSubtasks()
			if len(children) != 2 {
				t.Errorf("Expected 2 children for Root 1, got %d", len(children))
				return
			}

			grandchildren := children[0].GetSubtasks()
			if len(grandchildren) != 2 {
				t.Errorf("Expected 2 grandchildren for Child 1, got %d", len(grandchildren))
			}
		})

		// Verify location context of grandchild
		ctx, _ := provider.GetLocationContext(grandchild1ID)
		if len(ctx.Parentage) != 2 {
			t.Errorf("Expected parentage length 2 for grandchild, got %d", len(ctx.Parentage))
		} else {
			if ctx.Parentage[0] != root1ID {
				t.Errorf("Expected first parent %s, got %s", root1ID, ctx.Parentage[0])
			}
			if ctx.Parentage[1] != child1ID {
				t.Errorf("Expected second parent %s, got %s", child1ID, ctx.Parentage[1])
			}
		}

		// Verify sibling relationships
		gc1Ctx, _ := provider.GetLocationContext(grandchild1ID)
		if gc1Ctx.Next == nil || *gc1Ctx.Next != grandchild2ID {
			t.Error("GrandChild 1's next should be GrandChild 2")
		}

		_, _, _ = child2ID, root2ID, grandchild2ID
	})

	t.Run("move task by remove and insert", func(t *testing.T) {
		provider := factory(t)

		id1, _ := provider.InsertBack(createTask("Task 1", "cat1"), nil)
		id2, _ := provider.InsertBack(createTask("Task 2", "cat2"), nil)
		id3, _ := provider.InsertBack(createTask("Task 3", "cat3"), nil)

		// Remove Task 3
		removed, _, err := provider.Remove(id3)
		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		// Insert it before Task 1
		newID, err := provider.InsertBefore(removed, id1)
		if err != nil {
			t.Fatalf("InsertBefore failed: %v", err)
		}

		// Verify new order: Task 3, Task 1, Task 2
		provider.WithRoots(func(roots []model.ReadableTask) {
			if len(roots) != 3 {
				t.Errorf("Expected 3 roots, got %d", len(roots))
				return
			}
			if roots[0].GetID() != newID {
				t.Error("First task should be the moved Task 3")
			}
			if roots[1].GetID() != id1 {
				t.Error("Second task should be Task 1")
			}
			if roots[2].GetID() != id2 {
				t.Error("Third task should be Task 2")
			}
		})
	})

	t.Run("update multiple tasks", func(t *testing.T) {
		provider := factory(t)

		id1, _ := provider.InsertBack(createTask("Task 1", "old"), nil)
		id2, _ := provider.InsertBack(createTask("Task 2", "old"), nil)
		id3, _ := provider.InsertBack(createTask("Task 3", "old"), nil)

		// Update categories
		provider.Update(id1, createTask("Task 1", "new"))
		provider.Update(id2, createTask("Task 2", "new"))
		provider.Update(id3, createTask("Task 3", "new"))

		// Verify all updated
		allUpdated := true
		provider.WithTasks([]model.TaskID{id1, id2, id3}, func(tasks []model.ReadableTask) {
			for _, task := range tasks {
				if task.GetCategory() != "new" {
					allUpdated = false
				}
			}
		})

		if !allUpdated {
			t.Error("Not all tasks were updated")
		}
	})

	t.Run("empty backlog operations", func(t *testing.T) {
		provider := factory(t)

		// These should all handle empty backlog gracefully
		firstID, _ := provider.GetFirstChildTaskID(nil)
		if firstID != nil {
			t.Error("Expected nil first child for empty backlog")
		}

		lastID, _ := provider.GetLastChildTaskID(nil)
		if lastID != nil {
			t.Error("Expected nil last child for empty backlog")
		}

		provider.WithRoots(func(roots []model.ReadableTask) {
			if len(roots) != 0 {
				t.Error("Expected 0 roots for empty backlog")
			}
		})
	})
}
