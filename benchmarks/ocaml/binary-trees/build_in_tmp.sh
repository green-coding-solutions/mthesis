#!/bin/sh
set -eu

SRC_ML="/tmp/ocaml-binary-trees.ml"
BIN="/tmp/ocaml-binary-trees"

cp /tmp/repo/benchmarks/ocaml/binary-trees/main.ml "$SRC_ML"

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

rm -f "$SRC_ML" /tmp/ocaml-binary-trees.cmi /tmp/ocaml-binary-trees.cmx /tmp/ocaml-binary-trees.o
