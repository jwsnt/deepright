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

write_windows_wsl2_launcher_ps1() {
  out_path="$1"
  expected_uname="$2"
  package_label="$3"

  cat > "$out_path" <<'__DEEPRIGHT_WSL2_POWERSHELL__'
#Requires -Version 5.1

[CmdletBinding()]
param(
  [string]$DistroName = "deepright",
  [switch]$SkipLaunch
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

# ---------- Fixed config ----------
$DISTRO_NAME   = $DistroName
$WSL_VHD_PATH  = Join-Path "C:\WSL" $DISTRO_NAME
$BUILD_DIR     = $PSScriptRoot
$LOG_FILE      = Join-Path $PSScriptRoot "install.log"
$WSL_SENTINEL  = "/home/deepright/.deepright_initialized"
$LOCAL_SENTINEL_DIR  = "C:\ProgramData\deepright"
$LOCAL_SENTINEL_FILE = Join-Path $LOCAL_SENTINEL_DIR ".deepright_installed"
$EXPECTED_WSL_ARCH = "__DEEPRIGHT_EXPECTED_UNAME__"
$PACKAGE_LABEL = "__DEEPRIGHT_PACKAGE_LABEL__"

# ---------- Log ----------
function L_Step($m) { $l="`n========================================  $m"; Write-Host $l -F Cyan;   Add-Content -Path $LOG_FILE -Value $l -Encoding UTF8 }
function L_OK($m)   {                     Write-Host "  [OK] $m" -F Green;  Add-Content -Path $LOG_FILE -Value "  [OK] $m" -Encoding UTF8 }
function L_Warn($m) {                     Write-Host "  [!] $m" -F Yellow;  Add-Content -Path $LOG_FILE -Value "  [WARN] $m" -Encoding UTF8 }
function L_Err($m)  {                     Write-Host "  [X] $m" -F Red;     Add-Content -Path $LOG_FILE -Value "  [ERROR] $m" -Encoding UTF8 }
function L_Info($m) {                     Write-Host "  [i] $m" -F Gray;    Add-Content -Path $LOG_FILE -Value "  [INFO] $m" -Encoding UTF8 }

# ---------- Helpers ----------
function Test-Admin {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    $p  = New-Object Security.Principal.WindowsPrincipal($id)
    return $p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}
function Get-WindowsBuild {
    return [int](Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion").CurrentBuild
}
function Test-WslInstalled {
    try { $null = & wsl.exe --status 2>&1; return ($LASTEXITCODE -eq 0) } catch { return $false }
}

function Test-DistroExists([string]$N) {
    for ($i = 1; $i -le 3; $i++) {
        $out = & wsl.exe -d $N -u root -- echo "ok" 2>&1 | Out-String
        if ($out -match "ok") {
            return $true
        }
        if ($i -lt 3) {
            Start-Sleep -Seconds 3
        }
    }
    return $false
}

function WslPath([string]$P) {
    $d = $P.Substring(0,1).ToLower()
    $r = $P.Substring(2) -replace '\\', '/'
    return "/mnt/$d$r"
}

function Nuke-Distro([string]$Name, [string]$VhdPath) {
    & wsl.exe --shutdown 2>&1 | Out-Null
    Start-Sleep -Seconds 2

    $out1 = & wsl.exe --unregister $Name 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "unregister $Name : $out1" -Encoding UTF8
    Start-Sleep -Seconds 2

    $out2 = & wsl.exe --unregister $Name 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "unregister $Name (2nd): $out2" -Encoding UTF8
    Start-Sleep -Seconds 1

    if (Test-Path $VhdPath) {
        Get-ChildItem -Path $VhdPath -Recurse -Force | Remove-Item -Force -EA SilentlyContinue
    } else {
        New-Item -ItemType Directory -Path $VhdPath -Force | Out-Null
    }
}

function Fix-WslConfig([string]$Path) {
    $content = "[wsl2]`r`nnetworkingMode=mirrored`r`n"
    [System.IO.File]::WriteAllText($Path, $content, [System.Text.Encoding]::ASCII)
}

function Test-WslTool([string]$cmd) {
    & wsl.exe -d $DISTRO_NAME -- bash -c "command -v $cmd > /dev/null 2>&1" | Out-Null
    return ($LASTEXITCODE -eq 0)
}

function Get-RootfsUrl {
    switch ($EXPECTED_WSL_ARCH) {
        "aarch64" { return "https://cloud-images.ubuntu.com/wsl/releases/noble/current/ubuntu-noble-wsl-arm64-wsl.rootfs.tar.gz" }
        default   { return "https://cloud-images.ubuntu.com/wsl/releases/noble/current/ubuntu-noble-wsl-amd64-wsl.rootfs.tar.gz" }
    }
}

function Assert-WSLArchitecture {
    $arch = (& wsl.exe -d $DISTRO_NAME -- bash -c "uname -m 2>&1" | Out-String).Trim()
    if ($arch -ne $EXPECTED_WSL_ARCH) {
        L_Err "The Ubuntu architecture is $arch, but this package requires $EXPECTED_WSL_ARCH ($PACKAGE_LABEL)"
        exit 1
    }
}

# ---------- Main ----------
$runHeader = "`n========================================  NEW RUN: $(Get-Date -F 'yyyy-MM-dd HH:mm:ss')"
Add-Content -Path $LOG_FILE -Value $runHeader -Encoding UTF8

L_Step "Deepright WSL2 Environment Installer"
L_Info "Script dir: $PSScriptRoot"
L_Info "Log file: $LOG_FILE"

$needsFullInstall = $true

if (Test-Path $LOCAL_SENTINEL_FILE) {
    if (Test-DistroExists -N $DISTRO_NAME) {
        $needsFullInstall = $false
        L_OK "Sentinel found + distro alive -- skipping installation"
        L_OK "To force re-install, delete: $LOCAL_SENTINEL_DIR"
    } else {
        L_Warn "Sentinel found but distro is missing (unregistered/deleted)"
        L_Warn "Deleting stale sentinel, re-installing..."
        Remove-Item $LOCAL_SENTINEL_DIR -Recurse -Force -EA SilentlyContinue
    }
}

if ($needsFullInstall) {

L_Step "Step 1/7: Check administrator privileges"
if (-not (Test-Admin)) {
    L_Err "This script requires Administrator privileges"
    L_Info "Launch via install.bat (auto-elevates)"
    exit 1
}
L_OK "Administrator confirmed"

L_Step "Step 2/7: Check Windows version"
$build = Get-WindowsBuild
L_Info "Windows Build: $build"
if ($build -lt 19041) {
    L_Err "WSL2 requires Windows 10 2004+ (Build 19041+)"
    exit 1
}
L_OK "Windows version OK (Build $build)"

L_Step "Step 3/7: Check WSL2"
if (Test-WslInstalled) {
    L_OK "WSL already installed"
} else {
    L_Info "Installing WSL (may take several minutes)..."
    $o = & wsl.exe --install --no-distribution 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "WSL install: $o" -Encoding UTF8
    if (-not (Test-WslInstalled)) {
        L_Info "Trying DISM fallback..."
        $d1 = & dism.exe /online /enable-feature /featurename:Microsoft-Windows-Subsystem-Linux /all /norestart 2>&1 | Out-String
        $d2 = & dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart 2>&1 | Out-String
        Add-Content -Path $LOG_FILE -Value "DISM1: $d1`nDISM2: $d2" -Encoding UTF8
        if (-not (Test-WslInstalled)) {
            L_Warn "WSL features enabled. REBOOT REQUIRED, then re-run install.bat"
            exit 0
        }
    }
    L_OK "WSL2 installation complete"
}
$null = & wsl.exe --set-default-version 2 2>&1

L_Step "Step 4/7: Install Ubuntu (deepright)"

L_Info "Writing clean .wslconfig (ASCII, mirrored only)..."
$wcPath = Join-Path $env:USERPROFILE ".wslconfig"
Fix-WslConfig -Path $wcPath
L_OK ".wslconfig written"

L_Info "Updating WSL..."
$wu = & wsl.exe --update 2>&1 | Out-String
Add-Content -Path $LOG_FILE -Value "wsl --update: $wu" -Encoding UTF8
L_OK "WSL updated"

L_Info "Shutting down WSL..."
& wsl.exe --shutdown 2>&1 | Out-Null
Start-Sleep -Seconds 3
L_OK "WSL restarted with clean config"

$distroExists = Test-DistroExists -N $DISTRO_NAME

if ($distroExists) {
    L_OK "Distro $DISTRO_NAME already exists"

    $cu = & wsl.exe -d $DISTRO_NAME -u root -- id -u deepright 2>&1 | Out-String
    if ($LASTEXITCODE -ne 0) {
        L_Info "Creating user deepright..."
        & wsl.exe -d $DISTRO_NAME -u root -- useradd -m -s /bin/bash deepright 2>&1 | Out-Null
        & wsl.exe -d $DISTRO_NAME -u root -- bash -c "echo 'deepright:deepright' | chpasswd" 2>&1 | Out-Null
        & wsl.exe -d $DISTRO_NAME -u root -- usermod -aG sudo deepright 2>&1 | Out-Null
        & wsl.exe -d $DISTRO_NAME -u root -- bash -c "echo 'deepright ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/deepright && chmod 440 /etc/sudoers.d/deepright" 2>&1 | Out-Null
        & wsl.exe -d $DISTRO_NAME -u root -- bash -c "printf '[user]\ndefault=deepright\n' > /etc/wsl.conf" 2>&1 | Out-Null
        & wsl.exe --shutdown 2>&1 | Out-Null
        Start-Sleep -Seconds 3
        L_OK "User deepright created"
    } else {
        L_OK "User deepright already exists"
        & wsl.exe -d $DISTRO_NAME -u root -- bash -c "printf '[user]\ndefault=deepright\n' > /etc/wsl.conf" 2>&1 | Out-Null
        L_OK "Default user set to deepright"
    }

} else {
    L_Info "deepright not found, performing full install..."

    L_Info "Cleaning stale registrations..."
    Nuke-Distro -Name $DISTRO_NAME -VhdPath $WSL_VHD_PATH
    & wsl.exe --unregister Ubuntu 2>&1 | Out-Null
    Start-Sleep -Seconds 2
    L_OK "Cleanup done"

    $rootfsFile = $null

    L_Info "Method 1: wsl --install -d Ubuntu..."
    $i1 = & wsl.exe --install -d Ubuntu --no-launch 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "wsl --install -d Ubuntu: $i1" -Encoding UTF8
    Start-Sleep -Seconds 10

    $ubuntuOk = Test-DistroExists -N "Ubuntu"
    if ($ubuntuOk) {
        L_OK "Ubuntu installed via Method 1"
        $TEMP_TAR = "C:\Temp\deepright-ubuntu.tar"
        if (-not (Test-Path "C:\Temp")) { New-Item -ItemType Directory -Path "C:\Temp" -Force | Out-Null }
        L_Info "Exporting Ubuntu..."
        & wsl.exe --export Ubuntu $TEMP_TAR 2>&1 | Out-Null
        if (Test-Path $TEMP_TAR) {
            L_OK "Ubuntu exported"
            & wsl.exe --unregister Ubuntu 2>&1 | Out-Null
            Start-Sleep -Seconds 2
            $rootfsFile = $TEMP_TAR
        } else {
            L_Warn "Export failed, trying Method 2"
        }
    } else {
        L_Warn "Method 1 failed, trying Method 2"
    }

    if (-not $rootfsFile) {
        L_Info "Method 2: Direct rootfs download..."
        $rootfsUrl = Get-RootfsUrl
        $dlDir = "C:\Temp"
        if (-not (Test-Path $dlDir)) { New-Item -ItemType Directory -Path $dlDir -Force | Out-Null }
        $rootfsFile = Join-Path $dlDir "ubuntu-rootfs.tar.gz"

        L_Info "URL: $rootfsUrl"
        try {
            [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
            $wc = New-Object System.Net.WebClient
            $wc.DownloadFile($rootfsUrl, $rootfsFile)
        } catch {
            L_Err "Download error: $_"; exit 1
        }
        if (-not (Test-Path $rootfsFile)) { L_Err "Download failed"; exit 1 }
        L_OK "Rootfs downloaded ($([math]::Round((Get-Item $rootfsFile).Length/1MB,1)) MB)"
    }

    L_Info "Final cleanup before import..."
    & wsl.exe --shutdown 2>&1 | Out-Null
    Start-Sleep -Seconds 2
    & wsl.exe --unregister $DISTRO_NAME 2>&1 | Out-Null
    Start-Sleep -Seconds 1
    if (Test-Path $WSL_VHD_PATH) {
        Get-ChildItem -Path $WSL_VHD_PATH -Recurse -Force | Remove-Item -Force -EA SilentlyContinue
    } else {
        New-Item -ItemType Directory -Path $WSL_VHD_PATH -Force | Out-Null
    }

    $rtar = (Get-Item $rootfsFile).FullName
    L_Info "Importing: $rtar -> $WSL_VHD_PATH"
    $imp = & wsl.exe --import $DISTRO_NAME $WSL_VHD_PATH $rtar 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "wsl --import: $imp" -Encoding UTF8

    Start-Sleep -Seconds 3
    $verifyOut = & wsl.exe -d $DISTRO_NAME -u root -- echo "imported_ok" 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "verify: $verifyOut" -Encoding UTF8

    if ($verifyOut -notmatch "imported_ok") {
        L_Err "Import failed. WSL output:"
        L_Info $imp
        L_Info "Verify output: $verifyOut"
        exit 1
    }
    L_OK "$DISTRO_NAME imported successfully"

    Remove-Item $rootfsFile -Force -EA SilentlyContinue

    L_Info "Creating user deepright..."
    & wsl.exe -d $DISTRO_NAME -u root -- useradd -m -s /bin/bash deepright 2>&1 | Out-Null
    & wsl.exe -d $DISTRO_NAME -u root -- bash -c "echo 'deepright:deepright' | chpasswd" 2>&1 | Out-Null
    & wsl.exe -d $DISTRO_NAME -u root -- usermod -aG sudo deepright 2>&1 | Out-Null
    L_OK "User deepright created"

    L_Info "Configuring passwordless sudo..."
    & wsl.exe -d $DISTRO_NAME -u root -- bash -c "echo 'deepright ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/deepright && chmod 440 /etc/sudoers.d/deepright" 2>&1 | Out-Null
    L_OK "Passwordless sudo configured"

    L_Info "Setting default user to deepright..."
    & wsl.exe -d $DISTRO_NAME -u root -- bash -c "printf '[user]\ndefault=deepright\n' > /etc/wsl.conf" 2>&1 | Out-Null
    L_OK "Default user configured"

    L_Info "Restarting WSL..."
    & wsl.exe --shutdown 2>&1 | Out-Null
    Start-Sleep -Seconds 3
    L_OK "WSL restarted"
}

L_Step "Step 5/7: Verify mirrored networking"
L_OK ".wslconfig already configured with networkingMode=mirrored"

L_Step "Step 6/7: Install tools (git, npm, python3)"

$needGit   = -not (Test-WslTool "git")
$needPy    = -not (Test-WslTool "python3")
$needNode  = -not (Test-WslTool "node")
$needBwrap = -not (Test-WslTool "bwrap")

if (-not ($needGit -or $needPy -or $needNode -or $needBwrap)) {
    L_OK "All tools already installed"
} else {
    L_Info "Updating apt..."
    & wsl.exe -d $DISTRO_NAME -- bash -c "sudo DEBIAN_FRONTEND=noninteractive apt-get update -qq" 2>&1 | Out-Null
    L_OK "apt updated"

    if ($needGit -or $needPy -or $needBwrap) {
        L_Info "Installing git, python3, pip, curl, build-essential, bubblewrap..."
        $ar = & wsl.exe -d $DISTRO_NAME -- bash -c "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y git python3 python3-pip curl build-essential bubblewrap 2>&1" | Out-String
        Add-Content -Path $LOG_FILE -Value "apt: $ar" -Encoding UTF8
        if ($LASTEXITCODE -eq 0) { L_OK "git, python3, pip, curl, build-essential, bubblewrap installed" } else { L_Warn "Some packages may have failed" }
    }

    if ($needNode) {
        L_Info "Installing Node.js 20.x LTS (npm)..."
        $nr = & wsl.exe -d $DISTRO_NAME -- bash -c "curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash - 2>&1 && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs 2>&1" | Out-String
        Add-Content -Path $LOG_FILE -Value "node: $nr" -Encoding UTF8
        if ($LASTEXITCODE -eq 0) {
            L_OK "Node.js 20.x LTS + npm installed"
        } else {
            L_Warn "NodeSource failed, trying system packages..."
            & wsl.exe -d $DISTRO_NAME -- bash -c "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs npm 2>&1" | Out-Null
            if ($LASTEXITCODE -eq 0) { L_OK "Node.js + npm (system) installed" } else { L_Err "Node.js install failed" }
        }
    }
}

L_Info "Verifying tools..."
$gitV   = (& wsl.exe -d $DISTRO_NAME -- bash -c "git --version 2>&1" | Out-String).Trim()
$nodeV  = (& wsl.exe -d $DISTRO_NAME -- bash -c "node --version 2>&1" | Out-String).Trim()
$npmV   = (& wsl.exe -d $DISTRO_NAME -- bash -c "npm --version 2>&1" | Out-String).Trim()
$pyV    = (& wsl.exe -d $DISTRO_NAME -- bash -c "python3 --version 2>&1" | Out-String).Trim()
$bwrapV = (& wsl.exe -d $DISTRO_NAME -- bash -c "bwrap --version 2>&1 | head -n 1" | Out-String).Trim()
L_OK "git:      $gitV"
L_OK "node:     $nodeV"
L_OK "npm:      $npmV"
L_OK "python3:  $pyV"
L_OK "bwrap:    $bwrapV"
Assert-WSLArchitecture

L_Step "Step 7/7: Copy build dir to WSL /app/"

$WSL_APP_TARGET = "/app"

if (Test-Path $BUILD_DIR) {
    $fs = Get-ChildItem -Path $BUILD_DIR -Recurse -File -EA SilentlyContinue
    if ($fs.Count -eq 0) {
        L_Warn "build dir is empty, skipping"
    } else {
        L_Info "Copying $($fs.Count) files..."
        $wp = WslPath -P $BUILD_DIR
        & wsl.exe -d $DISTRO_NAME -u root -- bash -c "mkdir -p ${WSL_APP_TARGET} 2>/dev/null; cp -r '${wp}'/* ${WSL_APP_TARGET}/ 2>/dev/null; chown -R deepright:deepright ${WSL_APP_TARGET}/; chmod -R u+rw ${WSL_APP_TARGET}/" 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            L_OK "Build artifacts copied to ${WSL_APP_TARGET}/"
            $cl = & wsl.exe -d $DISTRO_NAME -- bash -c "ls -la ${WSL_APP_TARGET}/" 2>&1 | Out-String
            Write-Host $cl -F DarkGray
        } else {
            L_Warn "Build artifact copy may have partially failed"
        }
    }
} else {
    L_Warn "build dir not found: $BUILD_DIR"
}

L_Info "Creating WSL start wrapper script..."
& wsl.exe -d $DISTRO_NAME -u deepright -- bash -c "printf '#!/bin/bash`nTERM=xterm-256color setsid /app/integration start`n' > /home/deepright/start-deepright.sh"
& wsl.exe -d $DISTRO_NAME -u deepright -- chmod +x /home/deepright/start-deepright.sh
L_OK "Wrapper script: /home/deepright/start-deepright.sh"

L_Step "Writing sentinel files"

Add-Content -Path $LOG_FILE -Value "DEBUG LOCAL_SENTINEL_DIR  = $LOCAL_SENTINEL_DIR" -Encoding UTF8
Add-Content -Path $LOG_FILE -Value "DEBUG LOCAL_SENTINEL_FILE = $LOCAL_SENTINEL_FILE" -Encoding UTF8

try {
    $sentinelDir = [System.IO.Path]::GetFullPath($LOCAL_SENTINEL_DIR)
    Add-Content -Path $LOG_FILE -Value "DEBUG Resolved dir = $sentinelDir" -Encoding UTF8

    if (-not (Test-Path $sentinelDir)) {
        $null = New-Item -ItemType Directory -Path $sentinelDir -Force
        Add-Content -Path $LOG_FILE -Value "DEBUG New-Item exit: dir now exists = $(Test-Path $sentinelDir)" -Encoding UTF8
        L_Info "Created directory: $sentinelDir"
    } else {
        L_Info "Directory already exists: $sentinelDir"
    }

    [System.IO.File]::WriteAllText($LOCAL_SENTINEL_FILE, "deepright", [System.Text.Encoding]::ASCII)
    Add-Content -Path $LOG_FILE -Value "DEBUG WriteAllText completed" -Encoding UTF8

    if (Test-Path $LOCAL_SENTINEL_FILE) {
        $content = Get-Content $LOCAL_SENTINEL_FILE -Raw
        Add-Content -Path $LOG_FILE -Value "DEBUG Sentinel file content: [$content]" -Encoding UTF8
        L_OK "Local sentinel OK: $LOCAL_SENTINEL_FILE"
    } else {
        Add-Content -Path $LOG_FILE -Value "DEBUG Test-Path returned FALSE for: $LOCAL_SENTINEL_FILE" -Encoding UTF8
        L_Err "Local sentinel NOT created: $LOCAL_SENTINEL_FILE"
        exit 1
    }
} catch {
    Add-Content -Path $LOG_FILE -Value "DEBUG EXCEPTION: $_" -Encoding UTF8
    L_Err "Local sentinel exception: $_"
    exit 1
}

try {
    $touchOut = & wsl.exe -d $DISTRO_NAME -u deepright -- touch "$WSL_SENTINEL" 2>&1 | Out-String
    Add-Content -Path $LOG_FILE -Value "DEBUG WSL touch output: $touchOut" -Encoding UTF8
    $null = & wsl.exe -d $DISTRO_NAME -u deepright -- test -f "$WSL_SENTINEL" 2>&1
    Add-Content -Path $LOG_FILE -Value "DEBUG WSL test-f exit: $LASTEXITCODE" -Encoding UTF8
    if ($LASTEXITCODE -eq 0) {
        L_OK "WSL sentinel OK: $WSL_SENTINEL"
    } else {
        L_Err "WSL sentinel NOT created: $WSL_SENTINEL"
        Add-Content -Path $LOG_FILE -Value "DEBUG WSL sentinel failed, continuing anyway" -Encoding UTF8
    }
} catch {
    Add-Content -Path $LOG_FILE -Value "DEBUG WSL sentinel exception: $_" -Encoding UTF8
    L_Warn "WSL sentinel write threw exception, continuing"
}

}  # end of needsFullInstall

if (-not $gitV)   { $gitV   = (& wsl.exe -d $DISTRO_NAME -- bash -c "git --version 2>&1" | Out-String).Trim() }
if (-not $nodeV)  { $nodeV  = (& wsl.exe -d $DISTRO_NAME -- bash -c "node --version 2>&1" | Out-String).Trim() }
if (-not $npmV)   { $npmV   = (& wsl.exe -d $DISTRO_NAME -- bash -c "npm --version 2>&1" | Out-String).Trim() }
if (-not $pyV)    { $pyV    = (& wsl.exe -d $DISTRO_NAME -- bash -c "python3 --version 2>&1" | Out-String).Trim() }
if (-not $bwrapV) { $bwrapV = (& wsl.exe -d $DISTRO_NAME -- bash -c "bwrap --version 2>&1 | head -n 1" | Out-String).Trim() }

L_Step "Installation Complete"
Write-Host ""
Write-Host "  ================================================" -F Cyan
Write-Host "    Deepright WSL2 Environment Ready" -F Green
Write-Host "  ================================================" -F Cyan
Write-Host ""
Write-Host "  Distro:  $DISTRO_NAME" -F White
Write-Host "  User:    deepright" -F White
Write-Host "  Pass:    deepright" -F White
Write-Host "  VHD:     $WSL_VHD_PATH" -F White
Write-Host "  Network: mirrored" -F White
Write-Host ""
Write-Host "  Tools:" -F White
Write-Host "    git:      $gitV" -F Gray
Write-Host "    node:     $nodeV" -F Gray
Write-Host "    npm:      $npmV" -F Gray
Write-Host "    python3:  $pyV" -F Gray
Write-Host "    bwrap:    $bwrapV" -F Gray
Write-Host ""
Write-Host "  Enter WSL: wsl -d $DISTRO_NAME" -F Yellow
Write-Host "  Log file:  $LOG_FILE" -F Gray
Write-Host ""

if ($SkipLaunch) {
    L_Info "SkipLaunch enabled, not starting integration"
    exit 0
}

L_Step "Starting integration service"
Write-Host ""
& wsl.exe -d $DISTRO_NAME -- bash /home/deepright/start-deepright.sh 2>&1 | Write-Host
Write-Host ""
L_OK "Integration started"
__DEEPRIGHT_WSL2_POWERSHELL__

  sed -i.bak \
    -e "s|__DEEPRIGHT_EXPECTED_UNAME__|$expected_uname|g" \
    -e "s|__DEEPRIGHT_PACKAGE_LABEL__|$package_label|g" \
    "$out_path"
  rm -f "$out_path.bak"
}

write_windows_wsl2_launcher_cmd() {
  out_path="$1"

  cat > "$out_path" <<'EOF'
@echo off
setlocal enabledelayedexpansion

echo ============================================
echo    Deepright WSL2 Installer
echo ============================================
echo.

:: Check admin
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo Requesting administrator privileges...
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

echo Running as Administrator
echo.

:: Run PowerShell script
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0install.ps1"
set "EXIT_CODE=%ERRORLEVEL%"

echo.
echo ============================================
if "%EXIT_CODE%"=="0" (
    echo Installation finished. Press any key to exit.
) else (
    echo Installation failed with exit code %EXIT_CODE%. Press any key to exit.
)
echo ============================================
pause >nul
exit /b %EXIT_CODE%
EOF
}

write_windows_wsl2_start_bat() {
  out_path="$1"

  cat > "$out_path" <<'EOF'
@echo off

echo ============================================
echo    Deepright - Start Integration
echo ============================================
echo.

:: Check if deepright distro is reachable
wsl -d deepright -- echo "ok" 2>nul >nul
if %errorlevel% neq 0 (
    echo [X] deepright distro not found or not running.
    echo.
    echo Please run install.bat first to set up the environment.
    echo ============================================
    pause
    exit /b 1
)

echo [i] Starting integration in deepright...
echo.

:: Run synchronously so browser has time to open.
:: The wrapper script sets TERM and uses setsid (required for browser auto-open).
wsl -d deepright -- /home/deepright/start-deepright.sh
set "EXIT_CODE=%ERRORLEVEL%"
timeout /t 3 >nul

if "%EXIT_CODE%"=="0" (
    echo [OK] Integration started (browser should open automatically).
) else (
    echo [X] Integration failed to start. Exit code: %EXIT_CODE%
)
echo ============================================
pause >nul
exit /b %EXIT_CODE%
EOF
}

package_windows_wsl2_launcher() {
  target_name="$1"
  target_release_dir="$RELEASE_DIR/linux/$target_name"
  expected_uname=""
  package_label=""

  case "$target_name" in
    x86)
      expected_uname="x86_64"
      package_label="linux/x86"
      ;;
    arm)
      expected_uname="aarch64"
      package_label="linux/arm"
      ;;
    *)
      echo "unknown linux target for windows wsl2 launcher: $target_name" >&2
      exit 1
      ;;
  esac

  echo "-> packaging Windows WSL2 launcher (linux/$target_name)"
  rm -f "$target_release_dir/install-wsl2.ps1" "$target_release_dir/install-wsl2.cmd"
  write_windows_wsl2_launcher_ps1 "$target_release_dir/install.ps1" "$expected_uname" "$package_label"
  write_windows_wsl2_launcher_cmd "$target_release_dir/install.bat"
  write_windows_wsl2_start_bat "$target_release_dir/start.bat"
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
