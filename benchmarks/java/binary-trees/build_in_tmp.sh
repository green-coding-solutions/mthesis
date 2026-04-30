#!/bin/sh
set -eu

WORKDIR="/tmp/java-build/binary-trees"

mkdir -p "$WORKDIR"
cp /tmp/repo/benchmarks/java/binary-trees/Main.java "$WORKDIR"/

cd "$WORKDIR"
javac -d . -cp . Main.java

# Prefer G1 when available (as in Oracle GraalVM builds), fall back to serial.
if ! native-image --silent --gc=G1 -cp . -O3 -march=native Main -o binarytrees.graalvmaot-7.graalvmaot_run; then
  native-image --silent --gc=serial -cp . -O3 -march=native Main -o binarytrees.graalvmaot-7.graalvmaot_run
fi
