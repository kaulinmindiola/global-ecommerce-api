#!/bin/sh
# ==============================================================================
# HEALTHCHECK SCRIPT - Global E-commerce API
# Purpose: Lightweight health check for Docker HEALTHCHECK instruction
# Used by: Dockerfile HEALTHCHECK CMD
# Requirements: Only uses wget (available in Alpine)
# ==============================================================================

set -e

# Configuration
HOST="${APP_HOST:-0.0.0.0}"
PORT="${APP_PORT:-8080}"
HEALTH_PATH="/health"
TIMEOUT=5

# Use wget (available in Alpine) instead of curl
# -q: Quiet mode
# -O-: Output to stdout
# -T: Timeout in seconds
# --spider: Don't download, just check
RESPONSE=$(wget \
    -q \
    -O- \
    -T "${TIMEOUT}" \
    "http://localhost:${PORT}${HEALTH_PATH}" 2>/dev/null) || exit 1

# Check response contains "healthy" or "degraded" (not "unhealthy")
echo "${RESPONSE}" | grep -q '"status":"healthy"\|"status":"degraded"' || exit 1

exit 0
