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

	spec.Run(t, "build-pass-fail", func(t *testing.T, when spec.G, it spec.S) {
		gt = gomega.NewGomegaWithT(t)

		when("a path to json file is not given", func() {
			when("there is no build/build.json", func() {
				it.Before(func() {
					cmd := exec.Command(compiledPath)
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("fails with an error", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("could not open build/build.json"))
					gt.Eventually(session).Should(gexec.Exit(1))
				})
			})

			when("there is a build/build.json", func() {
				it.Before(func() {
					_, err := os.Create(filepath.Join("build", "build.json"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					cmd := exec.Command(compiledPath)
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("opens and attempts to parse build/build.json", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("could not parse build/build.json"))
				})
			})
		}, spec.Nested())

		when("a path to a json file is given", func() {
			it.Before(func() {
				_, err := os.Create(filepath.Join("build", "a-passed-in-file.json"))
				gt.Expect(err).NotTo(gomega.HaveOccurred())

				cmd := exec.Command(compiledPath, "build/a-passed-in-file.json")
				session, err = gexec.Start(cmd, it.Out(), it.Out())
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it("opens and attempts to parse the file", func() {
				gt.Eventually(session.Err).Should(gbytes.Say("could not parse build/a-passed-in-file.json"))
			})
		}, spec.Nested())

		when("there is a well-formed JSON file", func() {
			when("the file represents a successful build", func() {
				it.Before(func() {
					completed, err := os.Create(filepath.Join("build", "successful-build.json"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = completed.WriteString(`{
						"status": "succeeded",
						"team_name": "team_name",
						"pipeline_name": "pipeline_name",
						"job_name": "job_name",
						"name": "333"
					}`)
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					cmd := exec.Command(compiledPath, "build/successful-build.json")
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("prints a success message", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("Build /teams/team_name/pipelines/pipeline_name/jobs/job_name/builds/333 succeeded"))
				})

				it("exits 0", func() {
					gt.Eventually(session).Should(gexec.Exit(0))
				})
			})

			when("the file represents an unsuccessful build", func() {
				it.Before(func() {
					completed, err := os.Create(filepath.Join("build", "unsuccessful-build.json"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = completed.WriteString(`{
						"status": "unsuccessful status for test",
						"team_name": "team_name",
						"pipeline_name": "pipeline_name",
						"job_name": "job_name",
						"name": "333"
					}`)
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					cmd := exec.Command(compiledPath, "build/unsuccessful-build.json")
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("prints a failure message", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("Build /teams/team_name/pipelines/pipeline_name/jobs/job_name/builds/333 was unsuccessful & finished with status 'unsuccessful status for test'"))
				})

				it("exits 1", func() {
					gt.Eventually(session).Should(gexec.Exit(1))
				})
			})
		}, spec.Nested(), spec.Parallel())

		when("the JSON file is malformed", func() {
			it.Before(func() {
				malformed, err := os.Create(filepath.Join("build", "malformed-build.json"))
				gt.Expect(err).NotTo(gomega.HaveOccurred())

				_, err = malformed.WriteString(`} {  [] {{ malformed JSON file: ""`)
				gt.Expect(err).NotTo(gomega.HaveOccurred())

				cmd := exec.Command(compiledPath, "build/malformed-build.json")
				session, err = gexec.Start(cmd, it.Out(), it.Out())
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it("fails with an error", func() {
				gt.Eventually(session.Err).Should(gbytes.Say("could not parse build/malformed-build.json"))
			})
		}, spec.Nested())
	}, spec.Report(report.Terminal{}))

	gexec.CleanupBuildArtifacts()

	err = os.RemoveAll("build")
	gt.Expect(err).NotTo(gomega.HaveOccurred())
}
