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
		filepath = fmt.Sprintf("%s/plan.json", os.Args[1])
	} else {
		filepath = "build/plan.json"
	}

	prettified, err := prettyjson.Prettify(filepath)
	if err != nil {
		log.Fatalf("could not show %s: %s", filepath, err.Error())
	}

	fmt.Println(prettified)
}
