package core

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/mikeocool/tape/devcontinaer"
	"golang.org/x/term"
)

const DevContainerCliImage = "devcontainer:latest"

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
	devConArgs := []string{"devcontainer", dc.Command, "--workspace-folder", dc.BoxConfig.Workspace}

	// Add config path argument if needed
	if dc.BoxConfig.Config != "" {
		//devConArgs = append(devConArgs, "--config", dc.BoxConfig.Config)
		devConArgs = append(devConArgs, "--config", "/tmp/devcontainer.json")
	}

	// Add any additional arguments
	devConArgs = append(devConArgs, dc.AdditionalArgs...)

	// Prepare the docker command and base arguments
	// Create a Docker client
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("error creating Docker client: %v", err)
	}
	defer cli.Close()

	// Configure container binds for volumes
	binds := []string{
		"/var/run/docker.sock:/var/run/docker.sock",
		fmt.Sprintf("%s:%s", dc.BoxConfig.Workspace, dc.BoxConfig.Workspace),
	}

	// Optional config path binding
	if dc.BoxConfig.Config != "" {
		//configDir := filepath.Dir(dc.BoxConfig.Config)
		//binds = append(binds, fmt.Sprintf("%s:%s", configDir, configDir))
	}

	// Create container config
	containerConfig := &container.Config{
		Image:        DevContainerCliImage,
		Cmd:          devConArgs,
		Tty:          true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
	}

	// Create host config with binds
	hostConfig := &container.HostConfig{
		Binds:      binds,
		AutoRemove: true,
	}

	// Set up terminal raw mode to properly handle control sequences
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("unable to set terminal to raw mode: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Create the container
	resp, err := cli.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		"",
	)
	if err != nil {
		return fmt.Errorf("error creating container: %v", err)
	}

	if dc.BoxConfig.Config != "" {
		// Load the config file, modify it, and serialize it to JSON
		config, err := LoadConfig(dc.BoxConfig.Config)
		if err != nil {
			return fmt.Errorf("error loading config: %v", err)
		}
		overrideConfigValues(dc.BoxConfig, config)

		// Serialize the config to JSON
		configJSON, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("error serializing config to JSON: %v", err)
		}

		containerID := resp.ID
		containerDest := "/tmp/devcontainer.json"

		var copyContent bytes.Buffer
		tarWriter := tar.NewWriter(&copyContent)
		defer tarWriter.Close()

		header := &tar.Header{
			Name: filepath.Base(containerDest),
			Mode: 0644,
			Size: int64(len(configJSON)),
		}

		// Write the header to the tar
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("error writing tar header: %v", err)
		}

		// Write the JSON content to the tar
		if _, err := tarWriter.Write(configJSON); err != nil {
			return fmt.Errorf("error writing config JSON to tar: %v", err)
		}
		tarWriter.Close()

		contentReader := bytes.NewReader(copyContent.Bytes())

		// Create a tar archive for copying into the container
		err = cli.CopyToContainer(ctx, containerID, filepath.Dir(containerDest), contentReader, container.CopyToContainerOptions{
			AllowOverwriteDirWithFile: true,
		})
		if err != nil {
			return fmt.Errorf("error copying config file to container: %v", err)
		}
	}

	out, err := cli.ContainerAttach(ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Stdin:  true,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to container: %w", err)
	}
	defer out.Close()

	go func() {
		// Copy container output directly to terminal
		// TODO test that we also get stderr -- tty mode seems to break stdcopy
		//_, err := stdcopy.StdCopy(os.Stdout, os.Stderr, out.Reader)
		_, err := io.Copy(os.Stdout, out.Reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error streaming output: %s\n", err)
		}
	}()

	// Set up goroutine to handle terminal input (if needed)
	go func() {
		if _, err := io.Copy(out.Conn, os.Stdin); err != nil {
			fmt.Fprintf(os.Stderr, "Error copying stdin: %s\n", err)
		}
		out.CloseWrite()
	}()

	// Start the container
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("error starting container: %v", err)
	}

	// TODO this is probably not strcitly necessary, or can at least fail silently
	defer func() {
		if err := cli.ContainerStop(ctx, resp.ID, container.StopOptions{}); err != nil {
			log.Printf("Warning: failed to stop container: %v", err)
		}
	}()

	waitC, errC := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errC:
		if err != nil {
			return fmt.Errorf("error waiting for container: %v", err)
		}
	case <-waitC:
		// Container is ready
	}

	// Give a small amount of time for final I/O operations to complete
	time.Sleep(100 * time.Millisecond)

	return nil
}

func LoadConfig(path string) (*devcontinaer.DevContainerConfig, error) {
	// Read the original devcontainer.json file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading devcontainer config: %v", err)
	}

	// Parse the devcontainer.json into our config structure
	return devcontinaer.ParseDevContainer(data)
}

func overrideConfigValues(boxConfig BoxConfig, config *devcontinaer.DevContainerConfig) {
	if !slices.Contains(config.RunArgs, "--name") {
		config.RunArgs = append(config.RunArgs, "--name", boxConfig.Name)
	}
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
