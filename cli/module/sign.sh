#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RELEASE_DIR="${DEEPRIGHT_RELEASE_DIR:-${SCRIPT_DIR}/release/mac}"
KEY_DIR="${DEEPRIGHT_KEY:-}"
APP_NAME="${DEEPRIGHT_MAC_APP_NAME:-DeepRight}"
APP_IDENTITY="${DEEPRIGHT_IDENTITY:-}"
INSTALLER_IDENTITY="${DEEPRIGHT_INSTALLER_IDENTITY:-}"
KEYCHAIN_PASSWORD="${DEEPRIGHT_KEYCHAIN_PASSWORD:-}"
P12_PASSWORD="${DEEPRIGHT_P12_PASSWORD:-}"
ARCHES="${DEEPRIGHT_SIGN_ARCHES:-x86 arm}"
REQUIRE_NOTARIZATION_CHECK="${DEEPRIGHT_REQUIRE_NOTARIZATION_CHECK:-0}"
NOTARIZE_MODE="${DEEPRIGHT_NOTARIZE:-auto}"
NOTARY_TIMEOUT="${DEEPRIGHT_NOTARY_TIMEOUT:-1h}"
NOTARY_RETRY_COUNT="${DEEPRIGHT_NOTARY_RETRY_COUNT:-3}"
NOTARY_RETRY_DELAY="${DEEPRIGHT_NOTARY_RETRY_DELAY:-5}"
NOTARY_PROFILE="${DEEPRIGHT_NOTARY_PROFILE:-}"
NOTARY_KEYCHAIN="${DEEPRIGHT_NOTARY_KEYCHAIN:-}"
NOTARY_KEY="${DEEPRIGHT_NOTARY_KEY:-}"
NOTARY_KEY_ID="${DEEPRIGHT_NOTARY_KEY_ID:-}"
NOTARY_ISSUER="${DEEPRIGHT_NOTARY_ISSUER:-}"
NOTARY_APPLE_ID="${DEEPRIGHT_NOTARY_APPLE_ID:-}"
NOTARY_PASSWORD="${DEEPRIGHT_NOTARY_PASSWORD:-}"
NOTARY_TEAM_ID="${DEEPRIGHT_NOTARY_TEAM_ID:-}"

KEYCHAIN_PATH=""
APP_USE_KEYCHAIN=0
INSTALLER_USE_KEYCHAIN=0
USE_RUNTIME_OPTIONS=0
NOTARIZE_ENABLED=0
NOTARY_AUTH_ARGS=()
RUN_COMPLETED=0
FAILED_LINE=""
FAILED_COMMAND=""

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

mktemp_file_path() {
  local prefix="$1"
  local suffix="${2:-}"
  local tmp_path
  tmp_path="$(mktemp "${TMPDIR:-/tmp}/${prefix}.XXXXXX")"
  rm -f "${tmp_path}"
  printf '%s%s\n' "${tmp_path}" "${suffix}"
}

load_notary_defaults_from_key_dir() {
  [[ -n "${KEY_DIR}" && -d "${KEY_DIR}" ]] || return 0

  if [[ -z "${NOTARY_PROFILE}" ]]; then
    NOTARY_PROFILE="$(read_first_existing \
      "${KEY_DIR}/notary_profile.txt" \
      "${KEY_DIR}/notarytool_profile.txt" || true)"
  fi

  if [[ -z "${NOTARY_KEYCHAIN}" ]]; then
    NOTARY_KEYCHAIN="$(read_first_existing \
      "${KEY_DIR}/notary_keychain.txt" \
      "${KEY_DIR}/notarytool_keychain.txt" || true)"
  fi

  if [[ -z "${NOTARY_KEY}" ]]; then
    NOTARY_KEY="$(read_first_existing \
      "${KEY_DIR}/notary_key.txt" \
      "${KEY_DIR}/notary_api_key_path.txt" || true)"
    if [[ -z "${NOTARY_KEY}" ]]; then
      local key_file
      key_file="$(find "${KEY_DIR}" -maxdepth 1 -type f \( -name 'AuthKey_*.p8' -o -name '*.p8' \) | sort | head -n 1)"
      if [[ -n "${key_file}" ]]; then
        NOTARY_KEY="${key_file}"
      fi
    fi
  fi

  if [[ -z "${NOTARY_KEY_ID}" ]]; then
    NOTARY_KEY_ID="$(read_first_existing \
      "${KEY_DIR}/notary_key_id.txt" \
      "${KEY_DIR}/api_key_id.txt" || true)"
  fi

  if [[ -z "${NOTARY_ISSUER}" ]]; then
    NOTARY_ISSUER="$(read_first_existing \
      "${KEY_DIR}/notary_issuer.txt" \
      "${KEY_DIR}/issuer_id.txt" || true)"
  fi

  if [[ -z "${NOTARY_APPLE_ID}" ]]; then
    NOTARY_APPLE_ID="$(read_first_existing \
      "${KEY_DIR}/notary_apple_id.txt" \
      "${KEY_DIR}/apple_id.txt" \
      "${KEY_DIR}/developer_account_email.txt" \
      "${KEY_DIR}/开发者账号邮箱.txt" || true)"
  fi

  if [[ -z "${NOTARY_PASSWORD}" ]]; then
    NOTARY_PASSWORD="$(read_first_existing \
      "${KEY_DIR}/notary_password.txt" \
      "${KEY_DIR}/app_specific_password.txt" \
      "${KEY_DIR}/App-Specific_Password.txt" || true)"
  fi

  if [[ -z "${NOTARY_TEAM_ID}" ]]; then
    NOTARY_TEAM_ID="$(read_first_existing \
      "${KEY_DIR}/notary_team_id.txt" \
      "${KEY_DIR}/team_id.txt" || true)"
  fi
}

print_artifact_paths() {
  local arch
  for arch in ${ARCHES}; do
    local arch_dir="${RELEASE_DIR}/${arch}"
    echo "${arch} artifact paths:"
    if [[ ! -d "${arch_dir}" ]]; then
      echo "  (missing arch directory: ${arch_dir})"
      continue
    fi

    local found_any=0
    local artifact_path
    while IFS= read -r artifact_path; do
      [[ -n "${artifact_path}" ]] || continue
      found_any=1
      echo "  ${artifact_path}"
    done < <(find "${arch_dir}" -maxdepth 1 -type f \( -name '*.dmg' -o -name '*.pkg' \) -print | sort)

    while IFS= read -r artifact_path; do
      [[ -n "${artifact_path}" ]] || continue
      found_any=1
      echo "  ${artifact_path}"
    done < <(find "${arch_dir}" -maxdepth 1 -type d -name '*.app' -print | sort)

    if [[ "${found_any}" -eq 0 ]]; then
      echo "  (no signed artifacts found)"
    fi
  done
}

