.PHONY: all build test test-real docker-build clean bump-patch bump-minor bump-major release-minimal

# Version management
VERSION ?= 0.1.0
IMAGE_NAME = hrexed/otelcol-spanconnector
IMAGE_TAG = $(IMAGE_NAME):$(VERSION)
LATEST_TAG = $(IMAGE_NAME):latest

# Directory variables
SRC_DIR=src
DIST_DIR=dist

# Platform selection (default: linux/amd64)
PLATFORM ?= linux/amd64
DOCKER_PLATFORM = --platform=$(PLATFORM)

# Container engine (default: docker, can be overridden with CONTAINER_ENGINE=podman)
CONTAINER_ENGINE ?= docker

all: build

# Build the custom collector using Docker/Podman (default: linux/amd64)
build:
	$(CONTAINER_ENGINE) build $(DOCKER_PLATFORM) -t $(IMAGE_TAG) -t $(LATEST_TAG) --build-arg VERSION=$(VERSION) .

# Run all Go tests
# Usage: make test
test:
	cd $(SRC_DIR) && go test ./...

# Run realistic tests (real data)
test-real:
	cd $(SRC_DIR) && go test -v -run TestRealisticSpanEventsToLog || true

# Build Docker/Podman image (alias for build)
docker-build: build

clean:
	rm -rf $(DIST_DIR)/*.o $(DIST_DIR)/otelcol-custom
	echo "Cleaned build artifacts."

# Version bumping
define bump_version
	awk -F. 'BEGIN {OFS="."} {$(1+$(2=="patch"?3:2))++; if("$(1)"=="bump-minor"){$$3=0} if("$(1)"=="bump-major"){$$2=0;$$3=0} print $$1,$$2,$$3}' VERSION > VERSION.tmp && mv VERSION.tmp VERSION
endef

bump-patch:
	@awk -F. '{
	  $$3++; print $$1"."$$2"."$$3
	}' VERSION > VERSION.tmp && mv VERSION.tmp VERSION

bump-minor:
	@awk -F. '{
	  $$2++; $$3=0; print $$1"."$$2"."$$3
	}' VERSION > VERSION.tmp && mv VERSION.tmp VERSION

bump-major:
	@awk -F. '{
	  $$1++; $$2=0; $$3=0; print $$1"."$$2"."$$3
	}' VERSION > VERSION.tmp && mv VERSION.tmp VERSION

# Build a minimal collector for debugging (uses minimal manifest)
release-minimal:
	$(CONTAINER_ENGINE) build $(DOCKER_PLATFORM) -t $(IMAGE_TAG)-minimal -f Dockerfile . --build-arg MANIFEST=ocb/manifest_minimal.yaml

# Usage:
#   make build [CONTAINER_ENGINE=docker|podman] [PLATFORM=linux/amd64|linux/arm64|...] [VERSION=x.y.z]
#   make docker-build [CONTAINER_ENGINE=docker|podman] [PLATFORM=linux/amd64|linux/arm64|...] [VERSION=x.y.z]
#   make release-minimal [CONTAINER_ENGINE=docker|podman] [PLATFORM=linux/amd64|linux/arm64|...] [VERSION=x.y.z]
#   make test
#   make test-real
#   make bump-patch | bump-minor | bump-major 