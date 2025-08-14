# Full multi-stage build: CLI + enforcer, with runtime image containing both binaries and rules

FROM golang:1.24 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Build promptshield CLI and ps-enforcer
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/promptshield ./cmd/promptshield \
 && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/ps-enforcer ./cmd/ps-enforcer


FROM alpine:3.20
RUN adduser -D -H -u 10001 app \
    && apk add --no-cache ca-certificates \
    && mkdir -p /rules
COPY --from=builder /out/promptshield /usr/local/bin/promptshield
COPY --from=builder /out/ps-enforcer /usr/local/bin/ps-enforcer
COPY rules /rules

USER app
EXPOSE 9090 9091
ENV PS_ENFORCER_ADDR=:9090 \
    PS_ENFORCER_GRPC_ADDR=:9091 \
    PS_ENFORCER_RULEPACK=/rules/basic-security.yaml

HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD wget -qO- http://127.0.0.1:9090/healthz || exit 1

# Default to enforcer; override with `docker run ... promptshield` for CLI
ENTRYPOINT ["/usr/local/bin/ps-enforcer"]

