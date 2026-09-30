package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"changeme/backend/scraper"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: go run ./cmd/test_detail <detail URL>")
	}

	detail, err := scraper.NewDetailParser().Parse(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	encoded, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(encoded))
}
