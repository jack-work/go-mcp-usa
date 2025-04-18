package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func GetAnthropicMessageParams(message []Message) *anthropic.MessageNewParams {
	return &anthropic.MessageNewParams{
		MaxTokens: 1024,
		Messages: map2(message, func(msg Message) anthropic.MessageParam {
			return GetAnthropicMessage(msg)
		}),
		Model: anthropic.ModelClaude3_7SonnetLatest,
	}
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

func SimpleMessage(input Message) (Message, error) {
	client, err := GetAnthropicClient()
	if err != nil {
		return nil, err
	}

	anthropicMessage := GetAnthropicMessageParams([]Message{input})
	if anthropicMessage == nil {
		return nil, errors.New(fmt.Sprintf("Bad input, %v", input))
	}

	message, err := client.Messages.New(context.TODO(), *anthropicMessage)
	if err != nil {
		return nil, err
	}

	return GetMessage(message)
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

func StreamMessage(input []Message, context context.Context, task func(string) error) (Message, error) {
	client, err := GetAnthropicClient()
	if err != nil {
		return nil, err
	}

	stream := client.Messages.NewStreaming(context, *GetAnthropicMessageParams(input))
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
			}
		}
	}
	return GetMessage(&message)
}
