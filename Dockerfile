# ============================================================
# Build context: repo root (casa-nova/)
#
# Build:
#   docker build -t <registry>/casa-nova-agent:latest .
#
# Push:
#   docker push <registry>/casa-nova-agent:latest
# ============================================================

# ── Builder ─────────────────────────────────────────────────
FROM golang:1.25-bookworm AS builder

# CGO dependencies required by confluent-kafka-go (builds librdkafka from source)
RUN apt-get update && apt-get install -y --no-install-recommends \
  gcc \
  g++ \
  make \
  pkg-config \
  libssl-dev \
  libsasl2-dev \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy service source
COPY . .

RUN go mod download

RUN CGO_ENABLED=1 GOOS=linux \
  go build -ldflags="-s -w" -o /bin/agent .

# ── Runtime ──────────────────────────────────────────────────
FROM debian:bookworm-slim AS runtime

# Runtime shared libraries needed by confluent-kafka-go and TLS
RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates \
  libssl3 \
  libsasl2-2 \
  && rm -rf /var/lib/apt/lists/*

RUN useradd -r -u 1001 -g root appuser

WORKDIR /app

COPY --from=builder /bin/agent .

USER appuser

EXPOSE 9000

ENTRYPOINT ["./agent"]
