#!/bin/sh
set -eu

SRC_ML="/tmp/ocaml-n-body.ml"
BIN="/tmp/ocaml-n-body"

cp /tmp/repo/benchmarks/ocaml/n-body/main.ml "$SRC_ML"

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

rm -f "$SRC_ML" /tmp/ocaml-n-body.cmi /tmp/ocaml-n-body.cmx /tmp/ocaml-n-body.o
