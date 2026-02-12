#! /usr/bin/env bash

set -euox pipefail

docker run \
-it \
--rm \
-u $(id -u):$(id -g) \
-v ~/.claude.json:/home/jamesmusselwhite/.claude.json \
-v ~/.claude:/home/jamesmusselwhite/.claude \
-v ~/.gitconfig:/home/jamesmusselwhite/.gitconfig \
-v ~/.ssh:/home/jamesmusselwhite/.ssh:ro \
-v .:/workspace \
oauth-tester:1 bash
