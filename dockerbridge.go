package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"

	"go-mcp-usa/mcp"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
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

func processContainerOutput(reader io.Reader, stdoutWriter io.Writer, stderrWriter io.Writer, doneCh chan error) {
	// Create buffers for stdout and stderr
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	// Use stdcopy to handle the Docker multiplexing
	fmt.Println("test")
	// _, err := stdcopy.StdCopy(stdoutWriter, stderrWriter, reader)
	_, err := stdcopy.StdCopy(stdoutBuf, stderrBuf, reader)
	fmt.Println("test2")
	if err != nil {
		doneCh <- fmt.Errorf("error processing container output: %v", err)
		return
	}

	// Process stdout for JSON messages
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		// Try to parse as JSON
		var jsonData interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err != nil {
			// If not valid JSON, print the raw line
			fmt.Println(line)
		} else {
			// Format the JSON with indentation
			prettyJSON, err := json.MarshalIndent(jsonData, "", "    ")
			if err != nil {
				fmt.Println(line)
			} else {
				fmt.Println(string(prettyJSON))
			}
		}
	}

	// Check if there's any stderr content to print
	if stderrBuf.Len() > 0 {
		fmt.Fprintf(os.Stderr, "Container stderr: %s\n", stderrBuf.String())
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
	go processContainerOutput(waiter.Reader, os.Stdout, os.Stderr, outputDone)
	// go func() {
	// 	// Use StdCopy to demultiplex the output stream
	// 	_, err := stdcopy.StdCopy(os.Stdout, os.Stderr, waiter.Reader)
	// 	outputDone <- err
	// }()
	// Create initialization message
	initMessage := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "0.1.0", // Note: "protocolVersion" not "version"
			"clientInfo": map[string]interface{}{
				"name":    "bach",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{
				"tools":     true,
				"prompts":   false,
				"resources": true,
			},
		},
	}

	// Marshal to JSON and add newline
	initMessageJSON, err := json.Marshal(initMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal initialization message: %v", err)
	}
	initMessageJSON = append(initMessageJSON, '\n')

	// Send the message to the container's stdin
	if _, err := waiter.Conn.Write(initMessageJSON); err != nil {
		return fmt.Errorf("failed to send initialization message: %v", err)
	}

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

func printTelemetry[T any](content T) {
	telem, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		fmt.Printf("Cannot print telemetry: %w", err)
	} else {
		fmt.Println(string(telem))
	}
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

	printTelemetry(containers)
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

	printTelemetry(images)
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
