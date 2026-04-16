FROM golang:1.26.1-bookworm

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    bash \
    ca-certificates \
    iproute2 \
    procps \
    python3 \
    ripgrep \
  && rm -rf /var/lib/apt/lists/*

ENV PYTHONUNBUFFERED=1
WORKDIR /workspace
