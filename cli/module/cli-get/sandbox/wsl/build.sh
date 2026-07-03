#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RELEASE_DIR="${SANDBOX_RELEASE_DIR:-${SCRIPT_DIR}/../release/wsl}"
GO_BIN="${GO_BIN:-go}"
GO_RELEASE_LDFLAGS="${GO_RELEASE_LDFLAGS:--s -w}"

mkdir -p "${RELEASE_DIR}"

build_one_arch() {
  local arch_dir="$1"
  local target_goarch="$2"
  local target_output="${RELEASE_DIR}/${arch_dir}"

  echo "-> building WSL CLI_SANDBOX for ${arch_dir} (${target_goarch})"
  mkdir -p "${target_output}"

  for mode in filepick net filepick_net; do
    local mode_output="${target_output}/${mode}"
    local binary_path="${mode_output}/CLI_SANDBOX"
    mkdir -p "${mode_output}"
    (
      cd "${SCRIPT_DIR}/.."
      GOOS=linux GOARCH="${target_goarch}" \
        "${GO_BIN}" build -trimpath -ldflags "${GO_RELEASE_LDFLAGS} -X main.defaultMode=${mode}" \
        -o "${binary_path}" \
        ./wsl/helper
    )
  done
}

build_one_arch "x86" "amd64"
build_one_arch "arm" "arm64"
