package main

import (
	"os"
	"log"
	"encoding/json"
)

func main() {
	buildInfoFile, err := os.Open("build/build.json")
	if err != nil {
		log.Fatalf("could not open build/build.json: %s", err.Error())
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
		log.Fatalf("could not parse build/build.json: %s", err.Error())
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
