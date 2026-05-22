package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
)

type LogLine struct {
	StepIndex int    `json:"step_index"`
	Source    string `json:"source"`
	Type      string `json:"type"`
	Content   string `json:"content"`
}

func main() {
	filePath := `C:\Users\humas\.gemini\antigravity\brain\7b2061c1-bbf6-4e3f-8417-2f5f691aece0\.system_generated\logs\overview.txt`
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	fmt.Printf("Total log lines: %d\n", len(lines))

	// Print the last 15 lines that are user inputs or model inputs
	count := 0
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] == "" {
			continue
		}
		var line LogLine
		err = json.Unmarshal([]byte(lines[i]), &line)
		if err != nil {
			continue
		}
		if line.Source == "USER_EXPLICIT" || line.Source == "USER" || line.Type == "USER_INPUT" {
			fmt.Printf("--- USER INPUT (Step %d) ---\n%s\n\n", line.StepIndex, line.Content)
			count++
			if count >= 5 {
				break
			}
		}
	}
}
