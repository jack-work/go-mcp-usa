package main

import (
	"context"
	"errors"
	"fmt"
	"go-mcp-usa/logging"
	"go-mcp-usa/mcp"
	"os"
	"reflect"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

func GetAnthropicMessageParams(message []Message, tools []mcp.Tool) *anthropic.MessageNewParams {
	return &anthropic.MessageNewParams{
		MaxTokens: 1024,
		Messages:  GetAnthropicMessages(message),
		Model:     anthropic.ModelClaude3_7SonnetLatest,
		Tools:     GetAnthropicTools(tools),
	}
}

func GetAnthropicTools(tools []mcp.Tool) []anthropic.ToolUnionParam {
	anTools := make([]anthropic.ToolUnionParam, len(tools))
	for idx, val := range tools {
		anTools[idx] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        val.Name,
				InputSchema: GetAnthropicInputSchema(val.InputSchema),
				Description: param.Opt[string]{
					Value: *val.Description,
				},
			},
		}
	}

	return anTools
}

func GetAnthropicInputSchema(schema mcp.ToolInputSchema) anthropic.ToolInputSchemaParam {
	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties,
		Type:       constant.Object(schema.Type),
	}
}

func GetAnthropicMessages(messages []Message) []anthropic.MessageParam {
	return map2(messages, func(msg Message) anthropic.MessageParam {
		return GetAnthropicMessage(msg)
	})
}

type AnthropicMessageEnvelope struct {
	Message *anthropic.Message
}

func (u AnthropicMessageEnvelope) GetID() string {
	return u.Message.ID
}

func (u AnthropicMessageEnvelope) GetContent() string {
	return strings.Join(map2(u.Message.Content, func(union anthropic.ContentBlockUnion) string {
		return union.Text
	}), " ")
}

func (u AnthropicMessageEnvelope) GetRole() string {
	return string(u.Message.Role)
}

func GetAnthropicMessage(message Message) anthropic.MessageParam {
	return anthropic.MessageParam{
		Role: anthropic.MessageParamRole(message.GetRole()),
		Content: []anthropic.ContentBlockParamUnion{{
			OfRequestTextBlock: &anthropic.TextBlockParam{Text: message.GetContent()},
		}},
	}
}

func GetMessage(message *anthropic.Message) (Message, error) {
	return &AnthropicMessageEnvelope{
		Message: message,
	}, nil
	// I don't have handling for specific types at the moment.
	// switch variant := message.Content[0].AsAny().(type) {
	// case anthropic.TextBlock:
	// 	return &AnthropicMessageEnvelope{
	// 		Message: message,
	// 	}, nil
	// case anthropic.ToolUseBlock:
	// 	return nil, errors.New("ToolUseBlock, Not supported")
	// case anthropic.ThinkingBlock:
	// 	return nil, errors.New("ThinkingBlock, Not supported")
	// case anthropic.RedactedThinkingBlock:
	// 	return nil, errors.New("RedactedThinkingBlock, Not supported")
	// }
	// return nil, errors.New("Bad response from anthropic sdk")
}

func GetAnthropicClient() (*anthropic.Client, error) {
	apiKey, ok := os.LookupEnv("ANTHROPIC_API_KEY")
	if !ok {
		return nil, errors.New("No ANTHROPIC_API_KEY found")
	}
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)
	return &client, nil
}

func StreamMessage(input []Message, context context.Context, task func(string) error, tools []mcp.Tool) (Message, error) {
	params := *GetAnthropicMessageParams(input, tools)
	result, err := StreamMessage2(params, context, task)
	if err != nil {
		return nil, err
	}

	return GetMessage(result)
}

// newer version that uses built in anthropic types
func StreamMessage2(input anthropic.MessageNewParams, context context.Context, task func(string) error) (*anthropic.Message, error) {
	logging.PrintTelemetry(input)
	client, err := GetAnthropicClient()
	if err != nil {
		return nil, err
	}

	stream := client.Messages.NewStreaming(context, input)
	if stream.Err() != nil {
		fmt.Println(stream.Err())
		return nil, stream.Err()
	}
	message := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		err := message.Accumulate(event)
		if err != nil {
			panic(err)
		}

		switch variant := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			switch deltaVariant := variant.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				task(deltaVariant.Text)
			default:
				task(reflect.TypeOf(deltaVariant).Name() + "\n")
			}
		}
	}

	return &message, nil
}
