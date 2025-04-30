package main

// As Theodore Roosevelt proclaimed, we shall "speak softly and carry a big stack"

import (
	"bufio"
	"context"
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
			fmt.Printf(test)
			return nil
		}, tools)

		if err != nil {
			fmt.Printf(err.Error())
			return
		}

		thesis = append(thesis, answer)
		fmt.Println()
		fmt.Print("> ")
	}
}

// Get available tools
// Iff error, []mcp.Tool will be nil
// Otherwise, []mcp.Tool will always have a non-nil value, even if empty list.
// If server does not return any tools by responding with nil rather than empty list, that's fine,
// it's interpreted to mean empty list for interest of compatibility.
func InitMcp() ([]mcp.Tool, error) {
	genericClient, err := Brave.Setup()
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
	response, err := StreamMessage([]Message{message}, context.Background(), nil, tools)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	logging.PrintTelemetry(response)
	return
}
