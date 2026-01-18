//go:build integration

package backend_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/provider"
	"github.com/ja-he/dayplan/internal/provider/backend"
)

// TestCachingServerClientDataProvider_EventProviderSuite runs the comprehensive EventProvider test suite
// against the CachingServerClientDataProvider implementation connected to a real server.
func TestCachingServerClientDataProvider_EventProviderSuite(t *testing.T) {
	// Check if podman is available
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman not available, skipping integration test")
	}

	// Set up infrastructure
	infra := setupTestInfrastructure(t)
	t.Cleanup(func() {
		infra.cleanup(t)
	})

	factory := func(t *testing.T) provider.EventProvider {
		tmpDir := t.TempDir()
		categoryProvider := &backend.MemoryCategoryProvider{M: make(map[model.CategoryName]*model.Category)}

		p, err := backend.NewCachingServerClientDataProvider(
			backend.CachingServerClientConfig{
				DBPath:    filepath.Join(tmpDir, "local_data.sqlite"),
				ServerURL: infra.serverURL,
			},
			categoryProvider,
		)
		if err != nil {
			t.Fatalf("Failed to create CachingServerClientDataProvider: %v", err)
		}

		// Login to the server
		if err := p.Login(infra.testUser, infra.testPassword); err != nil {
			p.Close()
			t.Fatalf("Failed to login: %v", err)
		}

		t.Cleanup(func() {
			p.Close()
		})

		return p
	}

	opts := EventProviderTestOptions{
		SkipCrossDayTests: false,
	}

	RunEventProviderTests(t, factory, opts)
}
