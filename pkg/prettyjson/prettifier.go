package prettyjson

import (
	"os"
	"github.com/TylerBrock/colorjson"
	"encoding/json"
	"fmt"
)

func Prettify(jsonpath string) (string, error) {
	jsonFile, err := os.Open(jsonpath)
	if err != nil {
		return "", fmt.Errorf("could not open %s: %s", jsonpath, err.Error())
	}

	var data map[string]interface{}
	err = json.NewDecoder(jsonFile).Decode(&data)
	if err != nil {
		return "", fmt.Errorf("could not parse %s: %s", jsonpath, err.Error())
	}

	formatter := colorjson.NewFormatter()
	formatter.Indent = 2
	prettified, err := formatter.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("could not prettify %s: %s", jsonpath, err.Error())
	}

	return string(prettified), nil
}
