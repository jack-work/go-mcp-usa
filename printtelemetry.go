package main

import (
	"encoding/json"
	"fmt"
)

func printTelemetry[T any](content T) {
	telem, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		fmt.Printf("Cannot print telemetry: %w", err)
	} else {
		fmt.Println(string(telem))
	}
}
