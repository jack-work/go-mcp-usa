package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"go-mcp-usa/jsonrpc"
	"go-mcp-usa/logging"
	"go-mcp-usa/mcp"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type DockerServer struct {
	mcp.Server
	ImageName     *string // used if container must be created
	ContainerName *string // if not specified and ImageName is specified, a new container will be created with a default name
}

func (server *DockerServer) Setup() error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("Docker client could not be created: %w", err)
	}
	defer cli.Close()

	id, err := getOrCreateContainer(ctx, cli, *server)
	if err != nil {
		return err
	}

	fmt.Println(id)
	err = attachToContainer(ctx, cli, *id)
	return err
}

func processContainerOutput(reader io.Reader, responseChan chan jsonrpc.Message, doneCh chan error) {
	// Process stdout for JSON messages
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		// Remove any non-printable characters at the beginning
		var cleanLine string
		if idx := strings.Index(line, "{"); idx != -1 {
			cleanLine = line[idx:]
		} else {
			cleanLine = line
		}

		// Try to parse as JSON
		var response jsonrpc.Message
		if err := json.Unmarshal([]byte(cleanLine), &response); err != nil {
			fmt.Println(err)
			doneCh <- scanner.Err()
		} else {
			responseChan <- response
		}
	}

	doneCh <- scanner.Err()
}

func attachToContainer(ctx context.Context, cli *client.Client, id string) error {
	// Wait for the container to finis
	// Attach to the container
	waiter, err := cli.ContainerAttach(ctx, id, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return err
	}
	defer waiter.Close()

	// Set up a goroutine to handle container output
	outputDone := make(chan error)
	responseChan := make(chan jsonrpc.Message)

	go processContainerOutput(waiter.Reader, responseChan, outputDone)
	go jsonrpc.ReceiveMessages(responseChan)

	message := jsonrpc.Message{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: jsonrpc.InitializeParams{
			ProtocolVersion: "0.1.0",
			ClientInfo: jsonrpc.ClientInfo{
				Name:    "bach",
				Version: "1.0.0",
			},
			Capabilities: jsonrpc.ClientCapabilities{
				Tools:     true,
				Prompts:   false,
				Resources: true,
			},
		},
	}

	jsonrpc.SendMessage(message, waiter.Conn)
	jsonrpc.SendMessage(jsonrpc.Message{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "request",
		Params:  map[string]string{},
	}, waiter.Conn)

	// Wait for either the output processing to complete or context cancellation
	select {
	case err := <-outputDone:
		if err != nil {
			return fmt.Errorf("error processing container output: %v", err)
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func getOrCreateContainer(ctx context.Context, cli *client.Client, server DockerServer) (id *string, err error) {
	var name string
	var isRunning bool
	if serverName := server.ContainerName; serverName != nil {
		id, isRunning, err = getContainerByName(ctx, cli, *serverName)
		name = *server.ContainerName
	} else if imgName := server.ImageName; imgName != nil {
		id, name, isRunning, err = getContainerFromImage(ctx, cli, *imgName, *server.Env)
	}

	if isRunning {
		fmt.Printf("🏹 Running container found with ID: %s\n", *id)
		return id, err
	}

	if id == nil || err != nil {
		return id, err
	}

	if err := cli.ContainerStart(ctx, *id, container.StartOptions{}); err != nil {
		return id, err
	}

	fmt.Printf("🏹 Container started with Name: %s and ID: %s\n", name, *id)
	return id, err
}

func getContainerByName(ctx context.Context, cli *client.Client, name string) (*string, bool, error) {
	return getContainer(ctx, cli, filters.KeyValuePair{
		Key:   "name",
		Value: name,
	})
}

func getContainer(ctx context.Context, cli *client.Client, args filters.KeyValuePair) (*string, bool, error) {
	filters := filters.NewArgs(args)

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		Filters: filters,
	})

	lenContainers := len(containers)
	if lenContainers == 0 {
		return nil, false, err
	} else if lenContainers > 1 {
		fmt.Println("Multiple containers found:")
	}

	logging.PrintTelemetry(containers)
	container := &containers[0]
	return &container.ID, container.State == "running", err
}

// returns container id and whether it is running or false and error
func getContainerFromImage(
	ctx context.Context,
	cli *client.Client,
	imageName string,
	env []string,
) (
	id *string,
	name string,
	isRunning bool,
	err error,
) {
	name = formatContainerName(imageName)

	// If there is already a container with the expected name then we use it
	containerId, isRunning, err := getContainerByName(ctx, cli, name)
	if err != nil || containerId != nil {
		return containerId, name, isRunning, err
	}

	// Check if image exists by filtering on reference
	images, err := getImages(ctx, imageName, cli)
	if err != nil {
		return nil, name, false, err
	}

	if len(images) == 0 {
		return nil, name, false, fmt.Errorf("No docker images found")
	}

	fmt.Printf("📜 Image %s found locally\n", imageName)

	logging.PrintTelemetry(images)
	if len(images) == 0 {
		return nil, name, false, fmt.Errorf("No images found with the provided name: %s", imageName)
	}

	variables, err := getEnvironmentVariables(env)
	if err != nil {
		return nil, name, false, err
	}

	// Replace all matches with empty string
	resp, err := cli.ContainerCreate(
		ctx,
		&container.Config{
			Image:        imageName,
			Env:          variables,
			AttachStdin:  true,
			OpenStdin:    true,
			StdinOnce:    false,
			AttachStdout: true,
			AttachStderr: true,
			Tty:          false,
		},
		&container.HostConfig{
			AutoRemove: true,
		},
		nil,
		nil,
		name,
	)

	id = &resp.ID

	return id, name, false, err
}

// Pattern matches a letter/digit followed by a letter/digit/underscore/dot/hyphen
func formatContainerName(imageName string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_.-]`)
	return re.ReplaceAllString(fmt.Sprintf("mcp-%s", imageName), ".")
}

// Tries to get the image of the provided image name from the provided client.
// this effectively implements write behind cache style interface using docker by attempting to
// pull the container if it cannot be found locally, thereby caching it.
func getImages(ctx context.Context, imageName string, cli *client.Client) ([]image.Summary, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("reference", imageName)

	images, err := cli.ImageList(ctx, image.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("Could not list Docker images: %w", err)
	}

	// If image doesn't exist locally, try to pull it
	if len(images) == 0 {
		images, err = tryPullImage(ctx, imageName, cli, filterArgs)
	}

	return images, err
}

func tryPullImage(ctx context.Context, imageName string, cli *client.Client, filterArgs filters.Args) ([]image.Summary, error) {
	fmt.Printf("Image %s not found locally. Pulling...\n", imageName)
	reader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf(`Failed to pull image "%s". Error: %w`, imageName, err)
	}

	io.Copy(os.Stdout, reader) // print output of pull
	reader.Close()

	return cli.ImageList(ctx, image.ListOptions{
		Filters: filterArgs,
	})
}

func getEnvironmentVariables(variableNames []string) ([]string, error) {
	retVal := make([]string, len(variableNames))
	for i, name := range variableNames {
		if env := os.Getenv(name); env != "" {
			fmt.Println(fmt.Sprintf("%s=%s", name, env))
			retVal[i] = fmt.Sprintf("%s=%s", name, env)
		}
	}
	return retVal, nil
}
