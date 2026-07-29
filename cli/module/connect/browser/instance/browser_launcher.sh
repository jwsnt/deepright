#!/bin/sh
# DeepRight browser launcher v1

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)

BROWSER_BIN="${DEEPRIGHT_BROWSER_BIN:-}"
if [ -z "$BROWSER_BIN" ]; then
  for candidate in "$SCRIPT_DIR/browser" "$SCRIPT_DIR/../browser"; do
    if [ -x "$candidate" ]; then
      BROWSER_BIN="$candidate"
      break
    fi
  done
fi

if [ -z "$BROWSER_BIN" ]; then
  printf '{"status":1,"message":"browser binary not found beside launcher"}\n'
  exit 1
fi

exec "$BROWSER_BIN" __wsl-instance acquire "$@"
