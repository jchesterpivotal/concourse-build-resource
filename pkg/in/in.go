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
	concourseInfo   atc.Info
	build           atc.Build
	resources       atc.BuildInputsOutputs
	buildId         int
}

func (i inner) In() (*config.InResponse, error) {
	if i.inRequest.Source.EnableTracing {
		log.Printf("Received InRequest: %+v", i.inRequest)
	}

	err := i.getConcourseInfo()
	if err != nil {
		return nil, err
	}

	err = i.getBuildId()
	if err != nil {
		return nil, err
	}

	// the build
	err = i.getBuild()
	if err != nil {
		return nil, err
	}

	err = i.writeBuild()
	if err != nil {
		return nil, err
	}

	// resources
	err = i.getResources()
	if err != nil {
		return nil, err
	}

	err = i.writeResources()
	if err != nil {
		return nil, err
	}

	// plan
	plan, found, err := i.concourseClient.BuildPlan(i.buildId)
	if err != nil {
		return nil, fmt.Errorf("error while fetching plan for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("build '%s' not found while fetching plan", i.inRequest.Version.BuildId)
	}

	i.writeJsonFile("plan", "json", plan)
	i.writeJsonFile(i.addDetailedPostfixTo("plan"), "json", plan)
	i.writeJsonFile(i.addBuildNumberPostfixTo("plan"), "json", plan)

	// job
	// if the concourse team was blank in source, we need to replace here based on the build response.
	if i.inRequest.Source.Team == "" {
		i.concourseTeam = i.concourseClient.Team(i.build.TeamName)
	}

	// use build information as team, pipeline and job names might not have been provided in source
	job, found, err := i.concourseTeam.Job(i.build.PipelineName, i.build.JobName)
	if err != nil {
		return nil, fmt.Errorf("error while fetching job information for pipeline/job '%s/%s': %s", i.build.PipelineName, i.build.JobName, err.Error())
	}
	if !found {
		return nil, fmt.Errorf("pipeline/job '%s/%s' not found while fetching pipeline/job information", i.build.PipelineName, i.build.JobName)
	}

	i.writeJsonFile("job", "json", job)
	i.writeJsonFile(i.addDetailedPostfixTo("job"), "json", job)
	i.writeJsonFile(i.addBuildNumberPostfixTo("job"), "json", job)

	// events
	err = i.renderEventsRepetitively()
	if err != nil {
		return nil, err
	}

	// K-V convenience files

	i.writeStringFile("team", i.build.TeamName)
	i.writeStringFile("pipeline", i.build.PipelineName)
	i.writeStringFile("job", i.build.JobName)
	i.writeStringFile("global_number", strconv.Itoa(i.build.ID))
	i.writeStringFile("job_number", i.build.Name)
	i.writeStringFile("started_time", strconv.Itoa(int(i.build.StartTime)))
	i.writeStringFile("ended_time", strconv.Itoa(int(i.build.EndTime)))
	i.writeStringFile("status", i.build.Status)
	i.writeStringFile("concourse_url", i.concourseUrl())
	i.writeStringFile("team_url", i.teamUrl())
	i.writeStringFile("pipeline_url", i.pipelineUrl())
	i.writeStringFile("job_url", i.jobUrl())
	i.writeStringFile("build_url", i.buildUrl())
	i.writeStringFile("concourse_build_resource_release", i.inRequest.ReleaseVersion)
	i.writeStringFile("concourse_build_resource_git_ref", i.inRequest.ReleaseGitRef)
	i.writeStringFile("concourse_build_resource_get_timestamp", strconv.Itoa(int(i.inRequest.GetTimestamp)))
	i.writeStringFile("concourse_build_resource_get_uuid", i.inRequest.GetUuid)
	i.writeStringFile("concourse_version", i.concourseInfo.Version)

	return &config.InResponse{
		Version: i.inRequest.Version,
		Metadata: []config.VersionMetadataField{
			{Name: "build_url", Value: i.buildUrl()},
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
	input.Source.ConcourseUrl = strings.TrimSuffix(input.Source.ConcourseUrl, "/")

	return inner{
		inRequest:       input,
		concourseClient: client,
		concourseTeam:   client.Team(input.Source.Team),
	}
}

func (i *inner) getConcourseInfo() error {
	var err error
	i.concourseInfo, err = i.concourseClient.GetInfo()
	if err != nil {
		return fmt.Errorf("could not get Concourse server information: %s", err.Error())
	}

	return nil
}

func (i *inner) getBuildId() error {
	var err error
	i.buildId, err = strconv.Atoi(i.inRequest.Version.BuildId)
	if err != nil {
		return fmt.Errorf("could not convert build id '%s' to an int: '%s", i.inRequest.Version.BuildId, err.Error())
	}

	return nil
}

func (i *inner) getBuild() error {
	var err error
	var found bool
	i.build, found, err = i.concourseClient.Build(i.inRequest.Version.BuildId)
	if err != nil {
		return fmt.Errorf("error while fetching build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	if !found {
		return fmt.Errorf("server could not find '%s/%s' while retrieving build '%s'", i.inRequest.Source.Pipeline, i.inRequest.Source.Job, i.inRequest.Version.BuildId)
	}

	return nil
}

func (i *inner) writeBuild() error {
	// TODO maybe actually handle the errors

	i.writeJsonFile("build", "json", i.build)
	i.writeJsonFile(i.addDetailedPostfixTo("build"), "json", i.build)
	i.writeJsonFile(i.addBuildNumberPostfixTo("build"), "json", i.build)

	return nil
}

func (i *inner) getResources() error {
	var err error
	var found bool
	i.resources, found, err = i.concourseClient.BuildResources(i.buildId)
	if err != nil {
		return fmt.Errorf("error while fetching resources for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	if !found {
		return fmt.Errorf("build '%s' not found while fetching resources", i.inRequest.Version.BuildId)
	}

	return nil
}

func (i *inner) writeResources() error {
	// TODO maybe actually handle the errors

	i.writeJsonFile("resources", "json", i.resources)
	i.writeJsonFile(i.addDetailedPostfixTo("resources"), "json", i.resources)
	i.writeJsonFile(i.addBuildNumberPostfixTo("resources"), "json", i.resources)

	return nil
}

func (i *inner) concourseUrl() string {
	return i.inRequest.Source.ConcourseUrl
}

func (i *inner) teamUrl() string {
	return fmt.Sprintf(
		"%s/teams/%s",
		i.concourseUrl(),
		i.build.TeamName,
	)
}

func (i *inner) pipelineUrl() string {
	return fmt.Sprintf(
		"%s/pipelines/%s",
		i.teamUrl(),
		i.build.PipelineName,
	)
}

func (i *inner) jobUrl() string {
	return fmt.Sprintf(
		"%s/jobs/%s",
		i.pipelineUrl(),
		i.build.JobName,
	)
}

func (i *inner) buildUrl() string {
	return fmt.Sprintf(
		"%s/builds/%s",
		i.jobUrl(),
		i.build.Name,
	)
}

func (i *inner) addDetailedPostfixTo(name string, ) string {
	return fmt.Sprintf(
		"%s_%s_%s_%s_%s",
		name,
		i.build.TeamName,
		i.build.PipelineName,
		i.build.JobName,
		i.build.Name)
}

func (i *inner) addBuildNumberPostfixTo(name string) string {
	return fmt.Sprintf("%s_%s", name, i.inRequest.Version.BuildId)
}

func (i *inner) writeJsonFile(filename string, extension string, object interface{}) error {
	builder := &strings.Builder{}

	err := json.NewEncoder(builder).Encode(object)
	if err != nil {
		return fmt.Errorf("could not encode response from server into '%s': %s", filename, err.Error())
	}

	getMetadataStr := fmt.Sprintf(
		`{"concourse_build_resource":{"release":"%s","git_ref":"%s","get_timestamp":%d,"concourse_version":"%s","get_uuid":"%s"},`,
		i.inRequest.ReleaseVersion,
		i.inRequest.ReleaseGitRef,
		i.inRequest.GetTimestamp,
		i.concourseInfo.Version,
		i.inRequest.GetUuid,
	)
	jsonStr := builder.String()
	jsonStr = strings.Replace(jsonStr, "{", getMetadataStr, 1)

	unadornedFilePath := fmt.Sprintf("%s.%s", filename, extension)
	err = ioutil.WriteFile(filepath.Join(i.inRequest.WorkingDirectory, unadornedFilePath), []byte(jsonStr), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (i *inner) writeStringFile(filename string, value string) error {
	file, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, filename))
	defer file.Close()
	if err != nil {
		return err
	}

	_, err = file.WriteString(value)

	return err
}

func (i *inner) renderEventsRepetitively() error {
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

	eventsFileDetailPostfixed, err := os.Create(filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.log", i.addDetailedPostfixTo("events"))))
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
