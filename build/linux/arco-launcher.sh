#!/bin/bash
set -e

SYSTEM_BIN="/usr/local/bin/arco-bin"
USER_DIR="$HOME/.local/share/arco"
USER_BIN="$USER_DIR/arco"

mkdir -p "$USER_DIR"

# Only copy on first run - self-update handles all subsequent updates
if [ ! -f "$USER_BIN" ]; then
    cp "$SYSTEM_BIN" "$USER_BIN"
    chmod +x "$USER_BIN"
fi

exec "$USER_BIN" "$@"
