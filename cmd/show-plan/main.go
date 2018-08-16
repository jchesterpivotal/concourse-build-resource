package main

import (
	"os"
	"log"
	"encoding/json"
	"fmt"
	"github.com/TylerBrock/colorjson"
)

func main() {
	planFile, err := os.Open("build/plan.json")
	if err != nil {
		log.Fatalf("could not open build/plan.json: %s", err.Error())
	}

	var plan map[string]interface{}
	err = json.NewDecoder(planFile).Decode(&plan)
	if err != nil {
		log.Fatalf("could not parse build/plan.json: %s", err.Error())
	}

	formatter := colorjson.NewFormatter()
	formatter.Indent = 2
	prettified, err := formatter.Marshal(plan)
	if err != nil {
		log.Fatalf("could not prettify plan: %s", err.Error())
	}

	fmt.Println(string(prettified))
}
