#!/usr/bin/make -f

SHELL=/bin/bash -o pipefail

MODULE   = $(shell env GO111MODULE=on go list -m)
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null)
BIN      = ./bin

LDFLAGS  = -X $(MODULE)/version.Version=$(VERSION)
LDFLAGS += -X $(MODULE)/version.BuildDate=$(DATE)

.PHONY: all
all: test build

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN)/func .
	@du -h $(BIN)/func

.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)"

.PHONY: clean
clean:
	@rm -rf $(BIN)

.PHONY: test
test:
	@go test ./...
