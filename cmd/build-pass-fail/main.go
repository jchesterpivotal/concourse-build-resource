package main

import (
	"os"
	"log"
	"encoding/json"
	"path/filepath"
	"strings"
	"fmt"
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

		cleanpath = fmt.Sprintf("%s/build.json", cleanpath)
	} else {
		cleanpath = "build/build.json"
	}

	buildInfoFile, err := os.Open(cleanpath)
	if err != nil {
		log.Fatalf("could not open %s: %s", cleanpath, err.Error())
	}

	var build struct {
		TeamName     string `json:"team_name"`
		PipelineName string `json:"pipeline_name"`
		JobName      string `json:"job_name"`
		Name         string `json:"name"`
		Status       string `json:"status"`
	}

	err = json.NewDecoder(buildInfoFile).Decode(&build)
	if err != nil {
		log.Fatalf("could not parse %s: %s", cleanpath, err.Error())
	}

	if build.Status == "succeeded" {
		log.Printf(
			"Build /teams/%s/pipelines/%s/jobs/%s/builds/%s succeeded\n",
			build.TeamName,
			build.PipelineName,
			build.JobName,
			build.Name,
		)

		os.Exit(0)
	} else {
		log.Fatalf(
			"Build /teams/%s/pipelines/%s/jobs/%s/builds/%s was unsuccessful & finished with status '%s'\n",
			build.TeamName,
			build.PipelineName,
			build.JobName,
			build.Name,
			build.Status,
		)
	}
}
