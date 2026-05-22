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

		// Extract conversation ID from path
		parts := strings.Split(path, string(filepath.Separator))
		convoID := "unknown"
		for i, part := range parts {
			if part == "brain" && i+1 < len(parts) {
				convoID = parts[i+1]
				break
			}
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
			if line.Source == "USER_EXPLICIT" || (line.Source == "USER" && line.StepIndex == 0) {
				outPath := fmt.Sprintf("scratch/step0_%s.txt", convoID)
				_ = ioutil.WriteFile(outPath, []byte(line.Content), 0644)
				fmt.Printf("Wrote: %s\n", outPath)
				break
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("failed: %v", err)
	}
}
