package check_test

import (
	"github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"testing"

	"github.com/concourse/atc"
	"github.com/concourse/go-concourse/concourse"
	fakes "github.com/concourse/go-concourse/concourse/concoursefakes"

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

			when("checking a particular job", func() {
				source := config.Source{
					ConcourseUrl: "https://example.com",
					Team:         "test-team",
					Pipeline:     "test-pipeline",
					Job:          "test-job",
				}

				when("there are new builds", func() {
					when("there are completed builds", func() {
						gt := gomega.NewGomegaWithT(t)
						var page concourse.Page

						it.Before(func() {
							faketeam.JobBuildsReturns([]atc.Build{
								{ID: 999, Status: string(atc.StatusFailed)},
								{ID: 555, Status: string(atc.StatusSucceeded)},
							}, concourse.Pagination{}, true, nil)

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
							response, err = checker.Check()
							gt.Expect(err).NotTo(gomega.HaveOccurred())

							_, _, page = faketeam.JobBuildsArgsForCall(0)
						})

						it("returns completed builds in order", func() {
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
								{ID: 999, Status: string(atc.StatusPending)},
								{ID: 777, Status: string(atc.StatusStarted)},
								{ID: 555, Status: string(atc.StatusSucceeded)},
							}, concourse.Pagination{}, true, nil)

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
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

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
							response, err = checker.Check()
							gt.Expect(err).NotTo(gomega.HaveOccurred())
						})

						it("returns the version given", func() {
							gt.Expect(response).To(gomega.Equal(&config.CheckResponse{config.Version{BuildId: "111"}}))
						})
					})
				}, spec.Nested())

				when("there are no new builds", func() {
					gt := gomega.NewGomegaWithT(t)

					it.Before(func() {
						faketeam.JobBuildsReturns([]atc.Build{
							{ID: 555, Status: string(atc.StatusSucceeded)},
							{ID: 777, Status: string(atc.StatusStarted)},
							{ID: 999, Status: string(atc.StatusPending)},
						}, concourse.Pagination{}, true, nil)

						checker := check.NewCheckerUsingClient(&config.CheckRequest{
							Version: config.Version{BuildId: "999"},
							Source:  source,
						}, fakeclient)
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

						checker := check.NewCheckerUsingClient(&config.CheckRequest{
							Version: config.Version{BuildId: "999"},
							Source:  source,
						}, fakeclient)
						response, err = checker.Check()
						gt.Expect(err).NotTo(gomega.HaveOccurred())
					})

					it("returns an empty version array", func() {
						gt.Expect(response).To(gomega.Equal(&config.CheckResponse{}))

					})
				}, spec.Nested())
			}, spec.Nested())

			when("checking all jobs in a pipeline", func() {
				source := config.Source{
					ConcourseUrl: "https://example.com",
					Team:         "test-team",
					Pipeline:     "test-pipeline",
				}

				when("there are new builds", func() {
					when("there are completed builds", func() {
						gt := gomega.NewGomegaWithT(t)
						var page concourse.Page

						it.Before(func() {
							faketeam.PipelineBuildsReturns([]atc.Build{
								{ID: 999, Status: string(atc.StatusFailed)},
								{ID: 555, Status: string(atc.StatusSucceeded)},
							}, concourse.Pagination{}, true, nil)

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
							response, err = checker.Check()
							gt.Expect(err).NotTo(gomega.HaveOccurred())

							_, page = faketeam.PipelineBuildsArgsForCall(0)
						})

						it("returns completed builds in order", func() {
							gt.Expect(response).To(gomega.Equal(&config.CheckResponse{{BuildId: "555"}, {BuildId: "999"}}))
						})

						it("asks to fetch 50 builds", func() {
							gt.Expect(page).To(gomega.Equal(concourse.Page{Limit: 50}))
						})
					}, spec.Nested())

					when("there is a mix of completed and uncompleted builds", func() {
						gt := gomega.NewGomegaWithT(t)

						it.Before(func() {
							faketeam.PipelineBuildsReturns([]atc.Build{
								{ID: 555, Status: string(atc.StatusSucceeded)},
								{ID: 777, Status: string(atc.StatusStarted)},
								{ID: 999, Status: string(atc.StatusPending)},
							}, concourse.Pagination{}, true, nil)

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
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
							faketeam.PipelineBuildsReturns([]atc.Build{
								{ID: 777, Status: string(atc.StatusStarted)},
								{ID: 999, Status: string(atc.StatusPending)},
							}, concourse.Pagination{}, true, nil)

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
							response, err = checker.Check()
							gt.Expect(err).NotTo(gomega.HaveOccurred())
						})

						it("returns the version given", func() {
							gt.Expect(response).To(gomega.Equal(&config.CheckResponse{config.Version{BuildId: "111"}}))
						})
					})
				}, spec.Nested())

				when("there are no new builds", func() {
					gt := gomega.NewGomegaWithT(t)

					it.Before(func() {
						faketeam.PipelineBuildsReturns([]atc.Build{
							{ID: 555, Status: string(atc.StatusSucceeded)},
							{ID: 777, Status: string(atc.StatusStarted)},
							{ID: 999, Status: string(atc.StatusPending)},
						}, concourse.Pagination{}, true, nil)

						checker := check.NewCheckerUsingClient(&config.CheckRequest{
							Version: config.Version{BuildId: "999"},
							Source:  source,
						}, fakeclient)
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
						faketeam.PipelineBuildsReturns([]atc.Build{}, concourse.Pagination{}, true, nil)

						checker := check.NewCheckerUsingClient(&config.CheckRequest{
							Version: config.Version{BuildId: "999"},
							Source:  source,
						}, fakeclient)
						response, err = checker.Check()
						gt.Expect(err).NotTo(gomega.HaveOccurred())
					})

					it("returns an empty version array", func() {
						gt.Expect(response).To(gomega.Equal(&config.CheckResponse{}))
					})
				}, spec.Nested())
			}, spec.Nested())

			when("checking all jobs in a team", func() {
				source := config.Source{
					ConcourseUrl: "https://example.com",
					Team:         "test-team",
				}

				when("there are new builds", func() {
					when("there are completed builds", func() {
						gt := gomega.NewGomegaWithT(t)
						var page concourse.Page

						it.Before(func() {
							faketeam.BuildsReturns([]atc.Build{
								{ID: 999, Status: string(atc.StatusFailed)},
								{ID: 555, Status: string(atc.StatusSucceeded)},
							}, concourse.Pagination{}, nil)

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
							response, err = checker.Check()
							gt.Expect(err).NotTo(gomega.HaveOccurred())

							page = faketeam.BuildsArgsForCall(0)
						})

						it("returns completed builds in order", func() {
							gt.Expect(response).To(gomega.Equal(&config.CheckResponse{{BuildId: "555"}, {BuildId: "999"}}))
						})

						it("asks to fetch 50 builds", func() {
							gt.Expect(page).To(gomega.Equal(concourse.Page{Limit: 50}))
						})
					}, spec.Nested())

					when("there is a mix of completed and uncompleted builds", func() {
						gt := gomega.NewGomegaWithT(t)

						it.Before(func() {
							faketeam.BuildsReturns([]atc.Build{
								{ID: 555, Status: string(atc.StatusSucceeded)},
								{ID: 777, Status: string(atc.StatusStarted)},
								{ID: 999, Status: string(atc.StatusPending)},
							}, concourse.Pagination{}, nil)

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
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
							faketeam.BuildsReturns([]atc.Build{
								{ID: 777, Status: string(atc.StatusStarted)},
								{ID: 999, Status: string(atc.StatusPending)},
							}, concourse.Pagination{}, nil)

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
							response, err = checker.Check()
							gt.Expect(err).NotTo(gomega.HaveOccurred())
						})

						it("returns the version given", func() {
							gt.Expect(response).To(gomega.Equal(&config.CheckResponse{config.Version{BuildId: "111"}}))
						})
					})
				}, spec.Nested())

				when("there are no new builds", func() {
					gt := gomega.NewGomegaWithT(t)

					it.Before(func() {
						faketeam.BuildsReturns([]atc.Build{
							{ID: 555, Status: string(atc.StatusSucceeded)},
							{ID: 777, Status: string(atc.StatusStarted)},
							{ID: 999, Status: string(atc.StatusPending)},
						}, concourse.Pagination{}, nil)

						checker := check.NewCheckerUsingClient(&config.CheckRequest{
							Version: config.Version{BuildId: "999"},
							Source:  source,
						}, fakeclient)
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
						faketeam.BuildsReturns([]atc.Build{}, concourse.Pagination{}, nil)

						checker := check.NewCheckerUsingClient(&config.CheckRequest{
							Version: config.Version{BuildId: "999"},
							Source:  source,
						}, fakeclient)
						response, err = checker.Check()
						gt.Expect(err).NotTo(gomega.HaveOccurred())
					})

					it("returns an empty version array", func() {
						gt.Expect(response).To(gomega.Equal(&config.CheckResponse{}))
					})
				}, spec.Nested())

			}, spec.Nested())

			when("checking all jobs in all teams", func() {
				source := config.Source{
					ConcourseUrl: "https://example.com",
				}

				when("there are new builds", func() {
					when("there are completed builds", func() {
						gt := gomega.NewGomegaWithT(t)
						var page concourse.Page

						it.Before(func() {
							fakeclient.BuildsReturns([]atc.Build{
								{ID: 999, Status: string(atc.StatusFailed)},
								{ID: 555, Status: string(atc.StatusSucceeded)},
							}, concourse.Pagination{}, nil)

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
							response, err = checker.Check()
							gt.Expect(err).NotTo(gomega.HaveOccurred())

							page = fakeclient.BuildsArgsForCall(0)
						})

						it("returns completed builds in order", func() {
							gt.Expect(response).To(gomega.Equal(&config.CheckResponse{{BuildId: "555"}, {BuildId: "999"}}))
						})

						it("asks to fetch 50 builds", func() {
							gt.Expect(page).To(gomega.Equal(concourse.Page{Limit: 50}))
						})
					}, spec.Nested())

					when("there is a mix of completed and uncompleted builds", func() {
						gt := gomega.NewGomegaWithT(t)

						it.Before(func() {
							fakeclient.BuildsReturns([]atc.Build{
								{ID: 555, Status: string(atc.StatusSucceeded)},
								{ID: 777, Status: string(atc.StatusStarted)},
								{ID: 999, Status: string(atc.StatusPending)},
							}, concourse.Pagination{}, nil)

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
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
							fakeclient.BuildsReturns([]atc.Build{
								{ID: 777, Status: string(atc.StatusStarted)},
								{ID: 999, Status: string(atc.StatusPending)},
							}, concourse.Pagination{}, nil)

							checker := check.NewCheckerUsingClient(&config.CheckRequest{
								Version: config.Version{BuildId: "111"},
								Source:  source,
							}, fakeclient)
							response, err = checker.Check()
							gt.Expect(err).NotTo(gomega.HaveOccurred())
						})

						it("returns the version given", func() {
							gt.Expect(response).To(gomega.Equal(&config.CheckResponse{config.Version{BuildId: "111"}}))
						})
					})
				}, spec.Nested())

				when("there are no new builds", func() {
					gt := gomega.NewGomegaWithT(t)

					it.Before(func() {
						fakeclient.BuildsReturns([]atc.Build{
							{ID: 555, Status: string(atc.StatusSucceeded)},
							{ID: 777, Status: string(atc.StatusStarted)},
							{ID: 999, Status: string(atc.StatusPending)},
						}, concourse.Pagination{}, nil)

						checker := check.NewCheckerUsingClient(&config.CheckRequest{
							Version: config.Version{BuildId: "999"},
							Source:  source,
						}, fakeclient)
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
						fakeclient.BuildsReturns([]atc.Build{}, concourse.Pagination{}, nil)

						checker := check.NewCheckerUsingClient(&config.CheckRequest{
							Version: config.Version{BuildId: "999"},
							Source:  source,
						}, fakeclient)
						response, err = checker.Check()
						gt.Expect(err).NotTo(gomega.HaveOccurred())
					})

					it("returns an empty version array", func() {
						gt.Expect(response).To(gomega.Equal(&config.CheckResponse{}))
					})
				}, spec.Nested())

			}, spec.Nested())
		}, spec.Nested())

		when("build ID is not defined, meaning this is the first check", func() {
			when("initial_build_id is not set", func() {
				gt := gomega.NewGomegaWithT(t)
				faketeam := new(fakes.FakeTeam)
				fakeclient := new(fakes.FakeClient)
				fakeclient.TeamReturns(faketeam)
				var page concourse.Page
				source := config.Source{
					ConcourseUrl: "https://example.com",
					Team:         "test-team",
					Pipeline:     "test-pipeline",
					Job:          "test-job",
				}

				it.Before(func() {
					faketeam.JobBuildsReturns([]atc.Build{{ID: 111, Status: string(atc.StatusSucceeded)}}, concourse.Pagination{}, true, nil)

					checker := check.NewCheckerUsingClient(&config.CheckRequest{Source: source}, fakeclient)
					_, err := checker.Check()
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, _, page = faketeam.JobBuildsArgsForCall(0)
				})

				it("only fetches 1 build", func() {
					gt.Expect(page).To(gomega.Equal(concourse.Page{Limit: 1}))
				})
			}, spec.Nested())

			when("initial_build_id has been set", func() {
				gt := gomega.NewGomegaWithT(t)
				faketeam := new(fakes.FakeTeam)
				fakeclient := new(fakes.FakeClient)
				fakeclient.TeamReturns(faketeam)
				source := config.Source{
					ConcourseUrl:   "https://example.com",
					InitialBuildId: 222,
				}
				var response *config.CheckResponse
				var err error

				it.Before(func() {
					checker := check.NewCheckerUsingClient(&config.CheckRequest{Source: source}, fakeclient)
					response, err = checker.Check()
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("returns that initial_build_id as the first version", func() {
					gt.Expect(response).To(gomega.Equal(&config.CheckResponse{{BuildId: "222"}}))
				})

				it("does not bother dialling out to the remote Concourse", func() {
					gt.Expect(fakeclient.BuildsCallCount()).To(gomega.BeZero())
				})
			})
		}, spec.Nested())

		when("build ID is defined, but is not a valid number", func() {
			gt := gomega.NewGomegaWithT(t)
			fakeclient := new(fakes.FakeClient)
			var response *config.CheckResponse
			var err error
			source := config.Source{
				ConcourseUrl: "https://example.com",
				Team:         "test-team",
				Pipeline:     "test-pipeline",
				Job:          "test-job",
			}

			it.Before(func() {
				checker := check.NewCheckerUsingClient(&config.CheckRequest{
					Version: config.Version{BuildId: "not numerical"},
					Source:  source,
				}, fakeclient)
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
				gt.Expect(err.Error()).To(gomega.ContainSubstring("could not retrieve builds for pipeline/job 'pipeline/job': kerfupsed"))
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
					Source: config.Source{
						ConcourseUrl: "https://example.com",
						Team:         "team-name",
						Pipeline:     "pipeline-name",
						Job:          "job-name",
					},
				}, fakeclient)
				response, err = checker.Check()
			})

			it("returns an error", func() {
				gt.Expect(response).To(gomega.BeNil())
				gt.Expect(err.Error()).To(gomega.ContainSubstring("server could not find pipeline/job 'pipeline-name/job-name'"))
			})
		}, spec.Nested())
	}, spec.Report(report.Terminal{}))
}
