package devcontinaer

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseDevContainer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *DevContainerConfig
		wantErr  bool
	}{
		{
			name:     "empty config",
			input:    "{}",
			expected: &DevContainerConfig{},
			wantErr:  false,
		},
		{
			name:  "basic config",
			input: `{"name": "test-container", "image": "ubuntu:latest"}`,
			expected: &DevContainerConfig{
				Name:  "test-container",
				Image: "ubuntu:latest",
			},
			wantErr: false,
		},
		{
			name:     "invalid json",
			input:    `{"name": "test-container"`,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDevContainer([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDevContainer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ParseDevContainer() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSerializeDevContainer(t *testing.T) {
	tests := []struct {
		name     string
		config   *DevContainerConfig
		expected string
		wantErr  bool
	}{
		{
			name:     "empty config",
			config:   &DevContainerConfig{},
			expected: "{}",
			wantErr:  false,
		},
		{
			name: "basic config",
			config: &DevContainerConfig{
				Name:  "test-container",
				Image: "ubuntu:latest",
			},
			expected: `{"name":"test-container","image":"ubuntu:latest"}`,
			wantErr:  false,
		},
		{
			name: "config with run args",
			config: &DevContainerConfig{
				Name:    "dev-container",
				Image:   "node:14",
				RunArgs: []string{"--name", "my-container", "-p", "3000:3000"},
			},
			expected: `{"name":"dev-container","runArgs":["--name","my-container","-p","3000:3000"],"image":"node:14"}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.expected {
				t.Errorf("json.Marshal() = %v, want %v", string(got), tt.expected)
			}
		})
	}
}

func TestParseAndSerializeDevContainer(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty config",
			input:   "{}",
			wantErr: false,
		},
		{
			name:    "basic config",
			input:   `{"name":"test-container","image":"ubuntu:latest"}`,
			wantErr: false,
		},
		{
			name:    "complex config",
			input:   `{"name":"dev-container","image":"node:14","runArgs":["--name","my-container","-p","3000:3000"],"features":{"ghcr.io/devcontainers/features/node:1":{"version":"lts"}},"forwardPorts":[3000,8080],"remoteUser":"node"}`,
			wantErr: false,
		},
		{
			name:    "config with app port",
			input:   `{"name":"web-app","image":"nginx","appPort":[80,"443:8443"]}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the JSON into our config structure
			var config DevContainerConfig
			err := json.Unmarshal([]byte(tt.input), &config)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Serialize back to JSON
			output, err := json.Marshal(config)
			if err != nil {
				t.Errorf("json.Marshal() error = %v", err)
				return
			}

			// Parse both the original and the re-serialized JSON to compare their structure
			var originalMap map[string]interface{}
			var outputMap map[string]interface{}

			err = json.Unmarshal([]byte(tt.input), &originalMap)
			if err != nil {
				t.Errorf("Failed to parse original JSON: %v", err)
				return
			}

			err = json.Unmarshal(output, &outputMap)
			if err != nil {
				t.Errorf("Failed to parse output JSON: %v", err)
				return
			}

			// Compare the two maps
			if !reflect.DeepEqual(originalMap, outputMap) {
				t.Errorf("Round-trip serialization failed.\nOriginal: %s\nOutput:   %s", tt.input, string(output))
			}
		})
	}
}

func TestAppPortValue(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantInt    int
		wantString string
		wantArray  bool
	}{
		{
			name:       "integer port",
			input:      `{"appPort": 3000}`,
			wantInt:    3000,
			wantString: "",
			wantArray:  false,
		},
		{
			name:       "string port",
			input:      `{"appPort": "3000:3000"}`,
			wantInt:    0,
			wantString: "3000:3000",
			wantArray:  false,
		},
		{
			name:       "array port",
			input:      `{"appPort": [3000, "3001:3001"]}`,
			wantInt:    0,
			wantString: "",
			wantArray:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config DevContainerConfig
			err := json.Unmarshal([]byte(tt.input), &config)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if got := config.AppPort.AsInt(); got != tt.wantInt {
				t.Errorf("AppPort.AsInt() = %v, want %v", got, tt.wantInt)
			}

			if got := config.AppPort.AsString(); got != tt.wantString {
				t.Errorf("AppPort.AsString() = %v, want %v", got, tt.wantString)
			}

			if (config.AppPort.AsArray() != nil) != tt.wantArray {
				t.Errorf("AppPort.AsArray() is array: %v, want %v", config.AppPort.AsArray() != nil, tt.wantArray)
			}
		})
	}
}

func TestCommandValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		isString bool
		isArray  bool
		isObject bool
	}{
		{
			name:     "string command",
			input:    `{"postCreateCommand": "npm install"}`,
			isString: true,
			isArray:  false,
			isObject: false,
		},
		{
			name:     "array command",
			input:    `{"postCreateCommand": ["npm", "install"]}`,
			isString: false,
			isArray:  true,
			isObject: false,
		},
		{
			name:     "object command",
			input:    `{"postCreateCommand": {"linux": "apt-get update", "windows": "choco install"}}`,
			isString: false,
			isArray:  false,
			isObject: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config DevContainerConfig
			err := json.Unmarshal([]byte(tt.input), &config)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if got := config.PostCreateCommand.IsString(); got != tt.isString {
				t.Errorf("PostCreateCommand.IsString() = %v, want %v", got, tt.isString)
			}

			if got := config.PostCreateCommand.IsArray(); got != tt.isArray {
				t.Errorf("PostCreateCommand.IsArray() = %v, want %v", got, tt.isArray)
			}

			if got := config.PostCreateCommand.IsObject(); got != tt.isObject {
				t.Errorf("PostCreateCommand.IsObject() = %v, want %v", got, tt.isObject)
			}
		})
	}
}

func TestComposeFileValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		isString bool
		isArray  bool
	}{
		{
			name:     "string compose file",
			input:    `{"dockerComposeFile": "docker-compose.yml"}`,
			isString: true,
			isArray:  false,
		},
		{
			name:     "array compose file",
			input:    `{"dockerComposeFile": ["docker-compose.yml", "docker-compose.override.yml"]}`,
			isString: false,
			isArray:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config DevContainerConfig
			err := json.Unmarshal([]byte(tt.input), &config)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if (config.DockerComposeFile.AsString() != "") != tt.isString {
				t.Errorf("DockerComposeFile.AsString() is string: %v, want %v", config.DockerComposeFile.AsString() != "", tt.isString)
			}

			if (config.DockerComposeFile.AsArray() != nil) != tt.isArray {
				t.Errorf("DockerComposeFile.AsArray() is array: %v, want %v", config.DockerComposeFile.AsArray() != nil, tt.isArray)
			}
		})
	}
}
