#!/bin/bash
# gen-context.sh — Generate AI context for a specific service
# Usage: ./scripts/gen-context.sh [service-name]
# Example: ./scripts/gen-context.sh proxy-service
# Output: full context to stdout, pipe to file or pbcopy

set -e

SERVICE=${1:-""}
ROOT_DIR=$(git rev-parse --show-toplevel 2>/dev/null || pwd)

echo_section() {
    echo ""
    echo "=================================================================="
    echo "=== $1"
    echo "=================================================================="
}

# System prompt
if [ -f "$ROOT_DIR/docs/ai/SYSTEM_PROMPT.txt" ]; then
    echo_section "SYSTEM PROMPT"
    cat "$ROOT_DIR/docs/ai/SYSTEM_PROMPT.txt"
fi

# CONTEXT.md (living document)
if [ -f "$ROOT_DIR/CONTEXT.md" ]; then
    echo_section "CURRENT PROJECT STATE (CONTEXT.md)"
    cat "$ROOT_DIR/CONTEXT.md"
fi

if [ -z "$SERVICE" ]; then
    # No service specified — output architecture context only
    echo_section "ARCHITECTURE"
    cat "$ROOT_DIR/docs/01-ARCHITECTURE.md" 2>/dev/null || true
    
    echo_section "DATABASE DESIGN"
    cat "$ROOT_DIR/docs/02-DATABASE-DESIGN.md" 2>/dev/null || true
    exit 0
fi

# Service-specific docs
echo_section "SERVICE DESIGN: $SERVICE"
find "$ROOT_DIR/docs/services" -name "*${SERVICE}*" -exec cat {} \; 2>/dev/null || true

# Architecture for context
echo_section "RELEVANT ARCHITECTURE (Events & APIs)"
grep -A 5 -B 2 "$SERVICE" "$ROOT_DIR/docs/01-ARCHITECTURE.md" 2>/dev/null || true

# Actual Go source files
SERVICE_DIR="$ROOT_DIR/services/$SERVICE"
if [ -d "$SERVICE_DIR" ]; then
    echo_section "DOMAIN LAYER: $SERVICE"
    find "$SERVICE_DIR/internal/domain" -name "*.go" 2>/dev/null | while read f; do
        echo "--- $f ---"
        cat "$f"
        echo ""
    done
    
    echo_section "USECASES: $SERVICE"
    find "$SERVICE_DIR/internal/usecase" -name "*.go" 2>/dev/null | while read f; do
        echo "--- $f ---"
        cat "$f"
        echo ""
    done
    
    echo_section "HANDLERS: $SERVICE"
    find "$SERVICE_DIR/internal/handler" -name "*.go" 2>/dev/null | while read f; do
        echo "--- $f ---"
        cat "$f"
        echo ""
    done
    
    echo_section "REPOSITORIES: $SERVICE"
    find "$SERVICE_DIR/internal/repository" -name "*.go" 2>/dev/null | while read f; do
        echo "--- $f ---"
        cat "$f"
        echo ""
    done
    
    echo_section "EVENTS: $SERVICE"
    find "$SERVICE_DIR/internal/events" -name "*.go" 2>/dev/null | while read f; do
        echo "--- $f ---"
        cat "$f"
        echo ""
    done
fi

echo ""
echo "=== END OF CONTEXT ==="
echo "Token estimate: $(wc -w <<< "$(cat)") words"
