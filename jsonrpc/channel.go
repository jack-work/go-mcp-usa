package jsonrpc

import (
	"encoding/json"
	"fmt"
	"go-mcp-usa/logging"
	"net"
)

func SendMessage(message Message, conn net.Conn) error {
	// Marshal to JSON and add newline
	messageJson, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal initialization message: %v", err)
	}
	messageJson = append(messageJson, '\n')

	// Send the message to the container's stdin
	if _, err := conn.Write(messageJson); err != nil {
		return fmt.Errorf("failed to send initialization message: %v", err)
	}

	return nil
}

func ReceiveMessages(responseChan chan Message) {
	for {
		select {
		case response := <-responseChan:
			logging.PrintTelemetry(response)
		}
	}
}
