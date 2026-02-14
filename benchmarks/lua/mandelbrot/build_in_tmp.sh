#!/bin/sh
set -eu

WORKDIR=/tmp/lua-build/mandelbrot
rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

if [ -d /tmp/repo/Include/lua ]; then
  cp -L /tmp/repo/Include/lua/* .
fi

cp -L /tmp/repo/benchmarks/lua/mandelbrot/main.lua mandelbrot.lua-6.lua
luac -o mandelbrot.lua-6.lua_run mandelbrot.lua-6.lua
