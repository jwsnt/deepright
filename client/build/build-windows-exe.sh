#!/bin/sh

set -eu

GO_BIN="${GO_BIN:-go}"
GO_RELEASE_LDFLAGS="${GO_RELEASE_LDFLAGS:--s -w}"
BUILD_SCOPE="${1:-all}"
SKIP_LINUX_BUILD="${DEEPRIGHT_SKIP_LINUX_BUILD:-0}"
KEEP_TMP_DIR="${DEEPRIGHT_KEEP_WINDOWS_EXE_TMP:-0}"
APP_NAME="${DEEPRIGHT_WINDOWS_EXE_APP_NAME:-DeepRight}"
UNINSTALLER_BUNDLE_NAMESPACE="${DEEPRIGHT_WINDOWS_EXE_UNINSTALLER_NAMESPACE:-${APP_NAME}-uninstaller}"

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
MODULE_DIR="$SCRIPT_DIR"
RELEASE_DIR="$MODULE_DIR/release"
LINUX_RELEASE_DIR="$RELEASE_DIR/linux"
WINDOWS_RELEASE_DIR="$RELEASE_DIR/windows"
WRAPPER_TEMPLATE_DIR="$MODULE_DIR/build/exe-wrapper"
BUILD_TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/deepright-windows-exe.XXXXXX")

cleanup_build_tmp() {
  if [ "$KEEP_TMP_DIR" = "1" ]; then
    echo "-> keeping temp dir: $BUILD_TMP_DIR"
    return 0
  fi
  rm -rf "$BUILD_TMP_DIR"
}
trap cleanup_build_tmp EXIT HUP INT TERM

print_usage() {
  echo "Usage: $0 [x86|arm|all]" >&2
  echo "  x86  Build the Windows single-file launcher set for linux/x86 release payload." >&2
  echo "  arm  Build the Windows single-file launcher set for linux/arm release payload." >&2
  echo "  all  Build both Windows single-file launcher sets." >&2
  echo "  omit Build both Windows single-file launcher sets." >&2
}

if [ "$#" -gt 1 ]; then
  print_usage
  exit 1
fi

BUILD_X86=0
BUILD_ARM=0
case "$BUILD_SCOPE" in
  ""|all)
    BUILD_X86=1
    BUILD_ARM=1
    ;;
  x86)
    BUILD_X86=1
    ;;
  arm)
    BUILD_ARM=1
    ;;
  *)
    print_usage
    exit 1
    ;;
esac

copy_release_asset() {
  src_path="$1"
  dst_path="$2"
  if [ -d "$src_path" ]; then
    rm -rf "$dst_path"
    mkdir -p "$dst_path"
    cp -R "$src_path/." "$dst_path/"
    return 0
  fi
  rm -f "$dst_path"
  mkdir -p "$(dirname "$dst_path")"
  cp -R "$src_path" "$dst_path"
}

clean_macos_metadata() {
  target_dir="$1"
  find "$target_dir" \( -name '.DS_Store' -o -name '._*' \) -exec rm -rf {} +
}

target_goarch_for_release() {
  case "$1" in
    x86)
      printf '%s' "amd64"
      ;;
    arm)
      printf '%s' "arm64"
      ;;
    *)
      echo "unknown Windows release target: $1" >&2
      exit 1
      ;;
  esac
}

target_installer_name() {
  case "$1" in
    x86)
      printf '%s' "${APP_NAME}-windows-x86-installer.exe"
      ;;
    arm)
      printf '%s' "${APP_NAME}-windows-arm-installer.exe"
      ;;
    *)
      echo "unknown installer target: $1" >&2
      exit 1
      ;;
  esac
}

target_uninstaller_name() {
  case "$1" in
    x86)
      printf '%s' "${APP_NAME}-windows-x86-uninstaller.exe"
      ;;
    arm)
      printf '%s' "${APP_NAME}-windows-arm-uninstaller.exe"
      ;;
    *)
      echo "unknown uninstaller target: $1" >&2
      exit 1
      ;;
  esac
}

required_linux_payload_paths() {
  target_name="$1"
  cat <<EOF
$LINUX_RELEASE_DIR/$target_name/integration
$LINUX_RELEASE_DIR/$target_name/config
$LINUX_RELEASE_DIR/$target_name/site
$LINUX_RELEASE_DIR/$target_name/install.bat
$LINUX_RELEASE_DIR/$target_name/install.ps1
$LINUX_RELEASE_DIR/$target_name/start.bat
$LINUX_RELEASE_DIR/$target_name/uninstall.bat
$LINUX_RELEASE_DIR/$target_name/uninstall.ps1
$LINUX_RELEASE_DIR/$target_name/USER_GUIDE.txt
EOF
}

