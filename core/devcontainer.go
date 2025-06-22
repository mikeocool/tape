package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/mikeocool/tape/container"
	"github.com/mikeocool/tape/devcontinaer"
)

const DevContainerCliImage = "devcontainer:latest"

const HostFolderLabel = "devcontainer.local_folder" // used to label containers created from a workspace/folder
const ConfigFileLabel = "devcontainer.config_file"

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

	// Configure container binds for volumes
	binds := []string{
		"/var/run/docker.sock:/var/run/docker.sock",
		fmt.Sprintf("%s:%s", dc.BoxConfig.Workspace, dc.BoxConfig.Workspace),
	}

	// Optional config path binding
	if dc.BoxConfig.Config != "" {
		configDir := filepath.Dir(dc.BoxConfig.Config)
		binds = append(binds, fmt.Sprintf("%s:%s", configDir, configDir))
		// TODO manage binding the Dockerfile
		// the build path is relative to the config file
		// if Dockerfile is in workspace -- maybe just mount the workspace?
		// though need to handle cases where we need to modify the devcontainer config?
	}

	cli, err := container.NewClient()
	if err != nil {
		return fmt.Errorf("error creating container client: %v", err)
	}
	defer cli.Close()

	config := container.ContainerConfig{
		Image:       DevContainerCliImage,
		Command:     devConArgs,
		Interactive: true,
		Binds:       binds,
	}
	ctx := context.Background()
	devContainer, err := cli.CreateContainer(ctx, config)
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

		// TOOD only show this when debugging
		fmt.Printf("Using devcontainer config:\n%s\n", string(configJSON))

		err = devContainer.CreateFile(ctx, "/tmp/devcontainer.json", configJSON)
		if err != nil {
			return fmt.Errorf("error creating config file: %v", err)
		}
	}

	err = devContainer.AttachAndRun(ctx, devConArgs)
	if err != nil {
		return fmt.Errorf("error attaching and running container: %v", err)
	}

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

func FindDevContainer(config BoxConfig) (*container.Container, error) {
	cli, err := container.NewClient()
	if err != nil {
		return nil, fmt.Errorf("error creating container client: %v", err)
	}
	defer cli.Close()

	ctx := context.Background()

	hostFolderLabel := fmt.Sprintf("%s=%s", HostFolderLabel, config.Workspace)
	labels := []string{
		hostFolderLabel,
		fmt.Sprintf("%s=%s", ConfigFileLabel, config.Config),
	}

	dc, err := cli.FindContainer(ctx, labels)
	if err != nil && container.IsContainerNotFound(err) {
		// seems like sometimes the config file label is wrong?
		// so matching the devcontainer-cli impl of just using the host folder label
		dc, err = cli.FindContainer(ctx, []string{hostFolderLabel})
	}

	if err != nil {
		return nil, err
	}

	return dc, nil
}
