#!/bin/sh

set -eu

GO_BIN="${GO_BIN:-go}"
GO_RELEASE_LDFLAGS="${GO_RELEASE_LDFLAGS:--s -w}"
DEEPRIGHT_RELEASE_PLUGINS="${DEEPRIGHT_RELEASE_PLUGINS:-}"
BUILD_SCOPE="${1:-all}"

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
MODULE_DIR="$SCRIPT_DIR"
INTEGRATION_DIR="$MODULE_DIR/integration"
CONNECT_DIR="$MODULE_DIR/connect"
SITE_DIR="$MODULE_DIR/site"
CONFIG_DIR="$MODULE_DIR/config"
CLI_GET_SANDBOX_DIR="$MODULE_DIR/cli-get/sandbox"
CLI_GET_SANDBOX_MAC_RELEASE_DIR="$CLI_GET_SANDBOX_DIR/release/mac"
CLI_GET_SANDBOX_WSL_RELEASE_DIR="$CLI_GET_SANDBOX_DIR/release/wsl"
RELEASE_DIR="$MODULE_DIR/release"
BUILD_TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/deepright-module-build.XXXXXX")
KEY_DIR="${DEEPRIGHT_KEY:-}"
SIGN_MODE="${DEEPRIGHT_SKIP_SIGN:-auto}"
SIGN_IDENTITY="${DEEPRIGHT_IDENTITY:-}"
SIGN_KEYCHAIN_PATH=""
SIGN_KEYCHAIN_PASSWORD="${DEEPRIGHT_KEYCHAIN_PASSWORD:-}"
SIGN_USE_KEYCHAIN=0
SIGN_ENABLED=0
MAC_APP_NAME="${DEEPRIGHT_MAC_APP_NAME:-DeepRight}"

# Normalize cwd immediately so later subshells do not inherit a deleted caller cwd.
cd "$SCRIPT_DIR"

print_usage() {
  echo "Usage: $0 [linux|mac|all]" >&2
  echo "  linux  Build Linux release artifacts, including the Windows WSL2 launcher." >&2
  echo "  mac    Build macOS release artifacts." >&2
  echo "  all    Build all release artifacts." >&2
  echo "  omit   Build all release artifacts." >&2
}

if [ "$#" -gt 1 ]; then
  print_usage
  exit 1
fi

BUILD_LINUX=0
BUILD_MAC=0
case "$BUILD_SCOPE" in
  ""|all)
    BUILD_LINUX=1
    BUILD_MAC=1
    ;;
  linux)
    BUILD_LINUX=1
    ;;
  mac)
    BUILD_MAC=1
    ;;
  *)
    print_usage
    exit 1
    ;;
esac

cleanup_build_tmp() {
  if [ -n "$SIGN_KEYCHAIN_PATH" ]; then
    security delete-keychain "$SIGN_KEYCHAIN_PATH" >/dev/null 2>&1 || true
  fi
  rm -rf "$BUILD_TMP_DIR"
}
trap cleanup_build_tmp EXIT

trim_file_content() {
  path="$1"
  tr -d '\r' < "$path" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'
}

read_first_existing() {
  for path in "$@"; do
    if [ -f "$path" ]; then
      trim_file_content "$path"
      return 0
    fi
  done
  return 1
}

plugin_selected_for_release() {
  plugin_name=$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')
  selected_plugins=$(printf '%s' "$DEEPRIGHT_RELEASE_PLUGINS" | tr ',;' '   ' | tr '[:upper:]' '[:lower:]')
  case "$selected_plugins" in
    ""|"all")
      return 0
      ;;
    "none")
      return 1
      ;;
  esac
  for selected in $selected_plugins; do
    if [ "$selected" = "$plugin_name" ]; then
      return 0
    fi
  done
  return 1
}

has_system_codesign_identity() {
  if ! command -v security >/dev/null 2>&1; then
    return 1
  fi
  security find-identity -v -p codesigning 2>/dev/null | grep -E 'Developer ID Application:|Apple Development:' >/dev/null 2>&1
}

