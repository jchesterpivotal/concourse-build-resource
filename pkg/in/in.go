package in

import (
	"encoding/json"
	"github.com/concourse/atc"
	"github.com/concourse/fly/eventstream"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"io"
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

type versionedResourceTypesWrapper struct {
	VersionedResourceTypes atc.VersionedResourceTypes `json:"versioned_resource_types"`
}

type inner struct {
	inRequest              *config.InRequest
	concourseClient        gc.Client
	concourseTeam          gc.Team
	concourseInfo          atc.Info
	build                  atc.Build
	resources              atc.BuildInputsOutputs
	plan                   atc.PublicBuildPlan
	job                    atc.Job
	versionedResourceTypes versionedResourceTypesWrapper
	buildId                int
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

	err = i.writeJsonFile("build", i.build)
	if err != nil {
		return nil, err
	}

	// resources
	err = i.getResources()
	if err != nil {
		return nil, err
	}

	err = i.writeJsonFile("resources", i.resources)
	if err != nil {
		return nil, err
	}

	// plan
	err = i.getPlan()
	if err != nil {
		return nil, err
	}

	err = i.writeJsonFile("plan", i.plan)
	if err != nil {
		return nil, err
	}

	// job
	err = i.getJob()
	if err != nil {
		return nil, err
	}

	err = i.writeJsonFile("job", i.job)
	if err != nil {
		return nil, err
	}

	// versioned resource types
	err = i.getVersionedResourceTypes()
	if err != nil {
		return nil, err
	}

	err = i.writeJsonFile("versioned_resource_types", i.versionedResourceTypes)
	if err != nil {
		return nil, err
	}

	// events
	err = i.writeEventsLogAndJson()
	if err != nil {
		return nil, err
	}

	// K-V convenience files
	err = i.writeConvenienceKeyValueFiles()
	if err != nil {
		return nil, err
	}

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

func (i *inner) getPlan() error {
	var err error
	var found bool
	i.plan, found, err = i.concourseClient.BuildPlan(i.buildId)
	if err != nil {
		return fmt.Errorf("error while fetching plan for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	if !found {
		return fmt.Errorf("build '%s' not found while fetching plan", i.inRequest.Version.BuildId)
	}

	return nil
}

func (i *inner) getJob() error {
	// if the concourse team was blank in source, we need to replace here based on the build response.
	if i.inRequest.Source.Team == "" {
		i.concourseTeam = i.concourseClient.Team(i.build.TeamName)
	}

	// use build information as team, pipeline and job names might not have been provided in source
	var err error
	var found bool
	i.job, found, err = i.concourseTeam.Job(i.build.PipelineName, i.build.JobName)
	if err != nil {
		return fmt.Errorf("error while fetching job information for pipeline/job '%s/%s': %s", i.build.PipelineName, i.build.JobName, err.Error())
	}
	if !found {
		return fmt.Errorf("pipeline/job '%s/%s' not found while fetching pipeline/job information", i.build.PipelineName, i.build.JobName)
	}

	return nil
}

func (i *inner) getVersionedResourceTypes() error {
	var err error
	var found bool
	verResTypes, found, err := i.concourseTeam.VersionedResourceTypes(i.build.PipelineName)
	if err != nil {
		return fmt.Errorf("error while fetching versioned resource type information for pipeline '%s': %s", i.build.PipelineName, err.Error())
	}
	if !found {
		return fmt.Errorf("pipeline '%s' not found while fetching versioned resource type information", i.build.PipelineName)
	}

	i.versionedResourceTypes = versionedResourceTypesWrapper{VersionedResourceTypes: verResTypes}

	return nil
}

// This method is gross because it needs to fetch events twice: once for the .log, once for the .json.
// This has to do with the clever way events are handled by Concourse and also to do with my unwillingness
// to completely and properly tease apart a smarter way to do this.
func (i *inner) writeEventsLogAndJson() error {
	//////////////////////// Rendered log ////////////////////////

	eventsForLogFile, err := i.concourseClient.BuildEvents(i.inRequest.Version.BuildId)
	// first, check if we are even authorised
	if err != nil && err.Error() == "not authorized" {
		log.Printf("was unauthorized to fetch events for build '%s', no logs will be written.", i.inRequest.Version.BuildId)
		return nil
	}
	if err != nil {
		return fmt.Errorf("error while fetching events for build '%s': '%s", i.inRequest.Version.BuildId, err.Error())
	}
	defer eventsForLogFile.Close()

	unadornedLogPath := filepath.Join(i.inRequest.WorkingDirectory, "events.log")
	eventsLogFile, err := os.Create(unadornedLogPath)
	if err != nil {
		return err
	}

	eventstream.Render(eventsLogFile, eventsForLogFile)
	eventsLogFile.Close()

	detailedLogPath := filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.log", i.addDetailedPostfixTo("events")))
	_, err = fileutils.CopyFile(unadornedLogPath, detailedLogPath)
	if err != nil {
		return err
	}

	numberedLogPath := filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.log", i.addBuildNumberPostfixTo("events")))
	_, err = fileutils.CopyFile(unadornedLogPath, numberedLogPath)
	if err != nil {
		return err
	}

	//////////////////////// JSON ////////////////////////

	eventsForJsonFile, err := i.concourseClient.BuildEvents(i.inRequest.Version.BuildId)
	defer eventsForJsonFile.Close()

	jsonBuilder := &strings.Builder{}
	jsonEnc := json.NewEncoder(jsonBuilder)
	for {
		ev, err := eventsForJsonFile.NextEvent()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		err = jsonEnc.Encode(ev)
		if err != nil {
			return err
		}
		jsonBuilder.WriteString(",")
	}
	jsonStr := fmt.Sprintf("[%s]", strings.TrimSuffix(jsonBuilder.String(), ","))
	unadornedJsonPath := filepath.Join(i.inRequest.WorkingDirectory, "events.json")
	err = ioutil.WriteFile(unadornedJsonPath, []byte(jsonStr), os.ModePerm)
	if err != nil {
		return err
	}

	detailedJsonPath := filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.json", i.addDetailedPostfixTo("events")))
	_, err = fileutils.CopyFile(unadornedJsonPath, detailedJsonPath)
	if err != nil {
		return err
	}

	numberedJsonPath := filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.json", i.addBuildNumberPostfixTo("events")))
	_, err = fileutils.CopyFile(unadornedJsonPath, numberedJsonPath)
	if err != nil {
		return err
	}

	return nil
}

func (i *inner) writeConvenienceKeyValueFiles() error {
	// TODO maybe actually handle the errors

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

func (i *inner) writeJsonFile(filename string, object interface{}) error {
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

	unadornedJsonPath := filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.json", filename))
	err = ioutil.WriteFile(unadornedJsonPath, []byte(jsonStr), os.ModePerm)
	if err != nil {
		return err
	}

	detailedJsonPath := filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.json", i.addDetailedPostfixTo(filename)))
	_, err = fileutils.CopyFile(unadornedJsonPath, detailedJsonPath)
	if err != nil {
		return err
	}

	numberedJsonPath := filepath.Join(i.inRequest.WorkingDirectory, fmt.Sprintf("%s.json", i.addBuildNumberPostfixTo(filename)))
	_, err = fileutils.CopyFile(unadornedJsonPath, numberedJsonPath)
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
