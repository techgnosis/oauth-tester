#! /usr/bin/env bash

GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o oauth-tester-mac --mod=vendor