#!/bin/sh
set -eu

WORKDIR=/tmp/lua-build/k-nucleotide
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

if [ -d /tmp/repo/Include/lua ]; then
  cp -L /tmp/repo/Include/lua/* .
fi

cp -L /tmp/repo/benchmarks/lua/k-nucleotide/main.lua knucleotide.lua-2.lua
luac -o knucleotide.lua-2.lua_run knucleotide.lua-2.lua
