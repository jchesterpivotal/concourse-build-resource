package main_test

import (
	"testing"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/gbytes"

	"os/exec"
	"os"
	"path/filepath"
)

func TestBuildPassFail(t *testing.T) {
	var compiledPath string
	var err error
	var session *gexec.Session
	var gt *gomega.GomegaWithT

	gt = gomega.NewGomegaWithT(t)

	compiledPath, err = gexec.Build("github.com/jchesterpivotal/concourse-build-resource/cmd/build-pass-fail")
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

	err = os.Mkdir("unsuccessful", os.ModeDir|os.ModePerm)
	if err != nil {
		gt.Expect(err).NotTo(gomega.MatchError("unsuccessful: file exists"))
	}

	spec.Run(t, "build-pass-fail", func(t *testing.T, when spec.G, it spec.S) {
		gt = gomega.NewGomegaWithT(t)

		when("a resource name is not given", func() {
			when("there is no build/status", func() {
				it.Before(func() {
					cmd := exec.Command(compiledPath)
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("fails with an error", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("could not read build/status"))
					gt.Eventually(session).Should(gexec.Exit(1))
				})
			})

			when("there is no build/url", func() {
				it.Before(func() {
					_, err := os.Create(filepath.Join("build", "status"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					cmd := exec.Command(compiledPath)
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("fails with an error", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("could not read build/url"))
					gt.Eventually(session).Should(gexec.Exit(1))
				})
			})
		}, spec.Nested(), spec.Sequential())

		when("a resource name is given and the directory contains a status file", func() {
			when("the file represents a successful build", func() {
				it.Before(func() {
					completed, err := os.Create(filepath.Join("successful", "status"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = completed.WriteString("succeeded")
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					url, err := os.Create(filepath.Join("successful", "url"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = url.WriteString("https://example.com/path/to/build")
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					cmd := exec.Command(compiledPath, "successful")
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("prints a success message", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("Build https://example.com/path/to/build succeeded"))
				})

				it("exits 0", func() {
					gt.Eventually(session).Should(gexec.Exit(0))
				})
			})

			when("the file represents an unsuccessful build", func() {
				it.Before(func() {
					completed, err := os.Create(filepath.Join("unsuccessful", "status"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = completed.WriteString("unsuccessful status for test")
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					url, err := os.Create(filepath.Join("unsuccessful", "url"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = url.WriteString("https://example.com/path/to/build")
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					cmd := exec.Command(compiledPath, "unsuccessful")
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("prints a failure message", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("Build https://example.com/path/to/build was unsuccessful & finished with status 'unsuccessful status for test'"))
				})

				it("exits 1", func() {
					gt.Eventually(session).Should(gexec.Exit(1))
				})
			})
		}, spec.Nested())
	}, spec.Report(report.Terminal{}))

	gexec.CleanupBuildArtifacts()
	gt.Expect(os.RemoveAll("build")).To(gomega.Succeed())
	gt.Expect(os.RemoveAll("successful")).To(gomega.Succeed())
	gt.Expect(os.RemoveAll("unsuccessful")).To(gomega.Succeed())
}
