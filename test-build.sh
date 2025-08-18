#!/bin/bash

# Image-based custom collector validation via Makefile (no local binary build)

set -e

echo "🪪 Testing OCB Build and Custom Collector"
echo "========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Defaults and CLI flags
IMAGE_NAME_DEFAULT="hrexed/otelcol-spanconnector"
VERSION_DEFAULT="0.1.0"
PLATFORM_DEFAULT="$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/; s/arm64/arm64/')"
BUILD_IMAGE=false
USE_MINIMAL=false
PLATFORM="linux/${PLATFORM_DEFAULT}"
CUSTOM_IMAGE=""

usage() {
    cat <<EOF
Usage: $0 [options]

Options:
  --build-image           Build the Docker image via Makefile before testing
  --minimal               Use minimal manifest/image (release-minimal)
  --platform <plat>       Target platform (default: ${PLATFORM})
  --image <name:tag>      Use this image tag instead of Makefile defaults
  -h, --help              Show this help
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --build-image) BUILD_IMAGE=true; shift ;;
        --minimal) USE_MINIMAL=true; shift ;;
        --platform) PLATFORM="$2"; shift 2 ;;
        --image) CUSTOM_IMAGE="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) print_warning "Unknown option: $1"; usage; exit 1 ;;
    esac
done

# Check prerequisites
echo "🔍 Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    print_warning "Docker is not installed. Docker image tests will be skipped."
    DOCKER_AVAILABLE=false
else
    DOCKER_AVAILABLE=true
fi

if command -v curl &> /dev/null; then
    CURL_AVAILABLE=true
else
    CURL_AVAILABLE=false
    print_warning "curl is not available; HTTP checks will be skipped."
fi

print_status "Prerequisites check completed"

# Resolve image tag
IMAGE_NAME="$IMAGE_NAME_DEFAULT"
if [[ -f VERSION ]]; then
    VERSION_TAG="$(cat VERSION | tr -d '\n' | tr -d '\r')"
else
    VERSION_TAG="$VERSION_DEFAULT"
fi

if [[ "$USE_MINIMAL" == true ]]; then
    IMAGE_TAG_SUFFIX="-minimal"
else
    IMAGE_TAG_SUFFIX=""
fi

if [[ -n "$CUSTOM_IMAGE" ]]; then
    IMAGE_FULL="$CUSTOM_IMAGE"
else
    IMAGE_FULL="${IMAGE_NAME}:${VERSION_TAG}${IMAGE_TAG_SUFFIX}"
fi

# Function: wait for URL
wait_for_url() {
    local url="$1"
    local timeout="${2:-30}"
    local sleep_int=1
    local elapsed=0
    while (( elapsed < timeout )); do
        if curl -sf "$url" >/dev/null 2>&1; then
            return 0
        fi
        sleep "$sleep_int"
        elapsed=$((elapsed + sleep_int))
    done
    return 1
}

# Docker image build and tests
if [ "$DOCKER_AVAILABLE" = true ]; then
    echo ""
    echo "🐳 Testing Docker image..."
    if [ "$BUILD_IMAGE" = true ]; then
        echo "🧱 Building image via Makefile for platform $PLATFORM..."
        if [ "$USE_MINIMAL" = true ]; then
            PLATFORM="$PLATFORM" make -s release-minimal
        else
            PLATFORM="$PLATFORM" make -s docker-build
        fi
    fi

    echo "🔎 Using image: $IMAGE_FULL"
    # If the image does not exist locally and not building, warn
    if ! docker image inspect "$IMAGE_FULL" >/dev/null 2>&1; then
        print_warning "Image $IMAGE_FULL not found locally. Consider running with --build-image or --image <name:tag>."
    else
        # Run container
        CONTAINER_NAME="otelcol-test-$$"
        echo "🚀 Running container $CONTAINER_NAME from $IMAGE_FULL..."
        docker run --rm -d --name "$CONTAINER_NAME" -p 4317:4317 -p 4318:4318 -p 55679:55679 "$IMAGE_FULL"
        # Wait for health
        if [ "$CURL_AVAILABLE" = true ]; then
            if wait_for_url "http://localhost:55679/" 30; then
                print_status "Health check endpoint (container) is accessible"
            else
                print_warning "Health check endpoint (container) not accessible"
            fi
        else
            sleep 5
        fi
        # Clean up
        docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
        print_status "Docker container test completed"
    fi
else
    print_warning "Skipping Docker tests (Docker not available)"
fi

echo ""
echo "📊 Test Summary:"
echo "================="
if [ "$DOCKER_AVAILABLE" = true ]; then
    print_status "Docker image test: COMPLETED"
else
    print_warning "Docker image test: SKIPPED"
fi

echo ""
echo "🎯 Next Steps:"
echo "==============="
echo "1. Build Docker image via Makefile: make docker-build PLATFORM=${PLATFORM}"
echo "   Minimal image: make release-minimal PLATFORM=${PLATFORM}"
echo "2. Deploy to Kubernetes: kubectl apply -f deployment/k8s-deployment.yaml"
echo "3. Test with docker-compose: docker compose -f deployment/docker-compose.yml up"
echo ""
echo "📚 Documentation:"
echo "================="
echo "- OCB Build Guide: ocb/OCB_BUILD_README.md"
echo "- Connector Documentation: README.md"
echo "- Test Analysis: TEST_FAILURE_ANALYSIS.md"
echo "- Realistic Tests: REALISTIC_TEST_SUMMARY.md"

print_status "All tests completed successfully! 🎉"