setup_codesign_identity() {
  case "$SIGN_MODE" in
    1|true|TRUE|yes|YES)
      SIGN_ENABLED=0
      return 0
      ;;
    0|false|FALSE|no|NO)
      SIGN_ENABLED=1
      ;;
    *)
      if [ -n "$KEY_DIR" ] || has_system_codesign_identity; then
        SIGN_ENABLED=1
      else
        SIGN_ENABLED=0
        return 0
      fi
      ;;
  esac

  if [ -z "$SIGN_KEYCHAIN_PASSWORD" ]; then
    SIGN_KEYCHAIN_PASSWORD="$(uuidgen)"
  fi

  P12_PASSWORD="${DEEPRIGHT_P12_PASSWORD:-}"
  if [ -z "$P12_PASSWORD" ] && [ -n "$KEY_DIR" ]; then
    P12_PASSWORD="$(read_first_existing \
      "$KEY_DIR/DeepRight_p12_password.txt" \
      "$KEY_DIR/p12_password.txt" \
      "$KEY_DIR/password.txt" || true)"
  fi

  if [ -z "$SIGN_IDENTITY" ] && [ -n "$KEY_DIR" ]; then
    SIGN_IDENTITY="$(read_first_existing \
      "$KEY_DIR/identity.txt" \
      "$KEY_DIR/developer_id_identity.txt" || true)"
  fi

  IMPORT_FAILED=0
  if [ -n "$KEY_DIR" ] && [ -d "$KEY_DIR" ]; then
    SIGN_KEYCHAIN_PATH="$BUILD_TMP_DIR/deepright-build.keychain-db"
    security create-keychain -p "$SIGN_KEYCHAIN_PASSWORD" "$SIGN_KEYCHAIN_PATH"
    security unlock-keychain -p "$SIGN_KEYCHAIN_PASSWORD" "$SIGN_KEYCHAIN_PATH"
    security set-keychain-settings -lut 21600 "$SIGN_KEYCHAIN_PATH"

    for cert in $(find "$KEY_DIR" -maxdepth 1 -type f -name '*.cer' | sort); do
      security import "$cert" -k "$SIGN_KEYCHAIN_PATH" -T /usr/bin/codesign -T /usr/bin/security >/dev/null
    done
    for archive in $(find "$KEY_DIR" -maxdepth 1 -type f -name '*.p12' | sort); do
      if ! security import "$archive" -k "$SIGN_KEYCHAIN_PATH" -P "$P12_PASSWORD" -T /usr/bin/codesign -T /usr/bin/security >/dev/null; then
        echo "warning: failed to import $archive into temporary keychain" >&2
        IMPORT_FAILED=1
      fi
    done

    if security find-identity -v -p codesigning "$SIGN_KEYCHAIN_PATH" >/dev/null 2>&1; then
      security set-key-partition-list -S apple-tool:,apple: -s -k "$SIGN_KEYCHAIN_PASSWORD" "$SIGN_KEYCHAIN_PATH" >/dev/null || true
    fi
  fi

  if [ -z "$SIGN_IDENTITY" ] && [ -n "$SIGN_KEYCHAIN_PATH" ]; then
    SIGN_IDENTITY="$(
      security find-identity -v -p codesigning "$SIGN_KEYCHAIN_PATH" \
        | sed -n 's/.*"Developer ID Application: \([^"]*\)"/Developer ID Application: \1/p' \
        | head -n 1
    )"
  fi
  if [ -z "$SIGN_IDENTITY" ] && [ -n "$SIGN_KEYCHAIN_PATH" ]; then
    SIGN_IDENTITY="$(
      security find-identity -v -p codesigning "$SIGN_KEYCHAIN_PATH" \
        | sed -n 's/.*"Apple Development: \([^"]*\)"/Apple Development: \1/p' \
        | head -n 1
    )"
  fi

  SIGN_USE_KEYCHAIN=1
  if [ -z "$SIGN_IDENTITY" ]; then
    if [ "$IMPORT_FAILED" -eq 1 ]; then
      echo "warning: no usable identity found in temporary keychain, falling back to system keychains" >&2
    fi
    SIGN_IDENTITY="$(
      security find-identity -v -p codesigning \
        | sed -n 's/.*"Developer ID Application: \([^"]*\)"/Developer ID Application: \1/p' \
        | head -n 1
    )"
    if [ -z "$SIGN_IDENTITY" ]; then
      SIGN_IDENTITY="$(
        security find-identity -v -p codesigning \
          | sed -n 's/.*"Apple Development: \([^"]*\)"/Apple Development: \1/p' \
          | head -n 1
      )"
    fi
    SIGN_USE_KEYCHAIN=0
  fi

  if [ -z "$SIGN_IDENTITY" ]; then
    echo "No usable codesign identity found for mac app signing" >&2
    exit 1
  fi

  echo "Using integration app identity: $SIGN_IDENTITY"
}

codesign_path() {
  target_path="$1"
  entitlements_path="$2"

  if [ -n "$entitlements_path" ]; then
    if [ "$SIGN_USE_KEYCHAIN" -eq 1 ]; then
      codesign --force --sign "$SIGN_IDENTITY" --keychain "$SIGN_KEYCHAIN_PATH" --entitlements "$entitlements_path" "$target_path"
    else
      codesign --force --sign "$SIGN_IDENTITY" --entitlements "$entitlements_path" "$target_path"
    fi
    return 0
  fi

  if [ "$SIGN_USE_KEYCHAIN" -eq 1 ]; then
    codesign --force --sign "$SIGN_IDENTITY" --keychain "$SIGN_KEYCHAIN_PATH" "$target_path"
  else
    codesign --force --sign "$SIGN_IDENTITY" "$target_path"
  fi
}

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
  find "$dst_path" \( -name '.DS_Store' -o -name '._*' \) -exec rm -rf {} +
}

