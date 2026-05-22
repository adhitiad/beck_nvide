package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

type LogLine struct {
	StepIndex int    `json:"step_index"`
	Source    string `json:"source"`
	Type      string `json:"type"`
	Content   string `json:"content"`
}

func main() {
	pattern := `C:\Users\humas\.gemini\antigravity\brain\*\.system_generated\logs\overview.txt`
	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatalf("glob failed: %v", err)
	}

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			continue
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
			if strings.Contains(line.Content, "### BAGIAN A:") {
				fmt.Printf("File: %s\n", file)
				// Print up to 1000 characters around ### BAGIAN A:
				idx := strings.Index(line.Content, "### BAGIAN A:")
				endIdx := idx + 2000
				if endIdx > len(line.Content) {
					endIdx = len(line.Content)
				}
				fmt.Println(line.Content[idx:endIdx])
				fmt.Println("==================================================")
			}
		}
	}
}
