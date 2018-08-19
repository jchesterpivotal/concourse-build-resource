package main

import (
	"log"
	"path/filepath"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	var jsonpath, cleanpath string
	if len(os.Args) > 1 {
		jsonpath = os.Args[1]

		cleanpath = filepath.Clean(jsonpath)
		if strings.HasPrefix(cleanpath, "/") ||
			strings.Contains(cleanpath, "..") ||
			strings.Count(cleanpath, "/") > 1 {
			log.Fatalf("malformed path")
		}

		cleanpath = filepath.Join(cleanpath, "events.log")
	} else {
		cleanpath = filepath.Join("build", "events.log")
	}

	contents, err := ioutil.ReadFile(cleanpath)
	if err != nil {
		log.Fatalf("could not open %s: %s", cleanpath, err.Error())
	}

	fmt.Println("----------------------------------- [ begin log ] -----------------------------------")
	fmt.Println(string(contents))
	fmt.Println("------------------------------------ [ end log ] ------------------------------------")
}
