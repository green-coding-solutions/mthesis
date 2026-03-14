#!/bin/sh
set -eu

SRC_ML="/tmp/ocaml-fannkuch-redux.ml"
BIN="/tmp/ocaml-fannkuch-redux"

cp /tmp/repo/benchmarks/ocaml/fannkuch-redux/main.ml "$SRC_ML"

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

rm -f "$SRC_ML" /tmp/ocaml-fannkuch-redux.cmi /tmp/ocaml-fannkuch-redux.cmx /tmp/ocaml-fannkuch-redux.o
