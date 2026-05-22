package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := `C:\Users\humas\.gemini\antigravity\brain`
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".txt" && ext != ".json" {
			return nil
		}
		if strings.Contains(path, "overview.txt") {
			// Skip overview.txt as we already scanned it
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)
		if strings.Contains(content, "A2.") || strings.Contains(content, "A2 ") || strings.Contains(content, "Bagian A") {
			fmt.Printf("File: %s\n", path)
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				if strings.Contains(line, "A2.") || strings.Contains(line, "A2 ") || strings.Contains(line, "Bagian A") {
					fmt.Printf("  Line: %s\n", strings.TrimSpace(line))
				}
			}
			fmt.Println("--------------------------------------------------")
		}
		return nil
	})

	if err != nil {
		log.Fatalf("walk failed: %v", err)
	}
}
