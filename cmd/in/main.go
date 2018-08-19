package main

import (
	"encoding/json"
	"os"
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"log"
	"github.com/jchesterpivotal/concourse-build-resource/pkg/in"
)

func main() {
	var request config.InRequest
	err := json.NewDecoder(os.Stdin).Decode(&request)
	if err != nil {
		log.Printf("failed to parse input JSON: %s", err)
		os.Exit(1)
		return
	}

	request.WorkingDirectory = os.Args[1]

	inResponse, err := in.In(&request)
	if err != nil {
		log.Printf("failed to perform 'in': %s", err)
		os.Exit(1)
		return
	}

	json.NewEncoder(os.Stdout).Encode(inResponse)
	if err != nil {
		log.Fatalf("failed to encode in.In response: %s", err.Error())
	}
}

