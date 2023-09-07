#!/bin/bash
docker run --rm -v "$PWD":/go/src/app -w /go/src/app -e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=amd64 golang:latest go build -o ./badminton/badminton.linux -ldflags "-s -w -linkmode external -extldflags -static" ./badminton/main.go
