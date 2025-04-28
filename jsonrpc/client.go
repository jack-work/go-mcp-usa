package jsonrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"go-mcp-usa/logging"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Client[TId any] struct {
	ctx                   context.Context
	reader                io.Reader
	conn                  net.Conn
	notificaticationChans map[string]chan Message[any]
	responseChans         map[string]chan Message[any]
	notLock               *sync.RWMutex
	// Do I need this?  I don't think I ever change this and I don't remember why I added it
	resLock *sync.RWMutex
}

// optional notification chan for auxiliary messages besides the response
// generates an id on behalf of the user if it is not provided
func (client *Client[string]) SendMessage(message Message[any], notificationCh *chan Message[any], errorChan *chan error, expectsResponse bool) (*Message[any], error) {
	id := uuid.New().String()
	message.ID = id

	resCh := make(chan Message[any])
	client.resLock.Lock()
	client.responseChans[id] = resCh
	client.resLock.Unlock()

	defer func() {
		client.resLock.Lock()
		delete(client.responseChans, id)
		close(resCh)
		client.resLock.Unlock()
	}()

	var errCh chan error
	if errorChan == nil {
		errCh = make(chan error)
	} else {
		errCh = *errorChan
	}

	// Send request asynchronously
	go func() {
		err := SendMessage(message, client.conn)
		if err != nil {
			errCh <- err
		}
	}()

	// Wait with timeout
	select {
	case resp := <-resCh:
		fmt.Println("test")
		return &resp, nil
	case err := <-errCh:
		return nil, err
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("request timed out after %v seconds", 10)
	}
}

func NewClient[TId comparable](ctx context.Context, connection net.Conn, reader io.Reader, notificationChannels map[string]chan Message[any], doneCh chan error) (*Client[TId], error) {
	// One go routine to process the output of the conn, which sends a message over a channel to:
	// A multiplexer below to fan out to at most three listeners
	// Listeners may be passed by the caller, or are registered internally to facilitate .
	// Currently, a caller can only create notification channels at construction time.
	// This is so to allow responses, listeners to which we expect to be fewer than those to notifications, be processed first
	// before iterating over notification channels.  This optimization may be premature, as it might take a while to
	// use this library in a sufficiently intense setting, but that's the way I wrote it so it's what we have for now.

	if notificationChannels == nil {
		notificationChannels = make(map[string]chan Message[any])
	}
	main := make(chan Message[any])
	// Parses the json in the reader
	go processOutput(reader, main, doneCh)

	notLock, resLock := sync.RWMutex{}, sync.RWMutex{}

	telemChan := make(chan Message[any])
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case result := <-telemChan:
				logging.PrintTelemetry(result)
			}
		}
	}()

	// Pass from the main channel to the response channels
	// The responseChan can contain type specific wrappers so that we can leverage the mcp in the other folder
	responseChans := make(map[string]chan Message[any], 0)
	// multiplexer
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case response, ok := <-main:
				if !ok {
					return
				}

				telemChan <- response
				SendChannel(&notLock, notificationChannels, response.Method, response)
				SendChannel(&resLock, responseChans, response.ID, response)
			}
			// todo: log failure
		}
	}()

	return &Client[TId]{
		ctx:                   ctx,
		reader:                reader,
		conn:                  connection,
		notificaticationChans: notificationChannels,
		notLock:               &notLock,
		resLock:               &resLock,
		responseChans:         responseChans,
	}, nil
}

func SendChannel(notLock *sync.RWMutex, chans map[string]chan Message[any], key string, value Message[any]) {
	notLock.Lock()
	respCh, exists := chans[key]
	notLock.Unlock()

	fmt.Println(len(chans))
	if exists {
		respCh <- value
	}
}

func processOutput(reader io.Reader, responseChan chan Message[any], doneCh chan error) {
	// Process stdout for JSON messages
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		// Remove any non-printable characters at the beginning
		clean := getCleanLine(line)

		// Try to parse as JSON
		var response Message[any]
		if err := json.Unmarshal([]byte(clean), &response); err != nil {
			fmt.Println(err)
			doneCh <- scanner.Err()
		} else {
			responseChan <- response
		}
	}

	doneCh <- scanner.Err()
}

func getCleanLine(line string) string {
	if idx := strings.Index(line, "{"); idx != -1 {
		return line[idx:]
	} else {
		return line
	}
}

func SendMessage(message Message[any], conn net.Conn) error {
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

func ReceiveMessages[TId any](responseChan chan Message[any]) {
	for {
		select {
		case response := <-responseChan:
			logging.PrintTelemetry(response)
		}
	}
}
