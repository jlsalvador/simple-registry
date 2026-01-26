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

BUILD_CURRENT_VERSION := $(strip $(shell git describe --tags --match='v[0-9]+.[0-9]+.[0-9]+' 2>/dev/null || printf v0.0.1))
BUILD_VERSION_MAJOR ?= $(word 1, $(subst v,,$(subst ., ,$(BUILD_CURRENT_VERSION))))
BUILD_VERSION_MINOR ?= $(word 2, $(subst ., ,$(BUILD_CURRENT_VERSION)))
BUILD_VERSION_PATCH ?= $(shell date --utc +%s)
export BUILD_VERSION := $(BUILD_VERSION_MAJOR).$(BUILD_VERSION_MINOR).$(BUILD_VERSION_PATCH)

MODULE_NAME := $(shell grep ^module go.mod | cut -d' ' -f2)
VERSION_PKG := $(MODULE_NAME)/internal/version

NPROCS = $(shell grep -c 'processor' /proc/cpuinfo || printf 1)
MAKEFLAGS += -j$(NPROCS)

BUILD_DIR ?= $(shell pwd)/build
LDFLAGS_COMMON=-X $(VERSION_PKG).AppVersion=${BUILD_VERSION}
LDFLAGS_PROD=$(LDFLAGS_COMMON) -w -s
LDFLAGS_DEBUG=$(LDFLAGS_COMMON)
GCFLAGS_COMMON =
GCFLAGS_PROD = $(GCFLAGS_COMMON)
GCFLAGS_DEBUG = $(GCFLAGS_COMMON) all=-N -l

BINARY_NAME=$(notdir $(MODULE_NAME))
REPO_URL=https://$(MODULE_NAME)
GO_SOURCE=$(wildcard *.go)

# Detect host platform and architecture
HOST_OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
HOST_ARCH_RAW := $(shell uname -m)
ifeq ($(HOST_ARCH_RAW),x86_64)
	HOST_ARCH := amd64
else ifeq ($(HOST_ARCH_RAW),aarch64)
	HOST_ARCH := arm64
else ifeq ($(HOST_ARCH_RAW),arm64)
	HOST_ARCH := arm64
else
	HOST_ARCH := $(HOST_ARCH_RAW)
endif

# Define platforms to build for
PLATFORMS_LIST := linux-amd64 linux-arm64

# Generate list of binaries
BINARIES=$(foreach PLATFORM, ${PLATFORMS_LIST}, ${BUILD_DIR}/${BINARY_NAME}_${BUILD_VERSION}_${PLATFORM})
BINARIES_LATEST=$(foreach PLATFORM, ${PLATFORMS_LIST}, ${BUILD_DIR}/${BINARY_NAME}_latest_${PLATFORM})

# Generate list of artifacts archives
ARTIFACTS_ARCHIVES=$(foreach PLATFORM, ${PLATFORMS_LIST}, ${BUILD_DIR}/${BINARY_NAME}_${BUILD_VERSION}_${PLATFORM}.tar.gz)

CONTAINER_TOOL ?= podman
PLATFORMS ?= linux/arm64,linux/amd64
IMG ?= ${BINARY_NAME}:latest
# Derive debug image tag
IMG_NAME := $(word 1,$(subst :, ,$(IMG)))
IMG_TAG  := $(or $(word 2,$(subst :, ,$(IMG))),latest)
ifeq ($(IMG_TAG),latest)
IMG_DEBUG := $(IMG_NAME):debug
else
IMG_DEBUG := $(IMG_NAME):$(IMG_TAG)-debug
endif

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

.PHONY: container-clean-prod
container-clean-prod:
	-$(CONTAINER_TOOL) rmi $(IMG)
	-$(CONTAINER_TOOL) manifest rm $(IMG)

.PHONY: container-clean-debug
container-clean-debug:
	-$(CONTAINER_TOOL) rmi $(IMG_DEBUG)
	-$(CONTAINER_TOOL) manifest rm $(IMG_DEBUG)

.PHONY: container-clean
container-clean: container-clean-prod container-clean-debug  ## Remove container images.

.PHONY: mrproper
mrproper: clean container-clean ## Remove all generated files.
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
cover: ${BUILD_DIR}/cover.txt ${BUILD_DIR}/cover.html ## Generate coverage reports.

# Rule to build linux-amd64 binaries
${BUILD_DIR}/%_linux-amd64: ${GO_SOURCE} _mkdir_build
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build \
			-trimpath \
			-ldflags="${LDFLAGS_PROD}" \
			-gcflags="${GCFLAGS_PROD}" \
			-o $@ \
			./cmd/${BINARY_NAME}

# Rule to build linux-arm64 binaries
${BUILD_DIR}/%_linux-arm64: ${GO_SOURCE} _mkdir_build
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		go build \
			-trimpath \
			-ldflags="${LDFLAGS_PROD}" \
			-gcflags="${GCFLAGS_PROD}" \
			-o $@ \
			./cmd/${BINARY_NAME}

.PHONY: build
build: ${BINARIES} ## Build project binaries.

