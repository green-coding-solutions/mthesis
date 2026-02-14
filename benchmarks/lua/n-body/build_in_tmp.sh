#!/bin/sh
set -eu

WORKDIR=/tmp/lua-build/n-body
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

if [ -d /tmp/repo/Include/lua ]; then
  cp -L /tmp/repo/Include/lua/* .
fi

cp -L /tmp/repo/benchmarks/lua/n-body/main.lua nbody.lua-2.lua
luac -o nbody.lua-2.lua_run nbody.lua-2.lua
