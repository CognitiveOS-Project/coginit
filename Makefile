SHELL := /bin/sh
.SHELLFLAGS := -eu -c
.ONESHELL:
.DELETE_ON_ERROR:

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
LDFLAGS := -ldflags '-s -w -X main.version=$(VERSION)'

.PHONY: build build-static clean test lint

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/coginit .

build-static:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a $(LDFLAGS) -extldflags '-static' -o bin/coginit .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a $(LDFLAGS) -extldflags '-static' -o bin/coginit-arm64 .

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin/
