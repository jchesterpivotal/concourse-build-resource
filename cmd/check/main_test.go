package main_test

import (
	"testing"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/concourse/atc"

	"os/exec"
	"bytes"
	"fmt"
	"net/http"
	"encoding/json"
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
				server.AppendHandlers(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")

					json.NewEncoder(w).Encode([]atc.Build{{ID: 999, Status: string(atc.StatusSucceeded)}})
				}))

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
				gt.Eventually(session.Out).Should(gbytes.Say(`[{"build_id":"999"}]`))
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
				gt.Eventually(session.Err).Should(gbytes.Say("failed to perform 'check': could not retrieve builds for 'p/j"))
				gt.Eventually(session).Should(gexec.Exit(1))
			})
		}, spec.Nested())
	}, spec.Report(report.Terminal{}))

	gexec.CleanupBuildArtifacts()
}
