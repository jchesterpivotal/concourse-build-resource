package showjson

import (
	"os"
	"log"
	"github.com/TylerBrock/colorjson"
	"fmt"
	"path/filepath"
	"encoding/json"
)

func Show(filename string) {
	path := filepath.Join("build", filename)

	planFile, err := os.Open(path)
	if err != nil {
		log.Fatalf("could not open %s: %s", path, err.Error())
	}

	var plan map[string]interface{}
	err = json.NewDecoder(planFile).Decode(&plan)
	if err != nil {
		log.Fatalf("could not parse %s: %s", path, err.Error())
	}

	formatter := colorjson.NewFormatter()
	formatter.Indent = 2
	prettified, err := formatter.Marshal(plan)
	if err != nil {
		log.Fatalf("could not prettify plan: %s", err.Error())
	}

	fmt.Println(string(prettified))
}
