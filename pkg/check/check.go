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
	CheckRequest    *config.CheckRequest
	ConcourseClient gc.Client
	ConcourseTeam   gc.Team
}

func (c checker) Check() (*config.CheckResponse, error) {
	pipeline := c.CheckRequest.Source.Pipeline
	job := c.CheckRequest.Source.Job
	version := c.CheckRequest.Version

	if version.BuildId == "" {
		// first run
		builds, _, found, err := c.ConcourseTeam.JobBuilds(pipeline, job, gc.Page{Limit: 1})
		if err != nil {
			return nil, fmt.Errorf("could not retrieve builds for '%s/%s: %s", pipeline, job, err.Error())
		}
		if !found {
			return nil, fmt.Errorf("could not find any builds for '%s/%s", pipeline, job)
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

		builds, _, found, err := c.ConcourseTeam.JobBuilds(pipeline, job, gc.Page{})
		if err != nil {
			return nil, fmt.Errorf("could not retrieve builds for '%s/%s: %s", pipeline, job, err.Error())
		}
		if !found {
			return nil, fmt.Errorf("could not find any builds for '%s/%s", pipeline, job)
		}

		newBuilds := make(config.CheckResponse, 0)

		for _, b := range builds {
			if b.ID > buildId && b.Status != string(atc.StatusStarted) && b.Status != string(atc.StatusPending) {
				newBuildId := strconv.Itoa(b.ID)
				newBuilds = append(newBuilds, config.Version{BuildId: newBuildId})
			}
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
	team := concourse.Team(input.Source.Team)

	return checker{
		CheckRequest: input,
		ConcourseClient: concourse,
		ConcourseTeam:team,
	}
}
