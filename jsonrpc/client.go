package jsonrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"figaro/logging"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Client interface {
	Notify(method string, params any) error
	SendActionMessage(method string) (*Message[any], error)
	SendMesage(method string, params any)
}

type StdioClient struct {
	reader                io.Reader
	conn                  net.Conn
	notificaticationChans map[string]chan Message[any]
	responseChans         map[string]chan Message[any]
	notLock               *sync.RWMutex
	resLock               *sync.RWMutex
}

type Connection struct {
	Conn   net.Conn
	Reader *bufio.Reader
}

func (client StdioClient) Notify(method string, params any) error {
	return notifyMessage(Message[any]{JSONRPC: "2.0", Method: method, Params: params}, client.conn)
}

// Action -> func with no params
func (client StdioClient) SendActionMessage(method string) (*Message[any], error) {
	return client.sendMessage(Message[any]{JSONRPC: "2.0", Method: method})
}

func (client StdioClient) SendMessage(method string, params any) (*Message[any], error) {
	return client.sendMessage(Message[any]{JSONRPC: "2.0", Method: method, Params: params})
}

// optional notification chan for auxiliary messages besides the response
// generates an id on behalf of the user if it is not provided
func (client *StdioClient) sendMessage(message Message[any]) (*Message[any], error) {
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

	errCh := make(chan error)
	go func() {
		err := notifyMessage(message, client.conn)
		if err != nil {
			errCh <- err
		}
	}()

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

func NewStdioClient[TId comparable](ctx context.Context, client *Connection) (*StdioClient, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	// One go routine to process the output of the conn, which sends a message over a channel to:
	// A multiplexer below to fan out to at most three listeners
	// Listeners may be passed by the caller, or are registered internally to facilitate .
	// Currently, a caller can only create notification channels at construction time.
	// This is so to allow responses, listeners to which we expect to be fewer than those to notifications, be processed first
	// before iterating over notification channels.  This optimization may be premature, as it might take a while to
	// use this library in a sufficiently intense setting, but that's the way I wrote it so it's what we have for now.
	notificationChannels := make(map[string]chan Message[any])
	main := make(chan Message[any])

	// Parses the json in the reader
	go processOutput(client.Reader, main, cancel)

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

	return &StdioClient{
		reader:                client.Reader,
		conn:                  client.Conn,
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

// if the reader dies, Scan will return false and this will fail.
// probably should run the scanner in a for select and handle ctx failure appropriately.
// this works for now, so TODO:
// probably can do something with timeout where every successful scan resets the timeout, otherwise
// cancel is called
func processOutput(reader io.Reader, responseChan chan Message[any], cancel context.CancelCauseFunc) {
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
			cancel(scanner.Err())
		} else {
			responseChan <- response
		}
	}

	cancel(scanner.Err())
}

func getCleanLine(line string) string {
	if idx := strings.Index(line, "{"); idx != -1 {
		return line[idx:]
	} else {
		return line
	}
}

// Sends a message to the server as a notification.  If a response is expected, channels
// must be registered ahead of time to look for a response possessing the same id.
func notifyMessage(message Message[any], conn net.Conn) error {
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
