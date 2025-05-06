package main

// As Theodore Roosevelt proclaimed, we shall "speak softly and carry a big stack"

import (
	"context"
	"encoding/json"
	"figaro/figaro"
	"figaro/logging"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// establish root context
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(ctx.Err())

	// setup tracer and defer cleanup
	tp, err := logging.InitTracer(logging.WithServiceName("figaro"))
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			logging.EzPrint(fmt.Sprintf("Error shutting down tracer: %v", err))
		}
	}()

	// Define flag with default value "default_value"
	modePtr := flag.String("m", "ModelClaude3_7SonnetLatest", "Specify the model to use")

	// Parse flags
	flag.Parse()

	// init MCP
	servers, err := GetServers()
	if err != nil {
		logging.EzPrint(err)
	}

	figaro, err := figaro.SummonFigaro(ctx, tp, *servers)
	if err != nil {
		return
	}

	// Use the flag value
	args := flag.Args()
	if len(args) > 0 {
		figaro.Request(args, modePtr)
		cancel(nil)
		return
	} else {
		logging.EzPrint("Nothing to say now.  Bye bye.")
		cancel(nil)
	}
}

func GetServers() (*figaro.ServerRegistry, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(homeDir, ".figaro", "servers.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Unmarshal into struct and add the ID
	var config figaro.ServerRegistry
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
