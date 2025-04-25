package logging

import (
	"encoding/json"
	"fmt"
)

func PrintTelemetry[T any](content T) {
	telem, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		fmt.Printf("Cannot print telemetry: %w", err)
	} else {
		fmt.Println(string(telem))
	}
}
