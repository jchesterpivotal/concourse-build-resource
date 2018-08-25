package main

import (
	"os"
	"log"
	"path/filepath"
	"strings"
	"fmt"
	"io/ioutil"
)

func main() {
	var jsonPath, cleanPath, statusPath, urlPath string
	if len(os.Args) > 1 {
		jsonPath = os.Args[1]

		cleanPath = filepath.Clean(jsonPath)
		if strings.HasPrefix(cleanPath, "/") ||
			strings.Contains(cleanPath, "..") ||
			strings.Count(cleanPath, "/") > 1 {
			log.Fatalf("malformed path")
		}

		statusPath = fmt.Sprintf("%s/status", cleanPath)
		urlPath = fmt.Sprintf("%s/url", cleanPath)
	} else {
		statusPath = "build/status"
		urlPath = "build/url"
	}

	buildStatus, err := ioutil.ReadFile(statusPath)
	if err != nil {
		log.Fatalf("could not read %s: %s", statusPath, err.Error())
	}

	buildUrl, err := ioutil.ReadFile(urlPath)
	if err != nil {
		log.Fatalf("could not read %s: %s", urlPath, err.Error())
	}

	if string(buildStatus) == "succeeded" {
		log.Printf("Build %s succeeded\n", buildUrl)

		os.Exit(0)
	} else {
		log.Fatalf("Build %s was unsuccessful & finished with status '%s'\n", buildUrl, buildStatus)
	}
}
