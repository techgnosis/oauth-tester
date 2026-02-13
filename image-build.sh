#! /usr/bin/env bash

set -euo pipefail

docker build --platform=linux/amd64 -t docker.io/techgnosis/oauth-tester:1 -f Dockerfile.app .

docker push docker.io/techgnosis/oauth-tester:1
