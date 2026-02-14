#!/bin/sh
set -eu

WORKDIR=/tmp/lua-build/fannkuch-redux
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

if [ -d /tmp/repo/Include/lua ]; then
  cp -L /tmp/repo/Include/lua/* .
fi

cp -L /tmp/repo/benchmarks/lua/fannkuch-redux/main.lua fannkuchredux.lua
luac -o fannkuchredux.lua_run fannkuchredux.lua
