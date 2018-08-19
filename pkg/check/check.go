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

func Check(input *config.CheckRequest) (*config.CheckResponse, error) {
	tr := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	concourse := gc.NewClient(input.Source.ConcourseUrl, client, false)

	team := concourse.Team(input.Source.Team)

	if input.Version.BuildId == "" {
		// first run
		builds, _, found, err := team.JobBuilds(input.Source.Pipeline, input.Source.Job, gc.Page{Limit: 1})
		if err != nil {
			return nil, fmt.Errorf("could not retrieve builds for '%s/%s: %s", input.Source.Pipeline, input.Source.Job, err.Error())
		}
		if !found {
			return nil, fmt.Errorf("could not find any builds for '%s/%s", input.Source.Pipeline, input.Source.Job)
		}

		buildId := strconv.Itoa(builds[0].ID)
		return &config.CheckResponse{
			config.Version{BuildId:buildId},
		}, nil
	} else {
		buildId, err := strconv.Atoi(input.Version.BuildId)
		if err != nil {
			return nil, fmt.Errorf("could not convert build id '%s' to an int: '%s", input.Version.BuildId, err.Error())
		}

		builds, _, found, err := team.JobBuilds(input.Source.Pipeline, input.Source.Job, gc.Page{})
		if err != nil {
			return nil, fmt.Errorf("could not retrieve builds for '%s/%s: %s", input.Source.Pipeline, input.Source.Job, err.Error())
		}
		if !found {
			return nil, fmt.Errorf("could not find any builds for '%s/%s", input.Source.Pipeline, input.Source.Job)
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
