package container

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

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

	return &Container{ID: resp.ID, State: "created"}, nil
}

func (c *Client) FindContainer(ctx context.Context, labels []string) (*Container, error) {
	containers, err := c.listContainers(ctx, labels)
	if err != nil {
		return nil, fmt.Errorf("error listing containers: %v", err)
	}

	// Filter out containers with status "Removing"
	var filteredContainers []container.Summary
	for _, c := range containers {
		// Skip containers that are in the process of being removed
		if c.Status != "Removing" {
			filteredContainers = append(filteredContainers, c)
		}
	}

	if len(filteredContainers) == 0 {
		return nil, &ContainerNotFoundError{Labels: labels}
	}

	container := c.summaryToContainer(filteredContainers[0])
	return &container, nil
}

func (c *Client) ListContainers(ctx context.Context, labels []string) ([]Container, error) {
	containerSummaries, err := c.listContainers(ctx, labels)
	if err != nil {
		return nil, fmt.Errorf("error listing containers: %v", err)
	}

	containers := make([]Container, len(containerSummaries))
	for i, summary := range containerSummaries {
		containers[i] = c.summaryToContainer(summary)
	}

	return containers, nil
}

func (c *Client) listContainers(ctx context.Context, labels []string) ([]container.Summary, error) {
	// Create filters for the labels
	labelFilters := filters.NewArgs()
	for _, label := range labels {
		labelFilters.Add("label", label)
	}

	// List containers with the specified filters
	containerSummaries, err := c.client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: labelFilters,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing containers: %v", err)
	}

	return containerSummaries, nil
}

func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	timeout := int(30 * time.Second)
	return c.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	return c.client.ContainerRemove(ctx, containerID, container.RemoveOptions{RemoveVolumes: true, RemoveLinks: false, Force: true})
}

func (c *Client) InspectContainer(ctx context.Context, containerID string) (container.InspectResponse, error) {
	// TODO re-export InspectResponse type?
	return c.client.ContainerInspect(ctx, containerID)
}

func (c *Client) summaryToContainer(summary container.Summary) Container {
	return Container{
		ID:     summary.ID,
		State:  summary.State,
		client: c.client,
	}
}

func StopContainer(ctx context.Context, containerID string) error {
	cli, err := NewClient()
	if err != nil {
		return fmt.Errorf("error creating container client: %v", err)
	}
	defer cli.Close()

	return cli.StopContainer(ctx, containerID)
}

func RemoveContainer(ctx context.Context, containerID string) error {
	cli, err := NewClient()
	if err != nil {
		return fmt.Errorf("error creating container client: %v", err)
	}
	defer cli.Close()

	return cli.RemoveContainer(ctx, containerID)
}
