#!/bin/sh
set -eu

SRC_ML="/tmp/ocaml-spectral-norm.ml"
BIN="/tmp/ocaml-spectral-norm"

cp /tmp/repo/benchmarks/ocaml/spectral-norm/main.ml "$SRC_ML"

opam exec -- ocamlopt \
  -noassert \
  -unsafe \
  -nodynlink \
  -inline 100 \
  -O3 \
  -ccopt -fPIC \
  -ccopt -march=ivybridge \
  "$SRC_ML" \
  -o "$BIN"

rm -f "$SRC_ML" /tmp/ocaml-spectral-norm.cmi /tmp/ocaml-spectral-norm.cmx /tmp/ocaml-spectral-norm.o
