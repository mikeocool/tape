package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

const HostFolderLabel = "devcontainer.local_folder" // used to label containers created from a workspace/folder
const ConfigFileLabel = "devcontainer.config_file"

// ContainerNotFoundError is returned when no containers match the specified criteria.
// This error can be identified at call sites using type assertion:
//
//	container, err := FindContainer(labels)
//	if err != nil {
//		if _, ok := err.(*ContainerNotFoundError); ok {
//			// Handle case where no container was found
//		} else {
//			// Handle other errors
//		}
//	}
type ContainerNotFoundError struct {
	Labels []string
}

// Error implements the error interface for ContainerNotFoundError
func (e *ContainerNotFoundError) Error() string {
	return "no matching containers found"
}

// IsContainerNotFound checks if an error is a ContainerNotFoundError
func IsContainerNotFound(err error) bool {
	_, ok := err.(*ContainerNotFoundError)
	return ok
}

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

func FindDevContainer(config BoxConfig) (container.Summary, error) {
	hostFolderLabel := fmt.Sprintf("%s=%s", HostFolderLabel, config.Workspace)
	labels := []string{
		hostFolderLabel,
		fmt.Sprintf("%s=%s", ConfigFileLabel, config.Config),
	}

	summary, err := FindContainer(labels)
	if err != nil && IsContainerNotFound(err) {
		// seems like sometimes the config file label is wrong?
		// so matching the devcontainer-cli impl of just using the host folder label
		summary, err = FindContainer([]string{hostFolderLabel})
	}

	if err != nil {
		return container.Summary{}, err
	}

	return summary, nil
}

func FindContainer(labels []string) (container.Summary, error) {
	containers, err := ListContainers(labels)
	if err != nil {
		return container.Summary{}, err
	}

	// Filter out containers with status "Removing"
	var filteredContainers []container.Summary
	for _, c := range containers {
		// Skip containers that are in the process of being removed
		if c.Status != "Removing" {
			filteredContainers = append(filteredContainers, c)
		}
	}
	if len(filteredContainers) > 0 {
		return filteredContainers[0], nil
	}
	return container.Summary{}, &ContainerNotFoundError{Labels: labels}
}

func InspectContainer(containerID string) (container.InspectResponse, error) {
	// Execute docker inspect command for the given container IDs
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	containerInfo, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return container.InspectResponse{}, err
	}

	return containerInfo, nil
}

func StopContainer(containerID string) error {
	// Create a Docker client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("error creating Docker client: %v", err)
	}
	defer cli.Close()

	// Set timeout for stopping the container (in seconds)
	timeout := int(30 * time.Second)

	// Stop the container
	err = cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		return fmt.Errorf("error stopping container %s: %v", containerID, err)
	}

	return nil
}

func RemoveContainer(containerID string) error {
	// Create a Docker client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("error creating Docker client: %v", err)
	}
	defer cli.Close()

	// Remove the container
	err = cli.ContainerRemove(ctx, containerID, container.RemoveOptions{RemoveVolumes: true, RemoveLinks: false, Force: true})
	if err != nil {
		return fmt.Errorf("error removing container %s: %v", containerID, err)
	}

	return nil
}

func ListContainers(labels []string) ([]container.Summary, error) {
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
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: labelFilters,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing containers: %v", err)
	}

	return containers, nil
}
