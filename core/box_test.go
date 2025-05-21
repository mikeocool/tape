package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadBoxConfigWithMounts(t *testing.T) {
	// Create a temporary directory for config files
	tempDir, err := os.MkdirTemp("", "test-config-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Temporarily set ConfigDir
	originalConfigDir := ConfigDir
	ConfigDir = tempDir
	defer func() { ConfigDir = originalConfigDir }()

	// Define the test environment name and config file path
	envName := "testenv"
	configFilePath := filepath.Join(ConfigDir, envName+".yaml")

	// YAML content for the test
	yamlContent := `
workspace: /path/to/workspace
mounts:
  - type=bind,source=/source,target=/target
  - type=volume,source=myvol,target=/app
`
	// Write the temporary config file
	err = os.WriteFile(configFilePath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp config file: %v", err)
	}

	// Load the box config
	boxConfig, err := LoadBoxConfig(envName)
	if err != nil {
		t.Fatalf("LoadBoxConfig failed: %v", err)
	}

	// Assertions
	expectedMounts := []string{
		"type=bind,source=/source,target=/target",
		"type=volume,source=myvol,target=/app",
	}
	assert.Equal(t, "/path/to/workspace", boxConfig.Workspace)
	assert.Equal(t, expectedMounts, boxConfig.Mounts)
	assert.Equal(t, envName, boxConfig.Name) // LoadBoxConfig should set the Name
}
