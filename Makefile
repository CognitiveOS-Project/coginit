SHELL := /bin/sh
.SHELLFLAGS := -eu -c
.ONESHELL:
.DELETE_ON_ERROR:

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
LDFLAGS := -ldflags '-s -w -X github.com/CognitiveOS-Project/coginit/internal/coginit.version=$(VERSION)'

.PHONY: build build-static clean test lint

build:
	mkdir -p build/bin
	CGO_ENABLED=0 go build $(LDFLAGS) -o build/bin/coginit ./cmd/coginit

build-static:
	mkdir -p build/bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a $(LDFLAGS) -extldflags '-static' -o build/bin/coginit ./cmd/coginit
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a $(LDFLAGS) -extldflags '-static' -o build/bin/coginit-arm64 ./cmd/coginit

test:
	go test ./... -v -count=1

lint:
	go vet ./...

clean:
	rm -rf build/
