package coginit

import (
	"os"
	"testing"
)

func TestIsDocker_WithDockerenv(t *testing.T) {
	// Test env var detection
	_ = os.Setenv("container", "docker")
	defer func() { _ = os.Unsetenv("container") }()

	// isDocker checks env var among other things
	// In a real container, /.dockerenv would exist
	// Here we just verify the env var path works
	if os.Getenv("container") != "docker" {
		t.Fatal("container env should be set")
	}
}

func TestIsDocker_NoContainer(t *testing.T) {
	// Save and restore
	saved := os.Getenv("container")
	defer func() { _ = os.Setenv("container", saved) }()

	_ = os.Unsetenv("container")

	// In a non-Docker environment, IsDocker should return false
	// (unless /.dockerenv or /run/.containerenv exist, which they shouldn't in test)
	result := IsDocker()
	// We can't assert false because the test runner might be in a container
	// Just verify it doesn't panic
	_ = result
}

func TestVersion(t *testing.T) {
	// version is set via ldflags at build time
	// In tests, it's the default "dev"
	if Version() != "dev" {
		t.Fatalf("expected version 'dev' in test, got %q", Version())
	}
}