verify_linux_release_target() {
  target_name="$1"
  required_linux_payload_paths "$target_name" | while IFS= read -r required_path; do
    if [ ! -e "$required_path" ]; then
      echo "missing linux release payload for $target_name: $required_path" >&2
      exit 1
    fi
  done
}

ensure_linux_release_artifacts() {
  if ! command -v zip >/dev/null 2>&1; then
    echo "zip is required to build the Windows single-file installer payload" >&2
    exit 1
  fi

  if [ "$SKIP_LINUX_BUILD" = "1" ]; then
    echo "-> reusing existing linux release payloads"
  else
    echo "-> building linux release payloads via build.sh"
    sh "$MODULE_DIR/build.sh" linux
  fi

  if [ "$BUILD_X86" -eq 1 ]; then
    verify_linux_release_target "x86"
  fi
  if [ "$BUILD_ARM" -eq 1 ]; then
    verify_linux_release_target "arm"
  fi
}

write_payload_manifest() {
  payload_root="$1"
  target_name="$2"
  launcher_kind="$3"
  manifest_path="$payload_root/BUILD_WINDOWS_EXE.txt"

  cat > "$manifest_path" <<EOF
DeepRight Windows single-file launcher payload
Target: $target_name
Payload-Type: $launcher_kind
Built-At-UTC: $(date -u '+%Y-%m-%dT%H:%M:%SZ')
Builder: cli/module/build-windows-exe.sh
EOF
}

stage_launcher_payload() {
  target_name="$1"
  launcher_kind="$2"
  payload_dir="$3"
  linux_target_dir="$LINUX_RELEASE_DIR/$target_name"

  rm -rf "$payload_dir"
  mkdir -p "$payload_dir"

  case "$launcher_kind" in
    installer)
      copy_release_asset "$linux_target_dir" "$payload_dir"
      ;;
    uninstaller)
      copy_release_asset "$linux_target_dir/uninstall.bat" "$payload_dir/uninstall.bat"
      copy_release_asset "$linux_target_dir/uninstall.ps1" "$payload_dir/uninstall.ps1"
      copy_release_asset "$linux_target_dir/USER_GUIDE.txt" "$payload_dir/USER_GUIDE.txt"
      ;;
    *)
      echo "unknown launcher kind: $launcher_kind" >&2
      exit 1
      ;;
  esac

  clean_macos_metadata "$payload_dir"
}

escape_go_string() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

required_wrapper_paths() {
  launcher_kind="$1"
  case "$launcher_kind" in
    installer)
      cat <<EOF
install.bat
install.ps1
start.bat
integration
config
site
EOF
      ;;
    uninstaller)
      cat <<EOF
uninstall.bat
uninstall.ps1
USER_GUIDE.txt
EOF
      ;;
    *)
      echo "unknown launcher kind: $launcher_kind" >&2
      exit 1
      ;;
  esac
}

write_wrapper_config() {
  config_path="$1"
  launcher_kind="$2"
  wrapper_app_name="$APP_NAME"
  wrapper_bundle_namespace="$APP_NAME"
  wrapper_default_mode="install"
  wrapper_default_batch_name=""
  wrapper_install_batch_name="install.bat"
  wrapper_start_batch_name="start.bat"
  wrapper_run_batch_name=""
  wrapper_supports_auto="true"
  wrapper_supports_install="true"
  wrapper_supports_start="true"
  wrapper_supports_run="false"
  wrapper_launcher_description="${APP_NAME} Windows installer launcher"

  case "$launcher_kind" in
    installer)
      ;;
    uninstaller)
      wrapper_bundle_namespace="$UNINSTALLER_BUNDLE_NAMESPACE"
      wrapper_default_mode="run"
      wrapper_default_batch_name="uninstall.bat"
      wrapper_install_batch_name=""
      wrapper_start_batch_name=""
      wrapper_run_batch_name="uninstall.bat"
      wrapper_supports_auto="false"
      wrapper_supports_install="false"
      wrapper_supports_start="false"
      wrapper_supports_run="true"
      wrapper_launcher_description="${APP_NAME} Windows uninstaller launcher"
      ;;
    *)
      echo "unknown launcher kind: $launcher_kind" >&2
      exit 1
      ;;
  esac

  required_paths_literal="$(
    required_wrapper_paths "$launcher_kind" | while IFS= read -r required_path; do
      if [ -z "$required_path" ]; then
        continue
      fi
      printf '    "%s",\n' "$(escape_go_string "$required_path")"
    done
  )"

  cat > "$config_path" <<EOF
