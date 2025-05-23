package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"github.com/mikeocool/tape/container"
	"gopkg.in/yaml.v2"
)

var ConfigDir string

func init() {
	ConfigDir = os.Getenv("TAPE_CONFIG_DIR")
	if ConfigDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to a default if we can't get the home directory
			ConfigDir = "/etc/tape"
		} else {
			ConfigDir = filepath.Join(home, ".tape")
		}
	}
}

type BoxConfig struct {
	Name      string
	Workspace string `yaml:"workspace" validate:"required"`
	Config    string `yaml:"config,omitempty"`
}

// ValidateConfig validates the BoxConfig using validator
func (b *BoxConfig) ValidateConfig() error {
	validate := validator.New()
	return validate.Struct(b)
}

// LoadBoxConfig loads a box configuration from a YAML file by environment name
func LoadBoxConfig(envName string) (*BoxConfig, error) {
	configFile := filepath.Join(ConfigDir, envName+".yml")
	yamlData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file %s: %v", configFile, err)
	}

	var config BoxConfig
	if err := yaml.Unmarshal(yamlData, &config); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %v", err)
	}
	config.Name = envName

	// Validate the configuration using validator
	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	// fill in defaults
	// Make workspace path absolute
	if !filepath.IsAbs(config.Workspace) {
		absPath, err := filepath.Abs(filepath.Join(ConfigDir, config.Workspace))
		if err != nil {
			return nil, fmt.Errorf("error converting workspace to absolute path: %v", err)
		}
		config.Workspace = absPath
	}

	// Remove trailing slash if present
	config.Workspace = filepath.Clean(config.Workspace)

	if config.Config == "" {
		config.Config = fmt.Sprintf("%s/.devcontainer/devcontainer.json", config.Workspace)
	} else {
		if !filepath.IsAbs(config.Config) {
			absConfigPath, err := filepath.Abs(filepath.Join(ConfigDir, config.Config))
			if err != nil {
				return nil, fmt.Errorf("error converting config to absolute path: %v", err)
			}
			config.Config = absConfigPath
		}
	}

	return &config, nil
}

// ListBoxConfigs returns a list of available box configurations by listing
// all YAML files in the sample-config directory and removing the .yml extension
func ListBoxConfigs() ([]string, error) {

	// Check if the directory exists
	if _, err := os.Stat(ConfigDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("config directory %s does not exist", ConfigDir)
	}

	// Read all files in the directory
	files, err := os.ReadDir(ConfigDir)
	if err != nil {
		return nil, fmt.Errorf("error reading config directory: %v", err)
	}

	var configs []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if filepath.Ext(filename) == ".yml" {
			// Remove the .yml extension to get the environment name
			envName := filename[:len(filename)-4]
			configs = append(configs, envName)
		}
	}

	return configs, nil
}

type BoxState string

const (
	BoxStateRunning      BoxState = "running"
	BoxStateStopped      BoxState = "stopped"
	BoxStateDoesNotExist BoxState = "does-not-exist"
	BoxStateUnknown      BoxState = "unknown"
)

type BoxSummary struct {
	EnvName     string
	State       BoxState
	ContainerID string
}

func GetBoxSummary(envName string) (*BoxSummary, error) {
	boxConfig, err := LoadBoxConfig(envName)
	if err != nil {
		return nil, err
	}

	state := BoxStateUnknown
	dc, err := FindDevContainer(*boxConfig)
	if err != nil {
		if container.IsContainerNotFound(err) {
			return &BoxSummary{
				EnvName: envName,
				State:   BoxStateDoesNotExist,
			}, nil
		}
		return nil, err
	}

	if dc.State == "running" {
		state = BoxStateRunning
	} else if dc.State == "exited" {
		state = BoxStateStopped
	}

	return &BoxSummary{
		EnvName:     envName,
		State:       state,
		ContainerID: dc.ID,
	}, nil

}
