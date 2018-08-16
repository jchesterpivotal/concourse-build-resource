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

	jsonFile, err := os.Open(path)
	if err != nil {
		log.Fatalf("could not open %s: %s", path, err.Error())
	}

	var data map[string]interface{}
	err = json.NewDecoder(jsonFile).Decode(&data)
	if err != nil {
		log.Fatalf("could not parse %s: %s", path, err.Error())
	}

	formatter := colorjson.NewFormatter()
	formatter.Indent = 2
	prettified, err := formatter.Marshal(data)
	if err != nil {
		log.Fatalf("could not prettify %s: %s", path, err.Error())
	}

	fmt.Println(string(prettified))
}
