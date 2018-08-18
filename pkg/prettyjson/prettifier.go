package prettyjson

import (
	"os"
	"github.com/TylerBrock/colorjson"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

func Prettify(jsonpath string) (string, error) {
	cleanedpath := filepath.Clean(jsonpath)
	if strings.HasPrefix(cleanedpath, "/") || strings.Contains(cleanedpath, "..") {
		return "", fmt.Errorf("malformed path")
	}

	jsonFile, err := os.Open(cleanedpath)
	if err != nil {
		return "", fmt.Errorf("could not open %s: %s", cleanedpath, err.Error())
	}

	var data map[string]interface{}
	err = json.NewDecoder(jsonFile).Decode(&data)
	if err != nil {
		return "", fmt.Errorf("could not parse %s: %s", cleanedpath, err.Error())
	}

	formatter := colorjson.NewFormatter()
	formatter.Indent = 2
	prettified, err := formatter.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("could not prettify %s: %s", cleanedpath, err.Error())
	}

	return string(prettified), nil
}
