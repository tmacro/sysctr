MODULE   = $(shell $(GO) list -m)
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || echo v0)
SCHEMAS := $(wildcard pkg/types/schemas/*.json)

BIN      = bin
GO      = go

.PHONY: all
all: generate fmt tidy build test

.PHONY: build
build: generate
	$(GO) build \
		-tags release \
		-ldflags '-X $(MODULE)/cmd.Version=$(VERSION) -X $(MODULE)/cmd.BuildDate=$(DATE)' \
		-o sysctr ./cmd

.PHONY: test
test:
	@go test ./... -cover

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

.PHONY: generate
generate:
	@go-jsonschema \
		--struct-name-from-title \
		--extra-imports \
		-p github.com/tmacro/sysctr/pkg/types \
		-o pkg/types/types_gen.go \
		$(SCHEMAS)

$(BIN):
	@mkdir -p $@

$(BIN)/%: | $(BIN)
	env GOBIN=$(abspath $(BIN)) $(GO) install $(PACKAGE)

$(BIN)/goreleaser: PACKAGE=github.com/goreleaser/goreleaser/v2@latest

GORELEASER = $(BIN)/goreleaser

release: $(GORELEASER)
	$(GORELEASER) release --clean

snapshot: $(GORELEASER)
	$(GORELEASER) release --snapshot --clean
