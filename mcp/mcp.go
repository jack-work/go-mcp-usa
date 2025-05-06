package mcp

import (
	"context"
	"figaro/jsonrpc"
	"figaro/logging"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Server interface {
	GetEnv() *[]string
}

type Client struct {
	TargetServer        Server
	Tools               []Tool
	jsonrpc.StdioClient `json:"-"`
	ConnectionDone      <-chan error
	CancelConnection    context.CancelFunc
	RpcDone             <-chan error
	CancelRpc           context.CancelFunc
	TracerProvider      trace.TracerProvider
}

// executes mcp handshake and initializes tools
func (client *Client) Initialize(ctx context.Context) error {
	tracer := client.TracerProvider.Tracer("mcp.Initialize")
	ctx, span := tracer.Start(ctx, "mcp.Initialize")
	defer span.End()

	res1, err := client.SendMessage(ctx,
		"initialize", InitializeRequestParams{
			ProtocolVersion: "0.1.0",
			ClientInfo: Implementation{
				Name:    "figaro",
				Version: "1.0.0",
			},
			Capabilities: ClientCapabilities{},
		})
	if err != nil {
		span.AddEvent("Error when calling initialize", trace.WithStackTrace(true))
		return err
	}

	logging.EzPrint(res1)
	span.AddEvent("Initialize response", trace.WithAttributes(
		attribute.String("res1", logging.EzMarshal(res1))))

	err = client.Notify(ctx, "notifications/initialized", InitializedNotification{})
	if err != nil {
		return err
	}

	// need to impl pagination
	untypedToolsResponse, err := client.SendActionMessage(ctx, "tools/list")
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
