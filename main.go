package main

// As Theodore Roosevelt proclaimed, we shall "speak softly and carry a big stack"

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go-mcp-usa/jsonrpc"
	"go-mcp-usa/logging"
	"go-mcp-usa/mcp"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

type Figaro struct {
	Clients []mcp.Client
	// Might get stale when we implement dynamic tool introduction
	toolsCache []mcp.Tool
}

func (figaro Figaro) GetClientForTool(toolName string) *mcp.Client {
	for _, client := range figaro.Clients {
		for _, tool := range client.Tools {
			if tool.Name == toolName {
				return &client
			}
		}
	}
	return nil
}

func (figaro Figaro) GetAllTools() []mcp.Tool {
	logging.PrintTelemetry(figaro.Clients[0])
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

type ServerRegistry struct {
	DockerServers []DockerServer `json:"docker_servers"`
}

type DockerServer struct {
	ID  *string   `json:"id"`
	Env *[]string `json:"env"`
	// used if container must be created
	ImageName *string `json:"image_name"`
	// if not specified and ImageName is specified, a new container will be created with a default name
	ContainerName *string `json:"container_name"`
}

func (s DockerServer) GetEnv() *[]string {
	return s.Env
}

func main() {
	// Define flag with default value "default_value"
	modePtr := flag.String("m", "ModelClaude3_7SonnetLatest", "Specify the model to use")

	// Parse flags
	flag.Parse()

	// init MCP
	servers, err := GetServers()
	if err != nil {
		logging.PrintTelemetry(err)
	}
	figaro, err := SummonFigaro(*servers)
	if err != nil {
		logging.PrintTelemetry(err)
		return
	}

	// Use the flag value
	args := flag.Args()
	if len(args) > 0 {
		OneShotAnswer(args, modePtr, figaro)
		return
	}

	// short buffer to be manually authored and compacted
	thesis := make([]Message, 0, 6)
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	ctx := context.Background()
	for scanner.Scan() {
		userPrompt := scanner.Text()

		// add / or tab or . options here; e.g. .help, .session, .agent, or / of the same.

		message := NewMessage(userPrompt, string(anthropic.MessageParamRoleUser))
		thesis = append(thesis, message)

		answer, err := StreamMessage(thesis, ctx, func(test string) error {
			fmt.Printf("%s", test)
			return nil
		}, figaro.GetAllTools())

		if err != nil {
			logging.PrintTelemetry(err.Error())
			return
		}

		thesis = append(thesis, answer)
		fmt.Println()
		fmt.Print("> ")
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
		logging.PrintTelemetry(err)
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
func SummonFigaro(servers ServerRegistry) (*Figaro, error) {
	mcpClients := make([]mcp.Client, len(servers.DockerServers))
	for i, server := range servers.DockerServers {
		genericClient, err := server.Setup()
		if err != nil {
			return nil, err
		}
		client, err := jsonrpc.NewClient[string](genericClient)
		if err != nil {
			return nil, err
		}
		mcpClient := mcp.Client{
			TargetServer: server,
			StdioClient:  *client,
		}
		err = mcpClient.Initialize()
		if err != nil {
			return nil, err
		}
		mcpClients[i] = mcpClient
	}

	return &Figaro{
		Clients: mcpClients,
	}, nil
}

func OneShotAnswer(args []string, modePtr *string, figaro *Figaro) error {
	tools := figaro.GetAllTools()
	input := strings.Join(args, " ")
	fmt.Printf("Model: %s\n\n", *modePtr)
	fmt.Printf("Input: %s\n\n", input)
	message := NewMessage(input, string(anthropic.MessageParamRoleUser))
	logging.PrintTelemetry(tools)
	logging.PrintTelemetry("yup")
	role := anthropic.MessageParamRole(string(anthropic.MessageParamRoleUser))

	conversation := make([]anthropic.MessageParam, 0, 1)
	for range 1 {
		conversation = append(conversation, anthropic.MessageParam{
			Content: []anthropic.ContentBlockParamUnion{{
				OfRequestTextBlock: &anthropic.TextBlockParam{Text: message.GetContent()},
			}},
			Role: role,
		})
		messageParams := GetMessageNewParams(conversation, tools)
		message, err := StreamMessage2(*messageParams, context.Background(), func(test string) error {
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
			for _, toolResponse := range toolResponses {
				toolResults = append(toolResults, anthropic.ContentBlockParamUnion{
					OfRequestToolResultBlock: &anthropic.ToolResultBlockParam{
						ToolUseID: toolResponse.ID,
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
			logging.PrintTelemetry(conversation)
		}
	}

	return nil
}

func GetMessageNewParams(conversation []anthropic.MessageParam, tools []mcp.Tool) *anthropic.MessageNewParams {
	anthropicTools := GetAnthropicTools(tools)
	messageParams := &anthropic.MessageNewParams{
		MaxTokens: 1024,
		Messages:  conversation,
		Model:     anthropic.ModelClaude3_7SonnetLatest,
		Tools:     anthropicTools,
	}
	return messageParams
}

func callTools(message *anthropic.Message, figaro *Figaro) ([]jsonrpc.Message[any], error) {
	tools := make([]jsonrpc.Message[any], 0, len(message.Content))
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
			logging.PrintTelemetry(response)
			tools = append(tools, *response)
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
