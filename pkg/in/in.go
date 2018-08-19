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
		return nil, fmt.Errorf("build '%s' not found: '%s", i.inRequest.Version.BuildId, err.Error())
	}

	buildFile, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, "build.json"))
	defer buildFile.Close()
	if err != nil {
		return nil, err
	}

	json.NewEncoder(buildFile).Encode(build)

	// resources
	resources, found, err := i.concourseClient.BuildResources(buildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching resources for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("build '%s' not found while fetching resources: '%s", i.inRequest.Version.BuildId, err.Error())
	}

	resFile, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, "resources.json"))
	defer resFile.Close()
	if err != nil {
		return nil, err
	}

	json.NewEncoder(resFile).Encode(resources)

	// plan
	plan, found, err := i.concourseClient.BuildPlan(buildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching plan for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("build '%s' not found while fetching plan: '%s", i.inRequest.Version.BuildId, err.Error())
	}

	planFile, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, "plan.json"))
	defer planFile.Close()
	if err != nil {
		return nil, err
	}

	json.NewEncoder(planFile).Encode(plan)

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
