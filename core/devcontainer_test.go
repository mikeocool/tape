package core

import (
	"os"
	"reflect"
	"testing"
)

func TestDevcontainerCommandExecuteWithMounts(t *testing.T) {
	boxCfg := BoxConfig{
		Name:      "testbox",
		Workspace: "/test/workspace",
		Mounts: []string{
			"type=bind,source=/src,target=/dst",
			"type=volume,source=myvol,target=/app",
		},
		Config: "", // Assuming no config file for simplicity in this test focus
	}
	cmd := DevcontainerCommand{
		BoxConfig:      boxCfg,
		Command:        "up",
		AdditionalArgs: []string{"--some-extra-arg"},
	}

	actualDevConArgs := buildDevcontainerArgs(&cmd)

	expectedDevConArgs := []string{
		"devcontainer", "up", // Base command
		"--workspace-folder", "/test/workspace", // Workspace
		// Mounts - order matters as they are appended sequentially
		"--mount", "type=bind,source=/src,target=/dst",
		"--mount", "type=volume,source=myvol,target=/app",
		// Additional args
		"--some-extra-arg",
	}

	if !reflect.DeepEqual(actualDevConArgs, expectedDevConArgs) {
		t.Errorf("Execute() devConArgs mismatch:\nExpected: %v\nActual:   %v", expectedDevConArgs, actualDevConArgs)
	}
}

// Add another test for when BoxConfig.Config is also specified,
// to ensure mounts are added correctly in that case too.
func TestDevcontainerCommandExecuteWithMountsAndConfigPath(t *testing.T) {
	boxCfg := BoxConfig{
		Name:      "testboxconfig",
		Workspace: "/test/workspaceconfig",
		Mounts: []string{
			"type=bind,source=/srcconfig,target=/dstconfig",
		},
		Config: "/test/workspaceconfig/.devcontainer/devcontainer.json", // Non-empty config
	}
	cmd := DevcontainerCommand{
		BoxConfig:      boxCfg,
		Command:        "up",
		AdditionalArgs: []string{},
	}

	actualDevConArgs := buildDevcontainerArgs(&cmd)

	expectedDevConArgs := []string{
		"devcontainer", "up",
		"--workspace-folder", "/test/workspaceconfig",
		"--config", "/tmp/devcontainer.json", // This is how it's handled in Execute()
		"--mount", "type=bind,source=/srcconfig,target=/dstconfig",
	}

	if !reflect.DeepEqual(actualDevConArgs, expectedDevConArgs) {
		t.Errorf("Execute() devConArgs with config mismatch:\nExpected: %v\nActual:   %v", expectedDevConArgs, actualDevConArgs)
	}
}
