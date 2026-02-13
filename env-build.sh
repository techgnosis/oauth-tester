#! /usr/bin/env bash

set euo -pipefail

cp "$(mkcert -CAROOT)/rootCA.pem" ./mkcert.pem
docker build -t oauth-tester-env:1 -f Dockerfile.env .
