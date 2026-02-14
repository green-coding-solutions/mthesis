#!/bin/sh
set -eu

WORKDIR="/tmp/java-build/binary-trees"

mkdir -p "$WORKDIR"
cp /tmp/repo/benchmarks/java/binary-trees/Main.java "$WORKDIR"/

cd "$WORKDIR"
javac -d . -cp . Main.java
native-image --silent -cp . -O3 -march=native Main -o binarytrees.graalvmaot-7.graalvmaot_run
