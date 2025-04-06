package container

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Client struct {
	client *client.Client
}

func NewClient() (*Client, error) {
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("error creating Docker client: %v", err)
	}

	return &Client{client: client}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) CreateContainer(ctx context.Context, config ContainerConfig) (*Container, error) {
	containerConfig := &container.Config{
		Image:        config.Image,
		Cmd:          config.Command,
		Tty:          config.Interactive,
		AttachStdout: config.Interactive,
		AttachStderr: config.Interactive,
		OpenStdin:    config.Interactive,
	}

	// Create host config with binds
	hostConfig := &container.HostConfig{
		Binds:      config.Binds,
		AutoRemove: true,
	}

	resp, err := c.client.ContainerCreate(
		ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("error creating container: %v", err)
	}

	return &Container{ID: resp.ID, Config: config}, nil
}
