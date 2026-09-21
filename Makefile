VERSION := $(shell git describe --always --abbrev=0 --tags --match "v*" 2>/dev/null || echo "v0.0.0")
BUILD := $(shell git rev-parse --short HEAD 2>/dev/null || echo "HEAD")
PROJECTNAME := "go-template"
# We pass that to the main module to generate the correct help text
PROGRAMNAME := $(PROJECTNAME)

BASEDIR := $(shell pwd)

SCRIPTS_HOME := "$(BASEDIR)/scripts"

# Protobuf setup related
PROTOC_HOME := "$(BASEDIR)/protoc"
PROTOC_SOURCES := "$(BASEDIR)/proto"
PROTOC_VERSION := "36.2"

BIN := "$(BASEDIR)/bin"
GOBUILD := "$(BASEDIR)/build"
GENERATED := "$(BASEDIR)/generated"
# Build tools are pinned via `tool` directives in go.mod and installed here,
# never into the global GOBIN.
TOOLS_BIN := "$(BASEDIR)/.bin"

# Go related
ifndef $(GOPATH)
    GOPATH=$(shell go env GOPATH)
    export GOPATH
endif

GOBIN := "$(GOPATH)/bin"
GOHOSTOS := $(shell go env GOHOSTOS)
GOHOSTARCH := $(shell go env GOHOSTARCH)
GOFILES := $(shell find . -type f -name '*.go' -not -path './vendor/*')
GOOS_DARWIN := "darwin"
GOOS_LINUX := "linux"
GOOS_WINDOWS := "windows"
GOARCH_AMD64 := "amd64"
GOARCH_ARM64 := "arm64"
GOARCH_ARM := "arm"

MODFLAGS=-mod=readonly

# Use linker flags to provide version/build settings
LDFLAGS=-ldflags "-w -s -X=main.Version=$(VERSION) -X=main.Build=$(BUILD) -X=main.ProgramName=$(PROGRAMNAME)"

# Redirect error output to a file, so we can show it in development mode.
STDERR := $(GOBUILD)/.$(PROJECTNAME)-stderr.txt

# PID file will keep the process id of the server
PID := $(GOBUILD)/.$(PROJECTNAME).pid

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent

.PHONY: default
## default: install, format, lint, build, test
default: install format lint build test

.PHONY: init
## init: Initializes a new repository from this template (OWNER, REPO, GOVERSION required)
init:
	@test -n "$(OWNER)"     || { echo "OWNER is required: make init OWNER=<owner> REPO=<repo> GOVERSION=<x.y>"; exit 1; }
	@test -n "$(REPO)"      || { echo "REPO is required: make init OWNER=<owner> REPO=<repo> GOVERSION=<x.y>"; exit 1; }
	@test -n "$(GOVERSION)" || { echo "GOVERSION is required: make init OWNER=<owner> REPO=<repo> GOVERSION=<x.y>"; exit 1; }
	go run ./internal/bootstrap --owner $(OWNER) --repo $(REPO) --go-version $(GOVERSION)

.PHONY: ci-checks
## ci-checks: install, lint (includes the gofmt check), test
ci-checks: install lint test

.PHONY: install
## install: Checks for missing dependencies and installs them
install: go-get go-install

.PHONY: format
## format: Formats Go source files
format: go-format

.PHONY: lint
## lint: Runs all linters including go vet, golangci-lint and format check
lint: generate-sources go-vet go-lint go-format-check

.PHONY: generate-sources
generate-sources: go-proto-gen

.PHONY: build
## build: Builds binaries for all supported platforms
build:
	@[ -d $(GOBUILD) ] || mkdir -p $(GOBUILD)
	@-mkdir -p $(GOBUILD)/completions
	@-touch $(STDERR)
	@-rm $(STDERR)
	@-$(MAKE) -s go-build #2> $(STDERR)
	# generate completions
	bin/$(PROGRAMNAME)-$(GOHOSTOS)-$(GOHOSTARCH) completion zsh > $(GOBUILD)/completions/_$(PROGRAMNAME)
	bin/$(PROGRAMNAME)-$(GOHOSTOS)-$(GOHOSTARCH) completion bash > $(GOBUILD)/completions/$(PROGRAMNAME).bash
	bin/$(PROGRAMNAME)-$(GOHOSTOS)-$(GOHOSTARCH) completion fish > $(GOBUILD)/completions/$(PROGRAMNAME).fish

	#@cat $(STDERR) | sed -e '1s/.*/\nError:\n/'  | sed 's/make\[.*/ /' | sed "/^/s/^/     /" 1>&2

.PHONY: test
## test: Runs all Go tests
test: generate-sources go-test

.PHONY: coverage
## coverage: Runs all Go tests and generates a coverage report
coverage: generate-sources
	go test $(MODFLAGS) -count=1 -coverpkg=./... -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: coverage-html
## coverage-html: Runs tests and opens the coverage report in a browser
coverage-html: coverage
	go tool cover -html=coverage.out

