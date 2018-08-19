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
		log.Fatalf("failed to parse input JSON: %s", err)
	}

	request.WorkingDirectory = os.Args[1]

	inResponse, err := in.NewInner(&request).In()
	if err != nil {
		log.Fatalf("failed to perform 'in': %s", err)
	}

	json.NewEncoder(os.Stdout).Encode(inResponse)
	if err != nil {
		log.Fatalf("failed to encode in.In response: %s", err.Error())
	}
}

