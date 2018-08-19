package check_test

import (
	"testing"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/onsi/gomega"

	fakes "github.com/concourse/go-concourse/concourse/concoursefakes"
	"github.com/concourse/atc"
	"github.com/concourse/go-concourse/concourse"

	"github.com/jchesterpivotal/concourse-build-resource/pkg/check"
	"github.com/jchesterpivotal/concourse-build-resource/pkg/config"

	"fmt"
)

func TestCheckPkg(t *testing.T) {
	spec.Run(t, "pkg/check", func(t *testing.T, when spec.G, it spec.S) {
		when("build ID is defined", func() {
			faketeam := new(fakes.FakeTeam)
			fakeclient := new(fakes.FakeClient)
			fakeclient.TeamReturns(faketeam)
			var response *config.CheckResponse
			var err error

			when("there are new builds", func() {
				when("there are completed builds", func() {
					gt := gomega.NewGomegaWithT(t)
					var page concourse.Page

					it.Before(func() {
						faketeam.JobBuildsReturns([]atc.Build{
							{ID: 555, Status: string(atc.StatusSucceeded)}, {ID: 999, Status: string(atc.StatusFailed)},
						}, concourse.Pagination{}, true, nil)

						checker := check.NewCheckerUsingClient(&config.CheckRequest{Version: config.Version{BuildId: "111"}}, fakeclient)
						response, err = checker.Check()
						gt.Expect(err).NotTo(gomega.HaveOccurred())

						_, _, page = faketeam.JobBuildsArgsForCall(0)
					})

					it("returns completed builds", func() {
						gt.Expect(response).To(gomega.Equal(&config.CheckResponse{{BuildId: "555"}, {BuildId: "999"}}))
					})

					it("asks to fetch 50 builds", func() {
						gt.Expect(page).To(gomega.Equal(concourse.Page{Limit: 50}))
					})
				}, spec.Nested())

				when("there is a mix of completed and uncompleted builds", func() {
					gt := gomega.NewGomegaWithT(t)

					it.Before(func() {
						faketeam.JobBuildsReturns([]atc.Build{
							{ID: 555, Status: string(atc.StatusSucceeded)},
							{ID: 777, Status: string(atc.StatusStarted)},
							{ID: 999, Status: string(atc.StatusPending)},
						}, concourse.Pagination{}, true, nil)

						checker := check.NewCheckerUsingClient(&config.CheckRequest{Version: config.Version{BuildId: "111"}}, fakeclient)
						response, err = checker.Check()
						gt.Expect(err).NotTo(gomega.HaveOccurred())
					})

					it("returns only the completed builds, ignoring uncompleted builds", func() {
						gt.Expect(response).To(gomega.Equal(&config.CheckResponse{{BuildId: "555"}}))
					})
				}, spec.Nested())

				when("there are only uncompleted builds", func() {
					gt := gomega.NewGomegaWithT(t)

					it.Before(func() {
						faketeam.JobBuildsReturns([]atc.Build{
							{ID: 777, Status: string(atc.StatusStarted)},
							{ID: 999, Status: string(atc.StatusPending)},
						}, concourse.Pagination{}, true, nil)

						checker := check.NewCheckerUsingClient(&config.CheckRequest{Version: config.Version{BuildId: "111"}}, fakeclient)
						response, err = checker.Check()
						gt.Expect(err).NotTo(gomega.HaveOccurred())
					})

					it("returns the version given", func() {
						gt.Expect(response).To(gomega.Equal(&config.CheckResponse{config.Version{BuildId: "111"}}))
					})
				})
			})

			when("there are no new builds", func() {
				gt := gomega.NewGomegaWithT(t)

				it.Before(func() {
					faketeam.JobBuildsReturns([]atc.Build{
						{ID: 555, Status: string(atc.StatusSucceeded)},
						{ID: 777, Status: string(atc.StatusStarted)},
						{ID: 999, Status: string(atc.StatusPending)},
					}, concourse.Pagination{}, true, nil)

					checker := check.NewCheckerUsingClient(&config.CheckRequest{Version: config.Version{BuildId: "999"}}, fakeclient)
					response, err = checker.Check()
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("returns the version it was given", func() {
					gt.Expect(response).To(gomega.Equal(&config.CheckResponse{config.Version{BuildId: "999"}}))
				})
			}, spec.Nested())

			when("there are no builds at all", func() {
				gt := gomega.NewGomegaWithT(t)

				it.Before(func() {
					faketeam.JobBuildsReturns([]atc.Build{}, concourse.Pagination{}, true, nil)

					checker := check.NewCheckerUsingClient(&config.CheckRequest{Version: config.Version{BuildId: "999"}}, fakeclient)
					response, err = checker.Check()
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("returns an empty version array", func() {
					gt.Expect(response).To(gomega.Equal(&config.CheckResponse{}))

				})
			}, spec.Nested())
		})

		when("build ID is not defined, meaning this is the first check", func() {
			gt := gomega.NewGomegaWithT(t)
			faketeam := new(fakes.FakeTeam)
			fakeclient := new(fakes.FakeClient)
			fakeclient.TeamReturns(faketeam)
			var page concourse.Page

			it.Before(func() {
				faketeam.JobBuildsReturns([]atc.Build{{ID: 111, Status: string(atc.StatusSucceeded)}}, concourse.Pagination{}, true, nil)

				checker := check.NewCheckerUsingClient(&config.CheckRequest{}, fakeclient)
				_, err := checker.Check()
				gt.Expect(err).NotTo(gomega.HaveOccurred())

				_, _, page = faketeam.JobBuildsArgsForCall(0)
			})

			it("only fetches 1 build", func() {
				gt.Expect(page).To(gomega.Equal(concourse.Page{Limit: 1}))
			})
		}, spec.Nested())

		when("build ID is defined, but is not a valid number", func() {
			gt := gomega.NewGomegaWithT(t)
			fakeclient := new(fakes.FakeClient)
			var response *config.CheckResponse
			var err error

			it.Before(func() {
				checker := check.NewCheckerUsingClient(&config.CheckRequest{Version: config.Version{BuildId: "not numerical"}}, fakeclient)
				response, err = checker.Check()
			})

			it("returns an error", func() {
				gt.Expect(response).To(gomega.BeNil())
				gt.Expect(err.Error()).To(gomega.ContainSubstring("could not convert build id 'not numerical' to an int:"))
			})
		}, spec.Nested())

		when("builds cannot be retrieved due to an error", func() {
			gt := gomega.NewGomegaWithT(t)
			faketeam := new(fakes.FakeTeam)
			fakeclient := new(fakes.FakeClient)
			fakeclient.TeamReturns(faketeam)
			var response *config.CheckResponse
			var err error

			it.Before(func() {
				faketeam.JobBuildsReturns([]atc.Build{}, concourse.Pagination{}, false, fmt.Errorf("kerfupsed"))

				checker := check.NewCheckerUsingClient(&config.CheckRequest{
					Version: config.Version{BuildId: "111"},
					Source:  config.Source{Pipeline: "pipeline", Job: "job"},
				}, fakeclient)
				response, err = checker.Check()
			})

			it("returns an error", func() {
				gt.Expect(response).To(gomega.BeNil())
				gt.Expect(err.Error()).To(gomega.ContainSubstring("could not retrieve builds for 'pipeline/job': kerfupsed"))
			})
		}, spec.Nested())

		when("the team, pipeline or job are not found", func() {
			gt := gomega.NewGomegaWithT(t)
			faketeam := new(fakes.FakeTeam)
			fakeclient := new(fakes.FakeClient)
			fakeclient.TeamReturns(faketeam)
			var response *config.CheckResponse
			var err error

			it.Before(func() {
				faketeam.JobBuildsReturns([]atc.Build{}, concourse.Pagination{}, false, nil)

				checker := check.NewCheckerUsingClient(&config.CheckRequest{
					Version: config.Version{BuildId: "111"},
					Source:  config.Source{Pipeline: "pipeline", Job: "job"},
				}, fakeclient)
				response, err = checker.Check()
			})

			it("returns an error", func() {
				gt.Expect(response).To(gomega.BeNil())
				gt.Expect(err.Error()).To(gomega.ContainSubstring("server could not find 'pipeline/job'"))
			})
		}, spec.Nested())
	}, spec.Report(report.Terminal{}))
}
