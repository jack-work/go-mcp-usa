package main

// As Theodore Roosevelt proclaimed, we shall "speak softly and carry a big stack"

import (
	"context"
	"encoding/json"
	"figaro/docker"
	"figaro/figaro"
	"figaro/jsonrpc"
	"figaro/logging"
	"figaro/mcp"
	"flag"
	"os"
	"path/filepath"
)

type ServerRegistry struct {
	DockerServers []docker.DockerServer `json:"docker_servers"`
}

func main() {
	// _, err := logging.InitTracer()
	// Define flag with default value "default_value"
	modePtr := flag.String("m", "ModelClaude3_7SonnetLatest", "Specify the model to use")

	// Parse flags
	flag.Parse()

	// init MCP
	servers, err := GetServers()
	if err != nil {
		logging.PrintTelemetry(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	figaro, err := SummonFigaro(ctx, *servers)
	if err != nil {
		return
	}

	// Use the flag value
	args := flag.Args()
	if len(args) > 0 {
		figaro.Request(args, modePtr)
		return
	}
}

func GetServers() (*ServerRegistry, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(homeDir, ".figaro", "servers.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Unmarshal into struct and add the ID
	var config ServerRegistry
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Initializes an instance of a Figaro application configured with the provided server list, and returns it.
// Iff error, return val will be nil
// Otherwise, it will always have a non-nil value, even if empty list.
// If server does not return any tools by responding with nil tools in result rather than empty list, that's fine,
// it's interpreted to mean empty list for interest of compatibility.
func SummonFigaro(ctx context.Context, servers ServerRegistry) (*figaro.Figaro, error) {
	mcpClients := make([]mcp.Client, len(servers.DockerServers))
	for i, server := range servers.DockerServers {
		// parent context for each pair
		serviceContext := context.WithoutCancel(ctx)

		// child context for connection
		connCtx, cancelConn := context.WithCancel(serviceContext)
		connection, connectionDone, err := server.Setup(connCtx)
		if err != nil {
			cancelConn()
			return nil, err
		}

		// child context for client
		rpcCtx, cancelRpc := context.WithCancel(serviceContext)
		client, err := jsonrpc.NewStdioClient[string](rpcCtx, connection)
		if err != nil {
			cancelConn()
			cancelRpc()
			return nil, err
		}
		mcpClient := mcp.Client{
			TargetServer:     server,
			StdioClient:      *client,
			ConnectionDone:   connectionDone,
			CancelConnection: cancelConn,
			// RpcDone: ,
			CancelRpc: cancelRpc,
		}
		err = mcpClient.Initialize()
		if err != nil {
			cancelConn()
			cancelRpc()
			return nil, err
		}
		// add failure logging at this level
		mcpClients[i] = mcpClient
	}

	return &figaro.Figaro{
		Clients: mcpClients,
	}, nil
}
