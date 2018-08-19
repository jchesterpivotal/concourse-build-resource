package in

import (
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"net/http"
	"time"
	"fmt"
	"os"
	"path/filepath"
	gc "github.com/concourse/go-concourse/concourse"
	"encoding/json"
	"crypto/tls"
	"strconv"
	"github.com/concourse/fly/eventstream"
)

type Inner interface {
	In() (*config.InResponse, error)
}
type inner struct {
	inRequest       *config.InRequest
	concourseClient gc.Client
	concourseTeam   gc.Team
}

func (i inner) In() (*config.InResponse, error) {
	buildId, err := strconv.Atoi(i.inRequest.Version.BuildId)
	if err != nil {
		return nil, fmt.Errorf("could not convert build id '%s' to an int: '%s", i.inRequest.Version.BuildId, err.Error())
	}

	// the build
	build, found, err := i.concourseClient.Build(i.inRequest.Version.BuildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("server could not find '%s/%s' while retrieving build '%s'", i.inRequest.Source.Pipeline, i.inRequest.Source.Job, i.inRequest.Version.BuildId)
	}

	buildFile, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, "build.json"))
	defer buildFile.Close()
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(buildFile).Encode(build)
	if err != nil {
		return nil, fmt.Errorf("could not encode build response from server: %s", err.Error())
	}

	// resources
	resources, found, err := i.concourseClient.BuildResources(buildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching resources for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("build '%s' not found while fetching resources", i.inRequest.Version.BuildId)
	}

	resFile, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, "resources.json"))
	defer resFile.Close()
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(resFile).Encode(resources)
	if err != nil {
		return nil, fmt.Errorf("could not encode resources response from server: %s", err.Error())
	}

	// plan
	plan, found, err := i.concourseClient.BuildPlan(buildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching plan for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("build '%s' not found while fetching plan", i.inRequest.Version.BuildId)
	}

	planFile, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, "plan.json"))
	defer planFile.Close()
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(planFile).Encode(plan)
	if err != nil {
		return nil, fmt.Errorf("could not encode plan response from server: %s", err.Error())
	}

	// events
	events, err := i.concourseClient.BuildEvents(i.inRequest.Version.BuildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching events for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}

	eventsFile, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, "events.log"))
	defer eventsFile.Close()
	if err != nil {
		return nil, err
	}

	eventstream.Render(eventsFile, events)

	return &config.InResponse{
		Version: i.inRequest.Version,
		Metadata: []config.VersionMetadataField{},
	}, nil
}

func NewInner(input *config.InRequest) Inner {
	tr := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	concourse := gc.NewClient(input.Source.ConcourseUrl, client, false)

	return NewInnerUsingClient(input, concourse)
}

func NewInnerUsingClient(input *config.InRequest, client gc.Client) Inner {
	return inner{
		inRequest: input,
		concourseClient: client,
		concourseTeam: client.Team(input.Source.Team),
	}
}
