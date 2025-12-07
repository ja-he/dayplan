//go:build integration

package main_test

import (
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"testing"
)

func TestAdd(t *testing.T) {
	tmpHome := t.TempDir()
	t.Logf("Using tempdir '%s'", tmpHome)

	cmd := exec.Command("go", "run", ".", "add", "-c", "default", "-s", "2025-12-07 14:00:00", "-e", "2025-12-07 14:30:00", "-n", "Foo")
	cmd.Env = append(os.Environ(),
		"GOPATH="+path.Join(tmpHome, "go"),
		"GOFLAGS=-modcacherw", // Keep module files writable
		"HOME="+tmpHome,
		"DAYPLAN_HOME="+path.Join(tmpHome, "dayplan"),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\nOutput: %s", err, output)
	}

	// Check that the expected file was created
	expectedFile := path.Join(tmpHome, "dayplan", "days", "2025-12-07")
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("failed to read expected file %s: %v", expectedFile, err)
	}

	// Verify one line
	lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d lines:\n%s", len(lines), content)
	}

	// Verify the line matches the expected pattern
	pattern := regexp.MustCompile(`^.*\|2025-12-07T14:00:00Z\|2025-12-07T14:30:00Z\|default\|Foo$`)
	if !pattern.MatchString(lines[0]) {
		t.Errorf("line does not match expected pattern:\ngot: %s\npattern: %s", lines[0], pattern.String())
	}

}
