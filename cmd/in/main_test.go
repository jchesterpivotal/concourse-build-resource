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
	"io"
	"net/http"
	"os"
	"os/exec"
)

func TestInCmd(t *testing.T) {
	gt := gomega.NewGomegaWithT(t)

	compiledPath, err := gexec.Build("github.com/jchesterpivotal/concourse-build-resource/cmd/in")
	if err != nil {
		gt.Expect(err).NotTo(gomega.HaveOccurred())
	}

	err = os.Mkdir("build", os.ModeDir|os.ModePerm)
	if err != nil {
		gt.Expect(err).NotTo(gomega.MatchError("build: file exists"))
	}

	spec.Run(t, "/opt/resource/in", func(t *testing.T, when spec.G, it spec.S) {
		when("given a valid directory and valid JSON", func() {
			when("in.In() works", func() {
				gt := gomega.NewGomegaWithT(t)
				var session *gexec.Session
				var server *ghttp.Server

				it.Before(func() {
					server = ghttp.NewServer()
					server.RouteToHandler("GET", "/api/v1/info", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(atc.Info{})
					}))
					server.RouteToHandler("GET", "/api/v1/builds/111", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(atc.Build{
							ID:           999,
							Status:       string(atc.StatusSucceeded),
							TeamName:     "t",
							PipelineName: "p",
							JobName:      "j",
							Name:         "111",
						})
					}))
					server.RouteToHandler("GET", "/api/v1/builds/111/resources", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(atc.BuildInputsOutputs{})
					}))
					server.RouteToHandler("GET", "/api/v1/builds/111/plan", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(atc.PublicBuildPlan{})
					}))
					server.RouteToHandler("GET", "/api/v1/builds/111/events", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						io.WriteString(w, "id: 0\nevent: end\ndata")
					}))
					server.RouteToHandler("GET", "/api/v1/teams/t/pipelines/p/jobs/j", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(atc.Job{})
						json.NewEncoder(w).Encode(atc.JobConfigs{})
						json.NewEncoder(w).Encode(atc.JobConfig{})
					}))

					cmd := exec.Command(compiledPath, "build")
					input := fmt.Sprintf(`{"version":{"build_id":"111"},"source":{"concourse_url":"%s","team":"t","pipeline":"p","job":"j"}}`, server.URL())
					cmd.Stdin = bytes.NewBufferString(input)
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it.After(func() {
					server.Close()
				})

				it("prints the version to stdout", func() {
					gt.Eventually(session.Out).Should(gbytes.Say(`"version":{"build_id":"111"}`))
					gt.Eventually(session).Should(gexec.Exit(0))
				})

				it("prints metadata to stdout", func() {
					gt.Eventually(session.Out).Should(gbytes.Say(`"metadata":\[{"name":"build_url","value":"http://127.0.0.1:(\d+)/teams/t/pipelines/p/jobs/j/builds/111"}\]`))
					gt.Eventually(session).Should(gexec.Exit(0))
				})
			}, spec.Nested())

			when("trace is enabled", func() {
				gt := gomega.NewGomegaWithT(t)
				var session *gexec.Session
				var server *ghttp.Server

				it.Before(func() {
					server = ghttp.NewServer()
					server.RouteToHandler("GET", "/api/v1/info", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(atc.Info{})
					}))
					server.RouteToHandler("GET", "/api/v1/builds/111", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(atc.Build{
							ID:           999,
							Status:       string(atc.StatusSucceeded),
							TeamName:     "t",
							PipelineName: "p",
							JobName:      "j",
						})
					}))
					server.RouteToHandler("GET", "/api/v1/builds/111/resources", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(atc.BuildInputsOutputs{})
					}))
					server.RouteToHandler("GET", "/api/v1/builds/111/plan", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(atc.PublicBuildPlan{})
					}))
					server.RouteToHandler("GET", "/api/v1/teams/t/pipelines/p/jobs/j", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(atc.Job{})
					}))
					server.RouteToHandler("GET", "/api/v1/builds/111/events", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						io.WriteString(w, "id: 0\nevent: end\ndata")
					}))

					cmd := exec.Command(compiledPath, "build")
					input := fmt.Sprintf(`{"version":{"build_id":"111"},"source":{"concourse_url":"%s","team":"t","pipeline":"p","job":"j","enable_tracing":true}}`, server.URL())
					cmd.Stdin = bytes.NewBufferString(input)
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it.After(func() {
					server.Close()
				})

				it("prints the trace", func() {
					gt.Eventually(session.Err).Should(gbytes.Say(`GET /api/v1/builds/111`))
					gt.Eventually(session).Should(gexec.Exit(0))
				})
			}, spec.Nested())

			when("something goes wrong with in.In()", func() {
				gt := gomega.NewGomegaWithT(t)
				var session *gexec.Session
				var server *ghttp.Server

				it.Before(func() {
					server = ghttp.NewServer()

					cmd := exec.Command(compiledPath, "build")
					input := fmt.Sprintf(`{"version":{"build_id":"111"},"source":{"concourse_url":"%s","team":"t","pipeline":"p","job":"j"}}`, server.URL())
					cmd.Stdin = bytes.NewBufferString(input)
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it.After(func() {
					server.Close()
				})

				it("fails with an error", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("failed to perform 'in':"))
					gt.Eventually(session).Should(gexec.Exit(1))
				})
			}, spec.Nested())
		}, spec.Nested())
	}, spec.Report(report.Terminal{}))

	gexec.CleanupBuildArtifacts()
	gt.Expect(os.RemoveAll("build")).To(gomega.Succeed())
}
