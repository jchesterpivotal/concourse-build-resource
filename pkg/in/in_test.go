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
				fakeclient.BuildReturns(atc.Build{ID: 111}, true, nil)
				fakeclient.BuildResourcesReturns(atc.BuildInputsOutputs{}, true, nil)
				fakeclient.BuildPlanReturns(atc.PublicBuildPlan{}, true, nil)
				fakeeventstream.NextEventReturns(nil, io.EOF)
				fakeclient.BuildEventsReturns(fakeeventstream, nil)

				inner := in.NewInnerUsingClient(&config.InRequest{
					Source:           config.Source{},
					Version:          config.Version{BuildId: "111"},
					Params:           config.InParams{},
					WorkingDirectory: "build",
				}, fakeclient)
				response, err = inner.In()
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it("returns the version it was given", func() {
				gt.Expect(response.Version).To(gomega.Equal(config.Version{BuildId: "111"}))
			})

			it("returns no metadata", func() {
				gt.Expect(response.Metadata).To(gomega.BeEmpty())
			})

			it("writes out the build.json file", func() {
				gt.Expect("build/build.json").To(gomega.BeAnExistingFile())
			})

			it("writes out the resources.json file", func() {
				gt.Expect("build/resources.json").To(gomega.BeAnExistingFile())
			})

			it("writes out the plan.json file", func() {
				gt.Expect("build/plan.json").To(gomega.BeAnExistingFile())
			})

			it("writes out the events.log", func() {
				gt.Expect("build/events.log").To(gomega.BeAnExistingFile())
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
