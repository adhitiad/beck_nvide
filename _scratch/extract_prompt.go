package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type LogLine struct {
	StepIndex int    `json:"step_index"`
	Source    string `json:"source"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	Content   string `json:"content"`
}

func main() {
	file, err := os.Open(`C:\Users\humas\.gemini\antigravity\brain\0a461e6a-7e58-44b2-b64b-8ba8bace2147\.system_generated\logs\overview.txt`)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	var line LogLine
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&line); err != nil {
		fmt.Printf("Error decoding first line: %v\n", err)
		return
	}

	err = os.WriteFile(`C:\Users\humas\.gemini\antigravity\brain\0a461e6a-7e58-44b2-b64b-8ba8bace2147\full_original_prompt.txt`, []byte(line.Content), 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}

	fmt.Println("Successfully wrote full untruncated original prompt to full_original_prompt.txt")
}
