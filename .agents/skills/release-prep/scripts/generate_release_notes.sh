#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || (cd "${SCRIPT_DIR}/../../../.." && pwd))"

cd "${REPO_ROOT}"

LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

if [ -z "${LAST_TAG}" ]; then
    echo "No prior git tags detected. Generating notes for all commits..."
    RANGE="HEAD"
    PREV_NAME="Initial Commit"
else
    echo "Latest tag found: ${LAST_TAG}"
    RANGE="${LAST_TAG}..HEAD"
    PREV_NAME="${LAST_TAG}"
fi

echo
echo "========================================================================"
echo "Draft Release Notes (${PREV_NAME} -> HEAD)"
echo "========================================================================"
echo

COMMITS=$(git log "${RANGE}" --oneline --no-merges 2>/dev/null || echo "")

if [ -z "${COMMITS}" ]; then
    echo "No commits found in range ${RANGE}."
    exit 0
fi

extract_commits() {
    local pattern="$1"
    git log "${RANGE}" --oneline --no-merges --grep="^${pattern}" --format="- %s (%h)" 2>/dev/null || true
}

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

echo
echo "========================================================================"
