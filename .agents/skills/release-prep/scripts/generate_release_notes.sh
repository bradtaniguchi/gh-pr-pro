#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || (cd "${SCRIPT_DIR}/../../../.." && pwd))"

OUTPUT_FILE=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        -o|--output)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        *)
            OUTPUT_FILE="$1"
            shift
            ;;
    esac
done

# If current commit is tagged, find the previous tag before it
if git describe --tags --exact-match HEAD >/dev/null 2>&1; then
    LAST_TAG=$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo "")
else
    LAST_TAG=$(git describe --tags --abbrev=0 HEAD 2>/dev/null || echo "")
fi

if [ -z "${LAST_TAG}" ]; then
    RANGE="HEAD"
    PREV_NAME="Initial Commit"
else
    RANGE="${LAST_TAG}..HEAD"
    PREV_NAME="${LAST_TAG}"
fi

extract_commits() {
    local pattern="$1"
    git log "${RANGE}" --oneline --no-merges --grep="^${pattern}" --format="- %s (%h)" 2>/dev/null || true
}

build_notes() {
    local COMMITS
    COMMITS=$(git log "${RANGE}" --oneline --no-merges 2>/dev/null || echo "")
    if [ -z "${COMMITS}" ]; then
        echo "_No commits found in range ${RANGE}._"
        return
    fi

    local FEATS FIXES DOCS PERF CI REFACTOR
    FEATS=$(extract_commits "feat")
    FIXES=$(extract_commits "fix")
    DOCS=$(extract_commits "docs")
    PERF=$(extract_commits "perf")
    CI=$(extract_commits "ci")
    REFACTOR=$(extract_commits "refactor")

    if [ -n "${FEATS}" ]; then
        echo "### 🚀 Features"
        echo "${FEATS}"
        echo
    fi

    if [ -n "${FIXES}" ]; then
        echo "### 🐛 Bug Fixes"
        echo "${FIXES}"
        echo
    fi

    if [ -n "${PERF}" ]; then
        echo "### ⚡ Performance Improvements"
        echo "${PERF}"
        echo
    fi

    if [ -n "${DOCS}" ]; then
        echo "### 📚 Documentation"
        echo "${DOCS}"
        echo
    fi

    if [ -n "${CI}" ] || [ -n "${REFACTOR}" ]; then
        echo "### 🛠️ Maintenance & CI"
        if [ -n "${CI}" ]; then echo "${CI}"; fi
        if [ -n "${REFACTOR}" ]; then echo "${REFACTOR}"; fi
        echo
    fi

    echo "### 📦 All Commits in Release"
    git log "${RANGE}" --oneline --no-merges --format="- %s (%h)"
}

if [ -n "${OUTPUT_FILE}" ]; then
    build_notes > "${OUTPUT_FILE}"
    echo "Release notes written to ${OUTPUT_FILE} (range: ${PREV_NAME} -> HEAD)"
else
    echo
    echo "========================================================================"
    echo "Draft Release Notes (${PREV_NAME} -> HEAD)"
    echo "========================================================================"
    echo
    build_notes
    echo
    echo "========================================================================"
fi
