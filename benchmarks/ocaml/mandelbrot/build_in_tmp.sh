#!/bin/sh
set -eu

SRC_ML="/tmp/ocaml-mandelbrot.ml"
BIN="/tmp/ocaml-mandelbrot"

cp /tmp/repo/benchmarks/ocaml/mandelbrot/main.ml "$SRC_ML"

opam exec -- ocamlopt \
  -noassert \
  -unsafe \
  -nodynlink \
  -inline 100 \
  -O3 \
  -I +unix unix.cmxa \
  -ccopt -fPIC \
  -ccopt -march=ivybridge \
  "$SRC_ML" \
  -o "$BIN"

rm -f "$SRC_ML" /tmp/ocaml-mandelbrot.cmi /tmp/ocaml-mandelbrot.cmx /tmp/ocaml-mandelbrot.o
