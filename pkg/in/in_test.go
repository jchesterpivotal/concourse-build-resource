package in_test

import (
	"testing"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/onsi/gomega"
	fakes "github.com/concourse/go-concourse/concourse/concoursefakes"
	"github.com/concourse/go-concourse/concourse/eventstream/eventstreamfakes"

	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"github.com/jchesterpivotal/concourse-build-resource/pkg/in"

	"github.com/concourse/atc"

	"os"
	"io"
	"fmt"
	"io/ioutil"
)

func TestInPkg(t *testing.T) {
	gt := gomega.NewGomegaWithT(t)

	err := os.Mkdir("build", os.ModeDir|os.ModePerm)
	if err != nil {
		gt.Expect(err).NotTo(gomega.MatchError("build: file exists"))
	}

	spec.Run(t, "pkg/in", func(t *testing.T, when spec.G, it spec.S) {
		when("given valid inputs", func() {
			gt := gomega.NewGomegaWithT(t)
			faketeam := new(fakes.FakeTeam)
			fakeclient := new(fakes.FakeClient)
			fakeclient.TeamReturns(faketeam)
			fakeeventstream := &eventstreamfakes.FakeEventStream{}
			var response *config.InResponse
			var err error

			it.Before(func() {
				fakeclient.BuildReturns(atc.Build{
					ID:           999,
					Name:         "111",
					TeamName:     "team",
					PipelineName: "pipeline",
					JobName:      "job",
					StartTime:    1010101010,
					EndTime:      1191919191,
					Status:       "succeeded",
					APIURL:       "/api/v1/builds/999",
				}, true, nil)
				fakeclient.BuildResourcesReturns(atc.BuildInputsOutputs{
					Inputs:  []atc.PublicBuildInput{},
					Outputs: []atc.VersionedResource{},
				}, true, nil)
				fakeclient.BuildPlanReturns(atc.PublicBuildPlan{}, true, nil)
				fakeeventstream.NextEventReturns(nil, io.EOF)
				fakeclient.BuildEventsReturns(fakeeventstream, nil)

				inner := in.NewInnerUsingClient(&config.InRequest{
					Source: config.Source{
						ConcourseUrl: "https://example.com",
						Team:         "team",
						Pipeline:     "pipeline",
						Job:          "job",
					},
					Version:          config.Version{BuildId: "999"},
					Params:           config.InParams{},
					WorkingDirectory: "build",
				}, fakeclient)
				response, err = inner.In()
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it("returns the version it was given", func() {
				gt.Expect(response.Version).To(gomega.Equal(config.Version{BuildId: "999"}))
			})

			it("returns metadata with the team, pipeline, job and job number", func() {
				gt.Expect(response.Metadata).To(gomega.ContainElement(config.VersionMetadataField{Name:"team", Value: "team"}))
				gt.Expect(response.Metadata).To(gomega.ContainElement(config.VersionMetadataField{Name:"pipeline", Value: "pipeline"}))
				gt.Expect(response.Metadata).To(gomega.ContainElement(config.VersionMetadataField{Name:"job", Value: "job"}))
				gt.Expect(response.Metadata).To(gomega.ContainElement(config.VersionMetadataField{Name:"name", Value: "111"}))
			})

			it("writes out the build.json file", func() {
				gt.Expect(AFileExistsContaining("build/build.json", `"api_url":"/api/v1/builds/999"`, gt)).To(gomega.BeTrue())
			})

			it("writes out the build-<team>-<pipeline>-<job>-<build number>.json file", func() {
				gt.Expect(AFileExistsContaining("build/build-team-pipeline-job-111.json", `"api_url":"/api/v1/builds/999"`, gt)).To(gomega.BeTrue())
			})

			it("writes out the build-<global build number>.json file", func() {
				gt.Expect(AFileExistsContaining("build/build-999.json", `"api_url":"/api/v1/builds/999"`, gt)).To(gomega.BeTrue())
			})

			it("writes out the resources.json file", func() {
				gt.Expect(AFileExistsContaining("build/resources.json", `"inputs":[`, gt)).To(gomega.BeTrue())
			})

			it("writes out the resources-<team>-<pipeline>-<job>-<build number>.json file", func() {
				gt.Expect(AFileExistsContaining("build/resources-team-pipeline-job-111.json", `"inputs":[`, gt)).To(gomega.BeTrue())
			})

			it("writes out the resources-<global build number>.json file", func() {
				gt.Expect(AFileExistsContaining("build/resources-999.json", `"inputs":[`, gt)).To(gomega.BeTrue())
			})

			it("writes out the plan.json file", func() {
				gt.Expect(AFileExistsContaining("build/plan.json", `"plan":`, gt)).To(gomega.BeTrue())
			})

			it("writes out the plan-<team>-<pipeline>-<job>-<build number>.json file", func() {
				gt.Expect(AFileExistsContaining("build/plan-team-pipeline-job-111.json", `"plan":`, gt)).To(gomega.BeTrue())
			})

			it("writes out the plan-<global build number>.json file", func() {
				gt.Expect(AFileExistsContaining("build/plan-999.json", `"plan":`, gt)).To(gomega.BeTrue())
			})

			// TODO: Tests for logs are less rigorous because mocking up the event streams is a PITA.
			it("writes out the events.log", func() {
				gt.Expect("build/events.log").To(gomega.BeAnExistingFile())
			})

			it("writes out the events-<team>-<pipeline>-<job>-<build number>.log", func() {
				gt.Expect("build/events-team-pipeline-job-111.log").To(gomega.BeAnExistingFile())
			})

			it("writes out the events-<global build number>.log", func() {
				gt.Expect("build/events-999.log").To(gomega.BeAnExistingFile())
			})

			it("writes out build/team", func() {
				gt.Expect(AFileExistsContaining("build/team", "team", gt)).To(gomega.BeTrue())
			})

			it("writes out build/pipeline", func() {
				gt.Expect(AFileExistsContaining("build/pipeline", "pipeline", gt)).To(gomega.BeTrue())
			})

			it("writes out build/job", func() {
				gt.Expect(AFileExistsContaining("build/job", "job", gt)).To(gomega.BeTrue())
			})

			it("writes out build/job-number", func() {
				gt.Expect(AFileExistsContaining("build/job-number", "111", gt)).To(gomega.BeTrue())
			})

			it("writes out build/global-number", func() {
				gt.Expect(AFileExistsContaining("build/global-number", "999", gt)).To(gomega.BeTrue())
			})

			it("writes out build/started-time", func() {
				gt.Expect(AFileExistsContaining("build/started-time", "1010101010", gt)).To(gomega.BeTrue())
			})

			it("writes out build/ended-time", func() {
				gt.Expect(AFileExistsContaining("build/ended-time", "1191919191", gt)).To(gomega.BeTrue())
			})

			it("writes out build/status", func() {
				gt.Expect(AFileExistsContaining("build/status", "succeeded", gt)).To(gomega.BeTrue())
			})

			it("writes out build/url", func() {
				gt.Expect(AFileExistsContaining("build/url", "https://example.com/teams/team/pipelines/pipeline/jobs/job/builds/111", gt)).To(gomega.BeTrue())
			})

		}, spec.Nested())

		when("something goes wrong", func() {
			gt := gomega.NewGomegaWithT(t)
			faketeam := new(fakes.FakeTeam)
			fakeclient := new(fakes.FakeClient)
			fakeclient.TeamReturns(faketeam)
			var response *config.InResponse
			var err error

			when("data cannot be retrieved due to an error", func() {
				it.Before(func() {
					fakeclient.BuildReturns(atc.Build{ID: 111}, false, fmt.Errorf("test error"))

					inner := in.NewInnerUsingClient(&config.InRequest{
						Source:           config.Source{},
						Version:          config.Version{BuildId: "111"},
						Params:           config.InParams{},
						WorkingDirectory: "build",
					}, fakeclient)
					response, err = inner.In()
				})

				it("returns an error", func() {
					gt.Expect(err.Error()).To(gomega.ContainSubstring("error while fetching build"))
					gt.Expect(response).To(gomega.BeNil())
				})
			}, spec.Nested())

			when("the team, pipeline or job are not found", func() {
				it.Before(func() {
					fakeclient.BuildReturns(atc.Build{ID: 111}, false, nil)

					inner := in.NewInnerUsingClient(&config.InRequest{
						Source:           config.Source{Pipeline: "pipeline", Job: "job"},
						Version:          config.Version{BuildId: "111"},
						Params:           config.InParams{},
						WorkingDirectory: "build",
					}, fakeclient)
					response, err = inner.In()
				})

				it("returns an error", func() {
					gt.Expect(err.Error()).To(gomega.ContainSubstring("server could not find 'pipeline/job' while retrieving build '111'"))
					gt.Expect(response).To(gomega.BeNil())
				})
			}, spec.Nested())
		}, spec.Nested())

		when("build ID is defined, but is not a valid number", func() {
			gt := gomega.NewGomegaWithT(t)
			faketeam := new(fakes.FakeTeam)
			fakeclient := new(fakes.FakeClient)
			fakeclient.TeamReturns(faketeam)
			var response *config.InResponse
			var err error

			it.Before(func() {
				fakeclient.BuildReturns(atc.Build{ID: 111}, true, nil)

				inner := in.NewInnerUsingClient(&config.InRequest{Version: config.Version{BuildId: "not numerical"}}, fakeclient)
				response, err = inner.In()
			})

			it("returns an error", func() {
				gt.Expect(err.Error()).To(gomega.ContainSubstring("could not convert build id 'not numerical' to an int:"))
				gt.Expect(response).To(gomega.BeNil())
			})
		}, spec.Nested())
	}, spec.Report(report.Terminal{}))

	gt.Expect(os.RemoveAll("build")).To(gomega.Succeed())
}

func AFileExistsContaining(filepath string, substring string, gt *gomega.GomegaWithT) bool {
	gt.Expect(filepath).To(gomega.BeAnExistingFile())
	contents, err := ioutil.ReadFile(filepath)
	gt.Expect(err).NotTo(gomega.HaveOccurred())
	gt.Expect(string(contents)).To(gomega.ContainSubstring(substring))

	return true
}
