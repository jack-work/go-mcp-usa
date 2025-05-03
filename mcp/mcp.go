package mcp

import (
	"fmt"
	"go-mcp-usa/jsonrpc"
	"go-mcp-usa/logging"

	"github.com/mitchellh/mapstructure"
)

type Server interface {
	GetEnv() *[]string
}

type Client struct {
	TargetServer        Server
	Tools               []Tool
	jsonrpc.StdioClient `json:"-"`
}

// executes mcp handshake and initializes tools
func (client *Client) Initialize() error {
	res1, err := client.SendMessage("initialize", InitializeRequestParams{
		ProtocolVersion: "0.1.0",
		ClientInfo: Implementation{
			Name:    "figaro",
			Version: "1.0.0",
		},
		Capabilities: ClientCapabilities{},
	})
	if err != nil {
		fmt.Println(err)
		return err
	}

	logging.PrintTelemetry(res1)
	err = client.Notify("notifications/initialized", InitializedNotification{})
	if err != nil {
		return err
	}

	// need to impl pagination
	untypedToolsResponse, err := client.SendActionMessage("tools/list")
	if err != nil {
		return err
	}

	var toolsResult ListToolsResult
	err = mapstructure.Decode(untypedToolsResponse.Result, &toolsResult)
	if err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	client.Tools = toolsResult.Tools
	return nil
}
