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
	file := `C:\Users\humas\.gemini\antigravity\brain\0a461e6a-7e58-44b2-b64b-8ba8bace2147\.system_generated\logs\overview.txt`
	data, err := ioutil.ReadFile(file)
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
		// Look for user explicit prompt
		if line.Source == "USER_EXPLICIT" || (line.Source == "USER" && line.StepIndex == 0) {
			err = ioutil.WriteFile("scratch/step0_content.txt", []byte(line.Content), 0644)
			if err != nil {
				log.Fatalf("failed to write file: %v", err)
			}
			fmt.Println("Wrote step0_content.txt successfully!")
			break
		}
	}
}
