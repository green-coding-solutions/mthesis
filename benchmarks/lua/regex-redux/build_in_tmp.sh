#!/bin/sh
set -eu

# regexredux.lua requires rex_pcre2 from lrexlib-pcre2.
if ! lua -e 'require("rex_pcre2")' >/dev/null 2>&1; then
  if command -v apk >/dev/null 2>&1; then
    apk add --no-cache gcc musl-dev make pcre2-dev
  fi
  luarocks install lrexlib-pcre2
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