print_final_success_summary() {
  echo "FINAL RESULT: SUCCESS - mac artifacts are signed, notarized, stapled, and verified for safe distribution."
  print_artifact_paths
}

print_final_failure_summary() {
  local exit_code="$1"
  echo "FINAL RESULT: FAILURE - mac artifacts are not confirmed safe to distribute." >&2
  if [[ -n "${FAILED_LINE}" || -n "${FAILED_COMMAND}" ]]; then
    echo "Failure detail: line ${FAILED_LINE:-unknown}, command: ${FAILED_COMMAND:-unknown}" >&2
  fi
  echo "Current artifact paths:" >&2
  print_artifact_paths >&2
  return "${exit_code}"
}

cleanup() {
  local exit_code=$?
  if [[ -n "${KEYCHAIN_PATH}" ]]; then
    security delete-keychain "${KEYCHAIN_PATH}" >/dev/null 2>&1 || true
  fi
  if [[ "${RUN_COMPLETED}" -ne 1 ]]; then
    print_final_failure_summary "${exit_code}"
  fi
  return "${exit_code}"
}
trap cleanup EXIT
trap 'FAILED_LINE="${LINENO}"; FAILED_COMMAND="${BASH_COMMAND}"' ERR

require_command() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "missing required command: ${name}" >&2
    exit 1
  fi
}

check_timestamp_service() {
  local timestamp_url="http://timestamp.apple.com/ts01"
  local output=""

  require_command curl
  if output="$(curl --head --silent --show-error --connect-timeout 5 --max-time 15 "${timestamp_url}" 2>&1 >/dev/null)"; then
    return 0
  fi

  echo "Unable to reach Apple's signing timestamp service: ${timestamp_url}" >&2
  if [[ -n "${output}" ]]; then
    echo "${output}" >&2
  fi
  echo "Developer ID signing requires a secure timestamp. Check DNS, proxy, VPN, or firewall settings and retry." >&2
  exit 1
}

assert_macos() {
  if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "sign.sh only supports macOS" >&2
    exit 1
  fi
}

find_identity_in_codesigning_keychain() {
  local label="$1"
  local keychain_path="${2:-}"
  if [[ -n "${keychain_path}" ]]; then
    security find-identity -v -p codesigning "${keychain_path}" 2>/dev/null
  else
    security find-identity -v -p codesigning 2>/dev/null
  fi \
    | sed -n "s/.*\"${label}: \\([^\"]*\\)\"/${label}: \\1/p" \
    | head -n 1
}

find_installer_identity() {
  local keychain_path="${1:-}"
  local result=""
  if [[ -n "${keychain_path}" ]]; then
    result="$(
      security find-identity -v -p basic "${keychain_path}" 2>/dev/null \
        | sed -n 's/.*"Developer ID Installer: \([^"]*\)"/Developer ID Installer: \1/p' \
        | head -n 1
    )"
    if [[ -z "${result}" ]]; then
      result="$(
        security find-identity -v "${keychain_path}" 2>/dev/null \
          | sed -n 's/.*"Developer ID Installer: \([^"]*\)"/Developer ID Installer: \1/p' \
          | head -n 1
      )"
    fi
  else
    result="$(
      security find-identity -v -p basic 2>/dev/null \
        | sed -n 's/.*"Developer ID Installer: \([^"]*\)"/Developer ID Installer: \1/p' \
        | head -n 1
    )"
    if [[ -z "${result}" ]]; then
      result="$(
        security find-identity -v 2>/dev/null \
          | sed -n 's/.*"Developer ID Installer: \([^"]*\)"/Developer ID Installer: \1/p' \
          | head -n 1
      )"
    fi
  fi
  printf '%s\n' "${result}"
}

keychain_has_identity() {
  local identity="$1"
  local keychain_path="$2"
  [[ -n "${identity}" && -n "${keychain_path}" ]] || return 1
  security find-identity -v "${keychain_path}" 2>/dev/null | grep -F "\"${identity}\"" >/dev/null 2>&1
}

team_id_from_identity() {
  local identity="$1"
  if [[ "${identity}" =~ \(([A-Z0-9]{10})\)$ ]]; then
    printf '%s\n' "${BASH_REMATCH[1]}"
    return 0
  fi
  return 1
}

