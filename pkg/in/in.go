package in

import (
	"encoding/json"
	"github.com/concourse/atc"
	"github.com/concourse/fly/eventstream"
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"io/ioutil"
	"log"
	"strings"

	gc "github.com/concourse/go-concourse/concourse"

	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
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
	if i.inRequest.Source.EnableTracing {
		log.Printf("Received InRequest: %+v", i.inRequest)
	}

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
	err = i.renderEventsRepetitively(build)
	if err != nil {
		return nil, err
	}

	// K-V convenience files

	i.writeStringFile("team", build.TeamName)
	i.writeStringFile("pipeline", build.PipelineName)
	i.writeStringFile("job", build.JobName)
	i.writeStringFile("global_number", strconv.Itoa(build.ID))
	i.writeStringFile("job_number", build.Name)
	i.writeStringFile("started_time", strconv.Itoa(int(build.StartTime)))
	i.writeStringFile("ended_time", strconv.Itoa(int(build.EndTime)))
	i.writeStringFile("status", build.Status)
	i.writeStringFile("concourse_url", i.concourseUrl(build))
	i.writeStringFile("team_url", i.teamUrl(build))
	i.writeStringFile("pipeline_url", i.pipelineUrl(build))
	i.writeStringFile("job_url", i.jobUrl(build))
	i.writeStringFile("build_url", i.buildUrl(build))
	i.writeStringFile("concourse_build_resource_release", i.inRequest.ReleaseVersion)
	i.writeStringFile("concourse_build_resource_git_ref", i.inRequest.ReleaseGitRef)
	i.writeStringFile("concourse_build_resource_get_timestamp", strconv.Itoa(int(i.inRequest.GetTimestamp)))

	return &config.InResponse{
		Version: i.inRequest.Version,
		Metadata: []config.VersionMetadataField{
			{Name: "build_url", Value: i.buildUrl(build)},
		},
	}, nil
}

func NewInner(input *config.InRequest) Inner {
	tr := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	concourse := gc.NewClient(input.Source.ConcourseUrl, client, input.Source.EnableTracing)

	return NewInnerUsingClient(input, concourse)
}

func NewInnerUsingClient(input *config.InRequest, client gc.Client) Inner {
	return inner{
		inRequest:       input,
		concourseClient: client,
		concourseTeam:   client.Team(input.Source.Team),
	}
}

func (i inner) concourseUrl(build atc.Build) string {
	return i.inRequest.Source.ConcourseUrl
}

func (i inner) teamUrl(build atc.Build) string {
	return fmt.Sprintf(
		"%s/teams/%s",
		i.concourseUrl(build),
		build.TeamName,
	)
}

func (i inner) pipelineUrl(build atc.Build) string {
	return fmt.Sprintf(
		"%s/pipelines/%s",
		i.teamUrl(build),
		build.PipelineName,
	)
}

func (i inner) jobUrl(build atc.Build) string {
	return fmt.Sprintf(
		"%s/jobs/%s",
		i.pipelineUrl(build),
		build.JobName,
	)
}

func (i inner) buildUrl(build atc.Build) string {
	return fmt.Sprintf(
		"%s/builds/%s",
		i.jobUrl(build),
		build.Name,
	)
}

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
	builder := &strings.Builder{}

	err := json.NewEncoder(builder).Encode(object)
	if err != nil {
		return fmt.Errorf("could not encode response from server into '%s': %s", filename, err.Error())
	}

	getMetadataStr := fmt.Sprintf(`{"concourse_build_resource":{"release":"%s","git_ref":"%s","get_timestamp":%d},`, i.inRequest.ReleaseVersion, i.inRequest.ReleaseGitRef, i.inRequest.GetTimestamp)
	jsonStr := builder.String()
	jsonStr = strings.Replace(jsonStr, "{", getMetadataStr, 1)

	err = ioutil.WriteFile(filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.%s", filename, extension)), []byte(jsonStr), os.ModePerm)
	if err != nil {
		return err
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

func (i inner) renderEventsRepetitively(build atc.Build) error {
	events, err := i.concourseClient.BuildEvents(i.inRequest.Version.BuildId)
	// first, check if we are even authorised
	if err != nil && err.Error() == "not authorized" {
		log.Printf("was unauthorized to fetch events for build '%s', no logs will be written.", i.inRequest.Version.BuildId)
		return nil
	}
	if err != nil {
		return fmt.Errorf("error while fetching events for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	defer events.Close()

	eventsFile, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, "events.log"))
	if err != nil {
		return err
	}
	defer eventsFile.Close()
	eventstream.Render(eventsFile, events)

	eventsDetailPostfixed, err := i.concourseClient.BuildEvents(i.inRequest.Version.BuildId)
	if err != nil {
		return fmt.Errorf("error while fetching events for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	defer eventsDetailPostfixed.Close()

	eventsFileDetailPostfixed, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.log", i.addDetailedPostfixTo("events", build))))
	if err != nil {
		return err
	}
	defer eventsFileDetailPostfixed.Close()
	eventstream.Render(eventsFileDetailPostfixed, eventsDetailPostfixed)

	eventsBuildPostfixed, err := i.concourseClient.BuildEvents(i.inRequest.Version.BuildId)
	if err != nil {
		return fmt.Errorf("error while fetching events for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	defer eventsBuildPostfixed.Close()

	eventsFileBuildPostfixed, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.log", i.addBuildNumberPostfixTo("events"))))
	if err != nil {
		return err
	}
	defer eventsFileBuildPostfixed.Close()
	eventstream.Render(eventsFileBuildPostfixed, eventsBuildPostfixed)

	return nil
}
