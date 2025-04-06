package container

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/term"
)

type ContainerConfig struct {
	Image       string
	Command     []string
	Interactive bool
	Binds       []string
}

type Container struct {
	ID     string
	Config ContainerConfig
	client *client.Client
}

func (c *Container) CreateFile(ctx context.Context, path string, content []byte) error {
	var copyContent bytes.Buffer
	tarWriter := tar.NewWriter(&copyContent)
	defer tarWriter.Close()

	header := &tar.Header{
		Name: filepath.Base(path),
		Mode: 0644,
		Size: int64(len(content)),
	}

	// Write the header to the tar
	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("error writing tar header: %v", err)
	}

	// Write the JSON content to the tar
	if _, err := tarWriter.Write(content); err != nil {
		return fmt.Errorf("error writing config JSON to tar: %v", err)
	}
	tarWriter.Close()

	contentReader := bytes.NewReader(copyContent.Bytes())

	// Create a tar archive for copying into the container
	err := c.client.CopyToContainer(ctx, c.ID, filepath.Dir(path), contentReader, container.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	if err != nil {
		return fmt.Errorf("error copying config file to container: %v", err)
	}
	return nil
}

func (c *Container) AttachAndRun(ctx context.Context, command []string) error {
	// Set up terminal raw mode to properly handle control sequences
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("unable to set terminal to raw mode: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	out, err := c.client.ContainerAttach(ctx, resp.ID, container.AttachOptions{
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
	if err := c.client.ContainerStart(ctx, c.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("error starting container: %v", err)
	}

	// TODO this is probably not strcitly necessary, or can at least fail silently
	// defer func() {
	// 	if err := cli.ContainerStop(ctx, resp.ID, container.StopOptions{}); err != nil {
	// 		log.Printf("Warning: failed to stop container: %v", err)
	// 	}
	// }()

	waitC, errC := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errC:
		if err != nil {
			return fmt.Errorf("error waiting for container: %v", err)
		}
	case <-waitC:
		// Container is not running anymore
	}

	// Give a small amount of time for final I/O operations to complete
	time.Sleep(100 * time.Millisecond)

	return nil
}
