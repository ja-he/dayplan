package backend_test

import (
	"testing"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/provider"
	"github.com/ja-he/dayplan/internal/provider/backend"
)

// TestFilesDataProvider_EventProviderSuite runs the comprehensive EventProvider test suite
// against the FilesDataProvider implementation.
func TestFilesDataProvider_EventProviderSuite(t *testing.T) {
	factory := func(t *testing.T) provider.EventProvider {
		tmpDir := t.TempDir()
		categoryProvider := &backend.MemoryCategoryProvider{M: make(map[model.CategoryName]*model.Category)}

		p, err := backend.NewFilesDataProvider(tmpDir, categoryProvider)
		if err != nil {
			t.Fatalf("Failed to create FilesDataProvider: %v", err)
		}

		return p
	}

	// The file-based implementation does not support events spanning multiple days,
	// so we skip cross-day tests.
	opts := EventProviderTestOptions{
		SkipCrossDayTests: true,
	}

	RunEventProviderTests(t, factory, opts)
}
