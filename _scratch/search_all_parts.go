package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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
	root := `C:\Users\humas\.gemini\antigravity\brain`
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.Contains(path, "overview.txt") {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil
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
			contentUpper := strings.ToUpper(line.Content)
			if strings.Contains(contentUpper, "BAGIAN B") || strings.Contains(contentUpper, "BAGIAN C") || strings.Contains(contentUpper, "BAGIAN D") {
				fmt.Printf("File: %s | Step: %d | Source: %s\n", path, line.StepIndex, line.Source)
				idx := strings.Index(contentUpper, "BAGIAN B")
				if idx == -1 {
					idx = strings.Index(contentUpper, "BAGIAN C")
				}
				if idx == -1 {
					idx = strings.Index(contentUpper, "BAGIAN D")
				}
				if idx != -1 {
					start := idx - 100
					if start < 0 {
						start = 0
					}
					end := idx + 3000
					if end > len(line.Content) {
						end = len(line.Content)
					}
					fmt.Println(line.Content[start:end])
				}
				fmt.Println("==================================================")
				return nil // Just print the first match and skip the rest of this file
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("walk failed: %v", err)
	}
}
