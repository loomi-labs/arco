#!/bin/bash
set -euo pipefail

# Check Docker socket access
if [ -S /var/run/docker.sock ]; then
    echo "✅ Docker socket available at /var/run/docker.sock"
    docker version --format '{{.Server.Version}}' || echo "⚠️  Docker daemon not accessible"
else
    echo "❌ Docker socket not found at /var/run/docker.sock"
    echo "   Please mount Docker socket: -v /var/run/docker.sock:/var/run/docker.sock"
    exit 1
fi

# Verify SSH keys are available
if [ -r /home/borg/.ssh/borg_test_key ]; then
    echo "✅ SSH keys are available"
else
    echo "❌ SSH private key not found or not readable"
    exit 1
fi

# Show network configuration if available
if [ -n "${TESTCONTAINERS_NETWORK_NAME}" ]; then
    echo "🌐 Using Docker network: ${TESTCONTAINERS_NETWORK_NAME}"
fi

# Run integration tests with output capture and summary
echo "🚀 Running integration tests..."
echo "   Client Borg: ${CLIENT_BORG_VERSION}"
echo "   Server Borg: ${SERVER_BORG_VERSION}"

# Create temporary file for JSON output capture
TEST_OUTPUT_FILE=$(mktemp)
trap 'rm -f "$TEST_OUTPUT_FILE"' EXIT

# Run tests with JSON output and real-time display
# Note: Compiled test binaries don't support -test.json, so we parse regular output
/usr/local/bin/integration-tests ${TEST_ARGS:-"-test.v"} 2>&1 | tee "$TEST_OUTPUT_FILE"
TEST_EXIT_CODE=${PIPESTATUS[0]}

# Generate summary
echo ""
echo "📊 TEST SUMMARY"
echo "==============="

# Extract pass/fail events with timing from regular test output
PASS_TESTS=$(grep "^--- PASS:" "$TEST_OUTPUT_FILE" | sed 's/^--- PASS: //' || echo "")
FAIL_TESTS=$(grep "^--- FAIL:" "$TEST_OUTPUT_FILE" | sed 's/^--- FAIL: //' || echo "")

# Count results
PASS_COUNT=$(echo "$PASS_TESTS" | grep -c '^' 2>/dev/null || echo "0")
FAIL_COUNT=$(echo "$FAIL_TESTS" | grep -c '^' 2>/dev/null || echo "0")

# Handle empty string cases
if [ -z "$PASS_TESTS" ] || [ "$PASS_TESTS" = "" ]; then
    PASS_COUNT=0
fi
if [ -z "$FAIL_TESTS" ] || [ "$FAIL_TESTS" = "" ]; then
    FAIL_COUNT=0
fi

# Display results with enhanced formatting
if [ "$PASS_COUNT" -gt 0 ]; then
    echo "✅ PASSED ($PASS_COUNT tests):"
    echo "$PASS_TESTS" | sed 's/^/  ✓ /'
fi

if [ "$FAIL_COUNT" -gt 0 ]; then
    echo "❌ FAILED ($FAIL_COUNT tests):"
    echo "$FAIL_TESTS" | sed 's/^/  ✗ /'
fi

# Overall summary with timing - extract from regular output
TOTAL_TIME=$(grep -E "^--- (PASS|FAIL):" "$TEST_OUTPUT_FILE" | grep -oE '\([0-9]+\.[0-9]+s\)' | sed 's/[()s]//g' | awk '{sum += $1} END {printf "%.2f", sum}' || echo "0.00")
echo "==============="
if [ -n "$TOTAL_TIME" ] && [ "$TOTAL_TIME" != "0.00" ]; then
    echo "⏱️  Total test time: ${TOTAL_TIME}s"
fi

if [ "$TEST_EXIT_CODE" -eq 0 ]; then
    echo "🎉 ALL TESTS PASSED ($PASS_COUNT/$((PASS_COUNT + FAIL_COUNT)))"
else
    echo "💥 SOME TESTS FAILED ($FAIL_COUNT/$((PASS_COUNT + FAIL_COUNT)))"
fi

# Exit with original test exit code
exit "$TEST_EXIT_CODE"