setup_notarization() {
  load_notary_defaults_from_key_dir

  local inferred_team_id=""
  inferred_team_id="$(team_id_from_identity "${APP_IDENTITY}" || true)"
  if [[ -z "${NOTARY_TEAM_ID}" && -n "${inferred_team_id}" ]]; then
    NOTARY_TEAM_ID="${inferred_team_id}"
  fi
  if [[ -n "${NOTARY_TEAM_ID}" && -n "${inferred_team_id}" && "${NOTARY_TEAM_ID}" != "${inferred_team_id}" ]]; then
    echo "Notary Team ID (${NOTARY_TEAM_ID}) does not match the signing identity Team ID (${inferred_team_id})" >&2
    echo "Update ${KEY_DIR}/team_id.txt or DEEPRIGHT_NOTARY_TEAM_ID before retrying." >&2
    exit 1
  fi

  case "${NOTARIZE_MODE}" in
    0|false|FALSE|no|NO)
      NOTARIZE_ENABLED=0
      echo "Notarization disabled"
      return 0
      ;;
    1|true|TRUE|yes|YES|auto)
      ;;
    *)
      echo "invalid DEEPRIGHT_NOTARIZE value: ${NOTARIZE_MODE}" >&2
      exit 1
      ;;
  esac

  if [[ -n "${NOTARY_PROFILE}" ]]; then
    NOTARY_AUTH_ARGS=(--keychain-profile "${NOTARY_PROFILE}")
    if [[ -n "${NOTARY_KEYCHAIN}" ]]; then
      NOTARY_AUTH_ARGS+=(--keychain "${NOTARY_KEYCHAIN}")
    fi
    NOTARIZE_ENABLED=1
    echo "Using notarytool keychain profile: ${NOTARY_PROFILE}"
    return 0
  fi

  if [[ -n "${NOTARY_KEY}" || -n "${NOTARY_KEY_ID}" || -n "${NOTARY_ISSUER}" ]]; then
    if [[ -z "${NOTARY_KEY}" || -z "${NOTARY_KEY_ID}" || -z "${NOTARY_ISSUER}" ]]; then
      echo "DEEPRIGHT_NOTARY_KEY, DEEPRIGHT_NOTARY_KEY_ID, and DEEPRIGHT_NOTARY_ISSUER must be set together" >&2
      exit 1
    fi
    NOTARY_AUTH_ARGS=(
      --key "${NOTARY_KEY}"
      --key-id "${NOTARY_KEY_ID}"
      --issuer "${NOTARY_ISSUER}"
    )
    NOTARIZE_ENABLED=1
    echo "Using notarytool App Store Connect API key authentication"
    return 0
  fi

  if [[ -n "${NOTARY_APPLE_ID}" || -n "${NOTARY_PASSWORD}" || -n "${NOTARY_TEAM_ID}" ]]; then
    if [[ -z "${NOTARY_APPLE_ID}" || -z "${NOTARY_PASSWORD}" || -z "${NOTARY_TEAM_ID}" ]]; then
      echo "DEEPRIGHT_NOTARY_APPLE_ID, DEEPRIGHT_NOTARY_PASSWORD, and DEEPRIGHT_NOTARY_TEAM_ID must be set together" >&2
      exit 1
    fi
    NOTARY_AUTH_ARGS=(
      --apple-id "${NOTARY_APPLE_ID}"
      --password "${NOTARY_PASSWORD}"
      --team-id "${NOTARY_TEAM_ID}"
    )
    NOTARIZE_ENABLED=1
    echo "Using notarytool Apple ID authentication: ${NOTARY_APPLE_ID}"
    return 0
  fi

  if [[ "${NOTARIZE_MODE}" != "auto" ]]; then
    echo "Notarization requested but no credentials were configured" >&2
    exit 1
  fi

  echo "warning: no notarization credentials configured; skipping notarization/stapling" >&2
  NOTARIZE_ENABLED=0
}

setup_identities() {
  if [[ -n "${KEY_DIR}" && ! -d "${KEY_DIR}" ]]; then
    echo "DEEPRIGHT_KEY does not point to a directory: ${KEY_DIR}" >&2
    exit 1
  fi

  local import_failed=0
  if [[ -n "${KEY_DIR}" ]]; then
    if [[ -z "${KEYCHAIN_PASSWORD}" ]]; then
      KEYCHAIN_PASSWORD="$(uuidgen)"
    fi
    if [[ -z "${P12_PASSWORD}" ]]; then
      P12_PASSWORD="$(read_first_existing \
        "${KEY_DIR}/DeepRight_p12_password.txt" \
        "${KEY_DIR}/p12_password.txt" \
        "${KEY_DIR}/password.txt" || true)"
    fi
    if [[ -z "${APP_IDENTITY}" ]]; then
      APP_IDENTITY="$(read_first_existing \
        "${KEY_DIR}/identity.txt" \
        "${KEY_DIR}/developer_id_identity.txt" || true)"
    fi
    if [[ -z "${INSTALLER_IDENTITY}" ]]; then
      INSTALLER_IDENTITY="$(read_first_existing \
        "${KEY_DIR}/installer_identity.txt" \
        "${KEY_DIR}/developer_id_installer_identity.txt" || true)"
    fi

    local tmp_dir
    tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/deepright-sign.XXXXXX")"
    KEYCHAIN_PATH="${tmp_dir}/deepright-sign.keychain-db"

    security create-keychain -p "${KEYCHAIN_PASSWORD}" "${KEYCHAIN_PATH}"
    security unlock-keychain -p "${KEYCHAIN_PASSWORD}" "${KEYCHAIN_PATH}"
    security set-keychain-settings -lut 21600 "${KEYCHAIN_PATH}"

    local cert
    while IFS= read -r -d '' cert; do
      security import "${cert}" -k "${KEYCHAIN_PATH}" -T /usr/bin/codesign -T /usr/bin/security >/dev/null
    done < <(find "${KEY_DIR}" -maxdepth 1 -type f -name '*.cer' -print0 | sort -z)

    local archive
    while IFS= read -r -d '' archive; do
      if ! security import "${archive}" -k "${KEYCHAIN_PATH}" -P "${P12_PASSWORD}" -T /usr/bin/codesign -T /usr/bin/security >/dev/null; then
        echo "warning: failed to import ${archive}" >&2
        import_failed=1
      fi
    done < <(find "${KEY_DIR}" -maxdepth 1 -type f -name '*.p12' -print0 | sort -z)

    if security find-identity -v -p codesigning "${KEYCHAIN_PATH}" >/dev/null 2>&1; then
      security set-key-partition-list -S apple-tool:,apple: -s -k "${KEYCHAIN_PASSWORD}" "${KEYCHAIN_PATH}" >/dev/null || true
    fi

    if [[ -z "${APP_IDENTITY}" ]]; then
      APP_IDENTITY="$(find_identity_in_codesigning_keychain "Developer ID Application" "${KEYCHAIN_PATH}")"
    fi
    if [[ -z "${APP_IDENTITY}" ]]; then
      APP_IDENTITY="$(find_identity_in_codesigning_keychain "Apple Development" "${KEYCHAIN_PATH}")"
    fi
    if keychain_has_identity "${APP_IDENTITY}" "${KEYCHAIN_PATH}"; then
      APP_USE_KEYCHAIN=1
    fi

    if [[ -z "${INSTALLER_IDENTITY}" ]]; then
      INSTALLER_IDENTITY="$(find_installer_identity "${KEYCHAIN_PATH}")"
    fi
    if keychain_has_identity "${INSTALLER_IDENTITY}" "${KEYCHAIN_PATH}"; then
      INSTALLER_USE_KEYCHAIN=1
    fi
  fi

  if [[ -z "${APP_IDENTITY}" ]]; then
    if [[ "${import_failed}" -eq 1 ]]; then
      echo "warning: no usable app identity found in DEEPRIGHT_KEY, trying system keychains" >&2
    fi
    APP_IDENTITY="$(find_identity_in_codesigning_keychain "Developer ID Application")"
    if [[ -z "${APP_IDENTITY}" ]]; then
      APP_IDENTITY="$(find_identity_in_codesigning_keychain "Apple Development")"
    fi
    APP_USE_KEYCHAIN=0
  fi

  if [[ -z "${INSTALLER_IDENTITY}" ]]; then
    if [[ "${import_failed}" -eq 1 ]]; then
      echo "warning: no usable installer identity found in DEEPRIGHT_KEY, trying system keychains" >&2
    fi
    INSTALLER_IDENTITY="$(find_installer_identity)"
    INSTALLER_USE_KEYCHAIN=0
  fi

  if [[ -z "${APP_IDENTITY}" ]]; then
    echo "No usable app signing identity found" >&2
    exit 1
  fi

  if [[ "${APP_IDENTITY}" == Developer\ ID\ Application:* ]]; then
    USE_RUNTIME_OPTIONS=1
  fi

  echo "Using app signing identity: ${APP_IDENTITY}"
  if [[ -n "${INSTALLER_IDENTITY}" ]]; then
    echo "Using installer signing identity: ${INSTALLER_IDENTITY}"
  else
    echo "warning: no Developer ID Installer identity available; .pkg artifacts cannot be signed" >&2
  fi
}

