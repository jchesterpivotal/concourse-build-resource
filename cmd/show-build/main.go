package main

import "github.com/jchesterpivotal/concourse-build-resource/pkg/prettyjson"

func main() {
	prettyjson.Prettify("build.json")
}
