FROM docker.io/golang:1.25

RUN useradd -m -u 501 -g 20 jamesmusselwhite

USER jamesmusselwhite
ENV PATH="/home/jamesmusselwhite/.local/bin:${PATH}"


RUN curl -fsSL https://claude.ai/install.sh | bash
RUN curl -sSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

WORKDIR /workspace
