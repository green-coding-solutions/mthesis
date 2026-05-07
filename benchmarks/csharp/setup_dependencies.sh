#!/bin/sh
set -eu

OUT_DIR="/tmp/csharp-deps"
MARKER_FILE="$OUT_DIR/.ready"

if [ -f "$MARKER_FILE" ]; then
  exit 0
fi

if ! command -v apt-get >/dev/null 2>&1; then
  echo "[ERROR] Command not found 'apt-get'" >&2
  exit 127
fi

mkdir -p "$OUT_DIR"

apt-get update
DEBIAN_FRONTEND=noninteractive \
apt-get install -y --no-install-recommends clang zlib1g-dev
rm -rf /var/lib/apt/lists/*

touch "$MARKER_FILE"
