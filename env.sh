#! /usr/bin/env bash

set -euox pipefail

docker run \
-it \
--rm \
-v ~/.claude.json:/home/jamesmusselwhite/.claude.json \
-v ~/.claude:/home/jamesmusselwhite/.claude \
-v ~/.gitconfig:/home/jamesmusselwhite/.gitconfig \
-v ~/.ssh:/home/jamesmusselwhite/.ssh:ro \
-v .:/workspace \
oauth-tester-env:1 bash
