package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// DevcontainerCommand represents a command to be executed against the devcontainer CLI
type DevcontainerCommand struct {
	BoxConfig      BoxConfig
	Command        string
	AdditionalArgs []string
}

// Execute builds and runs the devcontainer command
func (dc *DevcontainerCommand) Execute() error {
	// Prepare the docker command and base arguments
	dockerArgs := []string{
		"run", "--rm", "-it",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-v", fmt.Sprintf("%s:%s", dc.BoxConfig.Workspace, dc.BoxConfig.Workspace),
	}

	// Optional config path - add volume mount before the image name
	if dc.BoxConfig.Config != "" {
		configDir := filepath.Dir(dc.BoxConfig.Config)
		dockerArgs = append(dockerArgs,
			"-v", fmt.Sprintf("%s:%s", configDir, configDir))
	}

	// Add the image and the command
	dockerArgs = append(dockerArgs, "devcontainer", dc.Command)

	// Add the workspace folder argument
	dockerArgs = append(dockerArgs, "--workspace-folder", dc.BoxConfig.Workspace)

	// Add config path argument if needed
	if dc.BoxConfig.Config != "" {
		dockerArgs = append(dockerArgs, "--override-config", dc.BoxConfig.Config)
	}

	// Add any additional arguments
	dockerArgs = append(dockerArgs, dc.AdditionalArgs...)

	// Create and configure the command
	dockerCmd := exec.Command("docker", dockerArgs...)

	// Set up command output streams
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr
	dockerCmd.Stdin = os.Stdin

	// Run the command
	return dockerCmd.Run()
}

func FindDevContainer(labels []string) (string, error) {
	containers, err := ListContainers(labels)
	if err != nil {
		return "", err
	}

	return containers, nil
}

func InspectContainer(containerID string) (types.ContainerJSON, error) {
	// Execute docker inspect command for the given container IDs
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	containerInfo, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, err
	}

	return containerInfo, nil
}

func ListContainers(labels []string) ([]types.ContainerSummary, error) {
	// Create a Docker client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("error creating Docker client: %v", err)
	}
	defer cli.Close()

	// Create filters for the labels
	labelFilters := filters.NewArgs()
	for _, label := range labels {
		labelFilters.Add("label", label)
	}

	// List containers with the specified filters
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: labelFilters,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing containers: %v", err)
	}

	return containers, nil
}
