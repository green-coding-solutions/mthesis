#!/bin/sh
set -eu

WORKDIR="/tmp/java-build/n-body"

mkdir -p "$WORKDIR"
cp /tmp/repo/benchmarks/java/n-body/Main.java "$WORKDIR"/

cd "$WORKDIR"
javac -d . -cp . Main.java

# Prefer G1 when available (as in Oracle GraalVM builds), fall back to serial.
if ! native-image --silent --gc=G1 -cp . -O3 -march=native Main -o nbody.graalvmaot-4.graalvmaot_run; then
  native-image --silent --gc=serial -cp . -O3 -march=native Main -o nbody.graalvmaot-4.graalvmaot_run
fi
