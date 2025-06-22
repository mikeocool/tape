package core

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type GlobalConfig struct {
	DotfilesRepository string `yaml:"dotfiles-repository"`
}

func LoadGlobalConfig() (*GlobalConfig, error) {
	configFile := filepath.Join(ConfigDir, ".tape.yml")
	yamlData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file %s: %v", configFile, err)
	}

	var config GlobalConfig
	if err := yaml.Unmarshal(yamlData, &config); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %v", err)
	}

	// TODO validate config

	return &config, nil
}
