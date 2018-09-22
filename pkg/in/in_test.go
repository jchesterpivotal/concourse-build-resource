package in_test

import (
	fakes "github.com/concourse/go-concourse/concourse/concoursefakes"
	"github.com/concourse/go-concourse/concourse/eventstream/eventstreamfakes"
	"github.com/nu7hatch/gouuid"
	"github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"testing"

	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"
	"github.com/jchesterpivotal/concourse-build-resource/pkg/in"

	"github.com/concourse/atc"

	"fmt"
	"io"
	"io/ioutil"
	"os"
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
				fakeclient.GetInfoReturns(atc.Info{
					Version:       "3.99.11",
					WorkerVersion: "",
				}, nil)
			})

			when("the concourse, team, pipeline and job are specified", func() {
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
					faketeam.JobReturns(atc.Job{}, true, nil)
					faketeam.VersionedResourceTypesReturns(atc.VersionedResourceTypes{{ResourceType: atc.ResourceType{CheckEvery: "10m"}}}, true, nil)
					fakeeventstream.NextEventReturns(nil, io.EOF)
					fakeclient.BuildEventsReturns(fakeeventstream, nil)

					uuid, err := uuid.ParseHex("96d7128f-bacf-4f60-9ffd-1a9ca4c9e1d7")
					gt.Expect(err).NotTo(gomega.HaveOccurred())
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
						ReleaseVersion:   "v0.99.11",
						ReleaseGitRef:    "abcdef1234567890",
						GetTimestamp:     1234567890,
						GetUuid:          uuid.String(),
					}, fakeclient)
					response, err = inner.In()
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("returns the version it was given", func() {
					gt.Expect(response.Version).To(gomega.Equal(config.Version{BuildId: "999"}))
				})

				it("returns metadata with the build URL", func() {
					gt.Expect(response.Metadata).To(gomega.ContainElement(config.VersionMetadataField{Name: "build_url", Value: "https://example.com/teams/team/pipelines/pipeline/jobs/job/builds/111"}))
				})

				it("writes out the build.json file", func() {
					gt.Expect(AFileExistsContaining("build/build.json", `"api_url":"/api/v1/builds/999"`, gt)).To(gomega.BeTrue())
				})

				it("writes out the build-<team>_<pipeline>_<job>_<build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/build_team_pipeline_job_111.json", `"api_url":"/api/v1/builds/999"`, gt)).To(gomega.BeTrue())
				})

				it("writes out the build_<global build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/build_999.json", `"api_url":"/api/v1/builds/999"`, gt)).To(gomega.BeTrue())
				})

				it("writes out the resources.json file", func() {
					gt.Expect(AFileExistsContaining("build/resources.json", `"inputs":[`, gt)).To(gomega.BeTrue())
				})

				it("writes out the resources_<team>_<pipeline>_<job>_<build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/resources_team_pipeline_job_111.json", `"inputs":[`, gt)).To(gomega.BeTrue())
				})

				it("writes out the resources_<global build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/resources_999.json", `"inputs":[`, gt)).To(gomega.BeTrue())
				})

				it("writes out the plan.json file", func() {
					gt.Expect(AFileExistsContaining("build/plan.json", `"plan":`, gt)).To(gomega.BeTrue())
				})

				it("writes out the plan_<team>_<pipeline>_<job>_<build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/plan_team_pipeline_job_111.json", `"plan":`, gt)).To(gomega.BeTrue())
				})

				it("writes out the plan_<global build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/plan_999.json", `"plan":`, gt)).To(gomega.BeTrue())
				})

				it("writes out the job.json file", func() {
					gt.Expect(AFileExistsContaining("build/job.json", `"finished_build":`, gt)).To(gomega.BeTrue())
				})

				it("writes out the job_<build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/job_999.json", `"finished_build":`, gt)).To(gomega.BeTrue())
				})

				it("writes out the job_<team>_<pipeline>_<job>_<build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/job_team_pipeline_job_111.json", `"finished_build":`, gt)).To(gomega.BeTrue())
				})

				it("writes out the versioned_resource_types.json file", func() {
					gt.Expect(AFileExistsContaining("build/versioned_resource_types.json", `"check_every":`, gt)).To(gomega.BeTrue())
				})

				it("writes out the versioned_resource_types_<global build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/versioned_resource_types_999.json", `"check_every":`, gt)).To(gomega.BeTrue())
				})

				it("writes out the versioned_resource_types_<team>_<pipeline>_<job>_<build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/versioned_resource_types_team_pipeline_job_111.json", `"check_every":`, gt)).To(gomega.BeTrue())
				})

				it("adds resource version metadata to JSON files", func() {
					gt.Expect(AFileExistsContaining("build/build.json", `"concourse_build_resource":{"release":"v0.99.11","git_ref":"abcdef1234567890","get_timestamp":1234567890,"concourse_version":"3.99.11","get_uuid":"96d7128f-bacf-4f60-9ffd-1a9ca4c9e1d7"},`, gt)).To(gomega.BeTrue())
					gt.Expect(AFileExistsContaining("build/plan.json", `"concourse_build_resource":{"release":"v0.99.11","git_ref":"abcdef1234567890","get_timestamp":1234567890,"concourse_version":"3.99.11","get_uuid":"96d7128f-bacf-4f60-9ffd-1a9ca4c9e1d7"},`, gt)).To(gomega.BeTrue())
					gt.Expect(AFileExistsContaining("build/resources.json", `"concourse_build_resource":{"release":"v0.99.11","git_ref":"abcdef1234567890","get_timestamp":1234567890,"concourse_version":"3.99.11","get_uuid":"96d7128f-bacf-4f60-9ffd-1a9ca4c9e1d7"},`, gt)).To(gomega.BeTrue())
					gt.Expect(AFileExistsContaining("build/job.json", `"concourse_build_resource":{"release":"v0.99.11","git_ref":"abcdef1234567890","get_timestamp":1234567890,"concourse_version":"3.99.11","get_uuid":"96d7128f-bacf-4f60-9ffd-1a9ca4c9e1d7"},`, gt)).To(gomega.BeTrue())
				})

				it("writes out a concourse_build_resource_release file", func() {
					gt.Expect(AFileExistsContaining("build/concourse_build_resource_release", "v0.99.11", gt)).To(gomega.BeTrue())
				})

				it("writes out a concourse_build_resource_git_ref file", func() {
					gt.Expect(AFileExistsContaining("build/concourse_build_resource_git_ref", "abcdef1234567890", gt)).To(gomega.BeTrue())
				})

				it("writes out a concourse_build_resource_get_timestamp file", func() {
					gt.Expect(AFileExistsContaining("build/concourse_build_resource_get_timestamp", "1234567890", gt)).To(gomega.BeTrue())
				})

				it("writes out a concourse_build_resource_get_uuid file", func() {
					gt.Expect(AFileExistsContaining("build/concourse_build_resource_get_uuid", "96d7128f-bacf-4f60-9ffd-1a9ca4c9e1d7", gt)).To(gomega.BeTrue())
				})

				it("writes out a concourse_version file", func() {
					gt.Expect(AFileExistsContaining("build/concourse_version", "3.99.11", gt)).To(gomega.BeTrue())
				})

				// TODO: Tests for logs are less rigorous because mocking up the event streams is a PITA.
				it("writes out the events.log", func() {
					gt.Expect("build/events.log").To(gomega.BeAnExistingFile())
				})

				it("writes out the events_<team>_<pipeline>_<job>_<build number>.log", func() {
					gt.Expect("build/events_team_pipeline_job_111.log").To(gomega.BeAnExistingFile())
				})

				it("writes out the events_<global build number>.log", func() {
					gt.Expect("build/events_999.log").To(gomega.BeAnExistingFile())
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

				it("writes out build/job_number", func() {
					gt.Expect(AFileExistsContaining("build/job_number", "111", gt)).To(gomega.BeTrue())
				})

				it("writes out build/global_number", func() {
					gt.Expect(AFileExistsContaining("build/global_number", "999", gt)).To(gomega.BeTrue())
				})

				it("writes out build/started_time", func() {
					gt.Expect(AFileExistsContaining("build/started_time", "1010101010", gt)).To(gomega.BeTrue())
				})

				it("writes out build/ended_time", func() {
					gt.Expect(AFileExistsContaining("build/ended_time", "1191919191", gt)).To(gomega.BeTrue())
				})

				it("writes out build/status", func() {
					gt.Expect(AFileExistsContaining("build/status", "succeeded", gt)).To(gomega.BeTrue())
				})

				it("writes out the build/concourse_url", func() {
					gt.Expect(AFileExistsContaining("build/concourse_url", "https://example.com", gt)).To(gomega.BeTrue())
				})

				it("writes out the build/team_url", func() {
					gt.Expect(AFileExistsContaining("build/team_url", "https://example.com/teams/team", gt)).To(gomega.BeTrue())
				})

				it("writes out the build/pipeline_url", func() {
					gt.Expect(AFileExistsContaining("build/pipeline_url", "https://example.com/teams/team/pipelines/pipeline", gt)).To(gomega.BeTrue())
				})

				it("writes out the build/job_url", func() {
					gt.Expect(AFileExistsContaining("build/job_url", "https://example.com/teams/team/pipelines/pipeline/jobs/job", gt)).To(gomega.BeTrue())
				})

				it("writes out build/build_url", func() {
					gt.Expect(AFileExistsContaining("build/build_url", "https://example.com/teams/team/pipelines/pipeline/jobs/job/builds/111", gt)).To(gomega.BeTrue())
				})
			}, spec.Nested())

			when("the pipeline name or job name are unspecified", func() {
				it.Before(func() {
					os.Remove("build/job.json")
					os.Remove("build/job_999.json")
					os.Remove("build/job_team_pipeline-from-build_job-from-build_111.json")

					fakeclient.BuildReturns(atc.Build{
						ID:           999,
						Name:         "111",
						TeamName:     "team",
						PipelineName: "pipeline-from-build",
						JobName:      "job-from-build",
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
					faketeam.JobReturns(atc.Job{
						ID:            444,
						Name:          "job",
						PipelineName:  "pipeline",
						TeamName:      "team",
						FinishedBuild: &atc.Build{},
					}, true, nil)
					faketeam.VersionedResourceTypesReturns(atc.VersionedResourceTypes{{ResourceType: atc.ResourceType{CheckEvery: "10m"}}}, true, nil)
					fakeeventstream.NextEventReturns(nil, io.EOF)
					fakeclient.BuildEventsReturns(fakeeventstream, nil)

					uuid, err := uuid.ParseHex("96d7128f-bacf-4f60-9ffd-1a9ca4c9e1d7")
					gt.Expect(err).NotTo(gomega.HaveOccurred())
					inner := in.NewInnerUsingClient(&config.InRequest{
						Source: config.Source{
							ConcourseUrl: "https://example.com",
							Team:         "team",
						},
						Version:          config.Version{BuildId: "999"},
						Params:           config.InParams{},
						WorkingDirectory: "build",
						ReleaseVersion:   "v0.99.11",
						ReleaseGitRef:    "abcdef1234567890",
						GetTimestamp:     1234567890,
						GetUuid:          uuid.String(),
					}, fakeclient)
					response, err = inner.In()
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("uses information from the build to fetch the job", func() {
					pipeline, job := faketeam.JobArgsForCall(0)
					gt.Expect(pipeline).To(gomega.Equal("pipeline-from-build"))
					gt.Expect(job).To(gomega.Equal("job-from-build"))
				})

				it("writes out the job.json file", func() {
					gt.Expect(AFileExistsContaining("build/job.json", `"finished_build":`, gt)).To(gomega.BeTrue())
				})

				it("writes out the job_<build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/job_999.json", `"finished_build":`, gt)).To(gomega.BeTrue())
				})

				it("writes out the job_<team>_<pipeline>_<job>_<build number>.json file", func() {
					gt.Expect(AFileExistsContaining("build/job_team_pipeline-from-build_job-from-build_111.json", `"finished_build":`, gt)).To(gomega.BeTrue())
				})

				it("adds resource version metadata to JSON files", func() {
					gt.Expect(AFileExistsContaining("build/job.json", `"concourse_build_resource":{"release":"v0.99.11","git_ref":"abcdef1234567890","get_timestamp":1234567890,"concourse_version":"3.99.11","get_uuid":"96d7128f-bacf-4f60-9ffd-1a9ca4c9e1d7"},`, gt)).To(gomega.BeTrue())
				})
			}, spec.Nested())

			when("only the concourse URL was specified", func() {
				it.Before(func() {
					fakeclient.BuildReturns(atc.Build{
						TeamName: "team-from-build",
					}, true, nil)
					fakeclient.BuildResourcesReturns(atc.BuildInputsOutputs{}, true, nil)
					fakeclient.BuildPlanReturns(atc.PublicBuildPlan{}, true, nil)
					faketeam.JobReturns(atc.Job{}, true, nil)
					faketeam.VersionedResourceTypesReturns(atc.VersionedResourceTypes{{ResourceType: atc.ResourceType{CheckEvery: "10m"}}}, true, nil)
					fakeeventstream.NextEventReturns(nil, io.EOF)
					fakeclient.BuildEventsReturns(fakeeventstream, nil)

					inner := in.NewInnerUsingClient(&config.InRequest{
						Source: config.Source{
							ConcourseUrl: "https://example.com",
						},
						Version:          config.Version{BuildId: "999"},
						Params:           config.InParams{},
						WorkingDirectory: "build",
					}, fakeclient)
					response, err = inner.In()
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("uses the build's team name to fetch the job", func() {
					team := fakeclient.TeamArgsForCall(1)
					gt.Expect(team).To(gomega.Equal("team-from-build"))
				})
			}, spec.Nested())

			when("the concourse URL has a trailing slash", func() {
				it.Before(func() {
					fakeclient.BuildReturns(atc.Build{
						TeamName:     "team",
						PipelineName: "pipeline",
						JobName:      "job",
						Name:         "123",
					}, true, nil)
					fakeclient.BuildResourcesReturns(atc.BuildInputsOutputs{}, true, nil)
					fakeclient.BuildPlanReturns(atc.PublicBuildPlan{}, true, nil)
					faketeam.JobReturns(atc.Job{}, true, nil)
					faketeam.VersionedResourceTypesReturns(atc.VersionedResourceTypes{{ResourceType: atc.ResourceType{CheckEvery: "10m"}}}, true, nil)
					fakeeventstream.NextEventReturns(nil, io.EOF)
					fakeclient.BuildEventsReturns(fakeeventstream, nil)

					inner := in.NewInnerUsingClient(&config.InRequest{
						Source: config.Source{
							ConcourseUrl: "https://example.com/",
						},
						Version:          config.Version{BuildId: "999"},
						Params:           config.InParams{},
						WorkingDirectory: "build",
					}, fakeclient)
					response, err = inner.In()
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("strips the trailing slash", func() {
					gt.Expect(response.Metadata[0].Value).ToNot(gomega.ContainSubstring("https://example.com//teams"))
				})
			}, spec.Nested())
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

			when("the Concourse version cannot be retrieved", func() {
				it.Before(func() {
					fakeclient.GetInfoReturns(atc.Info{}, fmt.Errorf("test error"))
					inner := in.NewInnerUsingClient(&config.InRequest{
						Source:           config.Source{Pipeline: "pipeline", Job: "job"},
						Version:          config.Version{BuildId: "111"},
						Params:           config.InParams{},
						WorkingDirectory: "build",
					}, fakeclient)
					response, err = inner.In()
				})

				it("returns an error", func() {
					gt.Expect(err.Error()).To(gomega.ContainSubstring("could not get Concourse server information: test error"))
					gt.Expect(response).To(gomega.BeNil())
				})
			})
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
