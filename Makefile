.PHONY: all build build-binary docker-build test test-real clean bump-patch bump-minor bump-major release-minimal

# Version management
VERSION ?= 0.1.0
IMAGE_NAME = hrexed/otelcol-spanconnector
IMAGE_TAG = $(IMAGE_NAME):$(VERSION)
LATEST_TAG = $(IMAGE_NAME):latest

# Directory variables
DIST_DIR=dist

# Platform selection (default: linux on host CPU arch)
PLATFORM ?= linux/$(shell uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/; s/arm64/arm64/')
DOCKER_PLATFORM = --platform=$(PLATFORM)

# Container engine (default: docker, can be overridden with CONTAINER_ENGINE=podman)
CONTAINER_ENGINE ?= docker

all: build-binary

# Build the Go binary locally
build-binary:
	@echo "Building Go binary..."
	@mkdir -p $(DIST_DIR)
	go build -o $(DIST_DIR)/spanEventstoLog .

# Build the custom collector using Docker/Podman (default: linux/amd64)
build: docker-build

# Run all Go tests
# Usage: make test
test:
	go test -v ./...

# Run realistic tests (real data)
test-real:
	go test -v -run TestRealisticSpanEventsToLog || true

# Build Docker/Podman image
docker-build:
	$(CONTAINER_ENGINE) build $(DOCKER_PLATFORM) -t $(IMAGE_TAG) -t $(LATEST_TAG) --build-arg VERSION=$(VERSION) .

clean:
	rm -rf $(DIST_DIR)
	echo "Cleaned build artifacts."

# Version bumping - create VERSION file if it doesn't exist
VERSION_FILE = VERSION
$(VERSION_FILE):
	@echo "$(VERSION)" > $(VERSION_FILE)

bump-patch: $(VERSION_FILE)
	@awk -F. '{
	  $$3++; print $$1"."$$2"."$$3
	}' $(VERSION_FILE) > $(VERSION_FILE).tmp && mv $(VERSION_FILE).tmp $(VERSION_FILE)
	@echo "Bumped patch version to $$(cat $(VERSION_FILE))"

bump-minor: $(VERSION_FILE)
	@awk -F. '{
	  $$2++; $$3=0; print $$1"."$$2"."$$3
	}' $(VERSION_FILE) > $(VERSION_FILE).tmp && mv $(VERSION_FILE).tmp $(VERSION_FILE)
	@echo "Bumped minor version to $$(cat $(VERSION_FILE))"

bump-major: $(VERSION_FILE)
	@awk -F. '{
	  $$1++; $$2=0; $$3=0; print $$1"."$$2"."$$3
	}' $(VERSION_FILE) > $(VERSION_FILE).tmp && mv $(VERSION_FILE).tmp $(VERSION_FILE)
	@echo "Bumped major version to $$(cat $(VERSION_FILE))"

# Build a minimal collector for debugging (uses minimal manifest)
release-minimal:
	$(CONTAINER_ENGINE) build $(DOCKER_PLATFORM) -t $(IMAGE_TAG)-minimal -f Dockerfile . --build-arg MANIFEST=ocb/manifest_minimal.yaml

# Usage:
#   make build-binary                    # Build Go binary locally
#   make build [CONTAINER_ENGINE=docker|podman] [PLATFORM=linux/amd64|linux/arm64|...] [VERSION=x.y.z]
#   make docker-build [CONTAINER_ENGINE=docker|podman] [PLATFORM=linux/amd64|linux/arm64|...] [VERSION=x.y.z]
#   make release-minimal [CONTAINER_ENGINE=docker|podman] [PLATFORM=linux/amd64|linux/arm64|...] [VERSION=x.y.z]
#   make test
#   make test-real
#   make bump-patch | bump-minor | bump-major 