package check

import (
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"net/http"
	"time"
	"encoding/json"
	"fmt"
)

type builds []map[string]string

func Check(input *config.CheckRequest) (*config.CheckResponse, error) {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	listBuildsUrl := fmt.Sprintf(
		"%s/api/v1/teams/%s/pipelines/%s/jobs/%s/builds",
		input.Source.ConcourseUrl,
		input.Source.Team,
		input.Source.Pipeline,
		input.Source.Job,
	)

	listBuildsResponse, err := client.Get(listBuildsUrl)
	if err != nil {
		return nil, err
	}
	var listBuilds builds
	json.NewDecoder(listBuildsResponse.Body).Decode(listBuilds)

	newBuilds := make(config.CheckResponse, 0)
	for _, b := range listBuilds {
		if b["id"] > input.Version.BuildId {
			newBuilds = append(newBuilds, config.Version{BuildId: b["id"]})
		}
	}

	return &newBuilds, nil
}