package check

import (
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"

	"github.com/concourse/atc"
	gc "github.com/concourse/go-concourse/concourse"

	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	singleJobPageSize        int = 1
	defaultConcoursePageSize int = 50
)

type Checker interface {
	Check() (*config.CheckResponse, error)
}

type checker struct {
	checkRequest    *config.CheckRequest
	concourseClient gc.Client
	concourseTeam   gc.Team
}

func (c checker) Check() (*config.CheckResponse, error) {
	version := c.checkRequest.Version

	if version.BuildId == "" {
		builds, err := c.getBuilds(singleJobPageSize)
		if err != nil {
			return nil, err
		}

		buildId := strconv.Itoa(builds[0].ID)
		return &config.CheckResponse{
			config.Version{BuildId: buildId},
		}, nil
	}

	buildId, err := strconv.Atoi(version.BuildId)
	if err != nil {
		return nil, fmt.Errorf("could not convert build id '%s' to an int: '%s", version.BuildId, err.Error())
	}

	builds, err := c.getBuilds(defaultConcoursePageSize)
	if err != nil {
		return nil, err
	}

	if len(builds) == 0 { // there are no builds at all
		return &config.CheckResponse{}, nil
	}

	newBuilds := make(config.CheckResponse, 0)
	for _, b := range builds {
		if b.ID > buildId && b.Status != string(atc.StatusStarted) && b.Status != string(atc.StatusPending) {
			newBuildId := strconv.Itoa(b.ID)
			newBuilds = append(newBuilds, config.Version{BuildId: newBuildId})
		}
	}

	if len(newBuilds) == 0 { // there were no new builds
		return &config.CheckResponse{version}, nil
	}

	return &newBuilds, nil
}

func NewChecker(input *config.CheckRequest) Checker {
	tr := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	concourse := gc.NewClient(input.Source.ConcourseUrl, client, input.Source.EnableTracing)

	return NewCheckerUsingClient(input, concourse)
}

func NewCheckerUsingClient(input *config.CheckRequest, client gc.Client) Checker {
	return checker{
		checkRequest:    input,
		concourseClient: client,
		concourseTeam:   client.Team(input.Source.Team),
	}
}

func (c checker) getBuilds(limit int) ([]atc.Build, error) {
	concourseUrl := c.checkRequest.Source.ConcourseUrl
	team := c.checkRequest.Source.Team
	pipeline := c.checkRequest.Source.Pipeline
	job := c.checkRequest.Source.Job
	var builds []atc.Build
	var found bool
	var err error

	if job == "" && pipeline == "" && team == "" {
		builds, _, err = c.concourseClient.Builds(gc.Page{Limit: limit})
		if err != nil {
			return nil, fmt.Errorf("could not retrieve builds for '%s': %s", concourseUrl, err.Error())
		}
	} else if job == "" && pipeline == "" {
		builds, _, err = c.concourseTeam.Builds(gc.Page{Limit: limit})
		if err != nil {
			return nil, fmt.Errorf("could not retrieve builds for '%s': %s", team, err.Error())
		}
	} else if job == "" {
		builds, _, found, err = c.concourseTeam.PipelineBuilds(pipeline, gc.Page{Limit: limit})
		if err != nil {
			return nil, fmt.Errorf("could not retrieve builds for '%s': %s", pipeline, err.Error())
		}
		if !found {
			return nil, fmt.Errorf("server could not find '%s'", pipeline)
		}
	} else {
		builds, _, found, err = c.concourseTeam.JobBuilds(pipeline, job, gc.Page{Limit: limit})
		if err != nil {
			return nil, fmt.Errorf("could not retrieve builds for '%s/%s': %s", pipeline, job, err.Error())
		}
		if !found {
			return nil, fmt.Errorf("server could not find '%s/%s'", pipeline, job)
		}
	}

	return reverseOrder(builds), nil
}

func reverseOrder(builds []atc.Build) []atc.Build {
	for i, j := 0, len(builds)-1; i < j; i, j = i+1, j-1 {
		builds[i], builds[j] = builds[j], builds[i]
	}
	return builds
}
