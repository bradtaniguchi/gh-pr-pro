#!/usr/bin/env bash
# validate-conventional-commit.sh — Validate a Conventional Commits message subject.
#
# Usage:
#   validate-conventional-commit.sh <file>   # read from file (git commit-msg hook via Lefthook)
#   echo "<msg>" | validate-conventional-commit.sh  # read from stdin (CI PR title validation)
#
# Exits 0 if valid, 1 if invalid.

set -euo pipefail

# Read from file argument (Lefthook passes the commit-msg file path as $1),
# or from stdin if no argument is provided (CI pipeline usage).
MSG=$(cat "${1:-/dev/stdin}")

# Strip comment lines (e.g. from 'git commit -v' or merge commit templates).
SUBJECT=$(echo "$MSG" | grep -v '^#' | head -1)

PATTERN='^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-z0-9_\-\.\/]+\))?(!)?: .+'

if ! echo "$SUBJECT" | grep -qE "$PATTERN"; then
  echo ""
  echo "❌ Commit message does not follow Conventional Commits format."
  echo ""
  echo "   Expected: <type>(<scope>): <subject>"
  echo "   Example:  feat(cache): add delta-fetch support"
  echo ""
  echo "   Allowed types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"
  echo "   Common scopes (optional): api, cache, metrics, export, cmd, skills, ci, docs"
  echo ""
  echo "   Your message: $SUBJECT"
  echo ""
  echo "   (To bypass in an emergency, use 'git commit --no-verify')"
  exit 1
fi