build_mac_app_icon() {
  src_png="$1"
  out_icns="$2"

  if [ ! -f "$src_png" ]; then
    echo "missing mac app icon source: $src_png" >&2
    exit 1
  fi
  if ! command -v sips >/dev/null 2>&1; then
    echo "sips is required to build the mac app icon" >&2
    exit 1
  fi
  if ! command -v iconutil >/dev/null 2>&1; then
    echo "iconutil is required to build the mac app icon" >&2
    exit 1
  fi

  width="$(sips -g pixelWidth "$src_png" 2>/dev/null | awk '/pixelWidth:/ {print $2}')"
  height="$(sips -g pixelHeight "$src_png" 2>/dev/null | awk '/pixelHeight:/ {print $2}')"
  if [ -z "$width" ] || [ -z "$height" ]; then
    echo "failed to read icon source dimensions: $src_png" >&2
    exit 1
  fi

  crop_size="$width"
  if [ "$height" -lt "$crop_size" ]; then
    crop_size="$height"
  fi

  icon_tmp_dir="$(mktemp -d "$BUILD_TMP_DIR/mac-icon.XXXXXX")"
  iconset_dir="$icon_tmp_dir/AppIcon.iconset"
  cropped_png="$icon_tmp_dir/AppIcon-square.png"
  mkdir -p "$iconset_dir"

  sips -c "$crop_size" "$crop_size" "$src_png" --out "$cropped_png" >/dev/null
  for size in 16 32 128 256 512; do
    retina_size=$((size * 2))
    sips -z "$size" "$size" "$cropped_png" --out "$iconset_dir/icon_${size}x${size}.png" >/dev/null
    sips -z "$retina_size" "$retina_size" "$cropped_png" --out "$iconset_dir/icon_${size}x${size}@2x.png" >/dev/null
  done

  mkdir -p "$(dirname "$out_icns")"
  iconutil -c icns "$iconset_dir" -o "$out_icns"
}

build_windows_app_icon() {
  src_png="$1"
  out_ico="$2"
  fallback_ico="$MODULE_DIR/build/DeepRight.ico"

  if [ ! -f "$src_png" ]; then
    echo "missing windows app icon source: $src_png" >&2
    exit 1
  fi

  mkdir -p "$(dirname "$out_ico")"

  if command -v magick >/dev/null 2>&1; then
    magick "$src_png" -define icon:auto-resize=256,128,64,48,32,16 "$out_ico"
    return 0
  fi

  if command -v python3 >/dev/null 2>&1; then
    if SRC_PNG="$src_png" OUT_ICO="$out_ico" python3 - <<'PY'
from pathlib import Path
import os
import sys

try:
    from PIL import Image
except Exception:
    sys.exit(1)

src = Path(os.environ["SRC_PNG"])
out = Path(os.environ["OUT_ICO"])
img = Image.open(src).convert("RGBA")
img.save(out, format="ICO", sizes=[(256, 256), (128, 128), (64, 64), (48, 48), (32, 32), (16, 16)])
PY
    then
      return 0
    fi
  fi

  if [ -f "$fallback_ico" ]; then
    copy_release_asset "$fallback_ico" "$out_ico"
    return 0
  fi

  echo "failed to build windows app icon: install ImageMagick or python3+Pillow, or provide $fallback_ico" >&2
  exit 1
}

write_mac_dmg_background_svg() {
  svg_path="$1"
  target_name="$2"

  cat > "$svg_path" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="800" height="480" viewBox="0 0 800 480">
  <defs>
    <linearGradient id="bg" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#07111d"/>
      <stop offset="50%" stop-color="#0d1b2f"/>
      <stop offset="100%" stop-color="#08121f"/>
    </linearGradient>
    <linearGradient id="lineGlow" x1="0%" y1="50%" x2="100%" y2="50%">
      <stop offset="0%" stop-color="#66dfff" stop-opacity="0.25"/>
      <stop offset="50%" stop-color="#9af6df" stop-opacity="0.9"/>
      <stop offset="100%" stop-color="#66dfff" stop-opacity="0.25"/>
    </linearGradient>
  </defs>

  <rect width="800" height="480" fill="url(#bg)"/>
  <circle cx="164" cy="276" r="118" fill="#2ad6c4" fill-opacity="0.13"/>
  <circle cx="634" cy="278" r="126" fill="#58afff" fill-opacity="0.13"/>
  <circle cx="708" cy="92" r="144" fill="#0e79f5" fill-opacity="0.14"/>
  <circle cx="108" cy="100" r="120" fill="#1ad8d0" fill-opacity="0.10"/>
  <rect x="74" y="148" width="180" height="180" rx="36" fill="#ffffff" fill-opacity="0.03" stroke="#9ee9ff" stroke-opacity="0.12"/>
  <rect x="548" y="148" width="180" height="180" rx="36" fill="#ffffff" fill-opacity="0.03" stroke="#9ee9ff" stroke-opacity="0.12"/>

  <g transform="translate(294 200)">
    <path d="M0 38 H128" fill="none" stroke="url(#lineGlow)" stroke-width="9" stroke-linecap="round" stroke-dasharray="12 12"/>
    <path d="M98 6 L146 38 L98 70" fill="none" stroke="#a0f6e0" stroke-width="9" stroke-linecap="round" stroke-linejoin="round"/>
  </g>

  <text x="60" y="78" fill="#f4fbff" font-size="38" font-weight="700" font-family="Helvetica">Install ${MAC_APP_NAME}</text>
  <text x="60" y="112" fill="#93b8d9" font-size="17" font-family="Helvetica">Drag the large app icon into Applications</text>

  <text x="164" y="390" fill="#d9f7ff" font-size="26" font-weight="600" text-anchor="middle" font-family="Helvetica">${MAC_APP_NAME}</text>
  <text x="164" y="418" fill="#7eb7cf" font-size="14" text-anchor="middle" font-family="Helvetica">${target_name} build</text>

  <text x="638" y="390" fill="#d9f7ff" font-size="26" font-weight="600" text-anchor="middle" font-family="Helvetica">Applications</text>
  <text x="638" y="418" fill="#7eb7cf" font-size="14" text-anchor="middle" font-family="Helvetica">Drop here to install</text>
</svg>
EOF
}

