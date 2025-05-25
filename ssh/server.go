package ssh

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/crypto/ssh"
)

const (
	hostKeyPath = "hostkey"
	sshUser     = "dev"
	sshPassword = "dev"
	sshPort     = "2222"
	containerID = "602fab1d59b5"
)

func Start() {
	// Generate or load SSH host key
	hostKey, err := generateOrLoadHostKey(hostKeyPath)
	if err != nil {
		log.Fatalf("Failed to load host key: %v", err)
	}

	// SSH server configuration
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == sshUser && string(pass) == sshPassword {
				return nil, nil
			}
			return nil, fmt.Errorf("authentication failed")
		},
	}
	config.AddHostKey(hostKey)

	// Start SSH server
	listener, err := net.Listen("tcp", ":"+sshPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", sshPort, err)
	}
	defer listener.Close()

	log.Printf("SSH server listening on port %s", sshPort)
	log.Printf("Connect with: ssh %s@localhost -p %s", sshUser, sshPort)

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleConnection(conn, config)
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake: %v", err)
		return
	}
	defer sshConn.Close()

	log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

	// Handle global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for ch := range chans {
		if ch.ChannelType() != "session" {
			ch.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := ch.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
			continue
		}

		go handleChannel(channel, requests)
	}
}

func handleChannel(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	// Create Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Printf("Failed to create Docker client: %v", err)
		return
	}
	defer dockerClient.Close()

	ctx := context.Background()
	var execID string
	var hijackedResp types.HijackedResponse

	for req := range requests {
		switch req.Type {
		case "pty-req":
			// Parse terminal dimensions
			termLen := req.Payload[3]
			termType := string(req.Payload[4 : 4+termLen])
			w, h := parseDims(req.Payload[4+termLen:])

			log.Printf("PTY requested: %s %dx%d", termType, w, h)

			// Create exec instance with PTY
			execConfig := container.ExecOptions{
				AttachStdin:  true,
				AttachStdout: true,
				AttachStderr: true,
				Tty:          true,
				Cmd:          []string{"/bin/sh"},
			}

			execResp, err := dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
			if err != nil {
				log.Printf("Failed to create exec: %v", err)
				req.Reply(false, nil)
				continue
			}
			execID = execResp.ID

			req.Reply(true, nil)

		case "shell":
			if execID == "" {
				// Create exec without PTY if PTY wasn't requested
				execConfig := container.ExecOptions{
					AttachStdin:  true,
					AttachStdout: true,
					AttachStderr: true,
					Tty:          false,
					Cmd:          []string{"/bin/sh"},
				}

				execResp, err := dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
				if err != nil {
					log.Printf("Failed to create exec: %v", err)
					req.Reply(false, nil)
					continue
				}
				execID = execResp.ID
			}

			// Start exec
			startConfig := container.ExecAttachOptions{
				Tty: true,
			}

			hijackedResp, err = dockerClient.ContainerExecAttach(ctx, execID, startConfig)
			if err != nil {
				log.Printf("Failed to attach to exec: %v", err)
				req.Reply(false, nil)
				continue
			}

			req.Reply(true, nil)

			// Start streaming
			go streamDockerToSSH(channel, &hijackedResp)
			go streamSSHToDocker(channel, &hijackedResp)

		case "window-change":
			// Handle terminal resize
			w, h := parseDims(req.Payload)
			err := dockerClient.ContainerExecResize(ctx, execID, container.ResizeOptions{
				Height: uint(h),
				Width:  uint(w),
			})
			if err != nil {
				log.Printf("Failed to resize: %v", err)
			}

		case "env":
			// Environment variables can be set here if needed
			req.Reply(true, nil)

		default:
			req.Reply(false, nil)
		}
	}
}

func streamDockerToSSH(channel ssh.Channel, hijacked *types.HijackedResponse) {
	defer hijacked.Close()

	// For TTY mode, copy directly. For non-TTY, use stdcopy to demultiplex
	_, err := io.Copy(channel, hijacked.Reader)
	if err != nil && err != io.EOF {
		log.Printf("Error streaming from Docker to SSH: %v", err)
	}
	channel.CloseWrite()
}

func streamSSHToDocker(channel ssh.Channel, hijacked *types.HijackedResponse) {
	_, err := io.Copy(hijacked.Conn, channel)
	if err != nil && err != io.EOF {
		log.Printf("Error streaming from SSH to Docker: %v", err)
	}
}

func parseDims(b []byte) (w, h int) {
	if len(b) < 8 {
		return 80, 24 // default dimensions
	}
	w = int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	h = int(b[4])<<24 | int(b[5])<<16 | int(b[6])<<8 | int(b[7])
	return
}

func generateOrLoadHostKey(path string) (ssh.Signer, error) {
	// Check if key exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Generate new key
		key, err := generateSSHKey()
		if err != nil {
			return nil, err
		}

		// Save key
		if err := os.WriteFile(path, key, 0600); err != nil {
			return nil, err
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		return signer, nil
	}

	// Load existing key
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return signer, nil
}

// TOOO get rid of this
func generateSSHKey() ([]byte, error) {
	// For production, use proper key generation
	// This is a simplified example
	privateKey := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBYK6n+HjQBzNpGKEpCcaI0eZOBUJQPNdH1Tj1C5DoazQAAAJgHvSLmB70i
5gAAAAtzc2gtZWQyNTUxOQAAACBYK6n+HjQBzNpGKEpCcaI0eZOBUJQPNdH1Tj1C5DoazQ
AAAEBRy4LAA7S7h0VJNZMvA7V4LdGWTQQJLAz7cH5wbrfAO1grqf4eNAHM2kYoSkJxojR5
k4FQlA810fVOPULkOhrNAAAAFHVzZXJAZG9ja2VyLXNzaC1wcm94eQ==
-----END OPENSSH PRIVATE KEY-----`
	return []byte(privateKey), nil
}
