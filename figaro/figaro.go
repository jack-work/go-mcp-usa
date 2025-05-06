package figaro

import (
	"context"
	"encoding/json"
	"figaro/anthropicbridge"
	"figaro/dockerbridge"
	"figaro/jsonrpc"
	"figaro/logging"
	"figaro/mcp"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"go.opentelemetry.io/otel/trace"
)

const FigaroChi = "figaro"

type Figaro struct {
	Clients        []mcp.Client
	toolsCache     []mcp.Tool // Might get stale when we implement dynamic tool introduction
	tracerProvider trace.TracerProvider
}

type ServerRegistry struct {
	DockerServers []dockerbridge.ContainerDefinition `json:"docker_servers"`
}

// Initializes an instance of a Figaro application configured with the provided server list, and returns it.
// Iff error, return val will be nil
// Otherwise, it will always have a non-nil value, even if empty list.
// If server does not return any tools by responding with nil tools in result rather than empty list, that's fine,
// it's interpreted to mean empty list for interest of compatibility.
func SummonFigaro(ctx context.Context, tp trace.TracerProvider, servers ServerRegistry) (*Figaro, error) {
	mcpClients := make([]mcp.Client, len(servers.DockerServers))
	for i, server := range servers.DockerServers {
		// parent context for each pair
		serviceContext := context.WithoutCancel(ctx)

		// child context for connection
		connCtx, cancelConn := context.WithCancel(serviceContext)
		connection, connectionDone, err := dockerbridge.Setup(connCtx, server, tp)
		if err != nil {
			cancelConn()
			return nil, err
		}

		// child context for client
		rpcCtx, cancelRpc := context.WithCancel(serviceContext)
		client, rpcDone, err := jsonrpc.NewStdioClient[string](rpcCtx, connection)
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
			RpcDone:          rpcDone,
			CancelRpc:        cancelRpc,
			TracerProvider:   tp,
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

	return &Figaro{
		Clients:        mcpClients,
		tracerProvider: tp,
	}, nil
}

func (figaro *Figaro) GetClientForTool(toolName string) *mcp.Client {
	for _, client := range figaro.Clients {
		for _, tool := range client.Tools {
			if tool.Name == toolName {
				return &client
			}
		}
	}
	return nil
}

func (figaro *Figaro) GetAllTools() []mcp.Tool {
	if figaro.toolsCache != nil {
		return figaro.toolsCache
	}
	cumToolSize := 0
	for _, client := range figaro.Clients {
		cumToolSize += len(client.Tools)
	}
	result := make([]mcp.Tool, cumToolSize)
	for i, client := range figaro.Clients {
		for j, tool := range client.Tools {
			result[i+j] = tool
		}
	}
	figaro.toolsCache = result
	return result
}

func (figaro *Figaro) Request(args []string, modePtr *string) error {
	tools := figaro.GetAllTools()
	input := strings.Join(args, " ")

	logging.EzPrint(tools)
	role := anthropic.MessageParamRole(string(anthropic.MessageParamRoleUser))

	conversation := make([]anthropic.MessageParam, 0, 1)
	for range 1 {
		conversation = append(conversation, anthropic.MessageParam{
			Content: []anthropic.ContentBlockParamUnion{{
				OfRequestTextBlock: &anthropic.TextBlockParam{Text: input},
			}},
			Role: role,
		})
		messageParams := GetMessageNewParams(conversation, tools)
		message, err := anthropicbridge.StreamMessage2(*messageParams, context.Background(), func(test string) error {
			fmt.Print(test)
			return nil
		})

		modelResponse := make([]anthropic.ContentBlockParamUnion, 0, len(message.Content))
		for _, message := range message.Content {
			modelResponse = append(modelResponse, message.ToParam())
		}
		conversation = append(conversation, anthropic.MessageParam{
			Content: modelResponse,
			Role:    anthropic.MessageParamRoleAssistant,
		})

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return err
		}
		if message.StopReason == "tool_use" {
			toolResponses, err := callTools(message, figaro)
			if err != nil {
				return err
			}
			toolResults := make([]anthropic.ContentBlockParamUnion, 0)
			for id, toolResponse := range toolResponses {
				toolResults = append(toolResults, anthropic.ContentBlockParamUnion{
					OfRequestToolResultBlock: &anthropic.ToolResultBlockParam{
						ToolUseID: id,
						Content: []anthropic.ToolResultBlockParamContentUnion{{
							OfRequestTextBlock: &anthropic.TextBlockParam{
								Text: anyToString(toolResponse.Result),
							},
						}},
					},
				})
			}
			conversation = append(conversation, anthropic.MessageParam{
				Content: toolResults,
				Role:    anthropic.MessageParamRoleUser,
			})

			messageParams = GetMessageNewParams(conversation, tools)
			message, err := anthropicbridge.StreamMessage2(*messageParams, context.Background(), func(test string) error {
				fmt.Print(test)
				return nil
			})

			writeHostFile(conversation, ".conversation.json")
			logging.EzPrint(message)
		}
	}

	return nil
}

func writeHostFile(contents any, path ...string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	relative := filepath.Join(path...)
	filePath := filepath.Join(homeDir, relative)
	byteContents, err := json.Marshal(contents)
	err = os.WriteFile(filePath, byteContents, os.FileMode(os.O_TRUNC))
	if err != nil {
		return err
	}

	return nil
}

func GetMessageNewParams(conversation []anthropic.MessageParam, tools []mcp.Tool) *anthropic.MessageNewParams {
	anthropicTools := anthropicbridge.GetAnthropicTools(tools)
	messageParams := &anthropic.MessageNewParams{
		MaxTokens: 1024,
		Messages:  conversation,
		Model:     anthropic.ModelClaude3_7SonnetLatest,
		Tools:     anthropicTools,
	}
	return messageParams
}

func callTools(message *anthropic.Message, figaro *Figaro) (map[string]jsonrpc.Message[any], error) {
	tools := make(map[string]jsonrpc.Message[any], len(message.Content))
	for _, block := range message.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.ToolUseBlock:
			client := figaro.GetClientForTool(variant.Name)
			if client == nil {
				return nil, fmt.Errorf("Could not find mcp client for %v", variant.Name)
			}
			var args map[string]any
			err := json.Unmarshal(variant.Input, &args)
			if err != nil {
				return nil, err
			}
			response, err := client.SendMessage("tools/call", mcp.CallToolRequestParams{
				Name:      variant.Name,
				Arguments: args,
			})
			if err != nil {
				return nil, err
			}
			logging.EzPrint(response)
			tools[variant.ID] = *response
		}
	}
	return tools, nil
}

// Converting to string
func anyToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case map[string]any, []any:
		bytes, _ := json.Marshal(val)
		return string(bytes)
	default:
		return fmt.Sprintf("%v", val)
	}
}
