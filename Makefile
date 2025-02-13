GO                 := $(shell command -v go 2> /dev/null)
GO_MAJOR_VERSION   := $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_MINOR_VERSION   := $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)

VERSION            := $(shell git describe --tags --abbrev=0)
BUILD              := $(shell git rev-parse HEAD)

#Removing windows while waiting for https://github.com/glebarez/go-sqlite/pull/179 to be merged
PLATFORMS          := $(shell go tool dist list | grep -E 'linux|darwin' | grep -E 'amd64|arm64|386')

BIN                := xkcd
LDFLAGS            := -X github.com/jucrouzet/xkcd/cmd.VERSION=$(VERSION) \
                      -X github.com/jucrouzet/xkcd/cmd.BUILD=$(BUILD)

LINTER_TOOL        := github.com/golangci/golangci-lint/cmd/golangci-lint@latest
CHANGELOG_TOOL     := github.com/git-chglog/git-chglog/cmd/git-chglog@latest


all: clean build-bins

deps: go-check
	@echo ">> Installing dependencies..."
	@go mod tidy

install-hooks: go-check deps
	@echo ">> Installing hooks dependencies..."
	@go get github.com/leodido/go-conventionalcommits
	@go get github.com/leodido/go-conventionalcommits/parser
	@echo ">> Compiling commit-msg hook..."
	@go build -o .git/hooks/commit-msg .git-templates/commit-msg/main.go
	@echo ">> Installing pre-commit message hook..."
	@cp .git-templates/hooks/pre-commit .git/hooks/pre-commit

clean-bin:
	@echo ">> Cleaning old binaries"
	@rm -f dist/*

go-check:
	@[ "${GO}" ] || ( echo ">> Go is not installed"; exit 1 )
	@if [ $(GO_MAJOR_VERSION) -ne 1 ]; then \
		( echo ">> Go v1.x is required"; exit 1 )\
	elif [ $(GO_MINOR_VERSION) -lt 22 ] ; then \
		( echo ">> Go >= v1.22 is required"; exit 1 )\
	fi

go-format: go-check
	@echo ">> formatting code"
	@go fmt ./...

go-vet: go-check
	@echo ">> vetting code"
	@go vet ./...

go-lint: go-format go-vet 
	@echo ">> linting code"
	@go run ${LINTER_TOOL} run -c configs/golangci.yaml ./...

build-bins-platform: $(PLATFORMS)
build-bins: build-bins-platform $(PLATFORMS)

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

$(PLATFORMS):
	@echo ">> Building $(os)/$(arch) ..."
	@GOOS=$(os) GOARCH=$(arch) go build -ldflags "$(LDFLAGS)" -o 'dist/$(BIN)-$(os)-$(arch)' main.go

test: go-check
	@echo ">> Running tests"
	@go test -v ./...

changelog:
	@echo ">> Updating changelog ..."
	@go run ${CHANGELOG_TOOL} -c configs/git-chglog.yml -o CHANGELOG.md

clean: clean-bin
checks: go-check
lint: go-lint
prepare: install-hooks clean deps
build: build-bins
