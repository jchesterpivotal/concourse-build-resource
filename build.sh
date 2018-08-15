#!/usr/bin/env bash

mkdir assets

GOOS=linux GOARCH=amd64 go build -o assets/check cmd/check/main.go
GOOS=linux GOARCH=amd64 go build -o assets/in cmd/in/main.go

docker build . --tag gcr.io/cf-elafros-dog/concourse-build-resource
docker push gcr.io/cf-elafros-dog/concourse-build-resource

rm -r assets