render_mac_dmg_background() {
  target_name="$1"
  out_png="$2"
  bg_tmp_dir="$BUILD_TMP_DIR/dmg-background-$target_name"
  svg_path="$bg_tmp_dir/install-background.svg"
  rendered_png="$bg_tmp_dir/install-background.svg.png"

  rm -rf "$bg_tmp_dir"
  mkdir -p "$bg_tmp_dir"
  write_mac_dmg_background_svg "$svg_path" "$target_name"

  if ! qlmanage -t -s 800 -o "$bg_tmp_dir" "$svg_path" >/dev/null 2>&1; then
    return 1
  fi
  if [ ! -f "$rendered_png" ]; then
    return 1
  fi

  mkdir -p "$(dirname "$out_png")"
  mv "$rendered_png" "$out_png"
}

create_plain_mac_dmg() {
  dmg_volume_name="$1"
  dmg_stage_dir="$2"
  dmg_path="$3"

  rm -f "$dmg_path"
  hdiutil create \
    -volname "$dmg_volume_name" \
    -srcfolder "$dmg_stage_dir" \
    -ov \
    -format UDZO \
    "$dmg_path" >/dev/null
}

run_with_timeout() {
  timeout_seconds="$1"
  shift

  "$@" &
  timeout_target_pid=$!
  (
    sleep "$timeout_seconds"
    kill "$timeout_target_pid" >/dev/null 2>&1 || exit 0
    sleep 1
    kill -9 "$timeout_target_pid" >/dev/null 2>&1 || true
  ) &
  timeout_watchdog_pid=$!

  wait "$timeout_target_pid"
  timeout_status=$?
  kill "$timeout_watchdog_pid" >/dev/null 2>&1 || true
  wait "$timeout_watchdog_pid" 2>/dev/null || true
  return "$timeout_status"
}

style_mac_dmg_window() {
  mounted_volume_name="$1"
  applescript_slug=$(printf '%s' "$mounted_volume_name" | tr ' /' '__')
  applescript_path="$BUILD_TMP_DIR/dmg-style-${applescript_slug}.applescript"

  cat > "$applescript_path" <<EOF
tell application "Finder"
  tell disk "${mounted_volume_name}"
    open
    delay 0.5
    set current view of container window to icon view
    set toolbar visible of container window to false
    set statusbar visible of container window to false
    set bounds of container window to {150, 130, 950, 610}

    set viewOptions to the icon view options of container window
    set arrangement of viewOptions to not arranged
    set icon size of viewOptions to 160
    set text size of viewOptions to 14
    set background picture of viewOptions to file ".background:install-background.png"

    set position of item "${MAC_APP_NAME}.app" of container window to {164, 246}
    set position of item "Applications" of container window to {636, 246}

    update without registering applications
    delay 1
    close
  end tell
end tell
EOF

  run_with_timeout 20 osascript "$applescript_path"
}

