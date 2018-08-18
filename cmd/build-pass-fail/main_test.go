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
	spec.Run(t, "build-pass-fail", func(t *testing.T, when spec.G, it spec.S) {
		var compiledPath string
		var err error
		var cmd *exec.Cmd
		var session *gexec.Session

		gt := gomega.NewGomegaWithT(t)

		it.Before(func() {
			compiledPath, err = gexec.Build("github.com/jchesterpivotal/concourse-build-resource/cmd/build-pass-fail")
			if err != nil {
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			}

			cmd = exec.Command(compiledPath)
		})

		it.After(func() {
			gexec.CleanupBuildArtifacts()
		})

		when("there is no build/build.json", func() {
			it.Before(func() {
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
				err = os.Mkdir("build", os.ModeDir|os.ModePerm)
				if err != nil {
					gt.Expect(err).NotTo(gomega.MatchError("build: file exists"))
				}
			})

			it.After(func() {
				err = os.RemoveAll("build")
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			when("build.json represents a successful build", func() {
				it.Before(func() {
					completed, err := os.Create(filepath.Join("build", "build.json"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = completed.WriteString(`{
                        "api_url": "api_url",
                        "end_time": 1111111111,
                        "id": 222222,
                        "job_name": "job_name",
                        "name": "333",
                        "pipeline_name": "pipeline_name",
                        "start_time": 9999999999,
                        "status": "succeeded",
                        "team_name": "team_name"
                    }`)
					gt.Expect(err).NotTo(gomega.HaveOccurred())

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

			when("build.json represents an unsuccessful build", func() {
				it.Before(func() {
					completed, err := os.Create(filepath.Join("build", "build.json"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = completed.WriteString(`{
                        "api_url": "api_url",
                        "end_time": 1111111111,
                        "id": 222222,
                        "job_name": "job_name",
                        "name": "333",
                        "pipeline_name": "pipeline_name",
                        "start_time": 9999999999,
                        "status": "test status",
                        "team_name": "team_name"
                    }`)
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("prints a failure message", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("Build /teams/team_name/pipelines/pipeline_name/jobs/job_name/builds/333 was unsuccessful & finished with status 'test status'"))
				})

				it("exits 1", func() {
					gt.Eventually(session).Should(gexec.Exit(1))
				})
			})

			when("the build.json file is malformed", func() {
				it.Before(func() {
					malformed, err := os.Create(filepath.Join("build", "build.json"))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = malformed.WriteString(`} {  [] {{ malformed JSON file: ""`)
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("fails with an error", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("could not parse build/build.json"))
				})
			})
		})
	}, spec.Report(report.Terminal{}))
}
