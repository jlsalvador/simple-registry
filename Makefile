# Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.DEFAULT_GOAL := build

BUILD_CURRENT_VERSION := $(strip $(shell git describe --tags --match='[0-9]*.[0-9]*.[0-9]*' 2>/dev/null || printf 0.0.1))
BUILD_VERSION_MAJOR ?= $(word 1, $(subst ., ,$(BUILD_CURRENT_VERSION)))
BUILD_VERSION_MINOR ?= $(word 2, $(subst ., ,$(BUILD_CURRENT_VERSION)))
BUILD_VERSION_PATCH ?= $(shell date --utc +%s)
export BUILD_VERSION := $(BUILD_VERSION_MAJOR).$(BUILD_VERSION_MINOR).$(BUILD_VERSION_PATCH)

MODULE_NAME := $(shell grep ^module go.mod | cut -d' ' -f2)
VERSION_PKG := $(MODULE_NAME)/internal/version

NPROCS = $(shell grep -c 'processor' /proc/cpuinfo || printf 1)
MAKEFLAGS += -j$(NPROCS)

BUILD_DIR ?= $(shell pwd)/build
LDFLAGS=\
	-X $(VERSION_PKG).AppVersion=${BUILD_VERSION} \
	-extldflags=-static -w -s

BINARY_NAME=simple-registry
ARCHITECTURES=x86-64 arm64
GO_SOURCE=$(wildcard *.go)
BINARIES=$(foreach ARCH, ${ARCHITECTURES}, ${BUILD_DIR}/${BINARY_NAME}.${BUILD_VERSION}.${ARCH})
BINARIES_UPX=$(foreach BINARY, ${BINARIES}, ${BINARY}.upx)

CONTAINER_TOOL ?= podman
PLATFORMS ?= linux/arm64,linux/amd64
IMG ?= ${BINARY_NAME}:latest

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOCYCLO ?= $(LOCALBIN)/gocyclo
MISSPELL ?= $(LOCALBIN)/misspell

## Tool Versions
GOCYCLO_VERSION ?= latest
MISSPELL_VERSION ?= latest

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


##@ Cleaning targets

.PHONY: clean
clean: ## Remove build directory.
	go clean ; \
	rm -rf "${BUILD_DIR}"

.PHONE: container_clean
container_clean: ## Remove container images.
	-$(CONTAINER_TOOL) rmi $(IMG)
	-$(CONTAINER_TOOL) manifest rm $(IMG)

.PHONY: mrproper
mrproper: clean ## Remove all generated files.
	rm -rf "${LOCALBIN}"


##@ Build

.PHONY: _mkdir_build
_mkdir_build:
	mkdir -p "${BUILD_DIR}"

${BUILD_DIR}/cover.out: _mkdir_build
	go test \
		-covermode=count \
		-coverprofile "${BUILD_DIR}/cover.out" \
		$(shell go list ./... | grep -v /vendor/ | tr '\n' ' ')

${BUILD_DIR}/cover.txt: ${BUILD_DIR}/cover.out
	go tool cover -func="${BUILD_DIR}/cover.out" -o "${BUILD_DIR}/cover.txt"

${BUILD_DIR}/cover.html: ${BUILD_DIR}/cover.out
	go tool cover -html="${BUILD_DIR}/cover.out" -o "${BUILD_DIR}/cover.html"

.PHONY: cover
cover: ${BUILD_DIR}/cover.txt ${BUILD_DIR}/cover.html ## Generate coverture reports.

# -gcflags="all=-N -l"
%.x86-64: ${GO_SOURCE} _mkdir_build
	GOOS=linux GOARCH=amd64 \
		go build \
			-trimpath \
			-ldflags="${LDFLAGS}" \
			-o $@ \
			./cmd/${BINARY_NAME}
	ln -sf $(notdir $@) $(subst .${BUILD_VERSION}.,.latest.,$@)

%.arm64: ${GO_SOURCE} _mkdir_build
	GOOS=linux GOARCH=arm64 \
		go build \
			-trimpath \
			-ldflags="${LDFLAGS}" \
			-o $@ \
			./cmd/${BINARY_NAME}
	ln -sf $(notdir $@) $(subst .${BUILD_VERSION}.,.latest.,$@)

.PHONY: build
build: ${BINARIES} ## Build project binary.

.PHONY: container
container: container_clean ${GO_SOURCE} ## Build container images
	$(CONTAINER_TOOL) build \
		--manifest $(IMG) \
		--platform=$(PLATFORMS) \
		--build-arg="LDFLAGS=${LDFLAGS}" \
		--build-arg="TARGETBIN=$(BINARY_NAME)" \
		-f Dockerfile .

.PHONY: publish
publish: container ## Publish container images
	$(CONTAINER_TOOL) manifest push \
		--all \
		$(IMG)

.PHONY: all
all: | clean test build ## Execute all tipical targets before publish.


##@ Test Dependencies

.PHONY: gocyclo
gocyclo: $(GOCYCLO) ## Download gocyclo locally if necessary.
$(GOCYCLO): $(LOCALBIN)
	test -s $(LOCALBIN)/gocyclo || GOBIN=$(LOCALBIN) go install github.com/fzipp/gocyclo/cmd/gocyclo@$(GOCYCLO_VERSION)

.PHONY: misspell
misspell: $(MISSPELL) ## Download misspell locally if necessary.
$(MISSPELL): $(LOCALBIN)
	test -s $(LOCALBIN)/misspell || GOBIN=$(LOCALBIN) go install github.com/client9/misspell/cmd/misspell@$(MISSPELL_VERSION)


##@ Test

.PHONY: cyclo
test-cyclo: gocyclo ## Run gocyclo against code.
	$(GOCYCLO) -over 15 .

.PHONY: test-misspell
test-misspell: misspell ## Run misspell against code.
	$(MISSPELL) -error cmd docs internal pkg LICENSE Makefile README.md

.PHONY: test-go
test-go: ## Test code.
	go test ./... -cover

.PHONY: test
test: test-cyclo test-misspell test-go ## Execute all tests.


##@ Release

%.upx: %
	upx --best --lzma --no-progress -o $@ $(patsubst %.upx,%,$@)

.PHONY: upx
upx: ${BINARIES_UPX} ## Compress project binaries with UPX.
