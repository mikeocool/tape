package devcontinaer

import (
	"encoding/json"
	"fmt"
	"os"
)

// DevContainerConfig represents the root structure of a devcontainer.json file
type DevContainerConfig struct {
	// Common properties from devContainerCommon
	Name                        string                    `json:"name,omitempty"`
	Features                    map[string]interface{}    `json:"features,omitempty"`
	OverrideFeatureInstallOrder []string                  `json:"overrideFeatureInstallOrder,omitempty"`
	ForwardPorts                []interface{}             `json:"forwardPorts,omitempty"`
	PortsAttributes             map[string]PortAttributes `json:"portsAttributes,omitempty"`
	OtherPortsAttributes        *PortAttributes           `json:"otherPortsAttributes,omitempty"`
	UpdateRemoteUserUID         *bool                     `json:"updateRemoteUserUID,omitempty"`
	RemoteEnv                   map[string]*string        `json:"remoteEnv,omitempty"`
	RemoteUser                  string                    `json:"remoteUser,omitempty"`
	InitializeCommand           *CommandValue             `json:"initializeCommand,omitempty"`
	OnCreateCommand             *CommandValue             `json:"onCreateCommand,omitempty"`
	UpdateContentCommand        *CommandValue             `json:"updateContentCommand,omitempty"`
	PostCreateCommand           *CommandValue             `json:"postCreateCommand,omitempty"`
	PostStartCommand            *CommandValue             `json:"postStartCommand,omitempty"`
	PostAttachCommand           *CommandValue             `json:"postAttachCommand,omitempty"`
	WaitFor                     string                    `json:"waitFor,omitempty"`
	UserEnvProbe                string                    `json:"userEnvProbe,omitempty"`
	HostRequirements            *HostRequirements         `json:"hostRequirements,omitempty"`
	Customizations              map[string]interface{}    `json:"customizations,omitempty"`

	// Non-compose specific properties
	AppPort         *AppPortValue     `json:"appPort,omitempty"`
	ContainerEnv    map[string]string `json:"containerEnv,omitempty"`
	ContainerUser   string            `json:"containerUser,omitempty"`
	Mounts          []string          `json:"mounts,omitempty"`
	RunArgs         []string          `json:"runArgs,omitempty"`
	ShutdownAction  string            `json:"shutdownAction,omitempty"`
	OverrideCommand *bool             `json:"overrideCommand,omitempty"`
	WorkspaceFolder string            `json:"workspaceFolder,omitempty"`
	WorkspaceMount  string            `json:"workspaceMount,omitempty"`

	// Dockerfile specific properties
	Build      *BuildOptions `json:"build,omitempty"`
	DockerFile string        `json:"dockerFile,omitempty"`
	Context    string        `json:"context,omitempty"`

	// Image specific properties
	Image string `json:"image,omitempty"`

	// Docker Compose specific properties
	DockerComposeFile *ComposeFileValue `json:"dockerComposeFile,omitempty"`
	Service           string            `json:"service,omitempty"`
	RunServices       []string          `json:"runServices,omitempty"`
}

// AppPortValue represents an app port that can be an integer, string, or array of those
type AppPortValue struct {
	value interface{}
}

// UnmarshalJSON custom unmarshaler for AppPortValue
func (a *AppPortValue) UnmarshalJSON(data []byte) error {
	// Try as integer
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		a.value = i
		return nil
	}

	// Try as string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		a.value = s
		return nil
	}

	// Try as array of mixed integer/string
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err == nil {
		// Validate each element is either string or integer
		for _, v := range arr {
			switch v.(type) {
			case float64, string:
				// These are valid types in JSON for integer and string
			default:
				return fmt.Errorf("array contains invalid type: %T", v)
			}
		}
		a.value = arr
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into AppPortValue", data)
}

// MarshalJSON custom marshaler for AppPortValue
func (a AppPortValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.value)
}

// AsInt returns the port as an integer if it is an integer, otherwise returns 0
func (a AppPortValue) AsInt() int {
	if i, ok := a.value.(int); ok {
		return i
	}
	return 0
}

// AsString returns the port as a string if it is a string, otherwise returns empty string
func (a AppPortValue) AsString() string {
	if s, ok := a.value.(string); ok {
		return s
	}
	return ""
}

// AsArray returns the port as an array if it is an array, otherwise returns nil
func (a AppPortValue) AsArray() []interface{} {
	if arr, ok := a.value.([]interface{}); ok {
		return arr
	}
	return nil
}

