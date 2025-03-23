package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
)

type BoxConfig struct {
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
	configFile := filepath.Join("sample-config", envName+".yml")
	yamlData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file %s: %v", configFile, err)
	}

	var config BoxConfig
	if err := yaml.Unmarshal(yamlData, &config); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %v", err)
	}

	// Validate the configuration using validator
	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	return &config, nil
}
