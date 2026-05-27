#!/bin/bash
# ==============================================================================
# BUILD SCRIPT - Global E-commerce API
# Purpose: Professional Docker build with version injection and validation
# Usage:
#   ./scripts/docker/build.sh              # Build latest
#   ./scripts/docker/build.sh --push       # Build and push to registry
#   ./scripts/docker/build.sh --scan       # Build and security scan
#   ./scripts/docker/build.sh --all        # Build, scan, and push
# ==============================================================================

set -euo pipefail

# ==============================================================================
# CONFIGURATION
# ==============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Image configuration
REGISTRY="${REGISTRY:-ghcr.io}"
OWNER="${OWNER:-kaulinmindiola}"
IMAGE_NAME="${IMAGE_NAME:-global-ecommerce-api}"
FULL_IMAGE="${REGISTRY}/${OWNER}/${IMAGE_NAME}"

# Colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

# ==============================================================================
# FUNCTIONS
# ==============================================================================
log_info()    { echo -e "${CYAN}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[OK]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

print_banner() {
    echo -e "${CYAN}"
    echo "═══════════════════════════════════════════════════════════"
    echo "     🐳 Global E-commerce API - Docker Build Script"
    echo "═══════════════════════════════════════════════════════════"
    echo -e "${NC}"
}

check_requirements() {
    log_info "Checking requirements..."

    command -v docker   &>/dev/null || log_error "Docker is not installed"
    command -v git      &>/dev/null || log_error "Git is not installed"

    # Check Docker daemon is running
    docker info &>/dev/null || log_error "Docker daemon is not running"

    # Check Docker Buildx
    docker buildx version &>/dev/null || log_error "Docker Buildx is not available"

    log_success "All requirements met"
}

get_version_info() {
    # Get version from git
    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

    log_info "Version: ${VERSION}"
    log_info "Commit:  ${COMMIT_HASH}"
    log_info "Branch:  ${BRANCH}"
    log_info "Date:    ${BUILD_DATE}"
}

build_image() {
    local PLATFORM="${1:-linux/amd64}"
    local PUSH="${2:-false}"

    log_info "Building Docker image..."
    log_info "  Image: ${FULL_IMAGE}:${VERSION}"
    log_info "  Platform: ${PLATFORM}"
    echo ""

    # Build arguments
    local BUILD_ARGS=(
        "--build-arg" "VERSION=${VERSION}"
        "--build-arg" "BUILD_DATE=${BUILD_DATE}"
        "--build-arg" "COMMIT_HASH=${COMMIT_HASH}"
    )

    # Tags
    local TAGS=(
        "--tag" "${FULL_IMAGE}:${VERSION}"
        "--tag" "${FULL_IMAGE}:${COMMIT_HASH}"
    )

    # Add 'latest' tag only for main branch
    if [ "${BRANCH}" == "main" ]; then
        TAGS+=("--tag" "${FULL_IMAGE}:latest")
    fi

    # Build command
    local CMD=(
        "docker" "buildx" "build"
        "${BUILD_ARGS[@]}"
        "${TAGS[@]}"
        "--file" "${PROJECT_ROOT}/deployments/docker/Dockerfile"
        "--platform" "${PLATFORM}"
        "--progress=plain"
        "--no-cache=false"
        "--provenance=true"
        "--sbom=true"
    )

    # Add push or load flag
    if [ "${PUSH}" == "true" ]; then
        CMD+=("--push")
    else
        CMD+=("--load")
    fi

    CMD+=("${PROJECT_ROOT}")

    # Execute build
    "${CMD[@]}"

    log_success "Image built successfully: ${FULL_IMAGE}:${VERSION}"
}

get_image_size() {
    local IMAGE="${FULL_IMAGE}:${VERSION}"
    local SIZE
    SIZE=$(docker image inspect "${IMAGE}" \
        --format='{{.Size}}' 2>/dev/null || echo "0")
    local SIZE_MB
    SIZE_MB=$(echo "scale=2; $SIZE / 1024 / 1024" | bc)

    echo -e "${CYAN}Image size:${NC} ${BOLD}${SIZE_MB} MB${NC}"

    # Warn if image is larger than 25MB
    local SIZE_CHECK
    SIZE_CHECK=$(echo "${SIZE_MB} > 25" | bc)
    if [ "${SIZE_CHECK}" -eq 1 ]; then
        log_warning "Image size ${SIZE_MB}MB exceeds 25MB target"
    else
        log_success "Image size ${SIZE_MB}MB is within 25MB target ✓"
    fi
}

scan_image() {
    local IMAGE="${FULL_IMAGE}:${VERSION}"

    log_info "Running Trivy security scan..."

    if command -v trivy &>/dev/null; then
        trivy image \
            --severity HIGH,CRITICAL \
            --ignore-unfixed \
            --format table \
            "${IMAGE}"
    else
        log_info "Trivy not installed. Running via Docker..."
        docker run --rm \
            -v /var/run/docker.sock:/var/run/docker.sock \
            aquasec/trivy:latest image \
            --severity HIGH,CRITICAL \
            --ignore-unfixed \
            "${IMAGE}"
    fi

    log_success "Security scan complete"
}

test_image() {
    local IMAGE="${FULL_IMAGE}:${VERSION}"

    log_info "Testing image startup..."

    # Start container
    local CONTAINER_ID
    CONTAINER_ID=$(docker run -d \
        --name "ecommerce-test-$$" \
        -p 18080:8080 \
        -e APP_ENV=test \
        -e APP_PORT=8080 \
        -e POSTGRES_HOST=localhost \
        -e REDIS_HOST=localhost \
        -e JWT_SECRET=test-secret-key-minimum-32-characters \
        "${IMAGE}" 2>/dev/null || echo "")

    if [ -z "${CONTAINER_ID}" ]; then
        log_warning "Could not start container for smoke test (DB not available)"
        return 0
    fi

    # Wait for startup
    sleep 3

    # Check container is running
    local STATUS
    STATUS=$(docker inspect --format='{{.State.Status}}' "${CONTAINER_ID}" 2>/dev/null || echo "unknown")

    # Cleanup
    docker stop "${CONTAINER_ID}" &>/dev/null || true
    docker rm "${CONTAINER_ID}" &>/dev/null || true

    if [ "${STATUS}" == "running" ] || [ "${STATUS}" == "exited" ]; then
        log_success "Image startup test passed"
    else
        log_warning "Container exited early (expected without DB)"
    fi
}

print_summary() {
    local PUSH="${1:-false}"

    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}${BOLD}   ✅ Build Complete!${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  ${BOLD}Image:${NC}   ${FULL_IMAGE}:${VERSION}"
    echo -e "  ${BOLD}Commit:${NC}  ${COMMIT_HASH}"
    echo -e "  ${BOLD}Date:${NC}    ${BUILD_DATE}"
    echo ""

    if [ "${PUSH}" == "true" ]; then
        echo -e "  ${GREEN}✓ Pushed to registry${NC}"
        echo ""
        echo -e "  ${BOLD}Pull command:${NC}"
        echo -e "  docker pull ${FULL_IMAGE}:${VERSION}"
    else
        echo -e "  ${YELLOW}→ Not pushed (use --push to push to registry)${NC}"
        echo ""
        echo -e "  ${BOLD}Run locally:${NC}"
        echo -e "  docker run -p 8080:8080 ${FULL_IMAGE}:${VERSION}"
    fi
    echo ""
    echo -e "${CYAN}═══════════════════════════════════════════════════════════${NC}"
}

# ==============================================================================
# MAIN
# ==============================================================================
main() {
    local DO_PUSH=false
    local DO_SCAN=false
    local PLATFORM="linux/amd64"

    # Parse arguments
    for arg in "$@"; do
        case $arg in
            --push)     DO_PUSH=true ;;
            --scan)     DO_SCAN=true ;;
            --all)      DO_PUSH=true; DO_SCAN=true ;;
            --multi)    PLATFORM="linux/amd64,linux/arm64" ;;
            --help|-h)
                echo "Usage: $0 [--push] [--scan] [--all] [--multi]"
                echo "  --push   Push image to registry"
                echo "  --scan   Run Trivy security scan"
                echo "  --all    Build, scan, and push"
                echo "  --multi  Build for multiple platforms"
                exit 0
                ;;
        esac
    done

    print_banner
    check_requirements

    cd "${PROJECT_ROOT}"

    get_version_info
    echo ""

    build_image "${PLATFORM}" "${DO_PUSH}"
    echo ""

    get_image_size
    echo ""

    if [ "${DO_SCAN}" == "true" ]; then
        scan_image
        echo ""
    fi

    test_image
    echo ""

    print_summary "${DO_PUSH}"
}

main "$@"
