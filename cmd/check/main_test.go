package main_test

import (
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"testing"

	"github.com/concourse/atc"

	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
)

func TestCheckCmd(t *testing.T) {
	gt := gomega.NewGomegaWithT(t)

	compiledPath, err := gexec.Build("github.com/jchesterpivotal/concourse-build-resource/cmd/check")
	if err != nil {
		gt.Expect(err).NotTo(gomega.HaveOccurred())
	}

	spec.Run(t, "/opt/resource/check", func(t *testing.T, when spec.G, it spec.S) {
		when("given malformed JSON", func() {
			gt := gomega.NewGomegaWithT(t)
			var session *gexec.Session

			it.Before(func() {
				cmd := exec.Command(compiledPath, `} this is malformed[] JSON:`)
				session, err = gexec.Start(cmd, it.Out(), it.Out())
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it("fails with an error", func() {
				gt.Eventually(session.Err).Should(gbytes.Say("failed to parse input JSON:"))
				gt.Eventually(session).Should(gexec.Exit(1))
			})
		}, spec.Nested())

		when("given syntactically-valid but otherwise incorrect JSON", func() {
			gt := gomega.NewGomegaWithT(t)
			var session *gexec.Session

			it.Before(func() {
				cmd := exec.Command(compiledPath)
				cmd.Stdin = bytes.NewBufferString(`{"valid": "json", "but_not": "the json which is expected", "by_check": true}`)
				session, err = gexec.Start(cmd, it.Out(), it.Out())
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it("fails with an error", func() {
				gt.Eventually(session.Err).Should(gbytes.Say("failed to perform 'check':"))
				gt.Eventually(session).Should(gexec.Exit(1))
			})
		}, spec.Nested())

		when("given valid input", func() {
			gt := gomega.NewGomegaWithT(t)
			var session *gexec.Session
			var server *ghttp.Server

			it.Before(func() {
				server = ghttp.NewServer()
				server.AppendHandlers(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.Header().Add("Link", fmt.Sprintf(`<%s/api/v1/teams/t/pipelines/p/jobs/j/builds?until=211&limit=100>; rel="previous"`, server.URL()))

						json.NewEncoder(w).Encode([]atc.Build{{
							ID:           210,
							TeamName:     "t",
							Name:         "111",
							Status:       string(atc.StatusSucceeded),
							JobName:      "j",
							APIURL:       fmt.Sprintf("%s/api/v1", server.URL()),
							PipelineName: "p",
							StartTime:    111222333,
							EndTime:      444555666,
							ReapTime:     0,
						}})
					}),
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")

						json.NewEncoder(w).Encode([]atc.Build{{
							ID:           210,
							TeamName:     "t",
							Name:         "111",
							Status:       string(atc.StatusSucceeded),
							JobName:      "j",
							APIURL:       fmt.Sprintf("%s/api/v1", server.URL()),
							PipelineName: "p",
							StartTime:    111222333,
							EndTime:      444555666,
							ReapTime:     0,
						}})
					}),
				)

				cmd := exec.Command(compiledPath)
				input := fmt.Sprintf(`{"version":{"build_id":"111"},"source":{"concourse_url":"%s","team":"t","pipeline":"p","job":"j"}}`, server.URL())
				cmd.Stdin = bytes.NewBufferString(input)
				session, err = gexec.Start(cmd, it.Out(), it.Out())
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it.After(func() {
				server.Close()
			})

			it("prints an array of versions to stdout", func() {
				gt.Eventually(session.Out).Should(gbytes.Say(`[{"build_id":"210"}]`))
				gt.Eventually(session).Should(gexec.Exit(0))
			})
		}, spec.Nested())

		when("version is not given (ie, first run) and initial_build_id is set", func() {
			gt := gomega.NewGomegaWithT(t)
			var session *gexec.Session

			it.Before(func() {
				server := ghttp.NewServer()
				server.AppendHandlers(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						w.Header().Add("Link", fmt.Sprintf(`<%s/api/v1/builds?until=334&limit=100>; rel="previous"`, server.URL()))

						json.NewEncoder(w).Encode([]atc.Build{{
							ID:           210,
							TeamName:     "t",
							Name:         "111",
							Status:       string(atc.StatusSucceeded),
							JobName:      "j",
							APIURL:       fmt.Sprintf("%s/api/v1", server.URL()),
							PipelineName: "p",
							StartTime:    111222333,
							EndTime:      444555666,
							ReapTime:     0,
						}})
					}),
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")

						json.NewEncoder(w).Encode([]atc.Build{{
							ID:           210,
							TeamName:     "t",
							Name:         "111",
							Status:       string(atc.StatusSucceeded),
							JobName:      "j",
							APIURL:       fmt.Sprintf("%s/api/v1", server.URL()),
							PipelineName: "p",
							StartTime:    111222333,
							EndTime:      444555666,
							ReapTime:     0,
						}})
					}),
				)

				cmd := exec.Command(compiledPath)
				input := fmt.Sprintf(`{"source":{"concourse_url":"%s","initial_build_id": 222}}`, server.URL())
				cmd.Stdin = bytes.NewBufferString(input)
				session, err = gexec.Start(cmd, it.Out(), it.Out())
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it("returns a version array of everything after and including initial_build_id", func() {
				gt.Eventually(session.Out).Should(gbytes.Say(`[{"build_id":"222"},{"build_id":"333"}]`))
				gt.Eventually(session).Should(gexec.Exit(0))
			})
		}, spec.Nested())

		when("trace is enabled", func() {
			gt := gomega.NewGomegaWithT(t)
			var session *gexec.Session
			var server *ghttp.Server

			it.Before(func() {
				server = ghttp.NewServer()
				server.AppendHandlers(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")

					json.NewEncoder(w).Encode([]atc.Build{{ID: 999, Status: string(atc.StatusSucceeded)}})
				}))

				cmd := exec.Command(compiledPath)
				input := fmt.Sprintf(`{"version":{"build_id":"111"},"source":{"concourse_url":"%s","team":"t","pipeline":"p","job":"j","enable_tracing":true}}`, server.URL())
				cmd.Stdin = bytes.NewBufferString(input)
				session, err = gexec.Start(cmd, it.Out(), it.Out())
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it.After(func() {
				server.Close()
			})

			it("prints traces to stdout", func() {
				gt.Eventually(session.Err).Should(gbytes.Say(`GET /api/v1/teams/t/pipelines/p/jobs/j/builds`))
				gt.Eventually(session).Should(gexec.Exit(0))
			})
		}, spec.Nested())

		when("something goes wrong with check.Check()", func() {
			gt := gomega.NewGomegaWithT(t)
			var session *gexec.Session

			it.Before(func() {
				cmd := exec.Command(compiledPath)
				cmd.Stdin = bytes.NewBufferString(`{"version":{"build_id":"111"},"source":{"concourse_url":"c","team":"t","pipeline":"p","job":"j"}}`)
				session, err = gexec.Start(cmd, it.Out(), it.Out())
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it("fails with an error", func() {
				gt.Eventually(session.Err).Should(gbytes.Say("failed to perform 'check': could not retrieve builds for pipeline/job 'p/j"))
				gt.Eventually(session).Should(gexec.Exit(1))
			})
		}, spec.Nested())
	}, spec.Report(report.Terminal{}))

	gexec.CleanupBuildArtifacts()
}
