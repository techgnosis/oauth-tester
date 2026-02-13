#! /usr/bin/env bash

set euo -pipefail

docker build -t oauth-tester-env:1 -f Dockerfile.env .
