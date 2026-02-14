#!/bin/sh
set -eu

WORKDIR="/tmp/java-build/regex-redux"

mkdir -p "$WORKDIR" "$WORKDIR/Include/java"
cp /tmp/repo/benchmarks/java/regex-redux/Main.java "$WORKDIR"/

cd "$WORKDIR"
javac -d . -cp . --source-path Include/java Main.java

# Prefer G1 when available (as in Oracle GraalVM builds), fall back to serial.
if ! native-image --silent --gc=G1 -cp . -O3 -march=native Main -o regexredux.graalvmaot-3.graalvmaot_run; then
  native-image --silent --gc=serial -cp . -O3 -march=native Main -o regexredux.graalvmaot-3.graalvmaot_run
fi
