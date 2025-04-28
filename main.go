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
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

func main() {
	// Define flag with default value "default_value"
	modePtr := flag.String("m", "ModelClaude3_7SonnetLatest", "Specify the model to use")

	// Parse flags
	flag.Parse()

	// init MCP
	InitMcp()

	// Use the flag value
	args := flag.Args()
	if len(args) > 0 {
		OneShotAnswer(args, modePtr)
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
		})

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
func InitMcp() {
	genericClient, err := Brave.Setup()
	client, err := jsonrpc.NewClient[string](
		genericClient.Context,
		genericClient.Conn,
		genericClient.Reader,
		nil,
		genericClient.DoneChan)

	if err != nil {
		logging.PrintTelemetry(err)
		return
	}

	message := jsonrpc.Message[any]{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "initialize",
		Params: jsonrpc.InitializeParams{
			ProtocolVersion: "0.1.0",
			ClientInfo: jsonrpc.ClientInfo{
				Name:    "bach",
				Version: "1.0.0",
			},
			Capabilities: jsonrpc.ClientCapabilities{
				Tools:     true,
				Prompts:   false,
				Resources: true,
			},
		},
	}

	msg1, err := client.SendMessage(message, nil, nil, true)
	if err != nil {
		fmt.Println(err)
		return
	}
	logging.PrintTelemetry(msg1)

	// todo:  implement a call and response pattern here
	time.Sleep(1 * time.Second)
	client.SendMessage(jsonrpc.Message[any]{
		JSONRPC: "2.0",
		ID:      "2",
		Method:  "tools/list",
	}, nil, nil, true)
	time.Sleep(1 * time.Second)
	client.SendMessage(jsonrpc.Message[any]{
		JSONRPC: "2.0",
		ID:      "3",
		Method:  "tools/call",
		Params: mcp.CallToolRequestParams{
			Name: "brave_web_search",
			Arguments: map[string]any{
				"query": "search the internet for beetles",
				"count": 100,
			},
		},
	}, nil, nil, true)
}

func OneShotAnswer(args []string, modePtr *string) {
	input := strings.Join(args, " ")
	fmt.Printf("Model: %s\n\n", *modePtr)
	fmt.Printf("Input: %s\n\n", input)
	message := NewMessage(input, string(anthropic.MessageParamRoleUser))
	response, err := SimpleMessage(message)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(response)
	return
}
