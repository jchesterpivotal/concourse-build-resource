package in

import (
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"net/http"
	"time"
	"fmt"
	"os"
	"path/filepath"
	"io/ioutil"
	"log"
	"net/url"
)

func In(input *config.InRequest) (*config.InResponse, error) {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	concourseUrl, err := url.Parse(input.Source.ConcourseUrl)
	if err != nil {
		return &config.InResponse{}, fmt.Errorf("could not parse concourse_url '%s': '%s", input.Source.ConcourseUrl, err.Error())
	}

	getBuildInfo(client, concourseUrl, input, "resources")
	if err != nil {
		return &config.InResponse{}, fmt.Errorf("could not fetch build resources: %s", err.Error())
	}

	getBuildInfo(client, concourseUrl, input, "plan")
	if err != nil {
		return &config.InResponse{}, fmt.Errorf("could not fetch build plan: %s", err.Error())
	}

	getBuildInfo(client, concourseUrl, input, "events")
	if err != nil {
		return &config.InResponse{}, fmt.Errorf("could not fetch build plan: %s", err.Error())
	}



	return &config.InResponse{
		Version: input.Version,
		Metadata: []config.VersionMetadataField{
			{Name: "name", Value: "value"},
		},
	}, nil
}

func getBuildInfo(client *http.Client, concourseUrl *url.URL, input *config.InRequest, subpath string) error {
	path := fmt.Sprintf("/api/v1/builds/%s/%s", input.Version.BuildId, subpath)

	concourseUrl.Path = path
	log.Printf("fetching %s", concourseUrl.String())
	response, err := client.Get(concourseUrl.String())
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.json", subpath)
	resFile, err := os.Create(filepath.Join(input.WorkingDirectory, filename))
	defer resFile.Close()
	if err != nil {
		return err
	}

	respBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	resFile.Write(respBytes)

	return nil
}
