package main_test

import (
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"testing"

	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TestShowCommandsWhichPrettyPrintJson tests the three commands which pretty-print JSON files output by the resource.
// They are: show-build, show-plan and show-resource.
// They are actually thin wrappers around a module, so for convenience I consolidate their tests here.
// Not included in this test is show-logs, since it does not involve pretty printing JSON.
func TestShowCommandsWhichPrettyPrintJson(t *testing.T) {
	var compiledPath string
	var err error
	var session *gexec.Session
	var gt *gomega.GomegaWithT

	commandsToTest := map[string]string{
		"show-build":     "github.com/jchesterpivotal/concourse-build-resource/cmd/show-build",
		"show-plan":      "github.com/jchesterpivotal/concourse-build-resource/cmd/show-plan",
		"show-resources": "github.com/jchesterpivotal/concourse-build-resource/cmd/show-resources",
		"show-job":       "github.com/jchesterpivotal/concourse-build-resource/cmd/show-job",
	}

	for cmdName, cmdPath := range commandsToTest {
		gt = gomega.NewGomegaWithT(t)

		compiledPath, err = gexec.Build(cmdPath)
		if err != nil {
			gt.Expect(err).NotTo(gomega.HaveOccurred())
		}

		err = os.Mkdir("build", os.ModeDir|os.ModePerm)
		if err != nil {
			gt.Expect(err).NotTo(gomega.MatchError("build: file exists"))
		}

		err = os.Mkdir("valid", os.ModeDir|os.ModePerm)
		if err != nil {
			gt.Expect(err).NotTo(gomega.MatchError("valid: file exists"))
		}

		err = os.Mkdir("invalid", os.ModeDir|os.ModePerm)
		if err != nil {
			gt.Expect(err).NotTo(gomega.MatchError("invalid: file exists"))
		}

		spec.Run(t, cmdName, func(t *testing.T, when spec.G, it spec.S) {
			gt = gomega.NewGomegaWithT(t) // scopes suck
			jsonFileName := strings.TrimPrefix(cmdName, "show-")
			jsonFileName = fmt.Sprintf("%s.json", jsonFileName)

			when("there is a valid JSON file to pretty-print", func() {
				it.Before(func() {
					jsonfile, err := os.Create(filepath.Join("valid", jsonFileName))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = jsonfile.WriteString(`{"test_json": true, "a_second_key": "a second value"}`)
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					cmd := exec.Command(compiledPath, filepath.Join("valid"))
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("prints the prettied JSON to stdout", func() {
					gt.Eventually(session.Out).Should(gbytes.Say(`"a_second_key": "a second value"`))
					gt.Eventually(session.Out).Should(gbytes.Say(`"test_json": true`))
				})
			}, spec.Nested())

			when("no resource name is provided", func() {
				it(fmt.Sprintf("defaults to printing %s", jsonFileName), func() {
					jsonfile, err := os.Create(filepath.Join("build", jsonFileName))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = jsonfile.WriteString(`{"test_json": true, "default_path": "yep"}`)
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					cmd := exec.Command(compiledPath)
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("prints the prettied JSON to stdout", func() {
					gt.Eventually(session.Out).Should(gbytes.Say(`"default_path": "yep"`))
					gt.Eventually(session.Out).Should(gbytes.Say(`"test_json": true`))
				})
			}, spec.Nested())

			when("something goes wrong", func() {
				it.Before(func() {
					jsonfile, err := os.Create(filepath.Join("invalid", jsonFileName))
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					_, err = jsonfile.WriteString(`this is {} not valid json`)
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					cmd := exec.Command(compiledPath, filepath.Join("invalid"))
					session, err = gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())
				})

				it("fails with an error", func() {
					gt.Eventually(session.Err).Should(gbytes.Say("could not parse invalid/"))
					gt.Eventually(session).Should(gexec.Exit(1))
				})
			}, spec.Nested())
		}, spec.Report(report.Terminal{}))

		gexec.CleanupBuildArtifacts()
		gt.Expect(os.RemoveAll("build")).To(gomega.Succeed())
		gt.Expect(os.RemoveAll("valid")).To(gomega.Succeed())
		gt.Expect(os.RemoveAll("invalid")).To(gomega.Succeed())
	}
}

func TestFileSystemTraversalsArePrevented(t *testing.T) {
	commandsToTest := map[string]string{
		"build-pass-fail": "github.com/jchesterpivotal/concourse-build-resource/cmd/build-pass-fail",
		"show-build":      "github.com/jchesterpivotal/concourse-build-resource/cmd/show-build",
		"show-plan":       "github.com/jchesterpivotal/concourse-build-resource/cmd/show-plan",
		"show-resources":  "github.com/jchesterpivotal/concourse-build-resource/cmd/show-resources",
		"show-job":        "github.com/jchesterpivotal/concourse-build-resource/cmd/show-job",
		"show-logs":       "github.com/jchesterpivotal/concourse-build-resource/cmd/show-logs",
	}

	for cmdName, cmdPath := range commandsToTest {
		gt := gomega.NewGomegaWithT(t)

		compiledPath, err := gexec.Build(cmdPath)
		if err != nil {
			gt.Expect(err).NotTo(gomega.HaveOccurred())
		}

		spec.Run(t, cmdName, func(t *testing.T, when spec.G, it spec.S) {
			when("given paths intended to perform directory traversal", func() {
				gt := gomega.NewGomegaWithT(t)

				it("rejects relative path traversals", func() {
					cmd := exec.Command(compiledPath, "./.././../../sensitive/file")
					session, err := gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					gt.Eventually(session.Err).Should(gbytes.Say("malformed path"))
				})

				it("rejects absolute path traversals", func() {
					cmd := exec.Command(compiledPath, "/absolute/path/to/sensitive/file")
					session, err := gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					gt.Eventually(session.Err).Should(gbytes.Say("malformed path"))
				})
			}, spec.Nested())

			when("any kind of argument other than a single-level directory name is given", func() {
				gt := gomega.NewGomegaWithT(t)

				it("rejects nested directory paths", func() {
					cmd := exec.Command(compiledPath, "nested/directory/path")
					session, err := gexec.Start(cmd, it.Out(), it.Out())
					gt.Expect(err).NotTo(gomega.HaveOccurred())

					gt.Eventually(session.Err).Should(gbytes.Say("malformed path"))
				})
			}, spec.Nested())
		}, spec.Report(report.Terminal{}))

		gexec.CleanupBuildArtifacts()
	}
}
