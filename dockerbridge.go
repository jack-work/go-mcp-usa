package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"go-mcp-usa/mcp"
)

type telemetry struct {
	image.Summary
	IsSelected bool
}

func RegisterServer(server mcp.Server) error {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("Docker client could not be created: %w", err)
	}
	defer cli.Close()

	imageName := *server.ImageName
	// Check if image exists by filtering on reference
	images, err := getImages(ctx, imageName, cli)
	if err != nil {
		return err
	}

	if len(images) == 0 {
		return fmt.Errorf("No docker images found")
	}

	fmt.Printf("📜 Image %s found locally\n", imageName)

	err = logTelemetry(images)
	if err != nil {
		return err
	}

	if len(images) == 0 {
		return fmt.Errorf("No images found with the provided name: %s", imageName)
	}

	// We always select the first image...for now
	image := images[0]

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: image.ID,
	}, nil, nil, nil, "")
	if err != nil {
		return err
	}

	fmt.Println(resp)
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}
	fmt.Println("don")

	fmt.Printf("🏹 Container started with ID: %s\n", resp.ID)
	return nil
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

func logTelemetry(images []image.Summary) error {
	telem := make([]telemetry, len(images))
	for i, img := range images {
		telem[i] = telemetry{
			Summary:    img,
			IsSelected: i == 0,
		}
	}
	result, err := json.MarshalIndent(telem, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(result))
	return nil
}
