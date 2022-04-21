# Set VERSION to the latest version tag name. Assuming version tags are formatted 'v*'
VERSION := $(shell git describe --always --abbrev=0 --tags --match "v*" $(git rev-list --tags --max-count=1))
BUILD := $(shell git rev-parse $(VERSION))
PROJECTNAME := "go-template"
# We pass that to the main module to generate the correct help text
PROGRAMNAME := $(PROJECTNAME)

BASEDIR := $(shell pwd)

SCRIPTS_HOME := "$(BASEDIR)/scripts"

# Protobuf setup related
PROTOC_HOME := "$(BASEDIR)/protoc"
PROTOC_SOURCES := "$(BASEDIR)/proto"
PROTOC_VERSION := "3.20.0"

BIN := "$(BASEDIR)/bin"
BUILD := "$(BASEDIR)/build"
GENERATED := "$(BASEDIR)/generated"

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

MODFLAGS= #-mod=readonly

# Use linker flags to provide version/build settings
LDFLAGS=-ldflags "-X=main.Version=$(VERSION) -X=main.Build=$(BUILD) -X=main.ProgramName=$(PROGRAMNAME)"

# Redirect error output to a file, so we can show it in development mode.
STDERR := $(BUILD)/.$(PROJECTNAME)-stderr.txt

# PID file will keep the process id of the server
PID := $(BUILD)/.$(PROJECTNAME).pid

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent

default: install format lint build test

ci-checks: install format lint test

install: go-get go-install

format: go-format

lint: generate-sources go-lint

generate-sources: go-proto-gen

.PHONY: build
build:
	@[ -d $(BUILD) ] || mkdir -p $(BUILD)
	@-mkdir -p $(BUILD)/completions
	@-touch $(STDERR)
	@-rm $(STDERR)
	@-$(MAKE) -s go-build #2> $(STDERR)
	# generate completions
	bin/$(PROGRAMNAME)-$(GOHOSTOS)-$(GOHOSTARCH) completion zsh > $(BUILD)/completions/_$(PROGRAMNAME)
	bin/$(PROGRAMNAME)-$(GOHOSTOS)-$(GOHOSTARCH) completion bash > $(BUILD)/completions/$(PROGRAMNAME).bash
	bin/$(PROGRAMNAME)-$(GOHOSTOS)-$(GOHOSTARCH) completion fish > $(BUILD)/completions/$(PROGRAMNAME).fish

	#@cat $(STDERR) | sed -e '1s/.*/\nError:\n/'  | sed 's/make\[.*/ /' | sed "/^/s/^/     /" 1>&2

test: generate-sources go-test

clean:
	@-rm $(BIN)/$(PROGRAMNAME)* 2> /dev/null
	@-rm -rf $(GENERATED)/* 2> /dev/null
	@-rm -rf $(BIN)/* 2> /dev/null
	@-rm -rf $(BUILD)/* 2> /dev/null
	@-$(MAKE) go-clean

go-lint:
	@echo "  >  Linting source files..."
	go vet $(MODFLAGS) -c=10 `go list $(MODFLAGS) ./...`

go-format:
	@echo "  >  Formating source files..."
	gofmt -s -w $(GOFILES)

go-build-current:
	@echo "  >  Building $(GOHOSTOS)/$(GOHOSTARCH) binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME) $(BASEDIR)/cmd

go-build: go-get generate-sources go-build-linux-amd64 go-build-linux-arm go-build-linux-arm64 go-build-darwin-amd64 go-build-darwin-arm64 go-build-windows-amd64 go-build-windows-arm

go-test:
	go test $(MODFLAGS) -covermode=count `go list $(MODFLAGS) ./...`

go-build-linux-amd64:
	@echo "  >  Building linux amd64 binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_AMD64) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_LINUX)-$(GOARCH_AMD64) $(BASEDIR)/cmd

go-build-linux-arm:
	@echo "  >  Building linux arm binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_ARM) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_LINUX)-$(GOARCH_ARM) $(BASEDIR)/cmd

go-build-linux-arm64:
	@echo "  >  Building linux arm64 binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_ARM64) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_LINUX)-$(GOARCH_ARM64) $(BASEDIR)/cmd

go-build-darwin-amd64:
	@echo "  >  Building darwin amd64 binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_DARWIN) GOARCH=$(GOARCH_AMD64) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_DARWIN)-$(GOARCH_AMD64) $(BASEDIR)/cmd

go-build-darwin-arm64:
	@echo "  >  Building darwin arm64 binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_DARWIN) GOARCH=$(GOARCH_ARM64) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_DARWIN)-$(GOARCH_ARM64) $(BASEDIR)/cmd

go-build-windows-amd64:
	@echo "  >  Building windows amd64 binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_WINDOWS) GOARCH=$(GOARCH_AMD64) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_WINDOWS)-$(GOARCH_AMD64).exe $(BASEDIR)/cmd

go-build-windows-arm:
	@echo "  >  Building windows arm binaries..."
	@GOPATH=$(GOPATH) GOOS=$(GOOS_WINDOWS) GOARCH=$(GOARCH_ARM) GOBIN=$(GOBIN) go build $(MODFLAGS) $(LDFLAGS) -o $(BIN)/$(PROGRAMNAME)-$(GOOS_WINDOWS)-$(GOARCH_ARM).exe $(BASEDIR)/cmd

go-proto-gen:
	@echo "  >  Generating protobuf sources..."
	@[ -d $(PROTOC_HOME) ] || "$(SCRIPTS_HOME)/setup_protoc.sh" "$(PROTOC_VERSION)" "$(PROTOC_HOME)"
	@[ -d $(GENERATED) ] || mkdir -p $(GENERATED)
	PATH=$$PATH:$(GOBIN) $(PROTOC_HOME)/bin/protoc --go_out=$(GENERATED) -I=$(PROTOC_HOME)/include -I=$(PROTOC_SOURCES) $(PROTOC_SOURCES)/*.proto

go-get:
	@echo "  >  Downloading build dependencies..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go get google.golang.org/protobuf/cmd/protoc-gen-go
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go get google.golang.org/grpc/cmd/protoc-gen-go-grpc
	# @GOPATH=$(GOPATH) GOBIN=$(GOBIN) go mod tidy

go-install:
	@echo "  >  Installing build dependencies..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

go-clean:
	@echo "  >  Cleaning build cache"
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean $(MODFLAGS) $(BASEDIR)
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean -modcache

goreleaser-release:
ifdef GITHUB_TOKEN
	@echo "  >  Releasing..."
	goreleaser release --rm-dist
else
	$(error GITHUB_TOKEN is not set)
endif

release: build goreleaser-release

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
