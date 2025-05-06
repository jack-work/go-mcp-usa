package dockerbridge

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"

	"figaro/jsonrpc"
	"figaro/logging"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ContainerDefinition struct {
	ID  *string   `json:"id"`
	Env *[]string `json:"env"`
	// used if container must be created
	ImageName *string `json:"image_name"`
	// if not specified and ImageName is specified, a new container will be created with a default name
	ContainerName *string `json:"container_name"`
}
type Container struct {
	ContainerDefinition
	Tracer trace.Tracer
}

func (s ContainerDefinition) GetEnv() *[]string {
	return s.Env
}

// Creates a json rpc connection object to the provided container definition
// TODO: Attach lifecycle management to the docker container if possible.  I would at least like a channel when it goes offline.
func Setup(ctx context.Context, def ContainerDefinition, tp trace.TracerProvider) (*jsonrpc.Connection, <-chan (error), error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(ctx.Err())

	tracer := tp.Tracer("figaro/dockerbridge")
	ctx, span := tracer.Start(ctx, "dockerbridge.Setup")
	defer span.End()

	span.AddEvent("Starting docker container initialization")

	server := Container{
		ContainerDefinition: def,
		Tracer:              tracer,
	}

	cli, err := client.NewClientWithOpts(client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithTraceProvider(tp))

	if err != nil {
		cli.Close()
		cancel(err)
		return nil, nil, err
	}

	id, err := server.getOrCreateContainer(ctx, cli)
	if err != nil {
		cli.Close()
		cancel(err)
		return nil, nil, err
	}

	// Wait for the container to finish
	// Attach to the container
	waiter, err := cli.ContainerAttach(ctx, *id, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	})

	if err != nil {
		cli.Close()
		cancel(err)
		return nil, nil, err
	}

	// Set up a goroutine to handle container output
	outputDone := make(chan error)
	go func() {
		<-ctx.Done()
		cli.Close()
		outputDone <- ctx.Err()
	}()

	return &jsonrpc.Connection{
		Conn:   waiter.Conn,
		Reader: waiter.Reader,
	}, outputDone, nil
}

// TODO:
//
//	func monitorContainer(ctx context.Context, cli *client.Client, containerID string) {
//	    // Create a channel to signal when container stops
//	    containerStopped := make(chan struct{})
//
//	    // Listen for Docker events
//	    messages, errs := cli.Events(ctx, types.EventsOptions{})
//
//	    // Monitor events in a goroutine
//	    go func() {
//	        for {
//	            select {
//	            case err := <-errs:
//	                if err != nil {
//	                    log.Printf("Error receiving events: %v", err)
//	                    return
//	                }
//	            case msg := <-messages:
//	                // Check if this is our container
//	                if msg.Actor.ID == containerID {
//	                    // Check for container stop/die events
//	                    if msg.Action == "die" || msg.Action == "stop" {
//	                        log.Printf("Container %s has stopped", containerID)
//	                        close(containerStopped)
//	                        return
//	                    }
//	                }
//	            case <-ctx.Done():
//	                return
//	            }
//	        }
//	    }()
//
//	    // Wait for container to stop
//	    <-containerStopped
//	    log.Println("Container stopped, performing cleanup...")
//	}

func (ctr *Container) getOrCreateContainer(ctx context.Context, cli *client.Client) (id *string, err error) {
	ctx, span := ctr.Tracer.Start(ctx, "dockerbridge.getOrCreateContainer")
	defer span.End()

	var name string
	var isRunning bool
	if serverName := ctr.ContainerName; serverName != nil {
		id, isRunning, err = getContainerByName(ctx, cli, *serverName, ctr.Tracer)
		name = *ctr.ContainerName
	} else if imgName := ctr.ImageName; imgName != nil {
		id, name, isRunning, err = getContainerFromImage(ctx, cli, *imgName, *ctr.Env, ctr.Tracer)
	}

	if isRunning {
		span.AddEvent(fmt.Sprintf("Running container found with ID: %s\n", *id))
		return id, err
	}

	if id == nil || err != nil {
		return id, err
	}

	if err := cli.ContainerStart(ctx, *id, container.StartOptions{}); err != nil {
		return id, err
	}

	span.AddEvent(fmt.Sprintf("Container started with Name: %s and ID: %s\n", name, *id))
	return id, err
}

func getContainerByName(ctx context.Context, cli *client.Client, name string, tracer trace.Tracer) (*string, bool, error) {
	return getContainer(ctx, cli, filters.KeyValuePair{
		Key:   "name",
		Value: name,
	}, tracer)
}

func getContainer(ctx context.Context, cli *client.Client, args filters.KeyValuePair, tracer trace.Tracer) (*string, bool, error) {
	ctx, span := tracer.Start(ctx, "dockerbridge.getContainer")
	defer span.End()

	filters := filters.NewArgs(args)

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		Filters: filters,
	})

	lenContainers := len(containers)
	if lenContainers == 0 {
		return nil, false, err
	} else if lenContainers > 1 {
		span.AddEvent(fmt.Sprintf("Multiple containers found: %v.  Using first", lenContainers))
	}

	container := &containers[0]

	span.AddEvent("Container located", trace.WithAttributes(
		attribute.String("container_id", container.ID),
		attribute.String("labels", container.Image)))

	return &container.ID, container.State == "running", err
}

// returns container id and whether it is running or false and error
func getContainerFromImage(
	ctx context.Context,
	cli *client.Client,
	imageName string,
	env []string,
	tracer trace.Tracer,
) (
	id *string,
	name string,
	isRunning bool,
	err error,
) {
	ctx, span := tracer.Start(ctx, "dockerbridge.getContainerFromImage")
	defer span.End()

	name = formatContainerName(imageName)

	// If there is already a container with the expected name then we use it
	containerId, isRunning, err := getContainerByName(ctx, cli, name, tracer)
	if err != nil || containerId != nil {
		return containerId, name, isRunning, err
	}

	// Check if image exists by filtering on reference
	images, err := getImages(ctx, imageName, cli, tracer)
	if err != nil {
		return nil, name, false, err
	}

	if len(images) == 0 {
		return nil, name, false, fmt.Errorf("No docker images found")
	}

	span.AddEvent(fmt.Sprintf("📜 Image %s found locally\n", imageName),
		trace.WithAttributes(attribute.Int("image_count", len(images))))

	logging.EzPrint(images)
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
func getImages(ctx context.Context, imageName string, cli *client.Client, tracer trace.Tracer) ([]image.Summary, error) {
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
		images, err = tryPullImage(ctx, imageName, cli, filterArgs, tracer)
	}

	return images, err
}

func tryPullImage(ctx context.Context, imageName string, cli *client.Client, filterArgs filters.Args, tracer trace.Tracer) ([]image.Summary, error) {
	ctx, span := tracer.Start(ctx, "dockerbridge.tryPullImage")
	defer span.End()

	span.AddEvent(fmt.Sprintf("Image %s not found locally. Pulling...\n", imageName))
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
