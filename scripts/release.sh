#!/usr/bin/env bash
set -euo pipefail

TAG="${TAG:-}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"
GORELEASER="${GORELEASER:-goreleaser}"

if [[ -z "${GITHUB_TOKEN}" ]]; then
    echo "ERROR: GITHUB_TOKEN is not set" >&2
    exit 1
fi

if [[ -z "${TAG}" ]]; then
    echo "ERROR: TAG is not set (e.g. make release TAG=v0.1.0)" >&2
    exit 1
fi

if git rev-parse "${TAG}" >/dev/null 2>&1; then
    echo "Tag ${TAG} already exists, skipping creation"
else
    echo "Creating tag ${TAG}..."
    git tag -a "${TAG}" -m "Release ${TAG}"
    git push origin "${TAG}"
fi

echo "Running goreleaser for ${TAG}..."
${GORELEASER} release --clean
