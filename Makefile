.PHONY: build test test-cover lint fmt vet clean check

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/dreadnought-inc/genbu/internal/cli.version=$(VERSION)"
GOLANGCI_LINT := $(shell command -v golangci-lint 2>/dev/null || echo "$(shell go env GOPATH)/bin/golangci-lint")

build:
	go build $(LDFLAGS) -o bin/genbu ./cmd/genbu

test:
	go test -race -count=1 ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	$(GOLANGCI_LINT) run ./...

fmt:
	gofmt -s -w .
	goimports -w .

vet:
	go vet ./...

check: vet lint test

clean:
	rm -rf bin/ coverage.out coverage.html dist/
