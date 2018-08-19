package main

import (
	"github.com/jchesterpivotal/concourse-build-resource/pkg/prettyjson"

	"os"
	"log"
	"fmt"
)

func main() {
	var filepath string
	if len(os.Args) > 1 {
		filepath = fmt.Sprintf("%s/resources.json", os.Args[1])
	} else {
		filepath = "build/resources.json"
	}

	prettified, err := prettyjson.Prettify(filepath)
	if err != nil {
		log.Fatalf("could not show %s: %s", filepath, err.Error())
	}

	fmt.Println(prettified)
}
