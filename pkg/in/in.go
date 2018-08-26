package in

import (
	"github.com/concourse/atc"
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"

	gc "github.com/concourse/go-concourse/concourse"
	"github.com/concourse/fly/eventstream"

	"net/http"
	"time"
	"fmt"
	"os"
	"path/filepath"
	"encoding/json"
	"crypto/tls"
	"strconv"
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

	i.writeJsonFile("build", "json", build)
	i.writeJsonFile(i.addDetailedPostfixTo("build", build), "json", build)
	i.writeJsonFile(i.addBuildNumberPostfixTo("build"), "json", build)

	// resources
	resources, found, err := i.concourseClient.BuildResources(buildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching resources for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("build '%s' not found while fetching resources", i.inRequest.Version.BuildId)
	}

	i.writeJsonFile("resources", "json", resources)
	i.writeJsonFile(i.addDetailedPostfixTo("resources", build), "json", resources)
	i.writeJsonFile(i.addBuildNumberPostfixTo("resources"), "json", resources)

	// plan
	plan, found, err := i.concourseClient.BuildPlan(buildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching plan for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("build '%s' not found while fetching plan", i.inRequest.Version.BuildId)
	}

	i.writeJsonFile("plan", "json", plan)
	i.writeJsonFile(i.addDetailedPostfixTo("plan", build), "json", plan)
	i.writeJsonFile(i.addBuildNumberPostfixTo("plan"), "json", plan)

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

	eventsFileDetailPostfixed, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.log", i.addDetailedPostfixTo("events", build))))
	defer eventsFileDetailPostfixed.Close()
	if err != nil {
		return nil, err
	}
	eventstream.Render(eventsFileDetailPostfixed, events)

	eventsFileBuildPostfixed, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.log", i.addBuildNumberPostfixTo("events"))))
	defer eventsFileBuildPostfixed.Close()
	if err != nil {
		return nil, err
	}
	eventstream.Render(eventsFileBuildPostfixed, events)

	// K-V convenience files

	i.writeStringFile("team", build.TeamName)
	i.writeStringFile("pipeline", build.PipelineName)
	i.writeStringFile("job", build.JobName)
	i.writeStringFile("global-number", strconv.Itoa(build.ID))
	i.writeStringFile("job-number", build.Name)
	i.writeStringFile("started-time", strconv.Itoa(int(build.StartTime)))
	i.writeStringFile("ended-time", strconv.Itoa(int(build.EndTime)))
	i.writeStringFile("status", build.Status)
	i.writeStringFile("url", i.fullUrl(build))

	return &config.InResponse{
		Version:  i.inRequest.Version,
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
		inRequest:       input,
		concourseClient: client,
		concourseTeam:   client.Team(input.Source.Team),
	}
}

//TODO: this won't work when source doesn't contain team name
func (i inner) fullUrl(build atc.Build) string {
	return fmt.Sprintf(
		"%s/teams/%s/pipelines/%s/jobs/%s/builds/%s",
		i.inRequest.Source.ConcourseUrl,
		build.TeamName,
		build.PipelineName,
		build.JobName,
		build.Name)
}

//TODO: this won't work when source doesn't contain team name
func (i inner) addDetailedPostfixTo(name string, build atc.Build) string {
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		name,
		build.TeamName,
		build.PipelineName,
		build.JobName,
		build.Name)
}

func (i inner) addBuildNumberPostfixTo(name string) string {
	return fmt.Sprintf("%s-%s", name, i.inRequest.Version.BuildId)
}

func (i inner) writeJsonFile(filename string, extension string, object interface{}) error {
	file, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.%s", filename, extension)))
	defer file.Close()
	if err != nil {
		return err
	}

	err = json.NewEncoder(file).Encode(object)
	if err != nil {
		return fmt.Errorf("could not encode response from server into '%s': %s", filename, err.Error())
	}

	return nil
}

func (i inner) writeStringFile(filename string, value string) error {
	file, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, filename))
	defer file.Close()
	if err != nil {
		return err
	}

	_, err = file.WriteString(value)

	return err
}