codesign_app_target() {
  local target="$1"
  local entitlements_path="${2:-}"
  local -a cmd=(codesign --force --sign "${APP_IDENTITY}" --verbose)

  if [[ "${APP_USE_KEYCHAIN}" -eq 1 ]]; then
    cmd+=(--keychain "${KEYCHAIN_PATH}")
  fi
  if [[ "${USE_RUNTIME_OPTIONS}" -eq 1 ]]; then
    cmd+=(--options runtime --timestamp)
  fi
  cmd+=(--generate-entitlement-der)
  if [[ -n "${entitlements_path}" ]]; then
    cmd+=(--entitlements "${entitlements_path}")
  fi
  cmd+=("${target}")
  "${cmd[@]}"
}

codesign_distribution_file() {
  local target="$1"
  local -a cmd=(codesign --force --sign "${APP_IDENTITY}" --verbose)
  if [[ "${APP_USE_KEYCHAIN}" -eq 1 ]]; then
    cmd+=(--keychain "${KEYCHAIN_PATH}")
  fi
  if [[ "${APP_IDENTITY}" == Developer\ ID\ Application:* ]]; then
    cmd+=(--timestamp)
  fi
  cmd+=("${target}")
  "${cmd[@]}"
}

productsign_pkg() {
  local input_path="$1"
  local output_path="$2"
  local -a cmd=(productsign --sign "${INSTALLER_IDENTITY}")
  if [[ "${INSTALLER_USE_KEYCHAIN}" -eq 1 ]]; then
    cmd+=(--keychain "${KEYCHAIN_PATH}")
  fi
  cmd+=("${input_path}" "${output_path}")
  "${cmd[@]}"
}

is_macho_file() {
  local path="$1"
  file -b "${path}" 2>/dev/null | grep -q 'Mach-O'
}