package main

var wrapperAppName = "$(escape_go_string "$wrapper_app_name")"
var wrapperBundleNamespace = "$(escape_go_string "$wrapper_bundle_namespace")"
var wrapperDefaultMode = "$(escape_go_string "$wrapper_default_mode")"
var wrapperDefaultBatchName = "$(escape_go_string "$wrapper_default_batch_name")"
var wrapperInstallBatchName = "$(escape_go_string "$wrapper_install_batch_name")"
var wrapperStartBatchName = "$(escape_go_string "$wrapper_start_batch_name")"
var wrapperRunBatchName = "$(escape_go_string "$wrapper_run_batch_name")"
var wrapperSupportsAuto = $wrapper_supports_auto
var wrapperSupportsInstall = $wrapper_supports_install
var wrapperSupportsStart = $wrapper_supports_start
var wrapperSupportsRun = $wrapper_supports_run
var wrapperLauncherDescription = "$(escape_go_string "$wrapper_launcher_description")"
var wrapperRequiredPaths = []string{
$required_paths_literal}
EOF
}

prepare_wrapper_source() {
  wrapper_src_dir="$1"
  payload_zip_path="$2"
  launcher_kind="$3"

  mkdir -p "$wrapper_src_dir"
  cp "$WRAPPER_TEMPLATE_DIR/main.go.tmpl" "$wrapper_src_dir/main.go"
  cp "$payload_zip_path" "$wrapper_src_dir/payload.zip"
  write_wrapper_config "$wrapper_src_dir/config.go" "$launcher_kind"
}

target_launcher_name() {
  target_name="$1"
  launcher_kind="$2"
  case "$launcher_kind" in
    installer)
      target_installer_name "$target_name"
      ;;
    uninstaller)
      target_uninstaller_name "$target_name"
      ;;
    *)
      echo "unknown launcher kind: $launcher_kind" >&2
      exit 1
      ;;
  esac
}

package_windows_launcher() {
  target_name="$1"
  launcher_kind="$2"
  windows_target_dir="$WINDOWS_RELEASE_DIR/$target_name"
  stage_dir="$BUILD_TMP_DIR/$target_name-$launcher_kind"
  payload_dir="$stage_dir/payload"
  wrapper_src_dir="$stage_dir/wrapper"
  payload_zip_path="$stage_dir/payload.zip"
  artifact_name="$(target_launcher_name "$target_name" "$launcher_kind")"
  output_path="$windows_target_dir/$artifact_name"
  output_hash_path="$output_path.sha256"
  target_goarch="$(target_goarch_for_release "$target_name")"

  echo "-> packaging Windows single-file $launcher_kind ($target_name)"
  rm -rf "$stage_dir"
  mkdir -p "$windows_target_dir"
  rm -f "$output_path" "$output_hash_path"

  stage_launcher_payload "$target_name" "$launcher_kind" "$payload_dir"
  write_payload_manifest "$payload_dir" "$target_name" "$launcher_kind"

  (
    cd "$payload_dir"
    zip -qr "$payload_zip_path" .
  )

  prepare_wrapper_source "$wrapper_src_dir" "$payload_zip_path" "$launcher_kind"

  (
    cd "$wrapper_src_dir"
    GO111MODULE=off GOOS=windows GOARCH="$target_goarch" \
      "$GO_BIN" build -trimpath -ldflags "$GO_RELEASE_LDFLAGS" -o "$output_path" main.go config.go
  )

  if command -v shasum >/dev/null 2>&1; then
    (
      cd "$windows_target_dir"
      LC_ALL=C LANG=C shasum -a 256 "$artifact_name" > "$output_hash_path"
    )
  fi

  echo "   created: $output_path"
}

echo "Building Windows single-file launchers..."
mkdir -p "$WINDOWS_RELEASE_DIR"
ensure_linux_release_artifacts

if [ "$BUILD_X86" -eq 1 ]; then
  package_windows_launcher "x86" "installer"
  package_windows_launcher "x86" "uninstaller"
fi
if [ "$BUILD_ARM" -eq 1 ]; then
  package_windows_launcher "arm" "installer"
  package_windows_launcher "arm" "uninstaller"
fi

echo "Windows launcher build completed:"
find "$WINDOWS_RELEASE_DIR" -mindepth 1 | sort | while IFS= read -r path; do
  echo "  $path"
done
