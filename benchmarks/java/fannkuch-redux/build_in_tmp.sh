#!/bin/sh
set -eu

WORKDIR="/tmp/java-build/fannkuch-redux"

mkdir -p "$WORKDIR"
cp /tmp/repo/benchmarks/java/fannkuch-redux/Main.java "$WORKDIR"/

cd "$WORKDIR"
javac -d . -cp . Main.java

# Prefer G1 when available (as in Oracle GraalVM builds), fall back to serial.
if ! native-image --silent --gc=G1 -cp . -O3 -march=native Main -o fannkuchredux.graalvmaot_run; then
  native-image --silent --gc=serial -cp . -O3 -march=native Main -o fannkuchredux.graalvmaot_run
fi
