#!/bin/sh
set -eu

WORKDIR="/tmp/java-build/k-nucleotide"
LIBDIR="/tmp/java-libs"
FASTUTIL_JAR="/opt/src/java-libs/fastutil-8.3.1.jar"
FALLBACK_JAR="$LIBDIR/fastutil-8.3.1.jar"

mkdir -p "$WORKDIR" "$LIBDIR"
cp /tmp/repo/benchmarks/java/k-nucleotide/Main.java "$WORKDIR"/

if [ -f "$FASTUTIL_JAR" ]; then
  JAR="$FASTUTIL_JAR"
else
  JAR="$FALLBACK_JAR"
  if [ ! -f "$JAR" ]; then
    if command -v curl >/dev/null 2>&1; then
      curl -fsSL -o "$JAR" "https://repo1.maven.org/maven2/it/unimi/dsi/fastutil/8.3.1/fastutil-8.3.1.jar"
    elif command -v wget >/dev/null 2>&1; then
      wget -q -O "$JAR" "https://repo1.maven.org/maven2/it/unimi/dsi/fastutil/8.3.1/fastutil-8.3.1.jar"
    else
      echo "Neither curl nor wget is available to download fastutil jar." >&2
      exit 1
    fi
  fi
fi

cd "$WORKDIR"
CP=".:$JAR"
javac -d . -cp "$CP" Main.java

# Prefer G1 when available (as in Oracle GraalVM builds), fall back to serial.
if ! native-image --silent --gc=G1 -cp "$CP" -O3 -march=native Main -o knucleotide.graalvmaot_run; then
  native-image --silent --gc=serial -cp "$CP" -O3 -march=native Main -o knucleotide.graalvmaot_run
fi
