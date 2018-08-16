#!/usr/bin/env bash

mkdir assets

GOOS=linux GOARCH=amd64 go build -o assets/check                cmd/check/main.go
GOOS=linux GOARCH=amd64 go build -o assets/in                   cmd/in/main.go

GOOS=linux GOARCH=amd64 go build -o assets/build-pass-fail      cmd/build-pass-fail/main.go

GOOS=linux GOARCH=amd64 go build -o assets/show-build           cmd/show-build/main.go
GOOS=linux GOARCH=amd64 go build -o assets/show-plan            cmd/show-plan/main.go
GOOS=linux GOARCH=amd64 go build -o assets/show-resources       cmd/show-resources/main.go

docker build . --tag gcr.io/cf-elafros-dog/concourse-build-resource
docker push gcr.io/cf-elafros-dog/concourse-build-resource

rm -r assets
