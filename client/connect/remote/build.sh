#!/bin/sh

set -eu

GO_BIN="${GO_BIN:-go}"
GO_RELEASE_LDFLAGS="${GO_RELEASE_LDFLAGS:--s -w}"

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
CONNECT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
RELEASE_DIR="$SCRIPT_DIR/release"

echo "Building remote release artifacts..."
rm -rf "$RELEASE_DIR"
mkdir -p "$RELEASE_DIR/plugins"

echo "-> building remote"
(
  cd "$CONNECT_DIR"
  "$GO_BIN" build -trimpath -ldflags "$GO_RELEASE_LDFLAGS" -o "$RELEASE_DIR/remote" ./remote
)

echo "-> removing runtime state from release"
rm -rf \
  "$RELEASE_DIR/.remote" \
  "$RELEASE_DIR/remote.json" \
  "$RELEASE_DIR/remote.log"

echo "Build completed:"
echo "  $RELEASE_DIR/remote"
echo "  $RELEASE_DIR/plugins"
