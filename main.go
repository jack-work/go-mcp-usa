package main

// As Theodore Roosevelt proclaimed, we shall "speak softly and carry a big stack"

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

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
	// err := RegisterServer("mcp/brave-search")
	// if err != nil {
	// 	fmt.Println("something went wrong: %w", err)
	// }
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
