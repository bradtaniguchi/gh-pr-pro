#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || (cd "${SCRIPT_DIR}/../../../.." && pwd))"

cd "${REPO_ROOT}"

echo "Validating cross-compilation for all target platforms..."
echo

TMP_DIR="$(mktemp -d)"
cleanup() {
    rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

TARGETS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

ALL_PASSED=true

for TARGET in "${TARGETS[@]}"; do
    IFS="/" read -r GOOS GOARCH <<< "${TARGET}"
    EXT=""
    if [ "${GOOS}" = "windows" ]; then
        EXT=".exe"
    fi
    OUTPUT="${TMP_DIR}/gh-pr-pro_${GOOS}_${GOARCH}${EXT}"

    printf "  Compiling for %-15s ... " "${TARGET}"
    if GOOS="${GOOS}" GOARCH="${GOARCH}" CGO_ENABLED=0 go build -o "${OUTPUT}" . 2>&1; then
        echo "✓ OK"
    else
        echo "✗ FAILED"
        ALL_PASSED=false
    fi
done

echo
if [ "${ALL_PASSED}" = true ]; then
    echo "Result: SUCCESS - All 5 target architectures compiled cleanly."
    exit 0
else
    echo "Result: FAILED - One or more targets failed cross-compilation."
    exit 1
fi
