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
	for _, lineStr := range lines {
		if lineStr == "" {
			continue
		}
		var line LogLine
		err = json.Unmarshal([]byte(lineStr), &line)
		if err != nil {
			continue
		}
		if line.Source == "MODEL" && line.Type == "PLANNER_RESPONSE" {
			// Print model response summaries
			fmt.Printf("=== MODEL PLANNER RESPONSE (Step %d) ===\n", line.StepIndex)
			if len(line.Content) > 500 {
				fmt.Println(line.Content[:500] + "...\n")
			} else {
				fmt.Println(line.Content + "\n")
			}
		}
	}
}
