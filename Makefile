export CGO_ENABLED:=0

DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
VERSION=$(shell git describe --tags --match=v* --always --dirty)
LD_FLAGS="-s -w -X github.com/tmacro/sysctr/version.Version=$(VERSION)"

REPO=github.com/tmacro/sysctr
LOCAL_REPO=tmacro/sysctr
IMAGE_REPO=ghcr.io/tmacro/sysctr
SCHEMAS := $(wildcard pkg/types/schemas/*.json)

.PHONY: all
all: tidy build test vet fmt

.PHONY: build
build: gen
	@go build -o sysctr -ldflags $(LD_FLAGS) $(REPO)/cmd

.PHONY: test
test:
	@go test ./... -cover

.PHONY: vet
vet:
	@go vet -all ./...

.PHONY: fmt
fmt:
	@test -z $$(go fmt ./...)

.PHONY: lint
lint:
	@golangci-lint run ./...

.PHONY: tidy
tidy:
	@go mod tidy

clean:
	@rm -f sysctr

.PHONY: gen
gen:
	@go-jsonschema \
		--struct-name-from-title \
		--extra-imports \
		-p github.com/tmacro/sysctr/pkg/types \
		-o pkg/types/types_gen.go \
		$(SCHEMAS)
