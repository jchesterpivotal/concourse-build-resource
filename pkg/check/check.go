package check

import (
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"net/http"
	"time"
	"fmt"
	gc "github.com/concourse/go-concourse/concourse"
	"crypto/tls"
	"strconv"
	"github.com/concourse/atc"
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
	pipeline := c.checkRequest.Source.Pipeline
	job := c.checkRequest.Source.Job
	version := c.checkRequest.Version

	if version.BuildId == "" {
		// first run
		builds, _, found, err := c.concourseTeam.JobBuilds(pipeline, job, gc.Page{Limit: 1})
		if err != nil {
			return nil, fmt.Errorf("could not retrieve builds for '%s/%s': %s", pipeline, job, err.Error())
		}
		if !found {
			return nil, fmt.Errorf("server could not find '%s/%s'", pipeline, job)
		}

		buildId := strconv.Itoa(builds[0].ID)
		return &config.CheckResponse{
			config.Version{BuildId: buildId},
		}, nil
	} else {
		buildId, err := strconv.Atoi(version.BuildId)
		if err != nil {
			return nil, fmt.Errorf("could not convert build id '%s' to an int: '%s", version.BuildId, err.Error())
		}

		builds, _, found, err := c.concourseTeam.JobBuilds(pipeline, job, gc.Page{})
		if err != nil {
			return nil, fmt.Errorf("could not retrieve builds for '%s/%s': %s", pipeline, job, err.Error())
		}
		if !found {
			return nil, fmt.Errorf("server could not find '%s/%s'", pipeline, job)
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

	return &config.CheckResponse{}, nil
}

func NewChecker(input *config.CheckRequest) Checker {
	tr := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	concourse := gc.NewClient(input.Source.ConcourseUrl, client, false)

	return NewCheckerUsingClient(input, concourse)
}

func NewCheckerUsingClient(input *config.CheckRequest, client gc.Client) Checker {
	return checker{
		checkRequest:    input,
		concourseClient: client,
		concourseTeam:   client.Team(input.Source.Team),
	}
}
