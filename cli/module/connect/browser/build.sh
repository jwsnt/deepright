#!/bin/sh

set -eu

GO_BIN="${GO_BIN:-go}"
GO_RELEASE_LDFLAGS="${GO_RELEASE_LDFLAGS:--s -w}"

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
CONNECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
RELEASE_DIR="$SCRIPT_DIR/release"

echo "Building browser release artifacts..."
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR"

echo "-> building browser"
(
  cd "$CONNECT_DIR"
  "$GO_BIN" build -trimpath -ldflags "$GO_RELEASE_LDFLAGS" -o "$RELEASE_DIR/browser" ./browser
)

echo "-> packaging browser launcher"
cp "$SCRIPT_DIR/instance/browser_launcher.sh" "$RELEASE_DIR/browser_launcher.sh"
chmod 755 "$RELEASE_DIR/browser_launcher.sh"

echo "-> removing runtime state from release"
rm -rf \
  "$RELEASE_DIR/.browser_playwright" \
  "$RELEASE_DIR/playwright/driver" \
  "$RELEASE_DIR/browser_instance.json" \
  "$RELEASE_DIR/browser.log"

echo "Build completed:"
echo "  $RELEASE_DIR/browser"
