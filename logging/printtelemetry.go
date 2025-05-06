package logging

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
)

func EzMarshal[T any](content T) string {
	telem, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Sprintf("Cannot print telemetry: %v", err)
	} else {
		return string(telem)
	}
}

func EzPrint[T any](content T) {
	telem, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		fmt.Printf("Call stack:\n%s\n", debug.Stack())
		fmt.Printf("Cannot print telemetry: %v", err)
	} else {
		fmt.Println(string(telem))
	}
}