// ComposeFileValue represents a docker-compose file that can be a string or array of strings
type ComposeFileValue struct {
	value interface{}
}

// UnmarshalJSON custom unmarshaler for ComposeFileValue
func (c *ComposeFileValue) UnmarshalJSON(data []byte) error {
	// Try as string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		c.value = s
		return nil
	}

	// Try as array of strings
	var a []string
	if err := json.Unmarshal(data, &a); err == nil {
		c.value = a
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into ComposeFileValue", data)
}

// MarshalJSON custom marshaler for ComposeFileValue
func (c ComposeFileValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.value)
}

// AsString returns the compose file as a string if it is a string, otherwise returns empty string
func (c ComposeFileValue) AsString() string {
	if s, ok := c.value.(string); ok {
		return s
	}
	return ""
}

// AsArray returns the compose file as an array if it is an array, otherwise returns nil
func (c ComposeFileValue) AsArray() []string {
	if a, ok := c.value.([]string); ok {
		return a
	}
	return nil
}

// CommandValue represents a command that can be a string, array of strings, or object
type CommandValue struct {
	value interface{}
}

// UnmarshalJSON custom unmarshaler for CommandValue to handle multiple types
func (c *CommandValue) UnmarshalJSON(data []byte) error {
	// Try as string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		c.value = s
		return nil
	}

	// Try as array of strings
	var a []string
	if err := json.Unmarshal(data, &a); err == nil {
		c.value = a
		return nil
	}

	// Try as object
	var o map[string]interface{}
	if err := json.Unmarshal(data, &o); err == nil {
		c.value = o
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into CommandValue", data)
}

// MarshalJSON custom marshaler for CommandValue
func (c CommandValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.value)
}

// IsString checks if the command is a string
func (c CommandValue) IsString() bool {
	_, ok := c.value.(string)
	return ok
}

// IsArray checks if the command is an array
func (c CommandValue) IsArray() bool {
	_, ok := c.value.([]string)
	return ok
}

// IsObject checks if the command is an object
func (c CommandValue) IsObject() bool {
	_, ok := c.value.(map[string]interface{})
	return ok
}

// AsString returns the command as a string if it is a string, otherwise returns empty string
func (c CommandValue) AsString() string {
	if s, ok := c.value.(string); ok {
		return s
	}
	return ""
}

// AsArray returns the command as an array if it is an array, otherwise returns nil
func (c CommandValue) AsArray() []string {
	if a, ok := c.value.([]string); ok {
		return a
	}
	return nil
}

// AsObject returns the command as an object if it is an object, otherwise returns nil
func (c CommandValue) AsObject() map[string]interface{} {
	if o, ok := c.value.(map[string]interface{}); ok {
		return o
	}
	return nil
}

// PortAttributes represents the attributes for a specific port
type PortAttributes struct {
	OnAutoForward    string `json:"onAutoForward,omitempty"`
	ElevateIfNeeded  *bool  `json:"elevateIfNeeded,omitempty"`
	Label            string `json:"label,omitempty"`
	RequireLocalPort *bool  `json:"requireLocalPort,omitempty"`
	Protocol         string `json:"protocol,omitempty"`
}

// HostRequirements represents the host hardware requirements
type HostRequirements struct {
	CPUs    int         `json:"cpus,omitempty"`
	Memory  string      `json:"memory,omitempty"`
	Storage string      `json:"storage,omitempty"`
	GPU     interface{} `json:"gpu,omitempty"`
}

// GPURequirements represents detailed GPU requirements when specified as an object
type GPURequirements struct {
	Cores  int    `json:"cores,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// BuildOptions represents Docker build-related options
type BuildOptions struct {
	Dockerfile string            `json:"dockerfile,omitempty"`
	Context    string            `json:"context,omitempty"`
	Target     string            `json:"target,omitempty"`
	Args       map[string]string `json:"args,omitempty"`
	CacheFrom  interface{}       `json:"cacheFrom,omitempty"`
}

// ParseDevContainer parses a devcontainer.json file into a DevContainer struct
func ParseDevContainer(data []byte) (*DevContainerConfig, error) {
	var container DevContainerConfig
	err := json.Unmarshal(data, &container)
	if err != nil {
		return nil, err
	}
	return &container, nil
}

// LoadDevContainerFromFile loads a devcontainer.json file from the given path
func LoadDevContainerFromFile(path string) (*DevContainerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseDevContainer(data)
}

// SaveDevContainerToFile saves a DevContainer to the given path
func (dc *DevContainerConfig) SaveDevContainerToFile(path string) error {
	data, err := json.MarshalIndent(dc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