create_styled_mac_dmg() {
  target_name="$1"
  dmg_volume_name="$2"
  app_dir="$3"
  dmg_path="$4"
  dmg_stage_dir="$5"
  rw_dmg_path="$BUILD_TMP_DIR/${MAC_APP_NAME}-${target_name}.rw.dmg"
  compressed_dmg_root="$BUILD_TMP_DIR/${MAC_APP_NAME}-${target_name}.styled"
  compressed_dmg_path="${compressed_dmg_root}.dmg"
  dmg_mount_dir=""
  dmg_background_path="$dmg_stage_dir/.background/install-background.png"
  stage_size_kb=""
  dmg_size_mb=0
  styled_dmg_attached=0
  attach_output=""
  mounted_volume_name=""

  cleanup_styled_mac_dmg() {
    if [ "$styled_dmg_attached" -eq 1 ] && [ -n "$dmg_mount_dir" ]; then
      hdiutil detach "$dmg_mount_dir" -quiet >/dev/null 2>&1 || true
    fi
  }

  trap cleanup_styled_mac_dmg EXIT HUP INT TERM

  if ! command -v osascript >/dev/null 2>&1; then
    cleanup_styled_mac_dmg
    trap - EXIT HUP INT TERM
    return 1
  fi
  if ! command -v qlmanage >/dev/null 2>&1; then
    cleanup_styled_mac_dmg
    trap - EXIT HUP INT TERM
    return 1
  fi

  rm -rf "$dmg_stage_dir"
  mkdir -p "$dmg_stage_dir/.background"
  copy_release_asset "$app_dir" "$dmg_stage_dir/${MAC_APP_NAME}.app"
  ln -s /Applications "$dmg_stage_dir/Applications"
  if ! render_mac_dmg_background "$target_name" "$dmg_background_path"; then
    cleanup_styled_mac_dmg
    trap - EXIT HUP INT TERM
    return 1
  fi

  stage_size_kb="$(du -sk "$dmg_stage_dir" | awk '{print $1}')"
  if [ -z "$stage_size_kb" ]; then
    cleanup_styled_mac_dmg
    trap - EXIT HUP INT TERM
    return 1
  fi
  dmg_size_mb=$((stage_size_kb / 1024 + 64))
  if [ "$dmg_size_mb" -lt 120 ]; then
    dmg_size_mb=120
  fi

  rm -f "$rw_dmg_path" "$compressed_dmg_path" "$dmg_path"
  if ! hdiutil create \
    -volname "$dmg_volume_name" \
    -fs HFS+ \
    -size "${dmg_size_mb}m" \
    -srcfolder "$dmg_stage_dir" \
    -ov \
    -format UDRW \
    "$rw_dmg_path" >/dev/null; then
    cleanup_styled_mac_dmg
    trap - EXIT HUP INT TERM
    return 1
  fi
  attach_output="$(hdiutil attach \
    "$rw_dmg_path" \
    -readwrite \
    -noverify \
    -noautoopen \
    -nobrowse)" || {
    cleanup_styled_mac_dmg
    trap - EXIT HUP INT TERM
    return 1
  }
  dmg_mount_dir="$(printf '%s\n' "$attach_output" | sed -n 's#.*\(/Volumes/.*\)$#\1#p' | tail -n 1)"
  if [ -z "$dmg_mount_dir" ] || [ ! -d "$dmg_mount_dir" ]; then
    cleanup_styled_mac_dmg
    trap - EXIT HUP INT TERM
    return 1
  fi
  mounted_volume_name="$(basename "$dmg_mount_dir")"
  styled_dmg_attached=1

  if ! style_mac_dmg_window "$mounted_volume_name"; then
    cleanup_styled_mac_dmg
    trap - EXIT HUP INT TERM
    return 1
  fi

  sync
  if ! hdiutil detach "$dmg_mount_dir" -quiet >/dev/null; then
    cleanup_styled_mac_dmg
    trap - EXIT HUP INT TERM
    return 1
  fi
  styled_dmg_attached=0

  if ! hdiutil convert \
    "$rw_dmg_path" \
    -ov \
    -format UDZO \
    -imagekey zlib-level=9 \
    -o "$compressed_dmg_root" >/dev/null; then
    cleanup_styled_mac_dmg
    trap - EXIT HUP INT TERM
    return 1
  fi
  mv "$compressed_dmg_path" "$dmg_path"
  rm -f "$rw_dmg_path"

  trap - EXIT HUP INT TERM
}

create_mac_dmg() {
  target_name="$1"
  target_release_dir="$RELEASE_DIR/mac/$target_name"
  app_dir="$target_release_dir/${MAC_APP_NAME}.app"
  dmg_path="$target_release_dir/${MAC_APP_NAME}-${target_name}.dmg"
  dmg_stage_dir="$BUILD_TMP_DIR/dmg-stage-$target_name"
  dmg_volume_name="${MAC_APP_NAME} ${target_name}"

  if [ ! -d "$app_dir" ]; then
    echo "missing mac app for dmg packaging (mac/$target_name): $app_dir" >&2
    exit 1
  fi
  if ! command -v hdiutil >/dev/null 2>&1; then
    echo "hdiutil is required to build the mac dmg" >&2
    exit 1
  fi

  echo "-> packaging ${MAC_APP_NAME}-${target_name}.dmg (mac/$target_name)"
  if create_styled_mac_dmg "$target_name" "$dmg_volume_name" "$app_dir" "$dmg_path" "$dmg_stage_dir"; then
    return 0
  fi

  echo "warning: styled dmg packaging unavailable, falling back to plain layout (mac/$target_name)" >&2
  rm -rf "$dmg_stage_dir"
  mkdir -p "$dmg_stage_dir"
  copy_release_asset "$app_dir" "$dmg_stage_dir/${MAC_APP_NAME}.app"
  ln -s /Applications "$dmg_stage_dir/Applications"
  create_plain_mac_dmg "$dmg_volume_name" "$dmg_stage_dir" "$dmg_path"
}

