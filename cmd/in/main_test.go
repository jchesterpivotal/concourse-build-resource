package main_test

import (
	"testing"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/concourse/atc"

	"os"
	"os/exec"
	"bytes"
	"net/http"
	"encoding/json"
	"fmt"
	"io"
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
					server.RouteToHandler("GET", "/api/v1/builds/111", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(atc.Build{ID: 999, Status: string(atc.StatusSucceeded)})
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
