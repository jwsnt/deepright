#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RELEASE_DIR="${SANDBOX_RELEASE_DIR:-${SCRIPT_DIR}/../../../release/mac}"
KEY_DIR="${DEEPRIGHT_KEY:-}"
SKIP_SIGN="${DEEPRIGHT_SKIP_SIGN:-0}"

if [[ "${SKIP_SIGN}" != "1" && -z "${KEY_DIR}" ]]; then
  echo "DEEPRIGHT_KEY is required and must point to the certificate directory" >&2
  exit 1
fi

if [[ "${SKIP_SIGN}" != "1" && ! -d "${KEY_DIR}" ]]; then
  echo "DEEPRIGHT_KEY does not point to a directory: ${KEY_DIR}" >&2
  exit 1
fi

trim_file_content() {
  local path="$1"
  tr -d '\r' < "${path}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'
}

read_first_existing() {
  local path
  for path in "$@"; do
    if [[ -f "${path}" ]]; then
      trim_file_content "${path}"
      return 0
    fi
  done
  return 1
}

KEYCHAIN_PASSWORD="${DEEPRIGHT_KEYCHAIN_PASSWORD:-}"
if [[ -z "${KEYCHAIN_PASSWORD}" ]]; then
  KEYCHAIN_PASSWORD="$(uuidgen)"
fi

P12_PASSWORD="${DEEPRIGHT_P12_PASSWORD:-}"
if [[ -z "${P12_PASSWORD}" ]]; then
  P12_PASSWORD="$(read_first_existing \
    "${KEY_DIR}/DeepRight_p12_password.txt" \
    "${KEY_DIR}/p12_password.txt" \
    "${KEY_DIR}/password.txt" || true)"
fi

IDENTITY="${DEEPRIGHT_IDENTITY:-}"
if [[ "${SKIP_SIGN}" != "1" && -z "${IDENTITY}" ]]; then
  IDENTITY="$(read_first_existing \
    "${KEY_DIR}/identity.txt" \
    "${KEY_DIR}/developer_id_identity.txt" || true)"
fi

mkdir -p "${RELEASE_DIR}"

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/deepright-sandbox-build.XXXXXX")"
KEYCHAIN_PATH="${TMP_DIR}/deepright-build.keychain-db"

cleanup() {
  security delete-keychain "${KEYCHAIN_PATH}" >/dev/null 2>&1 || true
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

USE_KEYCHAIN=0
if [[ "${SKIP_SIGN}" != "1" ]]; then
  security create-keychain -p "${KEYCHAIN_PASSWORD}" "${KEYCHAIN_PATH}"
  security unlock-keychain -p "${KEYCHAIN_PASSWORD}" "${KEYCHAIN_PATH}"
  security set-keychain-settings -lut 21600 "${KEYCHAIN_PATH}"

  while IFS= read -r cert; do
    [[ -n "${cert}" ]] || continue
    security import "${cert}" -k "${KEYCHAIN_PATH}" -T /usr/bin/codesign -T /usr/bin/security >/dev/null
  done < <(find "${KEY_DIR}" -maxdepth 1 -type f -name '*.cer' | sort)

  IMPORT_FAILED=0
  while IFS= read -r archive; do
    [[ -n "${archive}" ]] || continue
    if ! security import "${archive}" -k "${KEYCHAIN_PATH}" -P "${P12_PASSWORD}" -T /usr/bin/codesign -T /usr/bin/security >/dev/null; then
      echo "warning: failed to import ${archive} into temporary keychain" >&2
      IMPORT_FAILED=1
    fi
  done < <(find "${KEY_DIR}" -maxdepth 1 -type f -name '*.p12' | sort)

  if security find-identity -v -p codesigning "${KEYCHAIN_PATH}" >/dev/null 2>&1; then
    security set-key-partition-list -S apple-tool:,apple: -s -k "${KEYCHAIN_PASSWORD}" "${KEYCHAIN_PATH}" >/dev/null || true
  fi

  if [[ -z "${IDENTITY}" ]]; then
    IDENTITY="$(
      security find-identity -v -p codesigning "${KEYCHAIN_PATH}" \
        | sed -n 's/.*"Developer ID Application: \([^"]*\)"/Developer ID Application: \1/p' \
        | head -n 1
    )"
  fi

  if [[ -z "${IDENTITY}" ]]; then
    IDENTITY="$(
      security find-identity -v -p codesigning "${KEYCHAIN_PATH}" \
        | sed -n 's/.*"Apple Development: \([^"]*\)"/Apple Development: \1/p' \
        | head -n 1
    )"
  fi

  USE_KEYCHAIN=1
  if [[ -z "${IDENTITY}" ]]; then
    if [[ "${IMPORT_FAILED}" -eq 1 ]]; then
      echo "warning: no usable identity found in temporary keychain, falling back to system keychains" >&2
    fi
    IDENTITY="$(
      security find-identity -v -p codesigning \
        | sed -n 's/.*"Developer ID Application: \([^"]*\)"/Developer ID Application: \1/p' \
        | head -n 1
    )"
    if [[ -z "${IDENTITY}" ]]; then
      IDENTITY="$(
        security find-identity -v -p codesigning \
          | sed -n 's/.*"Apple Development: \([^"]*\)"/Apple Development: \1/p' \
          | head -n 1
      )"
    fi
    USE_KEYCHAIN=0
  fi

  if [[ -z "${IDENTITY}" ]]; then
    echo "No usable codesign identity found in ${KEY_DIR} or current system keychains" >&2
    exit 1
  fi

  echo "Using identity: ${IDENTITY}"
else
  echo "Skipping codesign and building unsigned sandbox app bundles"
fi
echo "Output directory: ${RELEASE_DIR}"

build_one_arch() {
  local arch_dir="$1"
  local target_goarch="$2"
  local target_output="${RELEASE_DIR}/${arch_dir}"

  echo "-> building sandbox apps for ${arch_dir} (${target_goarch})"
  mkdir -p "${target_output}"

  for mode in filepick net filepick_net; do
    local mode_output="${target_output}/${mode}"
    local bundle_id="cn.deepright.cli-sandbox.${mode}"
    GO_ARGS=(
      --sandbox-src ..
      --output-dir "${mode_output}"
      --bundle-id "${bundle_id}"
      --mode "${mode}"
      --target-goos darwin
      --target-goarch "${target_goarch}"
    )

    if [[ "${SKIP_SIGN}" != "1" ]]; then
      GO_ARGS+=(--identity "${IDENTITY}")
    else
      GO_ARGS+=(--skip-sign)
    fi

    if [[ "${USE_KEYCHAIN}" -eq 1 ]]; then
      GO_ARGS+=(--keychain "${KEYCHAIN_PATH}")
    fi

    go run . \
      "${GO_ARGS[@]}" \
      "$@"
  done
}

build_one_arch "arm" "arm64" "$@"
build_one_arch "x86" "amd64" "$@"
