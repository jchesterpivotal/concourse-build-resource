package main_test

import (
	"testing"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"os"
	"path/filepath"
	"os/exec"
)

func TestShowLogs(t *testing.T) {
	gt := gomega.NewGomegaWithT(t)

	compiledPath, err := gexec.Build("github.com/jchesterpivotal/concourse-build-resource/cmd/show-logs")
	if err != nil {
		gt.Expect(err).NotTo(gomega.HaveOccurred())
	}

	err = os.Mkdir("build", os.ModeDir|os.ModePerm)
	if err != nil {
		gt.Expect(err).NotTo(gomega.MatchError("build: file exists"))
	}

	err = os.Mkdir("successful", os.ModeDir|os.ModePerm)
	if err != nil {
		gt.Expect(err).NotTo(gomega.MatchError("successful: file exists"))
	}

	err = os.Mkdir("empty", os.ModeDir|os.ModePerm)
	if err != nil {
		gt.Expect(err).NotTo(gomega.MatchError("empty: file exists"))
	}

	err = os.Mkdir("wrappers", os.ModeDir|os.ModePerm)
	if err != nil {
		gt.Expect(err).NotTo(gomega.MatchError("wrappers: file exists"))
	}

	spec.Run(t, "show-logs", func(t *testing.T, when spec.G, it spec.S) {
		when("a resource name is given", func() {
			gt = gomega.NewGomegaWithT(t)

			when("the directory exists and contains an events.log", func() {
				var session *gexec.Session

				it.Before(func() {
					logfile, err := os.Create(filepath.Join("successful", "events.log"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = logfile.WriteString("successful/events.log log line 1\nlog line the 2nd\n")
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					cmd := exec.Command(compiledPath, "successful")
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("prints the logs to stdout", func() {
					gt.Eventually(session.Out).Should(gbytes.Say(`successful/events.log log line 1
log line the 2nd
`))
				})
			}, spec.Nested())

			when("the directory exists but does not contain an events.log", func() {
				var session *gexec.Session

				it.Before(func() {
					cmd := exec.Command(compiledPath, "empty")
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("prints a failure message", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("could not open empty/events.log"))
				})

				it("exits 1", func() {
					gt.Eventually(session).Should(gexec.Exit(1))
				})
			}, spec.Nested())
		}, spec.Nested())

		when("a resource name is not given", func() {
			gt = gomega.NewGomegaWithT(t)

			when("there is no build/events.log", func() {
				var session *gexec.Session

				it.Before(func() {
					cmd := exec.Command(compiledPath)
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("prints a failure message", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("could not open build/events.log"))
				})

				it("exits 1", func() {
					gt.Eventually(session).Should(gexec.Exit(1))
				})
			}, spec.Nested())

			when("there is a build/events.log", func() {
				var session *gexec.Session

				it.Before(func() {
					logfile, err := os.Create(filepath.Join("build", "events.log"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = logfile.WriteString("build/events.log log line 1\nlog line the 2nd\n")
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					cmd := exec.Command(compiledPath)
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("prints the logs to stdout", func() {
					gt.Eventually(session.Out).Should(gbytes.Say(`build/events.log log line 1
log line the 2nd
`))
				})
			}, spec.Nested(), spec.Sequential())

		}, spec.Nested())

		when("printing any events.log file", func() {
			gt = gomega.NewGomegaWithT(t)
			var session *gexec.Session

			it.Before(func() {
				logfile, err := os.Create(filepath.Join("wrappers", "events.log"))
				gt.Expect(err).NotTo(gomega.HaveOccurred())

				_, err = logfile.WriteString("wrappers/events.log log line 1")
				gt.Expect(err).NotTo(gomega.HaveOccurred())

				cmd := exec.Command(compiledPath, "wrappers")
				session, err = gexec.Start(cmd, it.Out(), it.Out())
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it("wraps the printed logs with 'begin' and 'end' lines for clarity", func() {
				gt.Eventually(session.Out).Should(gbytes.Say(`----------------------------------- \[ begin log \] -----------------------------------`))
				gt.Eventually(session.Out).Should(gbytes.Say(`wrappers/events\.log log line 1`))
				gt.Eventually(session.Out).Should(gbytes.Say(`------------------------------------ \[ end log \] ------------------------------------`))
			})
		}, spec.Nested())
	}, spec.Report(report.Terminal{}))

	gexec.CleanupBuildArtifacts()
	gt.Expect(os.RemoveAll("build")).To(gomega.Succeed())
	gt.Expect(os.RemoveAll("successful")).To(gomega.Succeed())
	gt.Expect(os.RemoveAll("empty")).To(gomega.Succeed())
	gt.Expect(os.RemoveAll("wrappers")).To(gomega.Succeed())
}