.PHONY: clean
## clean: Removes build artifacts
clean:
	@-rm $(BIN)/$(PROGRAMNAME)* 2> /dev/null
	@-rm -rf $(GENERATED)/* 2> /dev/null
	@-rm -rf $(BIN)/* 2> /dev/null
	@-rm -rf $(GOBUILD)/* 2> /dev/null
	@-rm -rf $(TOOLS_BIN)/* 2> /dev/null
	@-$(MAKE) go-clean

.PHONY: go-vet
go-vet:
	@echo "  >  Vetting source files..."
	go vet $(MODFLAGS) ./...

.PHONY: go-lint
go-lint:
	@echo "  >  Linting source files..."
	go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run ./...

.PHONY: go-format-check
go-format-check:
	@echo "  >  Checking formatting of source files..."
	@if [ -n "$$(gofmt -l $(GOFILES))" ]; then \
		echo "  >  Format check failed for the following files:"; \
		gofmt -l $(GOFILES); \
		exit 1; \
	fi

# Function-level complexity gate only (thresholds live in .golangci.yml).
# `make lint` also enforces these; this target exists so CI can surface a
# threshold breach as its own red check.
.PHONY: complexity
## complexity: Runs function-level complexity analysis (thresholds live in .golangci.yml)
complexity:
	@echo "  >  Running complexity analysis..."
	go tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run --default=none --enable=gocognit,gocyclo ./...

.PHONY: go-format
go-format:
	@echo "  >  Formating source files..."
	gofmt -s -w $(GOFILES)

.PHONY: go-build-current
go-build-current:
	@echo "  >  Building $(GOHOSTOS)/$(GOHOSTARCH) binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME) $(BASEDIR)/cmd

.PHONY: go-build
go-build: go-get generate-sources go-build-linux-amd64 go-build-linux-arm go-build-linux-arm64 go-build-darwin-amd64 go-build-darwin-arm64 go-build-windows-amd64 go-build-windows-arm64

.PHONY: go-test
go-test:
	go test $(MODFLAGS) `go list $(MODFLAGS) ./...`

.PHONY: go-build-linux-amd64
go-build-linux-amd64:
	@echo "  >  Building linux amd64 binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_AMD64) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_LINUX)-$(GOARCH_AMD64) $(BASEDIR)/cmd

.PHONY: go-build-linux-arm
go-build-linux-arm:
	@echo "  >  Building linux arm binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_ARM) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_LINUX)-$(GOARCH_ARM) $(BASEDIR)/cmd

.PHONY: go-build-linux-arm64
go-build-linux-arm64:
	@echo "  >  Building linux arm64 binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_ARM64) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_LINUX)-$(GOARCH_ARM64) $(BASEDIR)/cmd

.PHONY: go-build-darwin-amd64
go-build-darwin-amd64:
	@echo "  >  Building darwin amd64 binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_DARWIN) GOARCH=$(GOARCH_AMD64) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_DARWIN)-$(GOARCH_AMD64) $(BASEDIR)/cmd

.PHONY: go-build-darwin-arm64
go-build-darwin-arm64:
	@echo "  >  Building darwin arm64 binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_DARWIN) GOARCH=$(GOARCH_ARM64) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_DARWIN)-$(GOARCH_ARM64) $(BASEDIR)/cmd

.PHONY: go-build-windows-amd64
go-build-windows-amd64:
	@echo "  >  Building windows amd64 binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_WINDOWS) GOARCH=$(GOARCH_AMD64) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_WINDOWS)-$(GOARCH_AMD64).exe $(BASEDIR)/cmd

.PHONY: go-build-windows-arm64
go-build-windows-arm64:
	@echo "  >  Building windows arm64 binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_WINDOWS) GOARCH=$(GOARCH_ARM64) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_WINDOWS)-$(GOARCH_ARM64).exe $(BASEDIR)/cmd

.PHONY: go-proto-gen
go-proto-gen: go-install
	@echo "  >  Generating protobuf sources..."
	@[ -d $(PROTOC_HOME) ] || "$(SCRIPTS_HOME)/setup_protoc.sh" "$(PROTOC_VERSION)" "$(PROTOC_HOME)"
	@[ -d $(GENERATED) ] || mkdir -p $(GENERATED)
	PATH=$(TOOLS_BIN):$$PATH $(PROTOC_HOME)/bin/protoc --go_out=$(GENERATED) -I=$(PROTOC_HOME)/include -I=$(PROTOC_SOURCES) $(PROTOC_SOURCES)/*.proto

.PHONY: go-get
go-get:
	@echo "  >  Downloading module dependencies..."
	@go mod download

.PHONY: go-install
go-install:
	@echo "  >  Installing build tools into $(TOOLS_BIN)..."
	@GOBIN=$(TOOLS_BIN) go install tool

.PHONY: go-clean
go-clean:
	@echo "  >  Cleaning build cache"
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean $(MODFLAGS) $(BASEDIR)
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean -modcache

.PHONY: goreleaser-release
goreleaser-release:
ifdef GITHUB_TOKEN
	@echo "  >  Releasing..."
	goreleaser release --clean
else
	$(error GITHUB_TOKEN is not set)
endif

.PHONY: release
## release: Builds and publishes a release via GoReleaser (requires GITHUB_TOKEN)
release: build goreleaser-release

.PHONY: all
all: help

.PHONY: help
## help: Prints this help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
