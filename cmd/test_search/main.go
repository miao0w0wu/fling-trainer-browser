package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"changeme/backend/scraper"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: go run ./cmd/test_search <game name>")
	}

	agent := scraper.NewSearchAgent()
	results, err := agent.Search(strings.Join(os.Args[1:], " "))
	if err != nil {
		log.Fatal(err)
	}

	encoded, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}
