BIN=promptshield
PKG=github.com/promptshield/promptshield
VERSION?=0.2.0
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: build build-enforcer run tidy fmt lint test bench bench-quick bench-large

build:
	go build -ldflags "-X '$(PKG)/cmd.version=$(VERSION)' -X '$(PKG)/cmd.commit=$(COMMIT)' -X '$(PKG)/cmd.buildDate=$(DATE)'" -o bin/$(BIN) ./cmd/promptshield

build-enforcer:
    go build -ldflags "-X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.buildDate=$(DATE)'" -o bin/ps-enforcer ./cmd/ps-enforcer

run: build
	./bin/$(BIN) --help

tidy:
	go mod tidy

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

test:
	go test ./...

bench:
	go test -run '^$' -bench '.' -benchmem ./...

bench-quick:
	go test -run '^$' -bench '^BenchmarkP95L1L2$' -benchmem -count=3 ./internal/scanner

bench-large:
	go test -run '^$' -bench '^BenchmarkScanOneGiB$' -benchmem -count=1 ./internal/scanner


