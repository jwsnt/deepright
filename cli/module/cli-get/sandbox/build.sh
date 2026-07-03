#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RELEASE_DIR="${SANDBOX_RELEASE_DIR:-${SCRIPT_DIR}/release/mac}"

mkdir -p "${RELEASE_DIR}"

(
  cd "${SCRIPT_DIR}/mac"
  SANDBOX_RELEASE_DIR="${RELEASE_DIR}" ./build.sh "$@"
)
