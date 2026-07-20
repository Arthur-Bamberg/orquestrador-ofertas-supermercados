#!/usr/bin/env bash
# Point this clone at the versioned hooks under .githooks/ (once per clone).
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

git config core.hooksPath .githooks
chmod +x .githooks/pre-commit

echo "Git hooks installed (core.hooksPath=.githooks)."
echo "pre-commit will run: go test ./... for each module in go.work"
