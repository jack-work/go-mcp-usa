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
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/mitchellh/mapstructure"
)

type ServerRegistry interface {
	GetClient() mcp.Client
}

type ServerRegistryImpl struct {
	DockerServers []DockerServer `json:"docker_servers"`
}

type DockerServer struct {
	ID            *string   `json:"id"`
	Env           *[]string `json:"env"`
	ImageName     *string   // used if container must be created
	ContainerName *string   // if not specified and ImageName is specified, a new container will be created with a default name
}

func (s *DockerServer) GetEnv() *[]string {
	return s.Env
}

func main() {
	// Define flag with default value "default_value"
	modePtr := flag.String("m", "ModelClaude3_7SonnetLatest", "Specify the model to use")

	// Parse flags
	flag.Parse()

	// init MCP
	tools, err := InitMcp()
	if err != nil {
		logging.PrintTelemetry(err)
		return
	}
	logging.PrintTelemetry(tools)

	// Use the flag value
	args := flag.Args()
	if len(args) > 0 {
		OneShotAnswer(args, modePtr, tools)
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
		}, tools)

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
	// Read the JSON file
	data, err := os.ReadFile("~/.figaro/servers.json")
	if err != nil {
		logging.PrintTelemetry("found file")
		return nil, err
	}

	// Unmarshal into struct and add the ID
	var config ServerRegistry
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Get available tools
// Iff error, []mcp.Tool will be nil
// Otherwise, []mcp.Tool will always have a non-nil value, even if empty list.
// If server does not return any tools by responding with nil rather than empty list, that's fine,
// it's interpreted to mean empty list for interest of compatibility.
func InitMcp() ([]mcp.Tool, error) {
	servers, err := GetServers()
	if err != nil {
		logging.PrintTelemetry(err)
	}

	logging.PrintTelemetry(servers)
	// llmClient := jsonrpc.Client{
	// 	Servers: map[string]Server{
	// 		Brave.
	// 	}
	// }
	client, err := jsonrpc.NewClient[string](
		genericClient.Context,
		genericClient.Conn,
		genericClient.Reader,
		nil,
		genericClient.DoneChan)

	if err != nil {
		logging.PrintTelemetry(err)
		return nil, err
	}

	res1, err := client.SendMessage("initialize", mcp.InitializeRequestParams{
		ProtocolVersion: "0.1.0",
		ClientInfo: mcp.Implementation{
			Name:    "figaro",
			Version: "1.0.0",
		},
		Capabilities: mcp.ClientCapabilities{},
	})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	logging.PrintTelemetry(res1)
	err = client.Notify("notifications/initialized", mcp.InitializedNotification{})
	if err != nil {
		return nil, err
	}

	// need to impl pagination
	untypedToolsResponse, err := client.SendActionMessage("tools/list")
	if err != nil {
		return nil, err
	}

	var toolsResult mcp.ListToolsResult
	err = mapstructure.Decode(untypedToolsResponse.Result, &toolsResult)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return toolsResult.Tools, nil
}

func OneShotAnswer(args []string, modePtr *string, tools []mcp.Tool) {
	input := strings.Join(args, " ")
	fmt.Printf("Model: %s\n\n", *modePtr)
	fmt.Printf("Input: %s\n\n", input)
	message := NewMessage(input, string(anthropic.MessageParamRoleUser))
	anthropicTools := GetAnthropicTools(tools)
	role := anthropic.MessageParamRole(string(anthropic.MessageParamRoleUser))
	for range 1 {
		test := &anthropic.MessageNewParams{
			MaxTokens: 1024,
			Messages: []anthropic.MessageParam{{
				Content: []anthropic.ContentBlockParamUnion{{
					OfRequestTextBlock: &anthropic.TextBlockParam{Text: message.GetContent()},
				}},
				Role: role,
			}},
			Model: anthropic.ModelClaude3_7SonnetLatest,
			Tools: anthropicTools,
		}
		response, err := StreamMessage2(*test, context.Background(), func(test string) error {
			fmt.Print(test)
			return nil
		})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		switch response.Content.AsAny().(type) {

		}
	}
}
