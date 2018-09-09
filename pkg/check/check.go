package check

import (
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"log"
	"strings"

	"github.com/concourse/atc"
	gc "github.com/concourse/go-concourse/concourse"

	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Checker interface {
	Check() (*config.CheckResponse, error)
}

type checker struct {
	checkRequest    *config.CheckRequest
	concourseClient gc.Client
	concourseTeam   gc.Team
}

const defaultVersionPageSize = 100

func (c checker) Check() (*config.CheckResponse, error) {
	if c.checkRequest.Source.EnableTracing {
		log.Printf("Received CheckRequest: %+v", c.checkRequest)
	}

	version := c.checkRequest.Version
	initialBuildId := c.checkRequest.Source.InitialBuildId
	versionPageSize := c.checkRequest.Source.FetchPageSize
	if versionPageSize == 0 {
		versionPageSize = defaultVersionPageSize
	}

	if version.BuildId == "" && initialBuildId > 0 {
		return &config.CheckResponse{{BuildId: strconv.Itoa(initialBuildId)}}, nil
	}

	if version.BuildId == "" {
		builds, err := c.getBuilds(gc.Page{Limit: 1})
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

	builds, err := c.getBuilds(gc.Page{Since: buildId, Limit: versionPageSize})
	if err != nil {
		return nil, err
	}

	if len(builds) == 0 { // there are no builds at all
		return &config.CheckResponse{}, nil
	}

	newBuilds := make(config.CheckResponse, 0)
	for _, b := range builds {
		if b.Status != string(atc.StatusStarted) && b.Status != string(atc.StatusPending) {
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
	input.Source.ConcourseUrl = strings.TrimSuffix(input.Source.ConcourseUrl, "/")

	return checker{
		checkRequest:    input,
		concourseClient: client,
		concourseTeam:   client.Team(input.Source.Team),
	}
}

func (c checker) getBuilds(initialPage gc.Page) ([]atc.Build, error) {
	concourseUrl := c.checkRequest.Source.ConcourseUrl
	team := c.checkRequest.Source.Team
	pipeline := c.checkRequest.Source.Pipeline
	job := c.checkRequest.Source.Job
	builds := make([]atc.Build, 0)
	pageBuilds := make([]atc.Build, 0)
	var pagination gc.Pagination
	var found bool
	var err error

	// latest version only case
	if initialPage.Limit == 1 {
		if job == "" && pipeline == "" && team == "" {
			builds, _, err = c.concourseClient.Builds(initialPage)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve builds for concourse URL '%s': %s", concourseUrl, err.Error())
			}
		} else if job == "" && pipeline == "" {
			builds, _, err = c.concourseTeam.Builds(initialPage)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve builds for team '%s': %s", team, err.Error())
			}
		} else if job == "" {
			builds, _, found, err = c.concourseTeam.PipelineBuilds(pipeline, initialPage)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve builds for pipeline '%s': %s", pipeline, err.Error())
			}
			if !found {
				return nil, fmt.Errorf("server could not find pipeline '%s'", pipeline)
			}
		} else {
			builds, _, found, err = c.concourseTeam.JobBuilds(pipeline, job, initialPage)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve builds for pipeline/job '%s/%s': %s", pipeline, job, err.Error())
			}
			if !found {
				return nil, fmt.Errorf("server could not find pipeline/job '%s/%s'", pipeline, job)
			}
		}
	} else {
		// versions-since or initial_build_id cases
		if job == "" && pipeline == "" && team == "" {
			_, pagination, err = c.concourseClient.Builds(initialPage)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve builds for concourse URL '%s': %s", concourseUrl, err.Error())
			}

			for pagination.Previous != nil {
				pageBuilds, pagination, err = c.concourseClient.Builds(*pagination.Previous)
				if err != nil {
					return nil, fmt.Errorf("could not retrieve builds for concourse URL '%s': %s", concourseUrl, err.Error())
				}

				builds = append(pageBuilds, builds...)
			}
		} else if job == "" && pipeline == "" {
			_, pagination, err = c.concourseTeam.Builds(initialPage)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve builds for team '%s': %s", team, err.Error())
			}

			for pagination.Previous != nil {
				pageBuilds, pagination, err = c.concourseTeam.Builds(*pagination.Previous)
				if err != nil {
					return nil, fmt.Errorf("could not retrieve builds for concourse URL '%s': %s", concourseUrl, err.Error())
				}

				builds = append(pageBuilds, builds...)
			}
		} else if job == "" {
			_, pagination, found, err = c.concourseTeam.PipelineBuilds(pipeline, initialPage)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve builds for pipeline '%s': %s", pipeline, err.Error())
			}
			if !found {
				return nil, fmt.Errorf("server could not find pipeline '%s'", pipeline)
			}

			for pagination.Previous != nil {
				pageBuilds, pagination, found, err = c.concourseTeam.PipelineBuilds(pipeline, *pagination.Previous)
				if err != nil {
					return nil, fmt.Errorf("could not retrieve builds for concourse URL '%s': %s", concourseUrl, err.Error())
				}
				if !found {
					return nil, fmt.Errorf("server could not find pipeline '%s'", pipeline)
				}

				builds = append(pageBuilds, builds...)
			}
		} else {
			_, pagination, found, err = c.concourseTeam.JobBuilds(pipeline, job, initialPage)
			if err != nil {
				return nil, fmt.Errorf("could not retrieve builds for pipeline/job '%s/%s': %s", pipeline, job, err.Error())
			}
			if !found {
				return nil, fmt.Errorf("server could not find pipeline/job '%s/%s'", pipeline, job)
			}

			for pagination.Previous != nil {
				pageBuilds, pagination, found, err = c.concourseTeam.JobBuilds(pipeline, job, *pagination.Previous)
				if err != nil {
					return nil, fmt.Errorf("could not retrieve builds for pipeline/job '%s/%s': %s", pipeline, job, err.Error())
				}
				if !found {
					return nil, fmt.Errorf("server could not find pipeline/job '%s/%s'", pipeline, job)
				}

				builds = append(pageBuilds, builds...)
			}
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
