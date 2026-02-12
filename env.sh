#! /usr/bin/bash

set -euox pipefail

docker run \
-it \
--rm \
-u $(id -u):$(id -g) \
-v ~/.claude.json:/home/james/.claude.json \
-v ~/.claude:/home/james/.claude \
-v ~/.gitconfig:/home/james/.gitconfig \
-v ~/.ssh:/home/james/.ssh:ro \
-v .:/workspace \
golang-claude-beads:1 bash
