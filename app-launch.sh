#! /usr/bin/env bash

docker run \
-it \
--rm \
-p 8443:443 \
-e ISSUER_URL="https://oauth-tester.oauth-tester.svc.cluster.local" \
-e CLIENT_ID="apc" \
techgnosis/oauth-tester:1