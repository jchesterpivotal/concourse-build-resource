package main

import (
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"encoding/json"
	"os"
	"log"
	"github.com/jchesterpivotal/concourse-build-resource/pkg/check"
)

func main() {
	var request config.CheckRequest
	err := json.NewDecoder(os.Stdin).Decode(&request)
	if err != nil {
		log.Printf("failed to parse input JSON: %s", err)
		os.Exit(1)
		return
	}

	checkResponse, err := check.Check(&request)
	if err != nil {
		log.Printf("failed to perform 'in': %s", err)
		os.Exit(1)
		return
	}

	json.NewEncoder(os.Stdout).Encode(checkResponse)
}
