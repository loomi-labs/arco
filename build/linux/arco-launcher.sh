#!/bin/bash
set -e

SYSTEM_BIN="/usr/local/bin/arco-bin"
USER_DIR="$HOME/.local/share/arco"
USER_BIN="$USER_DIR/arco"

mkdir -p "$USER_DIR"

# Only copy on first run - self-update handles all subsequent updates
if [ ! -f "$USER_BIN" ]; then
    tmp="$(mktemp "$USER_DIR/.arco.XXXXXX")"
    install -m 0755 "$SYSTEM_BIN" "$tmp"
    mv -f "$tmp" "$USER_BIN"
fi

exec "$USER_BIN" "$@"
