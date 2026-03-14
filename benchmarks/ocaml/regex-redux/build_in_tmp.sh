#!/bin/sh
set -eu

SRC_ML="/tmp/ocaml-regex-redux.ml"
BIN="/tmp/ocaml-regex-redux"

cp /tmp/repo/benchmarks/ocaml/regex-redux/main.ml "$SRC_ML"

if ! opam exec -- ocamlfind query re >/dev/null 2>&1; then
  echo "Missing OCaml dependency: re (expected to be installed at container startup)." >&2
  exit 2
fi

RE_PACKAGE="re"
if opam exec -- ocamlfind query re.pcre >/dev/null 2>&1; then
  RE_PACKAGE="re.pcre"
fi

opam exec -- ocamlfind ocamlopt \
  -noassert \
  -unsafe \
  -nodynlink \
  -inline 100 \
  -O3 \
  -package "$RE_PACKAGE" \
  -package unix \
  -linkpkg \
  -ccopt -fPIC \
  -ccopt -march=ivybridge \
  "$SRC_ML" \
  -o "$BIN"

rm -f "$SRC_ML" /tmp/ocaml-regex-redux.cmi /tmp/ocaml-regex-redux.cmx /tmp/ocaml-regex-redux.o