cleanup_release_artifacts() {
  plugins_dir="$1"
  rm -rf \
    "$plugins_dir/.connect-plugin-cache.json" \
    "$plugins_dir/.browser_playwright" \
    "$plugins_dir/.remote" \
    "$plugins_dir/.browser_cookie.cache.json" \
    "$plugins_dir/browser_cookie.json" \
    "$plugins_dir/browser_instance.json" \
    "$plugins_dir/browser.log" \
    "$plugins_dir/browser.pid" \
    "$plugins_dir/data" \
    "$plugins_dir/data-shm" \
    "$plugins_dir/data-wal" \
    "$plugins_dir/email.state.json" \
    "$plugins_dir/email_artifacts" \
    "$plugins_dir/feishu.pending.json" \
    "$plugins_dir/feishu_artifacts" \
    "$plugins_dir/playwright" \
    "$plugins_dir/plugins" \
    "$plugins_dir/remote.json"
  rm -f \
    "$plugins_dir"/obscura/release/mac/*.tar.gz \
    "$plugins_dir"/obscura/release/linux/*.tar.gz
}

cleanup_workspace_binaries() {
  echo "-> cleaning workspace build artifacts"
  rm -rf \
    "$CONNECT_DIR/remote/release"
  if [ -d "$CONNECT_DIR/plugins" ]; then
    find "$CONNECT_DIR/plugins" -mindepth 1 ! -name '.gitkeep' -exec rm -rf {} +
  fi
  rm -f \
    "$CONNECT_DIR/browser/instance/browser_instance" \
    "$CONNECT_DIR/browser/playwright/browser_playwright" \
    "$CONNECT_DIR/connect" \
    "$CONNECT_DIR/email/email" \
    "$CONNECT_DIR/feishu/bin-feishu" \
    "$CONNECT_DIR/feishu/feishu" \
    "$CONNECT_DIR/playwright" \
    "$MODULE_DIR/cron/cron" \
    "$INTEGRATION_DIR/integration" \
    "$MODULE_DIR/proxy/proxy" \
    "$MODULE_DIR/skills/skill-scanner" \
    "$MODULE_DIR/static/static-server"
}

cleanup_intermediate_build_artifacts() {
  echo "-> cleaning intermediate build artifacts"
  find "$MODULE_DIR" -type d -name 'release' ! -path "$RELEASE_DIR" -prune -exec rm -rf {} +
  if [ -d "$RELEASE_DIR" ]; then
    find "$RELEASE_DIR" \( -name '.DS_Store' -o -name '._*' \) -exec rm -rf {} +
  fi
}

reset_release_dir() {
  mkdir -p "$RELEASE_DIR"
  # Preserve runtime state so repeated builds do not wipe the shared sqlite database.
  find "$RELEASE_DIR" -mindepth 1 -maxdepth 1 \
    ! -name 'data' \
    ! -name 'data-shm' \
    ! -name 'data-wal' \
    ! -name 'mac' \
    ! -name 'linux' \
    -exec rm -rf {} +
  rm -rf "$RELEASE_DIR/x86" "$RELEASE_DIR/arm"
}

reset_target_release_dir() {
  target_release_dir="$1"

  mkdir -p "$target_release_dir"
  find "$target_release_dir" -mindepth 1 -maxdepth 1 \
    ! -name 'data' \
    ! -name 'data-shm' \
    ! -name 'data-wal' \
    -exec rm -rf {} +
  mkdir -p "$target_release_dir/plugins" "$target_release_dir/site"
}

verify_browser_release_contract() {
  echo "-> verifying browser plugin contract"
  (
    cd "$CONNECT_DIR"
    "$GO_BIN" test ./browserplaywrightsvc ./browser
  )
}

should_skip_release_asset() {
  plugin_name="$1"
  asset_name="$2"
  case "$asset_name" in
    "$plugin_name"|*.log|*.pid|*.state.json|*.pending.json|.connect-plugin-cache.json|.browser_playwright|.remote|browser_instance.json|data|data-shm|data-wal|email_artifacts|feishu_artifacts|playwright|remote.json)
      return 0
      ;;
  esac
  return 1
}

build_plugin() {
  plugin_dir="$1"
  plugins_dir="$2"
  target_goos="$3"
  target_goarch="$4"
  plugin_name=$(basename "$plugin_dir")
  plugin_binary="$plugins_dir/$plugin_name"
  plugin_release_dir="$plugin_dir/release"

  if [ ! -f "$plugin_dir/main.go" ]; then
    return 0
  fi

  if ! plugin_selected_for_release "$plugin_name"; then
    echo "-> skipping plugin: $plugin_name (DEEPRIGHT_RELEASE_PLUGINS)"
    return 0
  fi

  echo "-> building plugin: $plugin_name ($target_goos/$target_goarch)"
  (
    cd "$CONNECT_DIR"
    GOOS="$target_goos" GOARCH="$target_goarch" "$GO_BIN" build -trimpath -ldflags "$GO_RELEASE_LDFLAGS" -o "$plugin_binary" "./$plugin_name"
  )

  if [ -d "$plugin_release_dir" ]; then
    echo "-> copying plugin runtime assets: $plugin_name"
    find "$plugin_release_dir" -mindepth 1 -maxdepth 1 ! -name '.*' | while IFS= read -r asset_path; do
      asset_name=$(basename "$asset_path")
      if should_skip_release_asset "$plugin_name" "$asset_name"; then
        continue
      fi
      copy_release_asset "$asset_path" "$plugins_dir/$asset_name"
    done
  fi

  if [ "$plugin_name" = "browser" ] && [ "$target_goos" = "linux" ] && [ -f "$plugin_dir/instance/browser_launcher.sh" ]; then
    echo "-> copying browser launcher script"
    copy_release_asset "$plugin_dir/instance/browser_launcher.sh" "$plugins_dir/browser_launcher.sh"
    chmod 755 "$plugins_dir/browser_launcher.sh"
  fi
}

build_target_release() {
  target_dir_name="$1"
  target_goos="$2"
  target_name="$3"
  target_goarch="$4"
  target_release_dir="$RELEASE_DIR/$target_dir_name/$target_name"
  target_plugins_dir="$target_release_dir/plugins"
  target_site_dir="$target_release_dir/site"

  echo "-> resetting ${target_dir_name}/${target_name} release directory"
  reset_target_release_dir "$target_release_dir"

  echo "-> building integration ($target_goos/$target_goarch)"
  (
    cd "$INTEGRATION_DIR"
    GOOS="$target_goos" GOARCH="$target_goarch" "$GO_BIN" build -trimpath -ldflags "$GO_RELEASE_LDFLAGS" -o "$target_release_dir/integration" ./
  )

  find "$CONNECT_DIR" -mindepth 1 -maxdepth 1 -type d ! -name '.*' | sort | while IFS= read -r plugin_dir; do
    build_plugin "$plugin_dir" "$target_plugins_dir" "$target_goos" "$target_goarch"
  done
  cleanup_release_artifacts "$target_plugins_dir"

  echo "-> copying site assets ($target_dir_name/$target_name)"
  cp "$SITE_DIR/index.html" "$target_site_dir/index.html"
  cp "$SITE_DIR/icon.png" "$target_site_dir/icon.png"
  cp "$SITE_DIR/sw.js" "$target_site_dir/sw.js"

  echo "-> copying config assets ($target_dir_name/$target_name)"
  copy_release_asset "$CONFIG_DIR" "$target_release_dir/config"
}

build_cli_sandbox_mac_release() {
  echo "-> building CLI_SANDBOX mac release artifacts"
  (
    cd "$CLI_GET_SANDBOX_DIR"
    DEEPRIGHT_SKIP_SIGN=1 ./build.sh
  )
}

build_cli_sandbox_wsl_release() {
  echo "-> building CLI_SANDBOX WSL release artifacts"
  (
    cd "$CLI_GET_SANDBOX_DIR/wsl"
    ./build.sh
  )
}

package_linux_sandbox() {
  target_name="$1"
  target_release_dir="$RELEASE_DIR/linux/$target_name"
  helpers_dir="$target_release_dir/helpers"

  echo "-> packaging CLI_SANDBOX binaries (linux/$target_name)"
  mkdir -p "$helpers_dir"
  for sandbox_mode in filepick net filepick_net; do
    sandbox_bin="$CLI_GET_SANDBOX_WSL_RELEASE_DIR/$target_name/$sandbox_mode/CLI_SANDBOX"
    if [ ! -f "$sandbox_bin" ]; then
      echo "missing sandbox binary for linux/$target_name mode=$sandbox_mode: $sandbox_bin" >&2
      exit 1
    fi
    copy_release_asset "$sandbox_bin" "$helpers_dir/$sandbox_mode/CLI_SANDBOX"
    chmod 755 "$helpers_dir/$sandbox_mode/CLI_SANDBOX"
  done
}

windows_wsl_rootfs_url_for_target() {
  target_name="$1"
  case "$target_name" in
    x86)
      printf '%s' "https://cloud-images.ubuntu.com/wsl/releases/noble/current/ubuntu-noble-wsl-amd64-wsl.rootfs.tar.gz"
      ;;
    arm)
      printf '%s' "https://cloud-images.ubuntu.com/wsl/releases/noble/current/ubuntu-noble-wsl-arm64-wsl.rootfs.tar.gz"
      ;;
    *)
      echo "unknown linux target for windows rootfs url: $target_name" >&2
      exit 1
      ;;
  esac
}

package_windows_wsl2_launcher() {
  target_name="$1"
  target_release_dir="$RELEASE_DIR/linux/$target_name"
  rootfs_url=""

  echo "-> packaging Windows WSL2 launcher (linux/$target_name)"
  rootfs_url="$(windows_wsl_rootfs_url_for_target "$target_name")"
  copy_release_asset "$MODULE_DIR/build/install.bat" "$target_release_dir/install.bat"
  copy_release_asset "$MODULE_DIR/build/start.bat" "$target_release_dir/start.bat"
  copy_release_asset "$MODULE_DIR/build/install.ps1" "$target_release_dir/install.ps1"
  build_windows_app_icon "$SITE_DIR/icon_white_bg.png" "$target_release_dir/DeepRight.ico"
  rm -f "$target_release_dir/install-wsl2.ps1" "$target_release_dir/install-wsl2.cmd"
  sed -i.bak \
    -e 's|^\$APP_DIR       = Join-Path \$PSScriptRoot "app"$|$APP_DIR       = $PSScriptRoot|' \
    -e "s|https://cloud-images.ubuntu.com/wsl/releases/noble/current/ubuntu-noble-wsl-amd64-wsl.rootfs.tar.gz|$rootfs_url|g" \
    "$target_release_dir/install.ps1"
  rm -f "$target_release_dir/install.ps1.bak"
}

write_integration_info_plist() {
  plist_path="$1"
  cat > "$plist_path" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>en</string>
  <key>CFBundleExecutable</key>
  <string>integration</string>
  <key>CFBundleIdentifier</key>
  <string>cn.deepright.integration</string>
  <key>CFBundleIconFile</key>
  <string>AppIcon</string>
  <key>CFBundleDisplayName</key>
  <string>${MAC_APP_NAME}</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>${MAC_APP_NAME}</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>1.0.0</string>
  <key>CFBundleVersion</key>
  <string>1</string>
</dict>
</plist>
EOF
}

package_mac_app() {
  target_name="$1"
  target_release_dir="$RELEASE_DIR/mac/$target_name"
  app_dir="$target_release_dir/${MAC_APP_NAME}.app"
  contents_dir="$app_dir/Contents"
  macos_dir="$contents_dir/MacOS"
  helpers_dir="$contents_dir/Helpers"
  resources_dir="$contents_dir/Resources"

  echo "-> packaging ${MAC_APP_NAME}.app (mac/$target_name)"
  rm -rf "$app_dir"
  mkdir -p "$macos_dir" "$helpers_dir" "$resources_dir"

  copy_release_asset "$target_release_dir/integration" "$macos_dir/integration"
  copy_release_asset "$target_release_dir/plugins" "$resources_dir/plugins"
  copy_release_asset "$target_release_dir/site" "$resources_dir/site"
  copy_release_asset "$target_release_dir/config" "$resources_dir/config"
  for sandbox_mode in filepick net filepick_net; do
    sandbox_app="$CLI_GET_SANDBOX_MAC_RELEASE_DIR/$target_name/$sandbox_mode/CLI_SANDBOX.app"
    if [ ! -d "$sandbox_app" ]; then
      echo "missing sandbox app for mac/$target_name mode=$sandbox_mode: $sandbox_app" >&2
      exit 1
    fi
    copy_release_asset "$sandbox_app" "$helpers_dir/$sandbox_mode/CLI_SANDBOX.app"
  done
  build_mac_app_icon "$SITE_DIR/icon_white_bg.png" "$resources_dir/AppIcon.icns"
  write_integration_info_plist "$contents_dir/Info.plist"

  if [ "$SIGN_ENABLED" -eq 1 ]; then
    echo "-> signing ${MAC_APP_NAME}.app (mac/$target_name)"
    for sandbox_mode in filepick net filepick_net; do
      sandbox_build_dir="$CLI_GET_SANDBOX_MAC_RELEASE_DIR/$target_name/$sandbox_mode/CLI_SANDBOX-build"
      codesign_path \
        "$helpers_dir/$sandbox_mode/CLI_SANDBOX.app/Contents/Helpers/CLI_SANDBOX" \
        "$sandbox_build_dir/inherit.entitlements.plist"
      codesign_path \
        "$helpers_dir/$sandbox_mode/CLI_SANDBOX.app" \
        "$sandbox_build_dir/app.entitlements.plist"
    done
    codesign_path "$app_dir" ""
    codesign --verify --deep --strict --verbose=2 "$app_dir"
  fi

  rm -rf \
    "$target_release_dir/data" \
    "$target_release_dir/data-shm" \
    "$target_release_dir/data-wal" \
    "$target_release_dir/integration" \
    "$target_release_dir/plugins" \
    "$target_release_dir/site" \
    "$target_release_dir/config"
}

build_linux_release_artifacts() {
  build_target_release "linux" "linux" "x86" "amd64"
  build_target_release "linux" "linux" "arm" "arm64"
  build_cli_sandbox_wsl_release
  package_linux_sandbox "x86"
  package_linux_sandbox "arm"
  package_windows_wsl2_launcher "x86"
  package_windows_wsl2_launcher "arm"
}

build_mac_release_artifacts() {
  build_target_release "mac" "darwin" "x86" "amd64"
  build_target_release "mac" "darwin" "arm" "arm64"
  build_cli_sandbox_mac_release
  package_mac_app "x86"
  create_mac_dmg "x86"
  package_mac_app "arm"
  create_mac_dmg "arm"
}

echo "Building integration release artifacts..."

if [ "$BUILD_MAC" -eq 1 ]; then
  setup_codesign_identity
fi

echo "-> resetting release directory"
reset_release_dir

if [ "$BUILD_LINUX" -eq 1 ]; then
  build_linux_release_artifacts
fi

if [ "$BUILD_MAC" -eq 1 ]; then
  build_mac_release_artifacts
fi

verify_browser_release_contract
rm -rf "$RELEASE_DIR/x86" "$RELEASE_DIR/arm"
cleanup_intermediate_build_artifacts
cleanup_workspace_binaries

echo "Build completed:"
find "$RELEASE_DIR" -mindepth 1 | sort | while IFS= read -r path; do
  echo "  $path"
done
