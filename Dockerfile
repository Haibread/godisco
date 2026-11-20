FROM golang:1.27.4-alpine3.24@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder

WORKDIR /usr/local/src/godisco

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /usr/local/bin/godisco ./app/godisco \
 && mkdir -p /app/config

FROM gcr.io/distroless/static-debian13:nonroot@sha256:aafe993665bf6200db38a02c11a9351e782a84b6a45e7f33b6695638c88e3931

LABEL org.opencontainers.image.source="https://github.com/Haibread/godisco"

COPY --from=builder /usr/local/bin/godisco /usr/local/bin/godisco
COPY --from=builder --chown=65532:65532 /app/config /app/config

# The app reads config.yaml and writes channels.db under ./config, and existing
# deployments mount their data at /app/config: moving it would orphan them.
WORKDIR /app
VOLUME ["/app/config"]

USER nonroot:nonroot

HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD ["/usr/local/bin/godisco", "healthcheck"]

ENTRYPOINT ["/usr/local/bin/godisco"]
