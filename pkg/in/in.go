package in

import (
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"net/http"
	"time"
	"fmt"
	"os"
	"path/filepath"
	"github.com/concourse/go-concourse/concourse"
	"strconv"
	"encoding/json"
	"crypto/tls"
	"github.com/concourse/fly/eventstream"
)

func In(input *config.InRequest) (*config.InResponse, error) {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	concourse := concourse.NewClient(input.Source.ConcourseUrl, client, false)

	buildId, err := strconv.Atoi(input.Version.BuildId)
	if err != nil {
		return nil, fmt.Errorf("could not convert build id '%s' to an int: '%s", input.Version.BuildId, err.Error())
	}

	// the build
	build, found, err := concourse.Build(input.Version.BuildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching build '%s': '%s", input.Version.BuildId, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("build '%s' not found: '%s", input.Version.BuildId, err.Error())
	}

	buildFile, err := os.Create(filepath.Join(input.WorkingDirectory, "build.json"))
	defer buildFile.Close()
	if err != nil {
		return nil, err
	}

	json.NewEncoder(buildFile).Encode(build)

	// resources
	resources, found, err := concourse.BuildResources(buildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching resources for build '%s': '%s", input.Version.BuildId, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("build '%s' not found while fetching resources: '%s", input.Version.BuildId, err.Error())
	}

	resFile, err := os.Create(filepath.Join(input.WorkingDirectory, "resources.json"))
	defer resFile.Close()
	if err != nil {
		return nil, err
	}

	json.NewEncoder(resFile).Encode(resources)

	// plan
	plan, found, err := concourse.BuildPlan(buildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching plan for build '%s': '%s", input.Version.BuildId, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("build '%s' not found while fetching plan: '%s", input.Version.BuildId, err.Error())
	}

	planFile, err := os.Create(filepath.Join(input.WorkingDirectory, "plan.json"))
	defer planFile.Close()
	if err != nil {
		return nil, err
	}

	json.NewEncoder(planFile).Encode(plan)

	// events
	events, err := concourse.BuildEvents(input.Version.BuildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching events for build '%s': '%s", input.Version.BuildId, err.Error())
	}

	eventsFile, err := os.Create(filepath.Join(input.WorkingDirectory, "events.log"))
	defer eventsFile.Close()
	if err != nil {
		return nil, err
	}

	eventstream.Render(eventsFile, events)

	return &config.InResponse{
		Version: input.Version,
		Metadata: []config.VersionMetadataField{
			{Name: "name", Value: "value"},
		},
	}, nil
}
