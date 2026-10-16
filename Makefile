# iceflow — Makefile
#
# Common developer tasks. Version metadata is derived from git so that
# `iceflow version` reports something meaningful for local builds.

BINARY   := iceflow
PKG      := github.com/aice109/ice-flow-cli
CMD      := ./cmd/$(BINARY)
BINDIR   := bin

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w \
	-X $(PKG)/internal/buildinfo.Version=$(VERSION) \
	-X $(PKG)/internal/buildinfo.Commit=$(COMMIT) \
	-X $(PKG)/internal/buildinfo.Date=$(DATE)

.PHONY: all build install uninstall test test-race vet fmt lint tidy clean run snapshot

all: build

## build: compile the binary into ./bin
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/$(BINARY) $(CMD)

## install: install into GOBIN (or GOPATH/bin)
install:
	go install -ldflags "$(LDFLAGS)" $(CMD)

## uninstall: remove a previously installed binary from GOBIN
uninstall:
	rm -f "$(shell go env GOBIN)/$(BINARY)" "$(shell go env GOPATH)/bin/$(BINARY)"

## test: run the unit tests
test:
	go test ./...

## test-race: run tests with the race detector
test-race:
	go test -race ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: format the tree with gofmt
fmt:
	gofmt -w .

## lint: fail if anything is unformatted, then vet
lint:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)
	go vet ./...

## tidy: sync go.mod/go.sum
tidy:
	go mod tidy

## run: build and run against temporary state
run: build
	ICEFLOW_HOME="$$(mktemp -d)" ./$(BINDIR)/$(BINARY)

## snapshot: build release artifacts locally without publishing
snapshot:
	goreleaser release --snapshot --clean

## clean: remove build output
clean:
	rm -rf $(BINDIR) dist