is_within_signable_bundle() {
  local path="$1"
  case "${path}" in
    *.app/*|*.framework/*|*.bundle/*|*.xpc/*|*.appex/*)
      return 0
      ;;
  esac
  return 1
}

extract_entitlements() {
  local target="$1"
  local out_path="$2"
  if ! codesign -d --entitlements :- "${target}" > "${out_path}" 2>/dev/null; then
    rm -f "${out_path}"
    return 1
  fi
  if ! grep -q '<plist' "${out_path}" 2>/dev/null; then
    rm -f "${out_path}"
    return 1
  fi
  return 0
}

sign_binary_if_needed() {
  local path="$1"
  local temp_dir="$2"
  if ! is_macho_file "${path}"; then
    return 0
  fi

  local entitlements_path=""
  local candidate="${temp_dir}/$(basename "${path}").entitlements.plist"
  if extract_entitlements "${path}" "${candidate}"; then
    entitlements_path="${candidate}"
  fi

  echo "  signing binary: ${path}"
  codesign_app_target "${path}" "${entitlements_path}"
}

clean_mutable_bundle_runtime_files() {
  local bundle_path="$1"
  local plugins_dir="${bundle_path}/Contents/Resources/plugins"
  local candidate
  local -a candidates=(
    "${bundle_path}/Contents/MacOS/data"
    "${bundle_path}/Contents/MacOS/data-shm"
    "${bundle_path}/Contents/MacOS/data-wal"
    "${bundle_path}/Contents/Resources/data"
    "${bundle_path}/Contents/Resources/data-shm"
    "${bundle_path}/Contents/Resources/data-wal"
    "${bundle_path}/Contents/Resources/plugins/.browser_playwright"
    "${bundle_path}/Contents/Resources/plugins/.remote"
    "${bundle_path}/Contents/Resources/plugins/email_artifacts"
    "${bundle_path}/Contents/Resources/plugins/feishu_artifacts"
    "${bundle_path}/Contents/Resources/plugins/playwright"
  )

  for candidate in "${candidates[@]}"; do
    [[ -e "${candidate}" ]] || continue
    echo "  removing mutable runtime file from bundle: ${candidate}"
    rm -rf "${candidate}"
  done

  if [[ -d "${plugins_dir}" ]]; then
    while IFS= read -r candidate; do
      [[ -n "${candidate}" ]] || continue
      echo "  removing mutable runtime file from bundle: ${candidate}"
      rm -rf "${candidate}"
    done < <(
      find "${plugins_dir}" -mindepth 1 -maxdepth 1 \
        \( -name '.connect-plugin-cache.json' \
        -o -name 'browser_instance.json' \
        -o -name 'data' \
        -o -name 'data-shm' \
        -o -name 'data-wal' \
        -o -name 'remote.json' \
        -o -name '*.log' \
        -o -name '*.pid' \
        -o -name '*.state.json' \
        -o -name '*.pending.json' \) \
        -print
    )
  fi
}

sign_bundle_tree() {
  local bundle_path="$1"
  local temp_dir="$2"

  if [[ "${bundle_path}" == *.app ]]; then
    clean_mutable_bundle_runtime_files "${bundle_path}"
  fi

  while IFS= read -r -d '' file_path; do
    sign_binary_if_needed "${file_path}" "${temp_dir}"
  done < <(find "${bundle_path}" -type f \
    ! -path '*/_CodeSignature/*' \
    ! -name 'CodeResources' \
    -print0)

  while IFS= read -r -d '' nested_bundle; do
    local entitlements_path=""
    local safe_name
    safe_name="$(printf '%s' "${nested_bundle}" | shasum | awk '{print $1}')"
    local candidate="${temp_dir}/${safe_name}.entitlements.plist"
    if extract_entitlements "${nested_bundle}" "${candidate}"; then
      entitlements_path="${candidate}"
    fi
    echo "  signing bundle: ${nested_bundle}"
    codesign_app_target "${nested_bundle}" "${entitlements_path}"
  done < <(
    find "${bundle_path}" \
      \( -type d \( -name '*.app' -o -name '*.framework' -o -name '*.bundle' -o -name '*.xpc' -o -name '*.appex' \) \) \
      ! -path "${bundle_path}" \
      -print \
      | awk '{ depth=gsub(/\//, "/"); print depth "\t" $0 }' \
      | sort -rn -k1,1 \
      | cut -f2- \
      | tr '\n' '\0'
  )

  local top_entitlements=""
  local top_candidate="${temp_dir}/top.entitlements.plist"
  if extract_entitlements "${bundle_path}" "${top_candidate}"; then
    top_entitlements="${top_candidate}"
  fi
  echo "  signing app: ${bundle_path}"
  codesign_app_target "${bundle_path}" "${top_entitlements}"
}

sign_unbundled_binaries_in_tree() {
  local root_path="$1"
  local temp_dir="$2"

  while IFS= read -r -d '' file_path; do
    if is_within_signable_bundle "${file_path}"; then
      continue
    fi
    sign_binary_if_needed "${file_path}" "${temp_dir}"
  done < <(find "${root_path}" -type f -print0)
}

sign_bundles_in_tree() {
  local root_path="$1"
  local temp_dir="$2"

  while IFS= read -r -d '' bundle_path; do
    sign_bundle_tree "${bundle_path}" "${temp_dir}"
    if [[ "${bundle_path}" == *.app ]]; then
      verify_app "${bundle_path}"
    fi
  done < <(
    find "${root_path}" \
      \( -type d \( -name '*.app' -o -name '*.framework' -o -name '*.bundle' -o -name '*.xpc' -o -name '*.appex' \) \) \
      -print \
      | awk '{ depth=gsub(/\//, "/"); print depth "\t" $0 }' \
      | sort -rn -k1,1 \
      | cut -f2- \
      | tr '\n' '\0'
  )
}

sign_pkgs_in_tree() {
  local root_path="$1"
  local pkg_path
  while IFS= read -r pkg_path; do
    [[ -n "${pkg_path}" ]] || continue
    sign_pkg "${pkg_path}"
  done < <(find "${root_path}" -type f -name '*.pkg' -print | sort)
}

sign_distribution_tree() {
  local root_path="$1"
  local temp_dir="$2"

  sign_unbundled_binaries_in_tree "${root_path}" "${temp_dir}"
  sign_bundles_in_tree "${root_path}" "${temp_dir}"
  sign_pkgs_in_tree "${root_path}"
}

extract_json_field() {
  local json_path="$1"
  local field_name="$2"
  sed -n "s/.*\"${field_name}\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "${json_path}" | head -n 1
}

print_notarytool_result_hint() {
  local result_path="$1"
  [[ -f "${result_path}" ]] || return 0

  if grep -Eqi 'HTTP status code:[[:space:]]*403|required agreement is missing or has expired|requires an in-effect agreement' "${result_path}"; then
    echo "Apple 拒绝了公证请求：必需的法律协议未签署，或协议已过期。" >&2
    if [[ -n "${NOTARY_TEAM_ID}" ]]; then
      echo "请登录 Team ID 为 ${NOTARY_TEAM_ID} 的 Apple Developer 或 App Store Connect 后台，签署所有待处理协议后再重试。" >&2
    else
      echo "请登录 Apple Developer 或 App Store Connect 后台，签署所有待处理协议后再重试。" >&2
    fi
    echo "这是 Apple 账号状态问题，不是当前已签名产物本身有问题。" >&2
    return 0
  fi

  if grep -Eqi 'HTTP status code:[[:space:]]*401|Invalid credentials|Unauthorized|authentication failed' "${result_path}"; then
    echo "Apple 拒绝了公证请求：当前配置的公证凭据未通过校验。" >&2
    echo "请检查 Apple ID、专用密码、App Store Connect API Key 或 keychain profile 是否配置正确，然后再重试。" >&2
    return 0
  fi
}

fetch_notary_log() {
  local submission_id="$1"
  local artifact_path="$2"
  [[ -n "${submission_id}" ]] || return 0

  local log_path="${artifact_path}.notarylog.json"
  local -a cmd=(xcrun notarytool log "${submission_id}" "${log_path}")
  cmd+=("${NOTARY_AUTH_ARGS[@]}")
  if "${cmd[@]}" >/dev/null 2>&1; then
    echo "  notarization log saved to ${log_path}" >&2
  fi
}

is_retryable_notarytool_exit() {
  local status="${1:-0}"
  case "${status}" in
    134|135|136|137|138|139)
      return 0
      ;;
  esac
  return 1
}

run_notarytool_with_retry() {
  local result_path="$1"
  local description="$2"
  shift 2

  local attempt=1
  local max_attempts="${NOTARY_RETRY_COUNT}"
  local delay_seconds="${NOTARY_RETRY_DELAY}"
  local status=0

  if ! [[ "${max_attempts}" =~ ^[0-9]+$ ]] || (( max_attempts < 1 )); then
    max_attempts=1
  fi
  if ! [[ "${delay_seconds}" =~ ^[0-9]+$ ]] || (( delay_seconds < 0 )); then
    delay_seconds=0
  fi

  while true; do
    if "$@" > "${result_path}" 2>&1; then
      return 0
    fi
    status=$?
    if ! is_retryable_notarytool_exit "${status}" || (( attempt >= max_attempts )); then
      return "${status}"
    fi
    echo "  ${description} crashed with exit ${status}; retrying (${attempt}/${max_attempts})" >&2
    if [[ -s "${result_path}" ]]; then
      cat "${result_path}" >&2
    fi
    attempt=$((attempt + 1))
    if (( delay_seconds > 0 )); then
      sleep "${delay_seconds}"
    fi
  done
}

start_notarization_submission() {
  local submit_path="$1"
  local artifact_path="$2"
  local result_path
  result_path="$(mktemp_file_path "$(basename "${artifact_path}").notary-result" ".json")"

  local -a cmd=(
    xcrun notarytool submit "${submit_path}"
    --output-format json
  )
  cmd+=("${NOTARY_AUTH_ARGS[@]}")

  echo "-> notarizing ${artifact_path}" >&2
  if ! run_notarytool_with_retry "${result_path}" "notary submit" "${cmd[@]}"; then
    cat "${result_path}" >&2
    print_notarytool_result_hint "${result_path}"
    local failed_submission_id
    failed_submission_id="$(extract_json_field "${result_path}" "id" || true)"
    fetch_notary_log "${failed_submission_id}" "${artifact_path}"
    rm -f "${result_path}"
    echo "公证失败：${artifact_path}" >&2
    exit 1
  fi

  local submission_id
  submission_id="$(extract_json_field "${result_path}" "id" || true)"
  if [[ -z "${submission_id}" ]]; then
    cat "${result_path}" >&2
    print_notarytool_result_hint "${result_path}"
    rm -f "${result_path}"
    echo "公证提交未返回 submission id：${artifact_path}" >&2
    exit 1
  fi

  rm -f "${result_path}"
  printf '%s\n' "${submission_id}"
}

wait_for_notarization() {
  local submission_id="$1"
  local artifact_path="$2"
  local result_path
  result_path="$(mktemp_file_path "$(basename "${artifact_path}").notary-result" ".json")"

  local -a cmd=(
    xcrun notarytool wait "${submission_id}"
    --timeout "${NOTARY_TIMEOUT}"
    --output-format json
  )
  cmd+=("${NOTARY_AUTH_ARGS[@]}")

  if ! run_notarytool_with_retry "${result_path}" "notary wait" "${cmd[@]}"; then
    cat "${result_path}" >&2
    fetch_notary_log "${submission_id}" "${artifact_path}"
    rm -f "${result_path}"
    echo "Notarization failed for ${artifact_path}" >&2
    exit 1
  fi

  local status
  status="$(extract_json_field "${result_path}" "status" || true)"
  if [[ "${status}" != "Accepted" ]]; then
    cat "${result_path}" >&2
    fetch_notary_log "${submission_id}" "${artifact_path}"
    rm -f "${result_path}"
    echo "Notarization did not complete successfully for ${artifact_path}: ${status:-unknown}" >&2
    exit 1
  fi

  rm -f "${result_path}"
  echo "  notarization accepted: ${artifact_path} (${submission_id})"
}

submit_notarization() {
  local submit_path="$1"
  local artifact_path="$2"
  local submission_id
  submission_id="$(start_notarization_submission "${submit_path}" "${artifact_path}")"
  wait_for_notarization "${submission_id}" "${artifact_path}"
}

staple_artifact() {
  local artifact_path="$1"
  echo "-> stapling ${artifact_path}"
  xcrun stapler staple -v "${artifact_path}"
  echo "  validating staple: ${artifact_path}"
  xcrun stapler validate -v "${artifact_path}"
}

notarize_app() {
  local app_path="$1"
  local temp_root
  temp_root="$(mktemp -d "${TMPDIR:-/tmp}/deepright-notary-app.XXXXXX")"
  local stage_root="${temp_root}/stage"
  local stage_app="${stage_root}/$(basename "${app_path}")"
  local archive_path
  archive_path="$(mktemp_file_path "$(basename "${app_path}" .app).notary" ".zip")"

  cleanup_notarize_app() {
    rm -rf "${temp_root}"
    rm -f "${archive_path}"
  }
  trap cleanup_notarize_app RETURN

  mkdir -p "${stage_root}"
  echo "  staging app for notarization: ${app_path}"
  ditto "${app_path}" "${stage_app}"
  clean_mutable_bundle_runtime_files "${stage_app}"
  verify_app "${stage_app}"

  ditto -c -k --keepParent "${stage_app}" "${archive_path}"
  submit_notarization "${archive_path}" "${app_path}"

  rm -rf "${app_path}"
  ditto "${stage_app}" "${app_path}"
  staple_artifact "${app_path}"
  verify_app "${app_path}" "post-notarization"
}

notarize_pkg() {
  local pkg_path="$1"
  submit_notarization "${pkg_path}" "${pkg_path}"
  staple_artifact "${pkg_path}"
  verify_pkg "${pkg_path}" "post-notarization"
}

notarize_dmg() {
  local dmg_path="$1"
  submit_notarization "${dmg_path}" "${dmg_path}"
  staple_artifact "${dmg_path}"
  verify_dmg "${dmg_path}" "post-notarization"
}

verify_app() {
  local app_path="$1"
  local phase="${2:-pre-notarization}"
  echo "  verifying codesign: ${app_path}"
  codesign --verify --deep --strict --verbose=2 "${app_path}"
  if [[ "${USE_RUNTIME_OPTIONS}" -eq 1 ]] && command -v spctl >/dev/null 2>&1; then
    if [[ "${phase}" == "pre-notarization" && "${NOTARIZE_ENABLED}" -eq 1 && "${REQUIRE_NOTARIZATION_CHECK}" != "1" ]]; then
      return 0
    fi
    echo "  verifying gatekeeper: ${app_path}"
    if ! spctl --assess --type execute --verbose=4 "${app_path}"; then
      if [[ "${phase}" == "post-notarization" || "${REQUIRE_NOTARIZATION_CHECK}" == "1" ]]; then
        echo "Gatekeeper assessment failed for ${app_path} (${phase})" >&2
        exit 1
      fi
      echo "warning: gatekeeper rejected ${app_path}; this is expected before notarization/stapling" >&2
    fi
  fi
}

resolve_app_targets() {
  local -a apps=()
  local arch app_path
  for arch in ${ARCHES}; do
    app_path="${RELEASE_DIR}/${arch}/${APP_NAME}.app"
    if [[ ! -d "${app_path}" ]]; then
      echo "warning: missing mac app for arch=${arch}: ${app_path}" >&2
      continue
    fi
    apps+=("${app_path}")
  done
  printf '%s\n' "${apps[@]}"
}

resolve_pkg_targets() {
  find "${RELEASE_DIR}" -type f -name '*.pkg' -print | sort
}

resolve_dmg_targets() {
  find "${RELEASE_DIR}" -type f -name '*.dmg' -print | sort
}

verify_pkg() {
  local pkg_path="$1"
  local phase="${2:-pre-notarization}"
  echo "  verifying installer signature: ${pkg_path}"
  pkgutil --check-signature "${pkg_path}"
  if command -v spctl >/dev/null 2>&1; then
    if [[ "${phase}" == "pre-notarization" && "${NOTARIZE_ENABLED}" -eq 1 && "${REQUIRE_NOTARIZATION_CHECK}" != "1" ]]; then
      return 0
    fi
    echo "  verifying gatekeeper install assessment: ${pkg_path}"
    if ! spctl --assess --type install --verbose=4 "${pkg_path}"; then
      if [[ "${phase}" == "post-notarization" || "${REQUIRE_NOTARIZATION_CHECK}" == "1" ]]; then
        echo "Gatekeeper install assessment failed for ${pkg_path} (${phase})" >&2
        exit 1
      fi
      echo "warning: gatekeeper rejected ${pkg_path}; this is expected before notarization/stapling" >&2
    fi
  fi
}

parse_mountpoints_from_hdiutil_output() {
  sed -n 's#^/dev/[^[:space:]]*[[:space:]].*[[:space:]]\(/.*\)$#\1#p'
}

list_mountpoints_for_image() {
  local dmg_path="$1"
  hdiutil info | awk -F '\t' -v target="${dmg_path}" '
    /^image-path[[:space:]]*:/ {
      current = substr($0, index($0, ":") + 2)
      active = (current == target)
      next
    }
    /^================================================/ {
      active = 0
      next
    }
    active && $1 ~ /^\/dev\// && NF >= 3 && $3 ~ /^\// {
      print $3
    }
  '
}

detach_image_mounts() {
  local dmg_path="$1"
  local mount_point
  while IFS= read -r mount_point; do
    [[ -n "${mount_point}" ]] || continue
    hdiutil detach "${mount_point}" -quiet >/dev/null 2>&1 || true
  done < <(list_mountpoints_for_image "${dmg_path}")
}

verify_dmg_contents_gatekeeper() {
  local dmg_path="$1"
  local attach_output=""
  local attached_here=0
  local -a mount_points=()

  cleanup_verify_dmg_contents() {
    local mount_point
    if [[ "${attached_here}" -eq 1 ]]; then
      for mount_point in "${mount_points[@]}"; do
        [[ -n "${mount_point}" ]] || continue
        hdiutil detach "${mount_point}" -quiet >/dev/null 2>&1 || true
      done
    fi
  }

  detach_image_mounts "${dmg_path}"

  if attach_output="$(hdiutil attach "${dmg_path}" -readonly -nobrowse 2>&1)"; then
    attached_here=1
    while IFS= read -r mount_point; do
      [[ -n "${mount_point}" ]] || continue
      mount_points+=("${mount_point}")
    done < <(printf '%s\n' "${attach_output}" | parse_mountpoints_from_hdiutil_output)
  else
    printf '%s\n' "${attach_output}" >&2
    while IFS= read -r mount_point; do
      [[ -n "${mount_point}" ]] || continue
      mount_points+=("${mount_point}")
    done < <(list_mountpoints_for_image "${dmg_path}")
  fi

  if [[ "${#mount_points[@]}" -eq 0 ]]; then
    echo "Unable to determine mounted volume for ${dmg_path}" >&2
    cleanup_verify_dmg_contents
    return 1
  fi

  local found_any=0
  local verify_failed=0
  local mount_point
  local app_path
  local pkg_path
  for mount_point in "${mount_points[@]}"; do
    while IFS= read -r app_path; do
      [[ -n "${app_path}" ]] || continue
      found_any=1
      echo "  verifying gatekeeper execute assessment for dmg app: ${app_path}"
      if ! spctl --assess --type execute --verbose=4 "${app_path}"; then
        verify_failed=1
      fi
    done < <(find "${mount_point}" -maxdepth 3 -name '*.app' -print | sort)

    while IFS= read -r pkg_path; do
      [[ -n "${pkg_path}" ]] || continue
      found_any=1
      echo "  verifying gatekeeper install assessment for dmg pkg: ${pkg_path}"
      if ! spctl --assess --type install --verbose=4 "${pkg_path}"; then
        verify_failed=1
      fi
    done < <(find "${mount_point}" -maxdepth 3 -name '*.pkg' -print | sort)
  done

  if [[ "${found_any}" -eq 0 ]]; then
    echo "No .app or .pkg payload found inside ${dmg_path} for fallback gatekeeper assessment" >&2
    cleanup_verify_dmg_contents
    return 1
  fi

  cleanup_verify_dmg_contents
  [[ "${verify_failed}" -eq 0 ]]
}

sign_pkg() {
  local pkg_path="$1"
  if [[ -z "${INSTALLER_IDENTITY}" ]]; then
    echo "No usable Developer ID Installer identity found for pkg signing: ${pkg_path}" >&2
    echo "Fix the .p12 password or install a Developer ID Installer certificate, then retry." >&2
    exit 1
  fi

  local signed_path
  signed_path="$(mktemp_file_path "$(basename "${pkg_path}" .pkg).signed" ".pkg")"
  echo "-> signing installer package ${pkg_path}"
  productsign_pkg "${pkg_path}" "${signed_path}"
  mv "${signed_path}" "${pkg_path}"
  verify_pkg "${pkg_path}"
}

verify_dmg() {
  local dmg_path="$1"
  local phase="${2:-pre-notarization}"
  echo "  verifying codesign: ${dmg_path}"
  codesign --verify --verbose=2 "${dmg_path}"
  if command -v spctl >/dev/null 2>&1; then
    if [[ "${phase}" == "pre-notarization" && "${NOTARIZE_ENABLED}" -eq 1 && "${REQUIRE_NOTARIZATION_CHECK}" != "1" ]]; then
      return 0
    fi
    echo "  verifying gatekeeper open assessment: ${dmg_path}"
    local assess_output=""
    if ! assess_output="$(spctl --assess --type open --verbose=4 "${dmg_path}" 2>&1)"; then
      printf '%s\n' "${assess_output}"
      if [[ "${assess_output}" == *"source=Insufficient Context"* ]]; then
        echo "  gatekeeper open assessment returned insufficient context; verifying dmg contents instead"
        if verify_dmg_contents_gatekeeper "${dmg_path}"; then
          return 0
        fi
      fi
      if [[ "${phase}" == "post-notarization" || "${REQUIRE_NOTARIZATION_CHECK}" == "1" ]]; then
        echo "Gatekeeper open assessment failed for ${dmg_path} (${phase})" >&2
        exit 1
      fi
      echo "warning: gatekeeper rejected ${dmg_path}; this is expected before notarization/stapling" >&2
    else
      printf '%s\n' "${assess_output}"
    fi
  fi
}

sign_app_bundle() {
  local app_path="$1"
  local temp_root
  temp_root="$(mktemp -d "${TMPDIR:-/tmp}/deepright-sign-app.XXXXXX")"

  local stage_root="${temp_root}/stage"
  local stage_app="${stage_root}/$(basename "${app_path}")"
  local sign_temp_dir="${temp_root}/sign"

  cleanup_sign_app_bundle() {
    rm -rf "${temp_root}"
  }
  trap cleanup_sign_app_bundle RETURN

  mkdir -p "${stage_root}" "${sign_temp_dir}"
  echo "  staging app for signing: ${app_path}"
  ditto "${app_path}" "${stage_app}"

  sign_bundle_tree "${stage_app}" "${sign_temp_dir}"
  verify_app "${stage_app}"

  rm -rf "${app_path}"
  ditto "${stage_app}" "${app_path}"
  verify_app "${app_path}"
}

sign_dmg() {
  local dmg_path="$1"
  local dmg_name
  dmg_name="$(basename "${dmg_path}" .dmg)"
  local repacked_dmg
  repacked_dmg="$(mktemp_file_path "${dmg_name}.repacked" ".dmg")"

  echo "-> repackaging and signing disk image contents ${dmg_path}"
  (
    set -euo pipefail

    local temp_root
    temp_root="$(mktemp -d "${TMPDIR:-/tmp}/${dmg_name}.contents.XXXXXX")"
    local mount_path="${temp_root}/mount"
    local stage_path="${temp_root}/stage"
    local sign_temp_dir="${temp_root}/sign"
    local dmg_volume_name=""
    local attached=0

    cleanup_sign_dmg() {
      if [[ "${attached}" -eq 1 ]]; then
        hdiutil detach "${mount_path}" -quiet >/dev/null 2>&1 || true
      fi
      rm -rf "${temp_root}"
    }
    trap cleanup_sign_dmg EXIT

    mkdir -p "${mount_path}" "${stage_path}" "${sign_temp_dir}"
    hdiutil attach "${dmg_path}" -readonly -nobrowse -mountpoint "${mount_path}" -quiet
    attached=1
    dmg_volume_name="$(diskutil info -plist "${mount_path}" | plutil -extract VolumeName raw -)"
    if [[ -z "${dmg_volume_name}" ]]; then
      echo "Unable to determine disk image volume name: ${dmg_path}" >&2
      exit 1
    fi
    ditto "${mount_path}" "${stage_path}"
    hdiutil detach "${mount_path}" -quiet >/dev/null
    attached=0

    sign_distribution_tree "${stage_path}" "${sign_temp_dir}"

    hdiutil create \
      -volname "${dmg_volume_name}" \
      -srcfolder "${stage_path}" \
      -ov \
      -format UDZO \
      "${repacked_dmg}" >/dev/null
  )

  mv "${repacked_dmg}" "${dmg_path}"
  echo "-> signing disk image ${dmg_path}"
  codesign_distribution_file "${dmg_path}"
  verify_dmg "${dmg_path}"
}

main() {
  assert_macos
  require_command codesign
  require_command security
  require_command file
  require_command shasum
  require_command find
  require_command sort
  require_command ditto

  if [[ ! -d "${RELEASE_DIR}" ]]; then
    echo "release directory not found: ${RELEASE_DIR}" >&2
    exit 1
  fi

  setup_identities
  if [[ "${USE_RUNTIME_OPTIONS}" -eq 1 ]]; then
    check_timestamp_service
  fi
  setup_notarization

  local app_path
  local found_any=0
  while IFS= read -r app_path; do
    [[ -n "${app_path}" ]] || continue
    found_any=1
    echo "-> signing ${app_path}"
    sign_app_bundle "${app_path}"
  done < <(resolve_app_targets)

  local pkg_path
  while IFS= read -r pkg_path; do
    [[ -n "${pkg_path}" ]] || continue
    found_any=1
    require_command productsign
    require_command pkgutil
    sign_pkg "${pkg_path}"
  done < <(resolve_pkg_targets)

  local dmg_path
  while IFS= read -r dmg_path; do
    [[ -n "${dmg_path}" ]] || continue
    found_any=1
    require_command hdiutil
    sign_dmg "${dmg_path}"
  done < <(resolve_dmg_targets)

  if [[ "${found_any}" -eq 0 ]]; then
    echo "No signable mac artifacts found under ${RELEASE_DIR}" >&2
    exit 1
  fi

  if [[ "${NOTARIZE_ENABLED}" -eq 1 ]]; then
    require_command xcrun
    local notarized_any=0

    while IFS= read -r app_path; do
      [[ -n "${app_path}" ]] || continue
      notarized_any=1
      notarize_app "${app_path}"
    done < <(resolve_app_targets)

    while IFS= read -r pkg_path; do
      [[ -n "${pkg_path}" ]] || continue
      notarized_any=1
      notarize_pkg "${pkg_path}"
    done < <(resolve_pkg_targets)

    while IFS= read -r dmg_path; do
      [[ -n "${dmg_path}" ]] || continue
      notarized_any=1
      notarize_dmg "${dmg_path}"
    done < <(resolve_dmg_targets)

    if [[ "${notarized_any}" -eq 0 ]]; then
      echo "warning: no artifacts available for notarization" >&2
    else
      echo "mac notarization, stapling, and gatekeeper validation completed"
    fi
  fi

  echo "mac distribution signing completed"
  RUN_COMPLETED=1
  print_final_success_summary
}

main "$@"
