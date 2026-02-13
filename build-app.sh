#! /usr/bin/env bash

set euo -pipefail

docker build -t docker.io/techgnosis/oauth-tester:1 -f Dockerfile.app .
