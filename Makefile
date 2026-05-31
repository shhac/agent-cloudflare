BINARY := agent-cloudflare
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GOCACHE ?= $(CURDIR)/.cache/go-build

build:
	GOCACHE=$(GOCACHE) go build -buildvcs=false -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/agent-cloudflare

build-mock:
	GOCACHE=$(GOCACHE) go build -buildvcs=false -o mockcloudflare ./cmd/mockcloudflare

mock:
	GOCACHE=$(GOCACHE) go run ./cmd/mockcloudflare

mock-dev:
	AGENT_CLOUDFLARE_BASE_URL=http://127.0.0.1:12112 CLOUDFLARE_API_TOKEN=cfut_mock GOCACHE=$(GOCACHE) go run ./cmd/agent-cloudflare $(ARGS)

test:
	GOCACHE=$(GOCACHE) go test ./... -count=1

test-short:
	GOCACHE=$(GOCACHE) go test ./... -count=1 -short

fmt:
	gofmt -w .
	@command -v goimports >/dev/null && goimports -w . || echo "goimports not installed (optional; install: go install golang.org/x/tools/cmd/goimports@latest)"

vet:
	GOCACHE=$(GOCACHE) go vet ./...

clean:
	rm -f $(BINARY)
	rm -f mockcloudflare
	rm -rf dist/

dev:
	GOCACHE=$(GOCACHE) go run ./cmd/agent-cloudflare $(ARGS)

.PHONY: build build-mock mock mock-dev test test-short fmt vet clean dev
