package figaro

import (
	"context"
	"encoding/json"
	"figaro/anthropicbridge"
	"figaro/jsonrpc"
	"figaro/logging"
	"figaro/mcp"
	"fmt"
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

func (figaro *Figaro) Request(args []string, modePtr *string) error {
	tools := figaro.GetAllTools()
	input := strings.Join(args, " ")

	logging.PrintTelemetry(tools)
	logging.PrintTelemetry("yup")
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
			logging.PrintTelemetry(message)
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
			logging.PrintTelemetry(response)
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
