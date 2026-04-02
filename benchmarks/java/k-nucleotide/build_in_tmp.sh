#!/bin/sh
set -eu

WORKDIR="/tmp/java-build/k-nucleotide"
LIBDIR="/tmp/java-libs"
JAR="$LIBDIR/fastutil-8.3.1.jar"

mkdir -p "$WORKDIR"
cp /tmp/repo/benchmarks/java/k-nucleotide/Main.java "$WORKDIR"/

if [ ! -f "$JAR" ]; then
  echo "Missing fastutil jar at $JAR. Expected setup-commands to download it." >&2
  exit 1
fi

cd "$WORKDIR"
CP=".:$JAR"
javac -d . -cp "$CP" Main.java

# Prefer G1 when available (as in Oracle GraalVM builds), fall back to serial.
if ! native-image --silent --gc=G1 -cp "$CP" -O3 -march=native Main -o knucleotide.graalvmaot_run; then
  native-image --silent --gc=serial -cp "$CP" -O3 -march=native Main -o knucleotide.graalvmaot_run
fi
