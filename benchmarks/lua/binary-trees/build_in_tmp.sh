#!/bin/sh
set -eu

WORKDIR=/tmp/lua-build/binary-trees
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

if [ -d /tmp/repo/Include/lua ]; then
  cp -L /tmp/repo/Include/lua/* .
fi

cp -L /tmp/repo/benchmarks/lua/binary-trees/main.lua binarytrees.lua-4.lua
luac -o binarytrees.lua-4.lua_run binarytrees.lua-4.lua
