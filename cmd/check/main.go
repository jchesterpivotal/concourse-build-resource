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
		log.Fatalf("failed to parse input JSON: %s", err)
	}

	checkResponse, err := check.NewChecker(&request).Check()
	if err != nil {
		log.Fatalf("failed to perform 'check': %s", err)
	}

	err = json.NewEncoder(os.Stdout).Encode(checkResponse)
	if err != nil {
		log.Fatalf("failed to encode check.Check response: %s", err.Error())
	}
}
