package main

import (
	"fmt"
	"time"
)

func ShowSpinner(done <-chan bool) {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	for {
		select {
		case <-done:
			fmt.Printf("\r") // Clear spinner
			return
		default:
			fmt.Printf("\r%s Processing...", spinner[i])
			i = (i + 1) % len(spinner)
			time.Sleep(100 * time.Millisecond)
		}
	}
}
