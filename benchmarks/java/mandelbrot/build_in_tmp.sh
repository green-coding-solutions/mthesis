#!/bin/sh
set -eu

WORKDIR="/tmp/java-build/mandelbrot"

mkdir -p "$WORKDIR"
cp /tmp/repo/benchmarks/java/mandelbrot/Main.java "$WORKDIR"/

cd "$WORKDIR"
javac -d . -cp . Main.java

# Prefer G1 when available (as in some GraalVM builds), fall back to serial.
if ! native-image --silent --gc=G1 -cp . -O3 -march=native Main -o mandelbrot.graalvmaot-6.graalvmaot_run; then
  native-image --silent --gc=serial -cp . -O3 -march=native Main -o mandelbrot.graalvmaot-6.graalvmaot_run
fi
