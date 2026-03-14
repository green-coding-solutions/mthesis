#!/bin/sh
set -eu

SRC_ML="/tmp/ocaml-k-nucleotide.ml"
BIN="/tmp/ocaml-k-nucleotide"

cp /tmp/repo/benchmarks/ocaml/k-nucleotide/main.ml "$SRC_ML"

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

rm -f "$SRC_ML" /tmp/ocaml-k-nucleotide.cmi /tmp/ocaml-k-nucleotide.cmx /tmp/ocaml-k-nucleotide.o
