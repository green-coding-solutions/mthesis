#!/bin/sh
set -eu

if ! lua -e 'require("rex_pcre2")' >/dev/null 2>&1; then
  echo "Missing rex_pcre2. Expected setup-commands to install lrexlib-pcre2." >&2
  exit 1
fi

WORKDIR=/tmp/lua-build/regex-redux
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

if [ -d /tmp/repo/Include/lua ]; then
  cp -L /tmp/repo/Include/lua/* .
fi

cp -L /tmp/repo/benchmarks/lua/regex-redux/main.lua regexredux.lua
luac -o regexredux.lua_run regexredux.lua
