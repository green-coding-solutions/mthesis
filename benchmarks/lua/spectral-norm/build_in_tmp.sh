#!/bin/sh
set -eu

WORKDIR=/tmp/lua-build/spectral-norm
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

if [ -d /tmp/repo/Include/lua ]; then
  cp -L /tmp/repo/Include/lua/* .
fi

cp -L /tmp/repo/benchmarks/lua/spectral-norm/main.lua spectralnorm.lua-7.lua
luac -o spectralnorm.lua-7.lua_run spectralnorm.lua-7.lua
