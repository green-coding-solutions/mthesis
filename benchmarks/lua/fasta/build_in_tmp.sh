#!/bin/sh
set -eu

WORKDIR=/tmp/lua-build/fasta
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

if [ -d /tmp/repo/Include/lua ]; then
  cp -L /tmp/repo/Include/lua/* .
fi

cp -L /tmp/repo/benchmarks/lua/fasta/main.lua fasta.lua-2.lua
luac -o fasta.lua-2.lua_run fasta.lua-2.lua
