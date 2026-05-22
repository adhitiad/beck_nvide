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
			contentLower := strings.ToLower(line.Content)
			if strings.Contains(contentLower, "a2.") || strings.Contains(contentLower, "a2 ") || strings.Contains(contentLower, "a.2") || strings.Contains(contentLower, "bagian a:") || strings.Contains(contentLower, "bagian a2") {
				fmt.Printf("File: %s | Step: %d | Source: %s\n", path, line.StepIndex, line.Source)
				idx := strings.Index(contentLower, "a2")
				if idx == -1 {
					idx = strings.Index(contentLower, "a.2")
				}
				if idx == -1 {
					idx = strings.Index(contentLower, "bagian a")
				}
				if idx != -1 {
					start := idx - 200
					if start < 0 {
						start = 0
					}
					end := idx + 1000
					if end > len(line.Content) {
						end = len(line.Content)
					}
					fmt.Println(line.Content[start:end])
				}
				fmt.Println("==================================================")
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("walk failed: %v", err)
	}
}
