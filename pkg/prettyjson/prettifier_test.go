package prettyjson_test

import (
	"testing"
	"github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"os"
	"path/filepath"
	"github.com/jchesterpivotal/concourse-build-resource/pkg/prettyjson"
)

func TestPrettifier(t *testing.T) {
	var gt *gomega.GomegaWithT
	gt = gomega.NewGomegaWithT(t)

	err := os.Mkdir("build", os.ModeDir|os.ModePerm)
	if err != nil {
		gt.Expect(err).NotTo(gomega.MatchError("build: file exists"))
	}

	spec.Run(t, "Prettify()", func(t *testing.T, when spec.G, it spec.S) {
		gt = gomega.NewGomegaWithT(t)

		when("given a file path to a JSON file", func() {
			var response string

			it.Before(func() {
				build, err := os.Create(filepath.Join("build", "valid.json"))
				gt.Expect(err).NotTo(gomega.HaveOccurred())

				_, err = build.WriteString(`{"zzz":false,"aaa":           111}`)
				gt.Expect(err).NotTo(gomega.HaveOccurred())

				response, err = prettyjson.Prettify("build/valid.json")
				gt.Expect(err).NotTo(gomega.HaveOccurred())
			})

			it("prettifies the contents", func() {
				gt.Expect(response).To(gomega.Equal("{\n  \"aaa\": 111,\n  \"zzz\": false\n}"))
			})
		}, spec.Nested())

		when("given a path to a non-existent file", func() {
			var err error

			it.Before(func() {
				_, err = prettyjson.Prettify("nosuchdirectory/nosuchfile.json")
			})

			it("returns an error", func() {
				gt.Expect(err.Error()).To(gomega.ContainSubstring("could not open nosuchdirectory/nosuchfile.json"))
			})
		}, spec.Nested())

		when("given a path to a malformed file", func() {
			var err error

			it.Before(func() {
				var build *os.File
				build, err = os.Create(filepath.Join("build", "invalid.json"))
				gt.Expect(err).NotTo(gomega.HaveOccurred())

				_, err = build.WriteString(`} this is not valid json: ][]`)
				gt.Expect(err).NotTo(gomega.HaveOccurred())

				_, err = prettyjson.Prettify("build/invalid.json")
			})

			it("returns an error", func() {
				gt.Expect(err.Error()).To(gomega.ContainSubstring("could not parse build/invalid.json"))
			})
		}, spec.Nested())
	}, spec.Report(report.Terminal{}))

	err = os.RemoveAll("build")
	gt.Expect(err).NotTo(gomega.HaveOccurred())
}
