package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
)

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
		var raw map[string]interface{}
		err = json.Unmarshal([]byte(lineStr), &raw)
		if err != nil {
			continue
		}
		stepIndexVal, ok := raw["step_index"].(float64)
		if !ok {
			continue
		}
		stepIndex := int(stepIndexVal)
		if stepIndex >= 540 {
			source := raw["source"].(string)
			if source == "MODEL" {
				fmt.Printf("=== STEP %d ===\n%s\n\n", stepIndex, lineStr)
			}
		}
	}
}
