.PHONY: build test clean run install

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

build:
	@echo "Building mingyue-agent..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/mingyue-agent ./cmd/agent

test:
	@echo "Running tests..."
	go test -v -race -cover ./...

clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

run: build
	@echo "Starting mingyue-agent..."
	./bin/mingyue-agent start

install: build
	@echo "Installing mingyue-agent..."
	install -m 755 bin/mingyue-agent /usr/local/bin/

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Linting code..."
	go vet ./...

tidy:
	@echo "Tidying dependencies..."
	go mod tidy

.DEFAULT_GOAL := build
