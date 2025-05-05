package anthropicbridge

import (
	"context"
	"errors"
	"figaro/logging"
	"figaro/mcp"
	"fmt"
	"os"
	"reflect"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

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
