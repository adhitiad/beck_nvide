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
	Content   string `json:"content"`
}

func main() {
	filePath := `C:\Users\humas\.gemini\antigravity\brain\7b2061c1-bbf6-4e3f-8417-2f5f691aece0\.system_generated\logs\overview.txt`
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || lines[0] == "" {
		log.Fatalf("empty file")
	}

	var firstLine LogLine
	err = json.Unmarshal([]byte(lines[0]), &firstLine)
	if err != nil {
		log.Fatalf("failed to unmarshal JSON: %v", err)
	}

	fmt.Println(firstLine.Content)
}