.PHONY: _create_symlinks
_create_symlinks: build
	@echo "Creating symbolic links..."
	@# Latest links
	@for platform in $(PLATFORMS_LIST); do \
		if [ -f "${BUILD_DIR}/${BINARY_NAME}_${BUILD_VERSION}_$${platform}" ]; then \
			ln -sf "${BINARY_NAME}_${BUILD_VERSION}_$${platform}" "${BUILD_DIR}/${BINARY_NAME}_latest_$${platform}"; \
		fi; \
	done
	@# Link for host platform
	@if [ -f "${BUILD_DIR}/${BINARY_NAME}_${BUILD_VERSION}_${HOST_OS}-${HOST_ARCH}" ]; then \
		ln -sf "${BINARY_NAME}_${BUILD_VERSION}_${HOST_OS}-${HOST_ARCH}" "${BUILD_DIR}/${BINARY_NAME}"; \
		echo "Created symlink for host platform: ${BUILD_DIR}/${BINARY_NAME} -> ${BINARY_NAME}_${BUILD_VERSION}_${HOST_OS}-${HOST_ARCH}"; \
	fi

.PHONY: _create_repo_url_file
_create_repo_url_file: _mkdir_build
	@echo "$(REPO_URL)" > "${BUILD_DIR}/REPOSITORY.txt"

# Create compressed archives for release
${BUILD_DIR}/%.tar.gz: ${BUILD_DIR}/% _create_repo_url_file
	@echo "Creating release archive: $@"
	@TEMP_DIR=$$(mktemp -d); \
	cp "$<" "$${TEMP_DIR}/${BINARY_NAME}"; \
	cp README.md "$${TEMP_DIR}/" 2>/dev/null || true; \
	cp LICENSE "$${TEMP_DIR}/" 2>/dev/null || true; \
	cp "${BUILD_DIR}/REPOSITORY.txt" "$${TEMP_DIR}/" 2>/dev/null || true; \
	tar -czf "$@" -C "$${TEMP_DIR}" .; \
	rm -rf "$${TEMP_DIR}"

.PHONY: artifacts-archives
artifacts-archives: ${ARTIFACTS_ARCHIVES} ## Create artifacts archives (.tar.gz).

.PHONY: artifacts-checksums
artifacts-checksums: artifacts-archives ## Generate SHA256 checksums for artifacts files.
	@echo "Generating SHA256 checksums..."
	@cd "${BUILD_DIR}" && \
	find . -type f \( -name "*.tar.gz" -o -name "${BINARY_NAME}_${BUILD_VERSION}_*" \) ! -name "*.sha256" | \
	sort | \
	xargs sha256sum > "${BINARY_NAME}_${BUILD_VERSION}.sha256"
	@echo "Checksums saved to: ${BUILD_DIR}/${BINARY_NAME}_${BUILD_VERSION}.sha256"

.PHONY: artifacts
artifacts: build _create_symlinks artifacts-archives artifacts-checksums ## Build complete GitHub artifacts (binaries, archives, checksums).
	@echo ""
	@echo "========================================"
	@echo "Artifacts build complete!"
	@echo "========================================"
	@echo "Version: ${BUILD_VERSION}"
	@echo "Build directory: ${BUILD_DIR}"
	@echo ""
	@echo "Release artifacts:"
	@ls -lh "${BUILD_DIR}"/${BINARY_NAME}_${BUILD_VERSION}_*
	@echo ""
	@echo "Upload these files:"
	@echo "  - ${BUILD_DIR}/${BINARY_NAME}_${BUILD_VERSION}_*.tar.gz"
	@echo "  - ${BUILD_DIR}/${BINARY_NAME}_${BUILD_VERSION}.sha256"

.PHONY: container-prod
container-prod: container-clean-prod ${GO_SOURCE}
	$(CONTAINER_TOOL) build \
		--target prod \
		--manifest $(IMG) \
		--platform=$(PLATFORMS) \
		--build-arg="GCFLAGS=${GCFLAGS_PROD}" \
		--build-arg="LDFLAGS=${LDFLAGS_PROD}" \
		--build-arg="TARGETBIN=$(BINARY_NAME)" \
		--build-arg="BUILD_TYPE=prod" \
		-f Dockerfile .

.PHONY: container-debug
container-debug: container-clean-debug ${GO_SOURCE}
	$(CONTAINER_TOOL) build \
		--target debug \
		--manifest $(IMG_DEBUG) \
		--platform=$(PLATFORMS) \
		--build-arg="GCFLAGS=${GCFLAGS_DEBUG}" \
		--build-arg="LDFLAGS=${LDFLAGS_DEBUG}" \
		--build-arg="TARGETBIN=$(BINARY_NAME)" \
		--build-arg="BUILD_TYPE=debug" \
		-f Dockerfile .

.PHONY: container
container: container-prod container-debug ## Build both prod and debug images.

.PHONY: all
all: | clean test build ## Execute all typical targets before publish.


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

.PHONY: test-cyclo
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

.PHONY: publish-prod
publish-prod: container-prod
	$(CONTAINER_TOOL) manifest push --all $(IMG)

.PHONY: publish-debug
publish-debug: container-debug
	$(CONTAINER_TOOL) manifest push --all $(IMG_DEBUG)

.PHONY: publish
publish: publish-prod publish-debug ## Publish container images
