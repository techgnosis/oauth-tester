FROM docker.io/golang:1.25

# Create the user to match your host (Assuming UID 1000)
RUN useradd -m -u 501 jamesmusselwhite

RUN curl -L https://github.com/dolthub/dolt/releases/latest/download/install.sh | bash

USER jamesmusselwhite
ENV PATH="/home/james/.local/bin:${PATH}"


RUN curl -fsSL https://claude.ai/install.sh | bash
RUN curl -sSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

WORKDIR /workspace
