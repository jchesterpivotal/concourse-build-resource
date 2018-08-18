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
	cleanpath := filepath.Clean(jsonpath)
	if strings.HasPrefix(cleanpath, "/") ||
		strings.Contains(cleanpath, "..") ||
		strings.Count(cleanpath, "/") > 1 {
		return "", fmt.Errorf("malformed path")
	}

	jsonFile, err := os.Open(cleanpath)
	if err != nil {
		return "", fmt.Errorf("could not open %s: %s", cleanpath, err.Error())
	}

	var data map[string]interface{}
	err = json.NewDecoder(jsonFile).Decode(&data)
	if err != nil {
		return "", fmt.Errorf("could not parse %s: %s", cleanpath, err.Error())
	}

	formatter := colorjson.NewFormatter()
	formatter.Indent = 2
	prettified, err := formatter.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("could not prettify %s: %s", cleanpath, err.Error())
	}

	return string(prettified), nil
}
