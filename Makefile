# (c) Copyright 2018 Hewlett Packard Enterprise Development LP

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

# http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GO_VERSION = 1.13
VERSION = $(shell git tag|tail -n1)
ifeq ($(VERSION),)
VERSION = v0.0.0
endif
ifndef BUILD_NUMBER
BUILD_NUMBER = 0
endif

# Where our code lives
PKG_PATH = ./
VEN_PATH = vendor
# This is the commit id of the branch we're building from
COMMIT = $(shell git log -n 1 --pretty=format:"%H")
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

# The version of make for OSX doesn't allow us to export, so
# we add these variables to the env in each invocation.
GOENV = PATH=$$PATH:$(GOPATH)/bin

# Our target binary is for Linux.  To build an exec for your local (non-linux)
# machine, use go build directly.
ifndef GOOS
GOOS = linux
endif
TEST_ENV  = GOOS=$(GOOS) GOARCH=amd64
BUILD_ENV = GOOS=$(GOOS) GOARCH=amd64 CGO_ENABLED=0 VERSION=$(VERSION) COMMIT=$(COMMIT)

# Add the version and commit id to the binary in the form of variables.
LD_FLAGS = '-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)'

# gometalinter allows us to have a single target that runs multiple linters in
# the same fashion.  This variable controls which linters are used.
LINTER_FLAGS = --disable-all --enable=vet --enable=vetshadow --enable=golint --enable=ineffassign --enable=goconst --enable=dupl --enable=gocyclo --deadline=240s

# list of packages
PACKAGE_LIST =   $(shell export $(GOENV) && go list ./$(PKG_PATH)...| grep -v vendor)

# prefixes to make things pretty
A1 = $(shell printf "»")
A2 = $(shell printf "»»")
A3 = $(shell printf "»»»")
S0 = 😁
S1 = 😔

.PHONY: help
help:
	@echo "Targets:"
	@echo "    tools          - Download and install go tooling required to build."
	@echo "    vendor         - Download dependencies."
	@echo "    lint           - Static analysis of source code.  Note that this must pass in order to build."
	@echo "    test           - Run unit tests."
	@echo "    int            - Run integration tests.  (Not implemented yet)."
	@echo "    clean          - Remove build artifacts."
	@echo "    debug          - Display make's view of the world."
	@echo "    all            - Build all cmds."
	@echo "    all_local      - Build all cmds for local OS (make sure GOOS is set)."
	@echo "    packages       - Build packages."

.PHONY: debug
debug:
	@echo "Debug:"
	@echo "  Go:           `go version`"
	@echo "  GOPATH:       $(GOPATH)"
	@echo "  Packages:     $(PACKAGE_LIST)"
	@echo "  VERSION:      $(VERSION)"
	@echo "  BRANCH:       $(BRANCH)"
	@echo "  COMMIT:       $(COMMIT)"
	@echo "  BUILD_NUMBER: $(BUILD_NUMBER)"
	@echo "  LD_FLAGS:     $(LD_FLAGS)"
	@echo "  BUILD_ENV:    $(BUILD_ENV)"
	@echo "  COMMOM_GIT:   $(COMMOM_GIT)"
	@echo "  GOENV:        $(GOENV)"
	@echo "$(S0)"

.PHONY : all
all : clean lint docker_run

.PHONY : all_local
all_local : clean debug lint packages test

# this is the target called from within the container
.PHONY: container_all
container_all: debug packages test

.PHONY: tools
tools: ; $(info $(A1) gettools)
	@echo "$(A2) get golangci-lint"
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

vendor: tools; $(info $(A1) vendor)
	@echo "$(A2) go mod vendor"
	go mod vendor
	@echo "$(S0)"

build: ; $(info $(A1) mkdir build)
	@mkdir build
	@echo "$(S0)"

.PHONY: lint
lint:
	@echo "Running lint"
	@go version
	export $(GOENV) $(BUILD_ENV) && golangci-lint run $(LINTER_FLAGS) --exclude vendor

.PHONY: clean
clean: ; $(info $(A1) clean)
	@echo "$(A2) remove build"
	@rm -rf build
	@echo "$(A2) remove src"
	@rm -rf src
	@echo "$(A2) remove bin"
	@rm -rf bin
	@echo "$(A2) remove pkg"
	@rm -rf pkg
	@echo "$(A2) remove vendor"
	@rm -rf vendor
	@echo "$(S0)"

.PHONY: test
test: packages; $(info $(A1) test)
	@echo "$(A2) unit tests"
ifeq ("$(GOOS)","linux")
	export $(GOENV) $(TEST_ENV) && ./package_tester.sh $(PACKAGE_LIST)
else
	@echo "Skipping tests... only linux is supported!"
endif
	@echo "$(S0)"

.PHONY: int
int: ; $(info $(A1) int)
	@echo "$(A2) There are no integration tests yet."
	@echo "$(S1)"

.PHONY: packages
packages: build ; $(info $(A1) packages)
	@echo "$(A2) build packages"
	export $(GOENV) $(BUILD_ENV) && ./package_builder.sh $(PACKAGE_LIST)
	@echo "$(S0)"

.PHONY: docker_run
docker_run: ; $(info $(A1) docker_run)
	@echo "$(A2) using docker image for build"
	docker run --env BACKEND --rm -t -v $(GOPATH):/go -w /go golang:$(GO_VERSION) sh -c "cd src/github.com/hpe-storage/common-host-libs && export XDG_CACHE_HOME=/tmp/.cache && make container_all"
	@echo "$(A2) leaving container happy $(S0)"
