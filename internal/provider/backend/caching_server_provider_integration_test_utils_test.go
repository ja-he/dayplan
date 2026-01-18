//go:build integration

package backend_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testInfrastructure holds the PostgreSQL container and server process for testing.
type testInfrastructure struct {
	postgresPort int
	serverPort   int
	containerID  string
	serverCmd    *exec.Cmd
	serverCancel context.CancelFunc
	databaseURL  string
	serverURL    string
	testUser     string
	testPassword string
}

// findFreePort returns an available TCP port.
func findFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

// waitForPort waits until a TCP port is accepting connections.
func waitForPort(t *testing.T, host string, port int, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("%s:%d", host, port)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", addr)
}

// waitForHTTP waits until an HTTP endpoint returns a successful response.
func waitForHTTP(t *testing.T, url string, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

// setupTestInfrastructure starts PostgreSQL container and the server binary.
func setupTestInfrastructure(t *testing.T) *testInfrastructure {
	t.Helper()

	infra := &testInfrastructure{
		testUser:     "testuser",
		testPassword: "testpw",
	}

	// Find free ports
	infra.postgresPort = findFreePort(t)
	infra.serverPort = findFreePort(t)

	infra.databaseURL = fmt.Sprintf(
		"postgres://dayplan-tester:dayplan-tester@localhost:%d/dayplan-tester?sslmode=disable",
		infra.postgresPort,
	)
	infra.serverURL = fmt.Sprintf("http://localhost:%d", infra.serverPort)

	// Start PostgreSQL container
	t.Logf("Starting PostgreSQL container on port %d", infra.postgresPort)
	cmd := exec.Command("podman", "run", "-d",
		"-e", "POSTGRES_USER=dayplan-tester",
		"-e", "POSTGRES_PASSWORD=dayplan-tester",
		"-e", "POSTGRES_DB=dayplan-tester",
		"-p", fmt.Sprintf("%d:5432", infra.postgresPort),
		"postgres:16",
	)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to start PostgreSQL container: %v", err)
	}
	infra.containerID = strings.TrimSpace(string(output))
	t.Logf("PostgreSQL container started: %s", infra.containerID[:12])

	// Wait for PostgreSQL to be ready using pg_isready inside the container
	t.Log("Waiting for PostgreSQL to be ready...")
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		checkCmd := exec.Command("podman", "exec", infra.containerID,
			"pg_isready", "-U", "dayplan-tester", "-d", "dayplan-tester")
		if err := checkCmd.Run(); err == nil {
			t.Log("PostgreSQL is ready")
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if time.Now().After(deadline) {
		infra.cleanup(t)
		t.Fatalf("PostgreSQL did not become ready in time")
	}

	// Find the server binary (should be in repo root)
	serverBinary, err := filepath.Abs("../../../server")
	if err != nil {
		infra.cleanup(t)
		t.Fatalf("failed to get server binary path: %v", err)
	}
	if _, err := os.Stat(serverBinary); os.IsNotExist(err) {
		infra.cleanup(t)
		t.Fatalf("server binary not found at %s", serverBinary)
	}

	// Run migrations (with retry since pg_isready can return before connections are fully accepted)
	t.Log("Running database migrations...")
	var migrateErr error
	var migrateOutput []byte
	for i := 0; i < 10; i++ {
		migrateCmd := exec.Command(serverBinary, "-migrate", "-reset-only")
		migrateCmd.Env = append(os.Environ(), "DATABASE_URL="+infra.databaseURL)
		migrateOutput, migrateErr = migrateCmd.CombinedOutput()
		if migrateErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if migrateErr != nil {
		infra.cleanup(t)
		t.Fatalf("failed to run migrations after retries: %v\nOutput: %s", migrateErr, migrateOutput)
	}

	// Create test user
	t.Logf("Creating test user: %s", infra.testUser)
	createUserCmd := exec.Command(serverBinary, "-create-user", fmt.Sprintf("%s:%s", infra.testUser, infra.testPassword))
	createUserCmd.Env = append(os.Environ(), "DATABASE_URL="+infra.databaseURL)
	if output, err := createUserCmd.CombinedOutput(); err != nil {
		infra.cleanup(t)
		t.Fatalf("failed to create user: %v\nOutput: %s", err, output)
	}

	// Start the server
	t.Logf("Starting server on port %d", infra.serverPort)
	ctx, cancel := context.WithCancel(context.Background())
	infra.serverCancel = cancel
	infra.serverCmd = exec.CommandContext(ctx, serverBinary, "-listen", fmt.Sprintf(":%d", infra.serverPort))
	infra.serverCmd.Env = append(os.Environ(), "DATABASE_URL="+infra.databaseURL)
	infra.serverCmd.Stdout = os.Stdout
	infra.serverCmd.Stderr = os.Stderr
	if err := infra.serverCmd.Start(); err != nil {
		infra.cleanup(t)
		t.Fatalf("failed to start server: %v", err)
	}

	// Wait for server to be ready
	t.Log("Waiting for server to be ready...")
	if err := waitForPort(t, "localhost", infra.serverPort, 10*time.Second); err != nil {
		infra.cleanup(t)
		t.Fatalf("server did not become ready: %v", err)
	}

	t.Log("Test infrastructure ready")
	return infra
}

// cleanup stops the server and removes the PostgreSQL container.
func (infra *testInfrastructure) cleanup(t *testing.T) {
	t.Helper()

	// Stop the server
	if infra.serverCancel != nil {
		t.Log("Stopping server...")
		infra.serverCancel()
		if infra.serverCmd != nil {
			infra.serverCmd.Wait()
		}
	}

	// Stop and remove the PostgreSQL container
	if infra.containerID != "" {
		t.Logf("Stopping PostgreSQL container %s...", infra.containerID[:12])
		exec.Command("podman", "stop", infra.containerID).Run()
		exec.Command("podman", "rm", infra.containerID).Run()
	}

	t.Log("Cleanup complete")
}
